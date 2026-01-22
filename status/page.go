package status

import (
	"html/template"
	"net/http"
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
}

// StatusHTML converts a known status string to template.HTML.
// It returns "success" for any unrecognized input.
func StatusHTML(s string) template.HTML {
	switch s {
	case "danger", "degraded", "success":
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
