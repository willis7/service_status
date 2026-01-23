package status

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatusHTML(t *testing.T) {
	tt := []struct {
		name     string
		input    string
		expected template.HTML
	}{
		{name: "danger status", input: "danger", expected: "danger"},
		{name: "degraded status", input: "degraded", expected: "degraded"},
		{name: "success status", input: "success", expected: "success"},
		{name: "maintenance status", input: "maintenance", expected: "maintenance"},
		{name: "empty status defaults to success", input: "", expected: "success"},
		{name: "unknown status defaults to success", input: "unknown", expected: "success"},
		{name: "warning defaults to success", input: "warning", expected: "success"},
		{name: "arbitrary string defaults to success", input: "<script>alert('xss')</script>", expected: "success"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if got := StatusHTML(tc.input); got != tc.expected {
				t.Errorf("StatusHTML(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestAPIStatus(t *testing.T) {
	tt := []struct {
		name               string
		page               Page
		expectedStatus     string
		expectedServices   int
		expectedMaintMsg   string
		expectedHTTPStatus int
	}{
		{
			name: "all services up",
			page: Page{
				Title:  "Test Status",
				Status: "success",
				Up:     []string{"service1", "service2"},
			},
			expectedStatus:     "OK",
			expectedServices:   2,
			expectedHTTPStatus: http.StatusOK,
		},
		{
			name: "some services down",
			page: Page{
				Title:  "Test Status",
				Status: "danger",
				Up:     []string{"service1"},
				Down:   map[string]int{"service2": 60},
			},
			expectedStatus:     "DOWN",
			expectedServices:   2,
			expectedHTTPStatus: http.StatusOK,
		},
		{
			name: "some services degraded",
			page: Page{
				Title:    "Test Status",
				Status:   "degraded",
				Up:       []string{"service1"},
				Degraded: map[string]int{"service2": 30},
			},
			expectedStatus:     "DEGRADED",
			expectedServices:   2,
			expectedHTTPStatus: http.StatusOK,
		},
		{
			name: "maintenance mode",
			page: Page{
				Title:              "Test Status",
				Status:             "maintenance",
				Up:                 []string{"service1", "service2"},
				MaintenanceMessage: "Scheduled maintenance in progress",
			},
			expectedStatus:     "MAINTENANCE",
			expectedServices:   2,
			expectedMaintMsg:   "Scheduled maintenance in progress",
			expectedHTTPStatus: http.StatusOK,
		},
		{
			name: "empty services",
			page: Page{
				Title:  "Test Status",
				Status: "success",
			},
			expectedStatus:     "OK",
			expectedServices:   0,
			expectedHTTPStatus: http.StatusOK,
		},
		{
			name: "mixed status with all types",
			page: Page{
				Title:    "Test Status",
				Status:   "danger",
				Up:       []string{"up-service"},
				Degraded: map[string]int{"degraded-service": 15},
				Down:     map[string]int{"down-service": 30},
			},
			expectedStatus:     "DOWN",
			expectedServices:   3,
			expectedHTTPStatus: http.StatusOK,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			handler := APIStatus(tc.page)
			req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.expectedHTTPStatus {
				t.Errorf("expected HTTP status %d, got %d", tc.expectedHTTPStatus, rec.Code)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			var response APIResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode JSON response: %v", err)
			}

			if response.OverallStatus != tc.expectedStatus {
				t.Errorf("expected overall status %q, got %q", tc.expectedStatus, response.OverallStatus)
			}

			if len(response.Services) != tc.expectedServices {
				t.Errorf("expected %d services, got %d", tc.expectedServices, len(response.Services))
			}

			if response.MaintenanceMessage != tc.expectedMaintMsg {
				t.Errorf("expected maintenance message %q, got %q", tc.expectedMaintMsg, response.MaintenanceMessage)
			}

			if response.Updated == "" {
				t.Error("expected Updated field to be set")
			}
			// Validate timestamp format (RFC 3339)
			if _, err := time.Parse(time.RFC3339, response.Updated); err != nil {
				t.Errorf("Updated field has invalid format: %s", response.Updated)
			}
		})
	}
}

func TestAPIStatusServiceStatuses(t *testing.T) {
	page := Page{
		Title:    "Test Status",
		Status:   "danger",
		Up:       []string{"healthy-service"},
		Degraded: map[string]int{"slow-service": 15},
		Down:     map[string]int{"broken-service": 60},
	}

	handler := APIStatus(page)
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var response APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	statusMap := make(map[string]string)
	for _, svc := range response.Services {
		statusMap[svc.Name] = svc.Status
	}

	if statusMap["healthy-service"] != "OK" {
		t.Errorf("expected healthy-service status OK, got %s", statusMap["healthy-service"])
	}
	if statusMap["slow-service"] != "DEGRADED" {
		t.Errorf("expected slow-service status DEGRADED, got %s", statusMap["slow-service"])
	}
	if statusMap["broken-service"] != "DOWN" {
		t.Errorf("expected broken-service status DOWN, got %s", statusMap["broken-service"])
	}
}

func TestAPIStatusMethodNotAllowed(t *testing.T) {
	page := Page{
		Title:  "Test Status",
		Status: "success",
		Up:     []string{"service1"},
	}

	handler := APIStatus(page)

	tt := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{name: "GET allowed", method: http.MethodGet, expectedStatus: http.StatusOK},
		{name: "HEAD allowed", method: http.MethodHead, expectedStatus: http.StatusOK},
		{name: "POST not allowed", method: http.MethodPost, expectedStatus: http.StatusMethodNotAllowed},
		{name: "PUT not allowed", method: http.MethodPut, expectedStatus: http.StatusMethodNotAllowed},
		{name: "DELETE not allowed", method: http.MethodDelete, expectedStatus: http.StatusMethodNotAllowed},
		{name: "PATCH not allowed", method: http.MethodPatch, expectedStatus: http.StatusMethodNotAllowed},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/status", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected HTTP status %d for %s, got %d", tc.expectedStatus, tc.method, rec.Code)
			}
		})
	}
}
