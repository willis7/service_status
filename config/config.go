// Package config provides configuration loading using Viper.
// It supports JSON and YAML configuration files, environment variable overrides,
// and sensible defaults.
package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/willis7/service_status/status"
)

// Config holds a list of services to be checked and notification settings.
type Config struct {
	Services  []status.Service        `mapstructure:"services" json:"services"`
	Notifiers []status.NotifierConfig `mapstructure:"notifiers" json:"notifiers,omitempty"`
	// AlertCooldown is the minimum time between alerts for the same service (in seconds)
	AlertCooldown int `mapstructure:"alert_cooldown" json:"alert_cooldown,omitempty"`
	// StoragePath is the path to the SQLite database for persistent storage.
	// If empty, storage is disabled (opt-in feature).
	StoragePath string `mapstructure:"storage_path" json:"storage_path,omitempty"`
	// MaintenanceFile is the path to a file containing maintenance message.
	// If the file exists and has content, the system enters maintenance mode.
	MaintenanceFile string `mapstructure:"maintenance_file" json:"maintenance_file,omitempty"`
	// MaintenanceMessage is an inline maintenance message.
	// If set, the system enters maintenance mode. MaintenanceFile takes precedence.
	MaintenanceMessage string `mapstructure:"maintenance_message" json:"maintenance_message,omitempty"`
	// IncidentHistoryLimit is the maximum number of past incidents to display (default: 10).
	IncidentHistoryLimit int `mapstructure:"incident_history_limit" json:"incident_history_limit,omitempty"`
	// MinIncidentDuration is the minimum incident duration in seconds to display (default: 0).
	// Incidents shorter than this duration are not shown in the history.
	MinIncidentDuration int `mapstructure:"min_incident_duration" json:"min_incident_duration,omitempty"`
}

// DefaultAlertCooldown is the default cooldown period between alerts for the same service.
const DefaultAlertCooldown = 5 * time.Minute

// SetDefaults configures default values for all configuration options.
func SetDefaults(v *viper.Viper) {
	v.SetDefault("alert_cooldown", int(DefaultAlertCooldown.Seconds()))
	v.SetDefault("incident_history_limit", 10)
	v.SetDefault("min_incident_duration", 0)
	v.SetDefault("storage_path", "")
	v.SetDefault("maintenance_file", "")
	v.SetDefault("maintenance_message", "")
	v.SetDefault("services", []status.Service{})
	v.SetDefault("notifiers", []status.NotifierConfig{})
}

// BindEnvVars binds environment variables with the SERVICE_STATUS_ prefix.
func BindEnvVars(v *viper.Viper) {
	v.SetEnvPrefix("SERVICE_STATUS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
}

// LoadConfig loads configuration from a file using Viper.
// It supports JSON, YAML, and other formats that Viper supports.
// Environment variables with SERVICE_STATUS_ prefix override config file values.
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	SetDefaults(v)

	// Bind environment variables
	BindEnvVars(v)

	// Set the config file path
	v.SetConfigFile(configPath)

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into Config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// LoadConfigWithViper loads configuration using a provided Viper instance.
// This is useful for testing or when you need more control over the Viper configuration.
func LoadConfigWithViper(v *viper.Viper) (*Config, error) {
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &config, nil
}

// CreateFactories returns a slice of Pinger concrete services.
func (c *Config) CreateFactories() ([]status.Pinger, error) {
	var checks []status.Pinger

	for _, service := range c.Services {
		switch service.Type {
		case "ping":
			pf := status.PingFactory{}
			p, err := pf.Create(service)
			if err != nil {
				return nil, fmt.Errorf("failed to create ping object: %w", err)
			}
			checks = append(checks, p)
		case "grep":
			gf := status.GrepFactory{}
			g, err := gf.Create(service)
			if err != nil {
				return nil, fmt.Errorf("failed to create grep object: %w", err)
			}
			checks = append(checks, g)
		case "tcp":
			tf := status.TCPFactory{}
			t, err := tf.Create(service)
			if err != nil {
				return nil, fmt.Errorf("failed to create tcp object: %w", err)
			}
			checks = append(checks, t)
		case "icmp":
			icf := status.ICMPFactory{}
			ic, err := icf.Create(service)
			if err != nil {
				return nil, fmt.Errorf("failed to create icmp object: %w", err)
			}
			checks = append(checks, ic)
		case "script":
			sf := status.ScriptFactory{}
			sc, err := sf.Create(service)
			if err != nil {
				return nil, fmt.Errorf("failed to create script object: %w", err)
			}
			checks = append(checks, sc)
		default:
			log.Printf("warning: unknown service type %q, skipping", service.Type)
		}
	}

	return checks, nil
}

// GetMaintenanceMessage returns the maintenance message if maintenance mode is active.
// It first checks for a maintenance file, then falls back to inline message.
// Returns empty string if maintenance mode is not active.
func (c *Config) GetMaintenanceMessage() string {
	// Check maintenance file first (takes precedence)
	if c.MaintenanceFile != "" {
		content, err := os.ReadFile(c.MaintenanceFile)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("warning: failed to read maintenance file %s: %v", c.MaintenanceFile, err)
			}
			// Fall through to inline message
		} else {
			msg := strings.TrimSpace(string(content))
			if msg != "" {
				return msg
			}
		}
	}
	// Fall back to inline message
	return c.MaintenanceMessage
}
