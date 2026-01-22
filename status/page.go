package status

import (
	"html/template"
	"net/http"
)

var tpl *template.Template

// Page represents the data of the status page
type Page struct {
	Title  string
	Status template.HTML
	Up     []string
	Down   map[string]int
	Time   string
}

// LoadTemplate parses the templates in the templates dir
func LoadTemplate() {
	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))
}

// Index is a HandlerFunc which closes over a Page data structure
func Index(p Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tpl.ExecuteTemplate(w, "status.gohtml", p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
