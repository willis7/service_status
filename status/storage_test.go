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
