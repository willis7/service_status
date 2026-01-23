package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfigJSON(t *testing.T) {
	// Use the existing config.json in the project root
	cfg, err := LoadConfig("../config.json")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Services) == 0 {
		t.Error("expected at least one service in config")
	}
}

func TestLoadConfigYAML(t *testing.T) {
	// Create a temporary YAML config file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
services:
  - type: ping
    url: http://example.com
    name: Example
  - type: grep
    url: http://test.com
    regex: "hello"
    name: Test Site

alert_cooldown: 120
storage_path: test.db
incident_history_limit: 5
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test yaml: %v", err)
	}

	cfg, err := LoadConfig(yamlPath)
	if err != nil {
		t.Fatalf("failed to load yaml config: %v", err)
	}

	if len(cfg.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(cfg.Services))
	}

	if cfg.Services[0].Type != "ping" {
		t.Errorf("expected first service type 'ping', got %s", cfg.Services[0].Type)
	}

	if cfg.Services[0].Name != "Example" {
		t.Errorf("expected first service name 'Example', got %s", cfg.Services[0].Name)
	}

	if cfg.AlertCooldown != 120 {
		t.Errorf("expected alert_cooldown 120, got %d", cfg.AlertCooldown)
	}

	if cfg.StoragePath != "test.db" {
		t.Errorf("expected storage_path 'test.db', got %s", cfg.StoragePath)
	}

	if cfg.IncidentHistoryLimit != 5 {
		t.Errorf("expected incident_history_limit 5, got %d", cfg.IncidentHistoryLimit)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Create a minimal config file
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")

	jsonContent := `{"services": [{"type": "ping", "url": "http://example.com"}]}`
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	cfg, err := LoadConfig(jsonPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Check defaults are applied
	if cfg.IncidentHistoryLimit != 10 {
		t.Errorf("expected default incident_history_limit 10, got %d", cfg.IncidentHistoryLimit)
	}

	if cfg.StoragePath != "" {
		t.Errorf("expected default storage_path '', got %s", cfg.StoragePath)
	}
}

func TestLoadConfigEnvironmentOverride(t *testing.T) {
	// Create a config file
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
		"services": [{"type": "ping", "url": "http://example.com"}],
		"alert_cooldown": 60
	}`
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	// Set environment variable
	os.Setenv("SERVICE_STATUS_ALERT_COOLDOWN", "300")
	defer os.Unsetenv("SERVICE_STATUS_ALERT_COOLDOWN")

	cfg, err := LoadConfig(jsonPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Environment variable should override config file value
	if cfg.AlertCooldown != 300 {
		t.Errorf("expected alert_cooldown 300 (from env), got %d", cfg.AlertCooldown)
	}
}

func TestLoadConfigStoragePathEnvOverride(t *testing.T) {
	// Create a config file without storage_path
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")

	jsonContent := `{"services": [{"type": "ping", "url": "http://example.com"}]}`
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	// Set environment variable
	os.Setenv("SERVICE_STATUS_STORAGE_PATH", "/tmp/status.db")
	defer os.Unsetenv("SERVICE_STATUS_STORAGE_PATH")

	cfg, err := LoadConfig(jsonPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Environment variable should set the value
	if cfg.StoragePath != "/tmp/status.db" {
		t.Errorf("expected storage_path '/tmp/status.db' (from env), got %s", cfg.StoragePath)
	}
}

func TestSetDefaults(t *testing.T) {
	v := viper.New()
	SetDefaults(v)

	// Verify defaults are set
	if v.GetInt("incident_history_limit") != 10 {
		t.Errorf("expected default incident_history_limit 10, got %d", v.GetInt("incident_history_limit"))
	}

	if v.GetInt("min_incident_duration") != 0 {
		t.Errorf("expected default min_incident_duration 0, got %d", v.GetInt("min_incident_duration"))
	}

	if v.GetString("storage_path") != "" {
		t.Errorf("expected default storage_path '', got %s", v.GetString("storage_path"))
	}
}

func TestLoadConfigWithViper(t *testing.T) {
	v := viper.New()
	SetDefaults(v)

	// Set values directly
	v.Set("services", []map[string]interface{}{
		{"type": "ping", "url": "http://example.com", "name": "Example"},
	})
	v.Set("alert_cooldown", 180)

	cfg, err := LoadConfigWithViper(v)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(cfg.Services))
	}

	if cfg.AlertCooldown != 180 {
		t.Errorf("expected alert_cooldown 180, got %d", cfg.AlertCooldown)
	}
}

func TestLoadConfigNotifiersYAML(t *testing.T) {
	// Create a YAML config with notifiers
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
services:
  - type: ping
    url: http://example.com

notifiers:
  - type: webhook
    webhook_url: http://webhook.example.com/notify
  - type: slack
    webhook_url: http://slack.webhook.url
    channel: "#alerts"
    username: StatusBot

alert_cooldown: 300
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test yaml: %v", err)
	}

	cfg, err := LoadConfig(yamlPath)
	if err != nil {
		t.Fatalf("failed to load yaml config: %v", err)
	}

	if len(cfg.Notifiers) != 2 {
		t.Errorf("expected 2 notifiers, got %d", len(cfg.Notifiers))
	}

	if cfg.Notifiers[0].Type != "webhook" {
		t.Errorf("expected first notifier type 'webhook', got %s", cfg.Notifiers[0].Type)
	}

	if cfg.Notifiers[1].Channel != "#alerts" {
		t.Errorf("expected slack channel '#alerts', got %s", cfg.Notifiers[1].Channel)
	}
}

func TestLoadConfigMaintenanceSettings(t *testing.T) {
	// Create a YAML config with maintenance settings
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
services:
  - type: ping
    url: http://example.com

maintenance_message: "System under maintenance"
min_incident_duration: 60
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test yaml: %v", err)
	}

	cfg, err := LoadConfig(yamlPath)
	if err != nil {
		t.Fatalf("failed to load yaml config: %v", err)
	}

	if cfg.MaintenanceMessage != "System under maintenance" {
		t.Errorf("expected maintenance_message 'System under maintenance', got %s", cfg.MaintenanceMessage)
	}

	if cfg.MinIncidentDuration != 60 {
		t.Errorf("expected min_incident_duration 60, got %d", cfg.MinIncidentDuration)
	}
}
