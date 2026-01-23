package status

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockPinger is a simple mock for testing CheckAllServices
type mockPinger struct {
	service Service
	result  StatusResult
}

func (m *mockPinger) GetService() *Service {
	return &m.service
}

func (m *mockPinger) Status() error {
	return m.result.Err
}

func (m *mockPinger) StatusWithTiming() StatusResult {
	return m.result
}

func TestCheckAllServicesEmpty(t *testing.T) {
	up, degraded, down := CheckAllServices(nil, nil, nil, "")

	if len(up) != 0 {
		t.Errorf("expected 0 up services, got %d", len(up))
	}
	if len(degraded) != 0 {
		t.Errorf("expected 0 degraded services, got %d", len(degraded))
	}
	if len(down) != 0 {
		t.Errorf("expected 0 down services, got %d", len(down))
	}
}

func TestCheckAllServicesMaintenanceMode(t *testing.T) {
	services := []Pinger{
		&mockPinger{
			service: Service{URL: "http://service1.example.com", Name: "Service 1"},
			result:  StatusResult{Err: ErrServiceUnavailable, ResponseTime: 100 * time.Millisecond},
		},
		&mockPinger{
			service: Service{URL: "http://service2.example.com", Name: "Service 2"},
			result:  StatusResult{Err: nil, ResponseTime: 50 * time.Millisecond},
		},
	}

	up, degraded, down := CheckAllServices(services, nil, nil, "Scheduled maintenance")

	// In maintenance mode, all services should be marked as up regardless of their actual status
	if len(up) != 2 {
		t.Errorf("expected 2 up services in maintenance mode, got %d", len(up))
	}
	if len(degraded) != 0 {
		t.Errorf("expected 0 degraded services in maintenance mode, got %d", len(degraded))
	}
	if len(down) != 0 {
		t.Errorf("expected 0 down services in maintenance mode, got %d", len(down))
	}

	// Response times should be 0 in maintenance mode
	for _, svc := range up {
		if svc.ResponseTime != 0 {
			t.Errorf("expected 0 response time in maintenance mode, got %v", svc.ResponseTime)
		}
	}
}

func TestCheckAllServicesAllUp(t *testing.T) {
	services := []Pinger{
		&mockPinger{
			service: Service{URL: "http://service1.example.com", Name: "Service 1"},
			result:  StatusResult{Err: nil, ResponseTime: 50 * time.Millisecond},
		},
		&mockPinger{
			service: Service{URL: "http://service2.example.com", Name: "Service 2"},
			result:  StatusResult{Err: nil, ResponseTime: 30 * time.Millisecond},
		},
	}

	up, degraded, down := CheckAllServices(services, nil, nil, "")

	if len(up) != 2 {
		t.Errorf("expected 2 up services, got %d", len(up))
	}
	if len(degraded) != 0 {
		t.Errorf("expected 0 degraded services, got %d", len(degraded))
	}
	if len(down) != 0 {
		t.Errorf("expected 0 down services, got %d", len(down))
	}

	// Verify service info
	if up[0].Name != "Service 1" {
		t.Errorf("expected 'Service 1', got '%s'", up[0].Name)
	}
	if up[0].ResponseTime != 50*time.Millisecond {
		t.Errorf("expected 50ms response time, got %v", up[0].ResponseTime)
	}
}

func TestCheckAllServicesWithDownService(t *testing.T) {
	services := []Pinger{
		&mockPinger{
			service: Service{URL: "http://service1.example.com", Name: "Service 1"},
			result:  StatusResult{Err: nil, ResponseTime: 50 * time.Millisecond},
		},
		&mockPinger{
			service: Service{URL: "http://service2.example.com", Name: "Service 2"},
			result:  StatusResult{Err: ErrServiceUnavailable, ResponseTime: 100 * time.Millisecond},
		},
	}

	up, degraded, down := CheckAllServices(services, nil, nil, "")

	if len(up) != 1 {
		t.Errorf("expected 1 up service, got %d", len(up))
	}
	if len(degraded) != 0 {
		t.Errorf("expected 0 degraded services, got %d", len(degraded))
	}
	if len(down) != 1 {
		t.Errorf("expected 1 down service, got %d", len(down))
	}

	// Verify down service info
	outage, ok := down["Service 2"]
	if !ok {
		t.Error("expected 'Service 2' to be in down map")
	}
	if outage.Minutes != defaultOutageMinutes {
		t.Errorf("expected %d minutes, got %d", defaultOutageMinutes, outage.Minutes)
	}
	if outage.ResponseTime != 100*time.Millisecond {
		t.Errorf("expected 100ms response time, got %v", outage.ResponseTime)
	}
}

func TestCheckAllServicesWithDegradedService(t *testing.T) {
	services := []Pinger{
		&mockPinger{
			service: Service{URL: "http://service1.example.com", Name: "Service 1"},
			result:  StatusResult{Err: nil, ResponseTime: 50 * time.Millisecond},
		},
		&mockPinger{
			service: Service{URL: "http://service2.example.com", Name: "Service 2"},
			result:  StatusResult{Err: ErrServiceDegraded, ResponseTime: 200 * time.Millisecond},
		},
	}

	up, degraded, down := CheckAllServices(services, nil, nil, "")

	if len(up) != 1 {
		t.Errorf("expected 1 up service, got %d", len(up))
	}
	if len(degraded) != 1 {
		t.Errorf("expected 1 degraded service, got %d", len(degraded))
	}
	if len(down) != 0 {
		t.Errorf("expected 0 down services, got %d", len(down))
	}

	// Verify degraded service info
	outage, ok := degraded["Service 2"]
	if !ok {
		t.Error("expected 'Service 2' to be in degraded map")
	}
	if outage.Minutes != defaultOutageMinutes {
		t.Errorf("expected %d minutes, got %d", defaultOutageMinutes, outage.Minutes)
	}
	if outage.ResponseTime != 200*time.Millisecond {
		t.Errorf("expected 200ms response time, got %v", outage.ResponseTime)
	}
}

