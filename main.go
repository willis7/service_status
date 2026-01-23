package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/willis7/service_status/status"
)

// defaultOutageMinutes is the default duration in minutes to display for service outages.
const defaultOutageMinutes = 60

func init() {
	status.LoadTemplate()
}

// Config holds a list of services to be checked and notification settings.
type Config struct {
	Services  []status.Service        `json:"services"`
	Notifiers []status.NotifierConfig `json:"notifiers,omitempty"`
	// AlertCooldown is the minimum time between alerts for the same service (in seconds)
	AlertCooldown int `json:"alert_cooldown,omitempty"`
	// StoragePath is the path to the SQLite database for persistent storage.
	// If empty, storage is disabled (opt-in feature).
	StoragePath string `json:"storage_path,omitempty"`
	// MaintenanceFile is the path to a file containing maintenance message.
	// If the file exists and has content, the system enters maintenance mode.
	MaintenanceFile string `json:"maintenance_file,omitempty"`
	// MaintenanceMessage is an inline maintenance message.
	// If set, the system enters maintenance mode. MaintenanceFile takes precedence.
	MaintenanceMessage string `json:"maintenance_message,omitempty"`
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

// LoadConfiguration takes a configuration file and returns a Config struct.
func LoadConfiguration(file string) (Config, error) {
	var config Config
	configFile, err := os.Open(file)
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	if err := jsonParser.Decode(&config); err != nil {
		return config, err
	}
	return config, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing path to config")
		os.Exit(2)
	}
	configPath := os.Args[1]

	fmt.Println("Starting the application...")
	// read the config file to determine which services need to be checked
	config, err := LoadConfiguration(configPath)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	services, err := config.CreateFactories()
	if err != nil {
		log.Fatalf("create factories: %v", err)
	}

	// Setup storage if configured (opt-in feature)
	var storage *status.Storage
	if config.StoragePath != "" {
		var storageErr error
		storage, storageErr = status.NewStorage(config.StoragePath)
		if storageErr != nil {
			log.Fatalf("failed to initialize storage: %v", storageErr)
		}
		defer storage.Close()
		log.Printf("storage enabled: %s", config.StoragePath)
	}

	// Setup notification manager
	cooldown := 5 * time.Minute // default
	if config.AlertCooldown > 0 {
		cooldown = time.Duration(config.AlertCooldown) * time.Second
	}
	notifyManager := status.NewNotificationManager(cooldown)

	// Connect storage to notification manager for recording alerts
	if storage != nil {
		notifyManager.SetStorage(storage)
	}

	// Add configured notifiers
	for _, notifierConfig := range config.Notifiers {
		notifier, err := status.CreateNotifier(notifierConfig)
		if err != nil {
			log.Printf("failed to create notifier %s: %v", notifierConfig.Type, err)
			continue
		}
		notifyManager.AddNotifier(notifier)
		log.Printf("added %s notifier", notifier.Type())
	}

	// Check for maintenance mode
	maintenanceMsg := config.GetMaintenanceMessage()

	down := make(map[string]status.OutageInfo)
	degraded := make(map[string]status.OutageInfo)
	var up []status.ServiceInfo

	// If in maintenance mode, skip status checks and mark all services as up
	if maintenanceMsg != "" {
		log.Printf("maintenance mode active: %s", maintenanceMsg)
		log.Printf("skipping status checks for %d services", len(services))
		for _, service := range services {
			svc := service.GetService()
			up = append(up, status.ServiceInfo{
				Name:         svc.DisplayName(),
				ResponseTime: 0,
			})
		}
	} else {
		for _, service := range services {
			result := service.StatusWithTiming()
			svc := service.GetService()
			displayName := svc.DisplayName()

			// Record status to storage if enabled
			if storage != nil {
				var errMsg string
				if result.Err != nil {
					errMsg = result.Err.Error()
				}
				isUp := status.IsOperational(result.Err)
				if storageErr := storage.RecordStatus(svc.URL, isUp, errMsg); storageErr != nil {
					log.Printf("storage: failed to record status: %v", storageErr)
				}
				if storageErr := storage.UpdateServiceState(svc.URL, isUp); storageErr != nil {
					log.Printf("storage: failed to update state: %v", storageErr)
				}
			}

			if result.Err != nil {
				if status.IsDegraded(result.Err) {
					degraded[displayName] = status.OutageInfo{
						Minutes:      defaultOutageMinutes,
						ResponseTime: result.ResponseTime,
					}
					notifyManager.CheckAndNotify(svc.URL, true) // Degraded is still partially available
				} else {
					down[displayName] = status.OutageInfo{
						Minutes:      defaultOutageMinutes,
						ResponseTime: result.ResponseTime,
					}
					notifyManager.CheckAndNotify(svc.URL, false)
				}
				continue
			}
			up = append(up, status.ServiceInfo{
				Name:         displayName,
				ResponseTime: result.ResponseTime,
			})
			notifyManager.CheckAndNotify(svc.URL, true)
		}
	}

	// Determine overall status (maintenance takes precedence over all other states)
	var overallStatus string
	switch {
	case maintenanceMsg != "":
		overallStatus = "maintenance"
	case len(down) > 0:
		overallStatus = "danger"
	case len(degraded) > 0:
		overallStatus = "degraded"
	default:
		overallStatus = "success"
	}

	p := status.Page{
		Title:              "My Status",
		Status:             status.StatusHTML(overallStatus),
		Up:                 up,
		Degraded:           degraded,
		Down:               down,
		Time:               time.Now().Format("2006-01-02 15:04:05"),
		MaintenanceMessage: maintenanceMsg,
	}

	// create and serve the page
	http.HandleFunc("/", status.Index(p))
	http.HandleFunc("/api/status", status.APIStatus(p))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
