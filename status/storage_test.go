package status

import (
	"os"
	"testing"
	"time"
)

func TestNewStorage(t *testing.T) {
	tmpFile := tempDBPath(t)
	t.Cleanup(func() { os.Remove(tmpFile) })

	storage, err := NewStorage(tmpFile)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	// Verify database file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestNewStorageInvalidPath(t *testing.T) {
	// Test with invalid path (directory that doesn't exist)
	_, err := NewStorage("/nonexistent/path/db.sqlite")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestRecordStatus(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	err := storage.RecordStatus("http://example.com", true, "")
	if err != nil {
		t.Fatalf("failed to record status: %v", err)
	}

	err = storage.RecordStatus("http://example.com", false, "connection refused")
	if err != nil {
		t.Fatalf("failed to record status: %v", err)
	}
}

func TestGetLastStatus(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"

	// Test with no records
	record, err := storage.GetLastStatus(serviceURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record != nil {
		t.Error("expected nil record for empty database")
	}

	// Insert some records
	if err := storage.RecordStatus(serviceURL, true, ""); err != nil {
		t.Fatalf("failed to record status: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	if err := storage.RecordStatus(serviceURL, false, "timeout"); err != nil {
		t.Fatalf("failed to record status: %v", err)
	}

	// Get last status
	record, err = storage.GetLastStatus(serviceURL)
	if err != nil {
		t.Fatalf("failed to get last status: %v", err)
	}
	if record == nil {
		t.Fatal("expected record, got nil")
	}
	if record.IsUp {
		t.Error("expected last status to be down (false)")
	}
	if record.Message != "timeout" {
		t.Errorf("expected message 'timeout', got '%s'", record.Message)
	}
}

func TestGetStatusHistory(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"

	// Insert multiple records
	for i := 0; i < 5; i++ {
		if err := storage.RecordStatus(serviceURL, i%2 == 0, ""); err != nil {
			t.Fatalf("failed to record status: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Get history with limit
	records, err := storage.GetStatusHistory(serviceURL, 3)
	if err != nil {
		t.Fatalf("failed to get status history: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}

	// Get all history
	records, err = storage.GetStatusHistory(serviceURL, 10)
	if err != nil {
		t.Fatalf("failed to get status history: %v", err)
	}
	if len(records) != 5 {
		t.Errorf("expected 5 records, got %d", len(records))
	}
}

func TestRecordAlert(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	alert := Alert{
		ServiceURL: "http://example.com",
		AlertType:  AlertTypeDown,
		Message:    "Service is down",
		Timestamp:  time.Now(),
	}

	err := storage.RecordAlert(alert)
	if err != nil {
		t.Fatalf("failed to record alert: %v", err)
	}
}

func TestGetLastAlert(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"

	// Test with no alerts
	record, err := storage.GetLastAlert(serviceURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record != nil {
		t.Error("expected nil record for empty database")
	}

	// Insert alerts
	if err := storage.RecordAlert(Alert{
		ServiceURL: serviceURL,
		AlertType:  AlertTypeDown,
		Message:    "down",
		Timestamp:  time.Now(),
	}); err != nil {
		t.Fatalf("failed to record alert: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := storage.RecordAlert(Alert{
		ServiceURL: serviceURL,
		AlertType:  AlertTypeRecovery,
		Message:    "recovered",
		Timestamp:  time.Now(),
	}); err != nil {
		t.Fatalf("failed to record alert: %v", err)
	}

	// Get last alert
	record, err = storage.GetLastAlert(serviceURL)
	if err != nil {
		t.Fatalf("failed to get last alert: %v", err)
	}
	if record == nil {
		t.Fatal("expected record, got nil")
	}
	if record.AlertType != string(AlertTypeRecovery) {
		t.Errorf("expected alert type 'recovery', got '%s'", record.AlertType)
	}
}

func TestGetRecentAlerts(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	// Insert alerts for different services
	services := []string{"http://a.com", "http://b.com", "http://c.com"}
	for _, svc := range services {
		if err := storage.RecordAlert(Alert{
			ServiceURL: svc,
			AlertType:  AlertTypeDown,
			Message:    "down",
			Timestamp:  time.Now(),
		}); err != nil {
			t.Fatalf("failed to record alert: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Get recent alerts
	records, err := storage.GetRecentAlerts(2)
	if err != nil {
		t.Fatalf("failed to get recent alerts: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	// Most recent should be first
	if records[0].ServiceURL != "http://c.com" {
		t.Errorf("expected most recent alert for 'http://c.com', got '%s'", records[0].ServiceURL)
	}
}

func TestUpdateServiceState(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"

	// Initial insert
	err := storage.UpdateServiceState(serviceURL, true)
	if err != nil {
		t.Fatalf("failed to update service state: %v", err)
	}

	// Verify state
	isUp, _, _, err := storage.GetServiceState(serviceURL)
	if err != nil {
		t.Fatalf("failed to get service state: %v", err)
	}
	if !isUp {
		t.Error("expected service to be up")
	}

	// Update state
	err = storage.UpdateServiceState(serviceURL, false)
	if err != nil {
		t.Fatalf("failed to update service state: %v", err)
	}

	// Verify updated state
	isUp, _, _, err = storage.GetServiceState(serviceURL)
	if err != nil {
		t.Fatalf("failed to get service state: %v", err)
	}
	if isUp {
		t.Error("expected service to be down")
	}
}

func TestUpdateLastAlert(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"

	// First create a service state entry
	err := storage.UpdateServiceState(serviceURL, true)
	if err != nil {
		t.Fatalf("failed to update service state: %v", err)
	}

	// Update last alert time
	alertTime := time.Now()
	err = storage.UpdateLastAlert(serviceURL, alertTime)
	if err != nil {
		t.Fatalf("failed to update last alert: %v", err)
	}

	// Verify
	_, _, lastAlert, err := storage.GetServiceState(serviceURL)
	if err != nil {
		t.Fatalf("failed to get service state: %v", err)
	}
	if lastAlert == nil {
		t.Error("expected last alert time to be set")
	}
}

func TestGetServiceStateNotFound(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	isUp, lastChecked, lastAlert, err := storage.GetServiceState("http://nonexistent.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isUp {
		t.Error("expected isUp to be false for nonexistent service")
	}
	if !lastChecked.IsZero() {
		t.Error("expected lastChecked to be zero for nonexistent service")
	}
	if lastAlert != nil {
		t.Error("expected lastAlert to be nil for nonexistent service")
	}
}

func TestGetAllServiceStates(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	// Insert multiple service states
	if err := storage.UpdateServiceState("http://a.com", true); err != nil {
		t.Fatalf("failed to update service state: %v", err)
	}
	if err := storage.UpdateServiceState("http://b.com", false); err != nil {
		t.Fatalf("failed to update service state: %v", err)
	}
	if err := storage.UpdateServiceState("http://c.com", true); err != nil {
		t.Fatalf("failed to update service state: %v", err)
	}

	states, err := storage.GetAllServiceStates()
	if err != nil {
		t.Fatalf("failed to get all service states: %v", err)
	}

	if len(states) != 3 {
		t.Errorf("expected 3 states, got %d", len(states))
	}
	if !states["http://a.com"] {
		t.Error("expected http://a.com to be up")
	}
	if states["http://b.com"] {
		t.Error("expected http://b.com to be down")
	}
	if !states["http://c.com"] {
		t.Error("expected http://c.com to be up")
	}
}

func TestPruneOldRecords(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"

	// Insert records
	for i := 0; i < 5; i++ {
		if err := storage.RecordStatus(serviceURL, true, ""); err != nil {
			t.Fatalf("failed to record status: %v", err)
		}
	}

	// Verify records exist
	records, err := storage.GetStatusHistory(serviceURL, 10)
	if err != nil {
		t.Fatalf("failed to get status history: %v", err)
	}
	if len(records) != 5 {
		t.Fatalf("expected 5 records, got %d", len(records))
	}

	// Prune records older than 0 (all records)
	deleted, err := storage.PruneOldRecords(0)
	if err != nil {
		t.Fatalf("failed to prune records: %v", err)
	}
	if deleted != 5 {
		t.Errorf("expected 5 deleted, got %d", deleted)
	}

	// Verify records are gone
	records, err = storage.GetStatusHistory(serviceURL, 10)
	if err != nil {
		t.Fatalf("failed to get status history: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records after prune, got %d", len(records))
	}
}

func TestBoolToInt(t *testing.T) {
	tt := []struct {
		name     string
		input    bool
		expected int
	}{
		{name: "true", input: true, expected: 1},
		{name: "false", input: false, expected: 0},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if got := boolToInt(tc.input); got != tc.expected {
				t.Errorf("boolToInt(%v) = %d, want %d", tc.input, got, tc.expected)
			}
		})
	}
}

func TestParseDBTime(t *testing.T) {
	tt := []struct {
		name     string
		input    string
		wantZero bool
	}{
		{name: "RFC3339", input: "2024-01-15T10:30:00Z", wantZero: false},
		{name: "RFC3339Nano", input: "2024-01-15T10:30:00.123456789Z", wantZero: false},
		{name: "simple datetime", input: "2024-01-15 10:30:00", wantZero: false},
		{name: "RFC3339 variant", input: "2024-01-15T10:30:00Z", wantZero: false},
		{name: "empty string", input: "", wantZero: true},
		{name: "invalid format", input: "not-a-date", wantZero: true},
		{name: "partial date", input: "2024-01-15", wantZero: true},
		{name: "invalid month", input: "2024-13-15T10:30:00Z", wantZero: true},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDBTime(tc.input)
			if tc.wantZero && !got.IsZero() {
				t.Errorf("parseDBTime(%q) = %v, want zero time", tc.input, got)
			}
			if !tc.wantZero && got.IsZero() {
				t.Errorf("parseDBTime(%q) returned zero time, want non-zero", tc.input)
			}
		})
	}
}

func TestParseDBTimeValues(t *testing.T) {
	// Test specific parsed values for valid inputs
	tt := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantHour  int
		wantMin   int
	}{
		{
			name:      "RFC3339",
			input:     "2024-01-15T10:30:00Z",
			wantYear:  2024,
			wantMonth: time.January,
			wantDay:   15,
			wantHour:  10,
			wantMin:   30,
		},
		{
			name:      "simple datetime",
			input:     "2024-06-20 14:45:30",
			wantYear:  2024,
			wantMonth: time.June,
			wantDay:   20,
			wantHour:  14,
			wantMin:   45,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDBTime(tc.input)
			if got.Year() != tc.wantYear {
				t.Errorf("year = %d, want %d", got.Year(), tc.wantYear)
			}
			if got.Month() != tc.wantMonth {
				t.Errorf("month = %v, want %v", got.Month(), tc.wantMonth)
			}
			if got.Day() != tc.wantDay {
				t.Errorf("day = %d, want %d", got.Day(), tc.wantDay)
			}
			if got.Hour() != tc.wantHour {
				t.Errorf("hour = %d, want %d", got.Hour(), tc.wantHour)
			}
			if got.Minute() != tc.wantMin {
				t.Errorf("minute = %d, want %d", got.Minute(), tc.wantMin)
			}
		})
	}
}

func TestStartIncident(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	err := storage.StartIncident("http://example.com", "Example Service", "connection timeout")
	if err != nil {
		t.Fatalf("failed to start incident: %v", err)
	}

	// Verify incident was created
	incident, err := storage.GetOngoingIncident("http://example.com")
	if err != nil {
		t.Fatalf("failed to get ongoing incident: %v", err)
	}
	if incident == nil {
		t.Fatal("expected ongoing incident, got nil")
	}
	if incident.ServiceName != "Example Service" {
		t.Errorf("expected service name 'Example Service', got '%s'", incident.ServiceName)
	}
	if incident.Message != "connection timeout" {
		t.Errorf("expected message 'connection timeout', got '%s'", incident.Message)
	}
	if !incident.IsOngoing {
		t.Error("expected incident to be ongoing")
	}
}

func TestEndIncident(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"

	// Start an incident
	err := storage.StartIncident(serviceURL, "Example Service", "error")
	if err != nil {
		t.Fatalf("failed to start incident: %v", err)
	}

	// End the incident
	time.Sleep(10 * time.Millisecond) // Ensure some duration
	err = storage.EndIncident(serviceURL)
	if err != nil {
		t.Fatalf("failed to end incident: %v", err)
	}

	// Verify incident was ended
	incident, err := storage.GetOngoingIncident(serviceURL)
	if err != nil {
		t.Fatalf("failed to get ongoing incident: %v", err)
	}
	if incident != nil {
		t.Error("expected no ongoing incident after ending")
	}
}

func TestGetOngoingIncidentNotFound(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	incident, err := storage.GetOngoingIncident("http://nonexistent.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if incident != nil {
		t.Error("expected nil for nonexistent ongoing incident")
	}
}

func TestGetRecentResolvedIncidents(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	// Create and resolve multiple incidents
	// Note: SQLite stores time with second precision, so we can't test sub-second durations
	services := []string{"http://a.com", "http://b.com", "http://c.com"}
	for _, svc := range services {
		if err := storage.StartIncident(svc, svc, "error"); err != nil {
			t.Fatalf("failed to start incident: %v", err)
		}
		time.Sleep(5 * time.Millisecond) // Small delay between operations
		if err := storage.EndIncident(svc); err != nil {
			t.Fatalf("failed to end incident: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Get recent incidents
	incidents, err := storage.GetRecentResolvedIncidents(10, 0)
	if err != nil {
		t.Fatalf("failed to get recent incidents: %v", err)
	}
	if len(incidents) != 3 {
		t.Errorf("expected 3 incidents, got %d", len(incidents))
	}

	// Most recently ended should be first
	if incidents[0].ServiceURL != "http://c.com" {
		t.Errorf("expected most recent incident for 'http://c.com', got '%s'", incidents[0].ServiceURL)
	}

	// Verify incidents are resolved (not ongoing)
	for _, inc := range incidents {
		if inc.IsOngoing {
			t.Error("expected resolved incident, got ongoing")
		}
	}
}

func TestGetRecentResolvedIncidentsMinDuration(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	// Note: SQLite stores time with second precision
	// We test the filtering logic by creating incidents with different durations
	// manually in the database rather than relying on real-time sleeps

	// Create incidents directly with specific timestamps
	now := time.Now().UTC()

	// Short incident: 10 seconds
	_, err := storage.db.Exec(
		"INSERT INTO incidents (service_url, service_name, started_at, ended_at, message) VALUES (?, ?, ?, ?, ?)",
		"http://short.com", "Short", now.Add(-20*time.Second), now.Add(-10*time.Second), "error",
	)
	if err != nil {
		t.Fatalf("failed to insert short incident: %v", err)
	}

	// Long incident: 60 seconds
	_, err = storage.db.Exec(
		"INSERT INTO incidents (service_url, service_name, started_at, ended_at, message) VALUES (?, ?, ?, ?, ?)",
		"http://long.com", "Long", now.Add(-120*time.Second), now.Add(-60*time.Second), "error",
	)
	if err != nil {
		t.Fatalf("failed to insert long incident: %v", err)
	}

	// Get incidents with minimum duration filter (30 seconds)
	incidents, err := storage.GetRecentResolvedIncidents(10, 30*time.Second)
	if err != nil {
		t.Fatalf("failed to get recent incidents: %v", err)
	}

	// Should only return the longer incident (60 seconds)
	if len(incidents) != 1 {
		t.Errorf("expected 1 incident with min duration filter, got %d", len(incidents))
	}
	if len(incidents) > 0 && incidents[0].ServiceURL != "http://long.com" {
		t.Errorf("expected 'http://long.com', got '%s'", incidents[0].ServiceURL)
	}
}

func TestGetRecentResolvedIncidentsLimit(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	// Create 5 resolved incidents
	for i := 0; i < 5; i++ {
		svc := "http://example.com"
		if err := storage.StartIncident(svc, "Example", "error"); err != nil {
			t.Fatalf("failed to start incident: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
		if err := storage.EndIncident(svc); err != nil {
			t.Fatalf("failed to end incident: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Get with limit
	incidents, err := storage.GetRecentResolvedIncidents(3, 0)
	if err != nil {
		t.Fatalf("failed to get recent incidents: %v", err)
	}
	if len(incidents) != 3 {
		t.Errorf("expected 3 incidents with limit, got %d", len(incidents))
	}
}

func TestGetServiceIncidents(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"

	// Create multiple incidents for the same service
	for i := 0; i < 3; i++ {
		if err := storage.StartIncident(serviceURL, "Example", "error"); err != nil {
			t.Fatalf("failed to start incident: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
		if err := storage.EndIncident(serviceURL); err != nil {
			t.Fatalf("failed to end incident: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Create an incident for a different service
	if err := storage.StartIncident("http://other.com", "Other", "error"); err != nil {
		t.Fatalf("failed to start incident: %v", err)
	}

	// Get incidents for the specific service
	incidents, err := storage.GetServiceIncidents(serviceURL, 10)
	if err != nil {
		t.Fatalf("failed to get service incidents: %v", err)
	}
	if len(incidents) != 3 {
		t.Errorf("expected 3 incidents for service, got %d", len(incidents))
	}

	for _, inc := range incidents {
		if inc.ServiceURL != serviceURL {
			t.Errorf("expected service URL '%s', got '%s'", serviceURL, inc.ServiceURL)
		}
	}
}

func TestRecordStatusTransitionUpToDown(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"
	serviceName := "Example Service"

	// First establish an up state using RecordStatusTransition
	// (it now updates state atomically)
	_, err := storage.RecordStatusTransition(serviceURL, serviceName, true, "")
	if err != nil {
		t.Fatalf("failed to set initial state: %v", err)
	}

	// Transition to down
	changed, err := storage.RecordStatusTransition(serviceURL, serviceName, false, "connection refused")
	if err != nil {
		t.Fatalf("failed to record transition: %v", err)
	}
	if !changed {
		t.Error("expected transition to be recorded")
	}

	// Verify incident was started
	incident, err := storage.GetOngoingIncident(serviceURL)
	if err != nil {
		t.Fatalf("failed to get ongoing incident: %v", err)
	}
	if incident == nil {
		t.Fatal("expected ongoing incident after up->down transition")
	}
	if incident.Message != "connection refused" {
		t.Errorf("expected message 'connection refused', got '%s'", incident.Message)
	}

	// Verify state was updated to down
	isUp, _, _, err := storage.GetServiceState(serviceURL)
	if err != nil {
		t.Fatalf("failed to get service state: %v", err)
	}
	if isUp {
		t.Error("expected service state to be down after transition")
	}
}

func TestRecordStatusTransitionDownToUp(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"
	serviceName := "Example Service"

	// First establish an "up" state, then transition to "down", then back to "up"
	// This simulates: initial up -> down (incident starts) -> up (incident ends)
	// Note: RecordStatusTransition now updates state atomically, so no separate
	// UpdateServiceState calls are needed between transitions.

	// 1. First check - service starts as up (sets initial state)
	changed, err := storage.RecordStatusTransition(serviceURL, serviceName, true, "")
	if err != nil {
		t.Fatalf("failed to record initial state: %v", err)
	}
	if changed {
		t.Error("expected no incident on first up state")
	}

	// 2. Transition to down - should start an incident and update state
	changed, err = storage.RecordStatusTransition(serviceURL, serviceName, false, "error")
	if err != nil {
		t.Fatalf("failed to record down transition: %v", err)
	}
	if !changed {
		t.Error("expected down transition to be recorded")
	}

	time.Sleep(10 * time.Millisecond)

	// 3. Transition back to up - should end the incident and update state
	changed, err = storage.RecordStatusTransition(serviceURL, serviceName, true, "")
	if err != nil {
		t.Fatalf("failed to record up transition: %v", err)
	}
	if !changed {
		t.Error("expected up transition to be recorded")
	}

	// Verify incident was ended
	incident, err := storage.GetOngoingIncident(serviceURL)
	if err != nil {
		t.Fatalf("failed to get ongoing incident: %v", err)
	}
	if incident != nil {
		t.Error("expected no ongoing incident after down->up transition")
	}

	// Verify incident is in history
	incidents, err := storage.GetRecentResolvedIncidents(10, 0)
	if err != nil {
		t.Fatalf("failed to get recent incidents: %v", err)
	}
	if len(incidents) != 1 {
		t.Errorf("expected 1 resolved incident, got %d", len(incidents))
	}

	// Verify final state is up
	isUp, _, _, err := storage.GetServiceState(serviceURL)
	if err != nil {
		t.Fatalf("failed to get service state: %v", err)
	}
	if !isUp {
		t.Error("expected service state to be up after recovery")
	}
}

func TestRecordStatusTransitionNoChange(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"
	serviceName := "Example Service"

	// First establish an up state using RecordStatusTransition
	_, err := storage.RecordStatusTransition(serviceURL, serviceName, true, "")
	if err != nil {
		t.Fatalf("failed to set initial state: %v", err)
	}

	// Record same state again (no transition)
	changed, err := storage.RecordStatusTransition(serviceURL, serviceName, true, "")
	if err != nil {
		t.Fatalf("failed to record transition: %v", err)
	}
	if changed {
		t.Error("expected no change for same state")
	}
}

func TestRecordStatusTransitionFirstCheck(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"
	serviceName := "Example Service"

	// First check - service is down
	changed, err := storage.RecordStatusTransition(serviceURL, serviceName, false, "initial failure")
	if err != nil {
		t.Fatalf("failed to record transition: %v", err)
	}
	if !changed {
		t.Error("expected incident to be recorded on first check when down")
	}

	// Verify incident was started
	incident, err := storage.GetOngoingIncident(serviceURL)
	if err != nil {
		t.Fatalf("failed to get ongoing incident: %v", err)
	}
	if incident == nil {
		t.Fatal("expected ongoing incident for first check when down")
	}
}

func TestRecordStatusTransitionFirstCheckUp(t *testing.T) {
	storage := setupTestStorage(t)
	defer storage.Close()

	serviceURL := "http://example.com"
	serviceName := "Example Service"

	// First check - service is up (no incident should be recorded)
	changed, err := storage.RecordStatusTransition(serviceURL, serviceName, true, "")
	if err != nil {
		t.Fatalf("failed to record transition: %v", err)
	}
	if changed {
		t.Error("expected no incident on first check when up")
	}

	// Verify no incident was started
	incident, err := storage.GetOngoingIncident(serviceURL)
	if err != nil {
		t.Fatalf("failed to get ongoing incident: %v", err)
	}
	if incident != nil {
		t.Error("expected no ongoing incident when first check is up")
	}
}

// Helper functions

func tempDBPath(t *testing.T) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-storage-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	return tmpFile.Name()
}

func setupTestStorage(t *testing.T) *Storage {
	t.Helper()
	tmpFile := tempDBPath(t)
	t.Cleanup(func() { os.Remove(tmpFile) })

	storage, err := NewStorage(tmpFile)
	if err != nil {
		t.Fatalf("failed to create test storage: %v", err)
	}
	return storage
}
