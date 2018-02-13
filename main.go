package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var tools = []string{"curl", "nc", "ping"}
var tpl *template.Template

type page struct {
	Title  string
	Status template.HTML
	Up     []string
	Down   map[string]int
	Time   string
}

func init() {
	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))
}

func status(p page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tpl.ExecuteTemplate(w, "status.gohtml", p)
	}
}

// Config holds a list of services to be
// checked
type Config []struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Port string `json:"port"`
}

// LoadConfiguration takes a configuration file and returns
// a Config struct
func LoadConfiguration(file string) (Config, error) {
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		return config, err
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config, nil
}

// IsInstalled checks the availability of a list of strings passed
// in as an array
func IsInstalled(tools []string) {
	for _, t := range tools {
		path, err := exec.LookPath(t)
		if err != nil {
			log.Fatalf("%s not found on this system", t)
		}
		fmt.Printf("%s is available at %s\n", t, path)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing path to config")
		os.Exit(2)
	}
	configPath := os.Args[1]

	IsInstalled(tools)

	fmt.Println("Starting the application...")
	// read the config file to determine which services need to be checked
	config, _ := LoadConfiguration(configPath)
	// :debug
	fmt.Println(config)

	// // For each of the services, get their status
	// var (
	// 	out []byte
	// 	err error
	// )
	// // PING
	// google := commands.Ping{URL: "google.com"}
	// if out, err = google.Command().Output(); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }
	// fmt.Println(string(out))

	// // CURL
	// google2 := commands.Curl{URL: "google.com"}
	// if out, err = google2.Command().Output(); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }
	// fmt.Println(string(out))

	// // NC
	// google3 := commands.NC{URL: "google.com", Port: "80"}
	// if out, err = google3.Command().Output(); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }
	// fmt.Println(string(out))

	statushtml := template.HTML(`<div class="alert alert-success" role="alert">
	<span class="glyphicon glyphicon-thumbs-up" aria-hidden="true"></span>
	All Systems Operational
</div>`)

	up := []string{"ping google.com"}
	down := map[string]int{
		"ping googlex.com":  60,
		"nc heisenberg.net": 30,
	}

	p := page{
		Title:  "My Status",
		Status: statushtml,
		Up:     up,
		Down:   down,
		Time:   time.Now().Format("2006-01-02 15:04:05"),
	}

	// create and serve the page
	http.HandleFunc("/", status(p))
	http.ListenAndServe(":8080", nil)
}
