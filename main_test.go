package main

import (
	"testing"

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
			config := Config{Services: tc.services}
			pingers, err := config.CreateFactories()

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

	config := Config{Services: services}
	pingers, err := config.CreateFactories()
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

func TestLoadConfigurationSuccess(t *testing.T) {
	config, err := LoadConfiguration("config.json")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(config.Services) == 0 {
		t.Error("expected at least one service in config")
	}
}

func TestLoadConfigurationFileNotFound(t *testing.T) {
	_, err := LoadConfiguration("nonexistent.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestConfigStoragePath(t *testing.T) {
	// Test that the config struct has the StoragePath field
	config := Config{
		Services:      []status.Service{},
		StoragePath:   "test.db",
		AlertCooldown: 300,
	}

	if config.StoragePath != "test.db" {
		t.Errorf("expected storage_path to be 'test.db', got %s", config.StoragePath)
	}
}

func TestConfigStoragePathEmpty(t *testing.T) {
	// Test that empty storage_path is valid (storage disabled by default)
	config := Config{
		Services:    []status.Service{},
		StoragePath: "",
	}

	if config.StoragePath != "" {
		t.Errorf("expected storage_path to be empty, got %s", config.StoragePath)
	}
}