func TestCheckAllServicesMixedStates(t *testing.T) {
	services := []Pinger{
		&mockPinger{
			service: Service{URL: "http://up.example.com", Name: "Up Service"},
			result:  StatusResult{Err: nil, ResponseTime: 50 * time.Millisecond},
		},
		&mockPinger{
			service: Service{URL: "http://degraded.example.com", Name: "Degraded Service"},
			result:  StatusResult{Err: ErrServiceDegraded, ResponseTime: 200 * time.Millisecond},
		},
		&mockPinger{
			service: Service{URL: "http://down.example.com", Name: "Down Service"},
			result:  StatusResult{Err: ErrServiceUnavailable, ResponseTime: 100 * time.Millisecond},
		},
	}

	up, degraded, down := CheckAllServices(services, nil, nil, "")

	if len(up) != 1 {
		t.Errorf("expected 1 up service, got %d", len(up))
	}
	if len(degraded) != 1 {
		t.Errorf("expected 1 degraded service, got %d", len(degraded))
	}
	if len(down) != 1 {
		t.Errorf("expected 1 down service, got %d", len(down))
	}
}

func TestCheckAllServicesUsesDisplayName(t *testing.T) {
	// Test that services without explicit names use URL as display name
	services := []Pinger{
		&mockPinger{
			service: Service{URL: "http://example.com"},
			result:  StatusResult{Err: nil, ResponseTime: 50 * time.Millisecond},
		},
	}

	up, _, _ := CheckAllServices(services, nil, nil, "")

	if len(up) != 1 {
		t.Fatalf("expected 1 up service, got %d", len(up))
	}
	if up[0].Name != "http://example.com" {
		t.Errorf("expected URL as display name, got '%s'", up[0].Name)
	}
}

func TestCheckAllServicesWithNotifyManager(t *testing.T) {
	nm := NewNotificationManager(time.Minute)
	nm.AddNotifier(NewLogNotifier())

	services := []Pinger{
		&mockPinger{
			service: Service{URL: "http://service.example.com", Name: "Service"},
			result:  StatusResult{Err: nil, ResponseTime: 50 * time.Millisecond},
		},
	}

	// First call - should record initial state
	CheckAllServices(services, nil, nm, "")

	state := nm.GetServiceState()
	if !state["http://service.example.com"] {
		t.Error("expected service to be tracked as up")
	}
}

func TestDetermineOverallStatus(t *testing.T) {
	tt := []struct {
		name           string
		maintenanceMsg string
		degraded       map[string]OutageInfo
		down           map[string]OutageInfo
		want           string
	}{
		{
			name:           "maintenance mode",
			maintenanceMsg: "Scheduled maintenance",
			degraded:       make(map[string]OutageInfo),
			down:           make(map[string]OutageInfo),
			want:           "maintenance",
		},
		{
			name:           "danger when down",
			maintenanceMsg: "",
			degraded:       make(map[string]OutageInfo),
			down:           map[string]OutageInfo{"Service 1": {Minutes: 60}},
			want:           "danger",
		},
		{
			name:           "degraded status",
			maintenanceMsg: "",
			degraded:       map[string]OutageInfo{"Service 1": {Minutes: 60}},
			down:           make(map[string]OutageInfo),
			want:           "degraded",
		},
		{
			name:           "success when all up",
			maintenanceMsg: "",
			degraded:       make(map[string]OutageInfo),
			down:           make(map[string]OutageInfo),
			want:           "success",
		},
		{
			name:           "down takes precedence",
			maintenanceMsg: "",
			degraded:       map[string]OutageInfo{"Service 1": {Minutes: 60}},
			down:           map[string]OutageInfo{"Service 2": {Minutes: 60}},
			want:           "danger",
		},
		{
			name:           "maintenance takes precedence over all",
			maintenanceMsg: "Maintenance in progress",
			degraded:       map[string]OutageInfo{"Service 1": {Minutes: 60}},
			down:           map[string]OutageInfo{"Service 2": {Minutes: 60}},
			want:           "maintenance",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := DetermineOverallStatus(tc.maintenanceMsg, tc.degraded, tc.down)
			if got != tc.want {
				t.Errorf("DetermineOverallStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCheckAllServicesIntegration(t *testing.T) {
	// Create real HTTP test servers for integration testing
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts2.Close()

	services := []Pinger{
		&Ping{Service: Service{URL: ts1.URL, Name: "Healthy Service"}},
		&Ping{Service: Service{URL: ts2.URL, Name: "Unhealthy Service"}},
	}

	up, degraded, down := CheckAllServices(services, nil, nil, "")

	if len(up) != 1 {
		t.Errorf("expected 1 up service, got %d", len(up))
	}
	if len(degraded) != 0 {
		t.Errorf("expected 0 degraded services, got %d", len(degraded))
	}
	if len(down) != 1 {
		t.Errorf("expected 1 down service, got %d", len(down))
	}

	// Verify the correct services are categorized correctly
	if up[0].Name != "Healthy Service" {
		t.Errorf("expected 'Healthy Service' to be up, got '%s'", up[0].Name)
	}
	if _, ok := down["Unhealthy Service"]; !ok {
		t.Error("expected 'Unhealthy Service' to be down")
	}

	// Verify response times are populated for real HTTP calls
	if up[0].ResponseTime == 0 {
		t.Error("expected non-zero response time for healthy service")
	}
	outage := down["Unhealthy Service"]
	if outage.ResponseTime == 0 {
		t.Error("expected non-zero response time for unhealthy service")
	}
}
