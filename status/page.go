package status

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"
)

var tpl *template.Template

// ServiceInfo holds information about a service including its response time.
type ServiceInfo struct {
	Name         string
	ResponseTime time.Duration
}

// OutageInfo holds information about a degraded or down service including its response time.
type OutageInfo struct {
	Minutes      int
	ResponseTime time.Duration
}

// IncidentInfo holds information about a past incident for display.
type IncidentInfo struct {
	ServiceName string
	StartedAt   string        // Formatted start time
	EndedAt     string        // Formatted end time (empty if ongoing)
	Duration    time.Duration // Duration of the incident
	Message     string        // Error message
	IsOngoing   bool          // Whether the incident is still active
}

// Page represents the data of the status page.
type Page struct {
	Title    string
	Status   template.HTML
	Up       []ServiceInfo
	Degraded map[string]OutageInfo
	Down     map[string]OutageInfo
	Time     string
	// MaintenanceMessage is displayed when the system is in maintenance mode.
	// Empty string indicates normal operation.
	MaintenanceMessage string
	// PastIncidents holds recent resolved incidents for display.
	PastIncidents []IncidentInfo
}

// StatusHTML converts a known status string to template.HTML.
// It returns "success" for any unrecognized input.
func StatusHTML(s string) template.HTML {
	switch s {
	case "danger", "degraded", "success", "maintenance":
		return template.HTML(s)
	default:
		return "success"
	}
}

// LoadTemplate parses the templates in the templates directory.
func LoadTemplate() {
	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))
}

// Index is a HandlerFunc that closes over a Page data structure.
func Index(p Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tpl.ExecuteTemplate(w, "status.gohtml", p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// ServiceStatus represents the status of a single service in the API response.
type ServiceStatus struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Updated      string `json:"updated"`
	ResponseTime int64  `json:"response_time_ms,omitempty"`
}

// APIIncident represents an incident in the JSON API response.
type APIIncident struct {
	ServiceName string `json:"service_name"`
	StartedAt   string `json:"started_at"`
	EndedAt     string `json:"ended_at,omitempty"`
	DurationMs  int64  `json:"duration_ms"`
	Message     string `json:"message,omitempty"`
	IsOngoing   bool   `json:"is_ongoing"`
}

// APIResponse represents the JSON API response for status endpoint.
type APIResponse struct {
	OverallStatus      string          `json:"overall_status"`
	Services           []ServiceStatus `json:"services"`
	Updated            string          `json:"updated"`
	MaintenanceMessage string          `json:"maintenance_message,omitempty"`
	PastIncidents      []APIIncident   `json:"past_incidents,omitempty"`
}

// APIStatus is a HandlerFunc that returns the status page data as JSON.
// Note: Service order is deterministic for "up" services (slice order preserved),
// but may vary for degraded/down services due to map iteration order.
func APIStatus(p Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		services := make([]ServiceStatus, 0)
		now := time.Now().UTC().Format(time.RFC3339)

		// Add services that are up
		for _, svc := range p.Up {
			services = append(services, ServiceStatus{
				Name:         svc.Name,
				Status:       "OK",
				Updated:      now,
				ResponseTime: svc.ResponseTime.Milliseconds(),
			})
		}

		// Add services that are degraded
		for name, info := range p.Degraded {
			services = append(services, ServiceStatus{
				Name:         name,
				Status:       "DEGRADED",
				Updated:      now,
				ResponseTime: info.ResponseTime.Milliseconds(),
			})
		}

		// Add services that are down
		for name, info := range p.Down {
			services = append(services, ServiceStatus{
				Name:         name,
				Status:       "DOWN",
				Updated:      now,
				ResponseTime: info.ResponseTime.Milliseconds(),
			})
		}

		// Determine overall status string for API
		overallStatus := "OK"
		switch string(p.Status) {
		case "danger":
			overallStatus = "DOWN"
		case "degraded":
			overallStatus = "DEGRADED"
		case "maintenance":
			overallStatus = "MAINTENANCE"
		}

		// Convert past incidents for API response
		var apiIncidents []APIIncident
		for _, inc := range p.PastIncidents {
			apiInc := APIIncident{
				ServiceName: inc.ServiceName,
				StartedAt:   inc.StartedAt,
				DurationMs:  inc.Duration.Milliseconds(),
				Message:     inc.Message,
				IsOngoing:   inc.IsOngoing,
			}
			if !inc.IsOngoing {
				apiInc.EndedAt = inc.EndedAt
			}
			apiIncidents = append(apiIncidents, apiInc)
		}

		response := APIResponse{
			OverallStatus:      overallStatus,
			Services:           services,
			Updated:            now,
			MaintenanceMessage: p.MaintenanceMessage,
			PastIncidents:      apiIncidents,
		}

		// Encode to buffer first to avoid partial writes on error
		data, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data) // Error ignored: client may have disconnected
	}
}
