package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/willis7/service_status/config"
	"github.com/willis7/service_status/status"
)

// defaultOutageMinutes is the default duration in minutes to display for service outages.
const defaultOutageMinutes = 60

func init() {
	status.LoadTemplate()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing path to config")
		os.Exit(2)
	}
	configPath := os.Args[1]

	fmt.Println("Starting the application...")
	// read the config file to determine which services need to be checked
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	services, err := cfg.CreateFactories()
	if err != nil {
		log.Fatalf("create factories: %v", err)
	}

	// Setup storage if configured (opt-in feature)
	var storage *status.Storage
	if cfg.StoragePath != "" {
		var storageErr error
		storage, storageErr = status.NewStorage(cfg.StoragePath)
		if storageErr != nil {
			log.Fatalf("failed to initialize storage: %v", storageErr)
		}
		defer storage.Close()
		log.Printf("storage enabled: %s", cfg.StoragePath)
	}

	// Setup notification manager
	cooldown := time.Duration(cfg.AlertCooldown) * time.Second
	notifyManager := status.NewNotificationManager(cooldown)

	// Connect storage to notification manager for recording alerts
	if storage != nil {
		notifyManager.SetStorage(storage)
	}

	// Add configured notifiers
	for _, notifierConfig := range cfg.Notifiers {
		notifier, err := status.CreateNotifier(notifierConfig)
		if err != nil {
			log.Printf("failed to create notifier %s: %v", notifierConfig.Type, err)
			continue
		}
		notifyManager.AddNotifier(notifier)
		log.Printf("added %s notifier", notifier.Type())
	}

	// Check for maintenance mode
	maintenanceMsg := cfg.GetMaintenanceMessage()

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

				// Track incident transitions and update state atomically
				if _, storageErr := storage.RecordStatusTransition(svc.URL, displayName, isUp, errMsg); storageErr != nil {
					log.Printf("storage: failed to record status transition: %v", storageErr)
				}

				if storageErr := storage.RecordStatus(svc.URL, isUp, errMsg); storageErr != nil {
					log.Printf("storage: failed to record status: %v", storageErr)
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

	// Fetch past incidents if storage is enabled
	var pastIncidents []status.IncidentInfo
	if storage != nil {
		minDuration := time.Duration(cfg.MinIncidentDuration) * time.Second

		incidents, err := storage.GetRecentResolvedIncidents(cfg.IncidentHistoryLimit, minDuration)
		if err != nil {
			log.Printf("storage: failed to get recent incidents: %v", err)
		} else {
			for _, inc := range incidents {
				pastIncidents = append(pastIncidents, inc.ToIncidentInfo())
			}
		}
	}

	p := status.Page{
		Title:              "My Status",
		Status:             status.StatusHTML(overallStatus),
		Up:                 up,
		Degraded:           degraded,
		Down:               down,
		Time:               time.Now().Format("2006-01-02 15:04:05"),
		MaintenanceMessage: maintenanceMsg,
		PastIncidents:      pastIncidents,
	}

	// create and serve the page
	http.HandleFunc("/", status.Index(p))
	http.HandleFunc("/api/status", status.APIStatus(p))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
