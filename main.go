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
type Config struct {
	Services []status.Service `json:"services"`
}

// CreateFactories will return a slice of Pinger concrete services
func (c *Config) CreateFactories() []status.Pinger {
	var checks []status.Pinger

	for _, service := range c.Services {
		switch service.Type {
		case "ping":
			p := status.PingFactory{}
			checks = append(checks, p.Create(service))
		case "grep":
			g := status.GrepFactory{}
			checks = append(checks, g.Create(service))
		}
	}

	return checks
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

	services := config.CreateFactories()

	down := make(map[string]int)
	var up []string

	for _, service := range services {
		err := service.Status()
		if err != nil {
			down[service.GetService().URL] = 60
			break
		}
		up = append(up, service.GetService().URL)
	}

	statushtml := template.HTML(`<div class="alert alert-success" role="alert">
	<span class="glyphicon glyphicon-thumbs-up" aria-hidden="true"></span>
	All Systems Operational
</div>`)

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
