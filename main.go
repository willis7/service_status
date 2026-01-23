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

	// Check all services and categorize by status
	up, degraded, down := status.CheckAllServices(services, storage, notifyManager, maintenanceMsg)

	// Determine overall status
	overallStatus := status.DetermineOverallStatus(maintenanceMsg, degraded, down)

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
