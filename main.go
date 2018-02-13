package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/willis7/status/status"
)

func init() {
	status.LoadTemplate()
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing path to config")
		os.Exit(2)
	}
	configPath := os.Args[1]

	fmt.Println("Starting the application...")
	// read the config file to determine which services need to be checked
	config, _ := LoadConfiguration(configPath)
	// :debug
	fmt.Println(config)

	statushtml := template.HTML(`<div class="alert alert-success" role="alert">
	<span class="glyphicon glyphicon-thumbs-up" aria-hidden="true"></span>
	All Systems Operational
</div>`)

	up := []string{"ping google.com"}
	down := map[string]int{
		"ping googlex.com":    60,
		"grep heisenberg.net": 30,
	}

	p := status.Page{
		Title:  "My Status",
		Status: statushtml,
		Up:     up,
		Down:   down,
		Time:   time.Now().Format("2006-01-02 15:04:05"),
	}

	// create and serve the page
	http.HandleFunc("/", status.Index(p))
	http.ListenAndServe(":8080", nil)
}
