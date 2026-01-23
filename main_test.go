package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/willis7/service_status/config"
	"github.com/willis7/service_status/status"
)

func TestCreateFactories(t *testing.T) {
	tt := []struct {
		name     string
		services []status.Service
		expected int
		wantErr  bool
	}{
		{
			name:     "empty services",
			services: []status.Service{},
			expected: 0,
			wantErr:  false,
		},
		{
			name: "single ping service",
			services: []status.Service{
				{Type: "ping", URL: "http://example.com"},
			},
			expected: 1,
			wantErr:  false,
		},
		{
			name: "single grep service",
			services: []status.Service{
				{Type: "grep", URL: "http://example.com", Regex: "test"},
			},
			expected: 1,
			wantErr:  false,
		},
		{
			name: "multiple mixed services",
			services: []status.Service{
				{Type: "ping", URL: "http://example1.com"},
				{Type: "grep", URL: "http://example2.com", Regex: "pattern1"},
				{Type: "ping", URL: "http://example3.com"},
				{Type: "grep", URL: "http://example4.com", Regex: "pattern2"},
			},
			expected: 4,
			wantErr:  false,
		},
		{
			name: "single tcp service",
			services: []status.Service{
				{Type: "tcp", URL: "localhost", Port: "8080"},
			},
			expected: 1,
			wantErr:  false,
		},
		{
			name: "mixed services with tcp",
			services: []status.Service{
				{Type: "ping", URL: "http://example1.com"},
				{Type: "tcp", URL: "localhost", Port: "3306"},
				{Type: "grep", URL: "http://example2.com", Regex: "pattern"},
				{Type: "tcp", URL: "localhost", Port: "5432"},
			},
			expected: 4,
			wantErr:  false,
		},
		{
			name: "unknown service type is skipped",
			services: []status.Service{
				{Type: "ping", URL: "http://example1.com"},
				{Type: "unknown", URL: "http://example2.com"},
				{Type: "grep", URL: "http://example3.com", Regex: "test"},
			},
			expected: 2,
			wantErr:  false,
		},
		{
			name: "single icmp service",
			services: []status.Service{
				{Type: "icmp", URL: "8.8.8.8"},
			},
			expected: 1,
			wantErr:  false,
		},
		{
			name: "mixed services with icmp",
			services: []status.Service{
				{Type: "ping", URL: "http://example1.com"},
				{Type: "icmp", URL: "8.8.8.8"},
				{Type: "grep", URL: "http://example2.com", Regex: "pattern"},
				{Type: "tcp", URL: "localhost", Port: "8080"},
			},
			expected: 4,
			wantErr:  false,
		},
		{
			name: "single script service",
			services: []status.Service{
				{Type: "script", Command: "/path/to/script.sh"},
			},
			expected: 1,
			wantErr:  false,
		},
		{
			name: "mixed services with script",
			services: []status.Service{
				{Type: "ping", URL: "http://example1.com"},
				{Type: "script", Command: "echo hello"},
				{Type: "grep", URL: "http://example2.com", Regex: "pattern"},
			},
			expected: 3,
			wantErr:  false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{Services: tc.services}
			pingers, err := cfg.CreateFactories()

			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(pingers) != tc.expected {
				t.Errorf("expected %d pingers, got %d", tc.expected, len(pingers))
			}
		})
	}
}

func TestCreateFactoriesIteratesAllServices(t *testing.T) {
	// This test verifies that CreateFactories iterates through ALL services
	// and creates Pinger objects for each valid service type
	services := []status.Service{
		{Type: "ping", URL: "http://service1.com"},
		{Type: "ping", URL: "http://service2.com"},
		{Type: "grep", URL: "http://service3.com", Regex: "pattern"},
	}

	cfg := &config.Config{Services: services}
	pingers, err := cfg.CreateFactories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all services were processed
	if len(pingers) != 3 {
		t.Errorf("expected 3 pingers, got %d", len(pingers))
	}

	// Verify URLs are correct (ensuring iteration processed each service)
	expectedURLs := map[string]bool{
		"http://service1.com": false,
		"http://service2.com": false,
		"http://service3.com": false,
	}

	for _, pinger := range pingers {
		url := pinger.GetService().URL
		if _, exists := expectedURLs[url]; !exists {
			t.Errorf("unexpected URL in pingers: %s", url)
		}
		expectedURLs[url] = true
	}

	for url, found := range expectedURLs {
		if !found {
			t.Errorf("service with URL %s was not processed", url)
		}
	}
}

