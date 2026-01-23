package status

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"
)

var tpl *template.Template

// Page represents the data of the status page.
type Page struct {
	Title    string
	Status   template.HTML
	Up       []string
	Degraded map[string]int
	Down     map[string]int
	Time     string
	// MaintenanceMessage is displayed when the system is in maintenance mode.
	// Empty string indicates normal operation.
	MaintenanceMessage string
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
	Name    string `json:"name"`
	Status  string `json:"status"`
	Updated string `json:"updated"`
}

// APIResponse represents the JSON API response for status endpoint.
type APIResponse struct {
	OverallStatus      string          `json:"overall_status"`
	Services           []ServiceStatus `json:"services"`
	Updated            string          `json:"updated"`
	MaintenanceMessage string          `json:"maintenance_message,omitempty"`
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
		for _, name := range p.Up {
			services = append(services, ServiceStatus{
				Name:    name,
				Status:  "OK",
				Updated: now,
			})
		}

		// Add services that are degraded
		for name := range p.Degraded {
			services = append(services, ServiceStatus{
				Name:    name,
				Status:  "DEGRADED",
				Updated: now,
			})
		}

		// Add services that are down
		for name := range p.Down {
			services = append(services, ServiceStatus{
				Name:    name,
				Status:  "DOWN",
				Updated: now,
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

		response := APIResponse{
			OverallStatus:      overallStatus,
			Services:           services,
			Updated:            now,
			MaintenanceMessage: p.MaintenanceMessage,
		}

		// Encode to buffer first to avoid partial writes on error
		data, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
