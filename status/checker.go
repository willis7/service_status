package status

import (
	"log"
)

// defaultOutageMinutes is the default duration in minutes to display for service outages.
const defaultOutageMinutes = 60

// CheckAllServices performs status checks on all services and categorizes them
// into up, degraded, and down states. It handles storage recording and notifications.
// If maintenanceMsg is non-empty, all services are marked as up without checking.
// Storage and notifyManager may be nil to disable those features.
func CheckAllServices(
	services []Pinger,
	storage *Storage,
	notifyManager *NotificationManager,
	maintenanceMsg string,
) (up []ServiceInfo, degraded, down map[string]OutageInfo) {
	down = make(map[string]OutageInfo)
	degraded = make(map[string]OutageInfo)

	// If in maintenance mode, skip status checks and mark all services as up
	if maintenanceMsg != "" {
		log.Printf("maintenance mode active: %s", maintenanceMsg)
		log.Printf("skipping status checks for %d services", len(services))
		for _, service := range services {
			svc := service.GetService()
			up = append(up, ServiceInfo{
				Name:         svc.DisplayName(),
				ResponseTime: 0,
			})
		}
		return up, degraded, down
	}

	// Check each service
	for _, pinger := range services {
		result := pinger.StatusWithTiming()
		svc := pinger.GetService()
		displayName := svc.DisplayName()

		// Record status to storage if enabled
		if storage != nil {
			var errMsg string
			if result.Err != nil {
				errMsg = result.Err.Error()
			}
			isUp := IsOperational(result.Err)

			// Track incident transitions and update state atomically
			if _, storageErr := storage.RecordStatusTransition(svc.URL, displayName, isUp, errMsg); storageErr != nil {
				log.Printf("storage: failed to record status transition: %v", storageErr)
			}

			if storageErr := storage.RecordStatus(svc.URL, isUp, errMsg); storageErr != nil {
				log.Printf("storage: failed to record status: %v", storageErr)
			}
		}

		if result.Err != nil {
			if IsDegraded(result.Err) {
				degraded[displayName] = OutageInfo{
					Minutes:      defaultOutageMinutes,
					ResponseTime: result.ResponseTime,
				}
				if notifyManager != nil {
					notifyManager.CheckAndNotify(svc.URL, true) // operational
				}
			} else {
				down[displayName] = OutageInfo{
					Minutes:      defaultOutageMinutes,
					ResponseTime: result.ResponseTime,
				}
				if notifyManager != nil {
					notifyManager.CheckAndNotify(svc.URL, false) // not operational
				}
			}
			continue
		}
		up = append(up, ServiceInfo{
			Name:         displayName,
			ResponseTime: result.ResponseTime,
		})
		if notifyManager != nil {
			notifyManager.CheckAndNotify(svc.URL, true) // operational
		}
	}

	return up, degraded, down
}

// DetermineOverallStatus calculates the overall status based on service states.
// It returns one of: "maintenance", "danger", "degraded", or "success".
//
// Priority order:
//  1. maintenance - if maintenance message is set
//  2. danger - if any services are down
//  3. degraded - if any services are degraded
//  4. success - all services are up
func DetermineOverallStatus(maintenanceMsg string, degraded, down map[string]OutageInfo) string {
	switch {
	case maintenanceMsg != "":
		return "maintenance"
	case len(down) > 0:
		return "danger"
	case len(degraded) > 0:
		return "degraded"
	default:
		return "success"
	}
}