func TestLoadConfigSuccess(t *testing.T) {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Services) == 0 {
		t.Error("expected at least one service in config")
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := config.LoadConfig("nonexistent.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestConfigStoragePath(t *testing.T) {
	// Test that the config struct has the StoragePath field
	cfg := &config.Config{
		Services:      []status.Service{},
		StoragePath:   "test.db",
		AlertCooldown: 300,
	}

	if cfg.StoragePath != "test.db" {
		t.Errorf("expected storage_path to be 'test.db', got %s", cfg.StoragePath)
	}
}

func TestConfigStoragePathEmpty(t *testing.T) {
	// Test that empty storage_path is valid (storage disabled by default)
	cfg := &config.Config{
		Services:    []status.Service{},
		StoragePath: "",
	}

	if cfg.StoragePath != "" {
		t.Errorf("expected storage_path to be empty, got %s", cfg.StoragePath)
	}
}

func TestGetMaintenanceMessageFromInline(t *testing.T) {
	cfg := &config.Config{
		MaintenanceMessage: "System under maintenance for upgrade",
	}

	msg := cfg.GetMaintenanceMessage()
	if msg != "System under maintenance for upgrade" {
		t.Errorf("expected inline message, got %q", msg)
	}
}

func TestGetMaintenanceMessageFromFile(t *testing.T) {
	// Create a temporary maintenance file
	tmpDir := t.TempDir()
	maintenanceFile := filepath.Join(tmpDir, "maintenance.txt")
	err := os.WriteFile(maintenanceFile, []byte("Scheduled maintenance in progress"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &config.Config{
		MaintenanceFile: maintenanceFile,
	}

	msg := cfg.GetMaintenanceMessage()
	if msg != "Scheduled maintenance in progress" {
		t.Errorf("expected file message, got %q", msg)
	}
}

func TestGetMaintenanceMessageFileTakesPrecedence(t *testing.T) {
	// Create a temporary maintenance file
	tmpDir := t.TempDir()
	maintenanceFile := filepath.Join(tmpDir, "maintenance.txt")
	err := os.WriteFile(maintenanceFile, []byte("File message"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &config.Config{
		MaintenanceFile:    maintenanceFile,
		MaintenanceMessage: "Inline message",
	}

	msg := cfg.GetMaintenanceMessage()
	if msg != "File message" {
		t.Errorf("expected file message to take precedence, got %q", msg)
	}
}

func TestGetMaintenanceMessageEmptyFileUsesInline(t *testing.T) {
	// Create an empty temporary maintenance file
	tmpDir := t.TempDir()
	maintenanceFile := filepath.Join(tmpDir, "maintenance.txt")
	err := os.WriteFile(maintenanceFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &config.Config{
		MaintenanceFile:    maintenanceFile,
		MaintenanceMessage: "Inline message",
	}

	msg := cfg.GetMaintenanceMessage()
	if msg != "Inline message" {
		t.Errorf("expected inline message when file is empty, got %q", msg)
	}
}

func TestGetMaintenanceMessageNonexistentFileUsesInline(t *testing.T) {
	cfg := &config.Config{
		MaintenanceFile:    "/nonexistent/path/maintenance.txt",
		MaintenanceMessage: "Inline message",
	}

	msg := cfg.GetMaintenanceMessage()
	if msg != "Inline message" {
		t.Errorf("expected inline message when file doesn't exist, got %q", msg)
	}
}

func TestGetMaintenanceMessageNoMaintenance(t *testing.T) {
	cfg := &config.Config{}

	msg := cfg.GetMaintenanceMessage()
	if msg != "" {
		t.Errorf("expected empty message when no maintenance configured, got %q", msg)
	}
}

func TestGetMaintenanceMessageTrimsWhitespace(t *testing.T) {
	// Create a temporary maintenance file with whitespace
	tmpDir := t.TempDir()
	maintenanceFile := filepath.Join(tmpDir, "maintenance.txt")
	err := os.WriteFile(maintenanceFile, []byte("  Message with whitespace  \n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &config.Config{
		MaintenanceFile: maintenanceFile,
	}

	msg := cfg.GetMaintenanceMessage()
	if msg != "Message with whitespace" {
		t.Errorf("expected trimmed message, got %q", msg)
	}
}

func TestGetMaintenanceMessageWhitespaceOnlyFileUsesInline(t *testing.T) {
	// Create a temporary maintenance file with only whitespace
	tmpDir := t.TempDir()
	maintenanceFile := filepath.Join(tmpDir, "maintenance.txt")
	err := os.WriteFile(maintenanceFile, []byte("   \n\t  \n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := &config.Config{
		MaintenanceFile:    maintenanceFile,
		MaintenanceMessage: "Inline fallback",
	}

	msg := cfg.GetMaintenanceMessage()
	if msg != "Inline fallback" {
		t.Errorf("expected inline message when file has only whitespace, got %q", msg)
	}
}
