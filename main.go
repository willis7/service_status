package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/willis7/service_status/status"
)

func init() {
	status.LoadTemplate()
}

// Config holds a list of services to be checked.
type Config struct {
	Services []status.Service `json:"services"`
}

// CreateFactories returns a slice of Pinger concrete services.
func (c *Config) CreateFactories() ([]status.Pinger, error) {
	var checks []status.Pinger

	for _, service := range c.Services {
		switch service.Type {
		case "ping":
			pf := status.PingFactory{}
			p, err := pf.Create(service)
			if err != nil {
				return nil, fmt.Errorf("failed to create ping object: %w", err)
			}
			checks = append(checks, p)
		case "grep":
			gf := status.GrepFactory{}
			g, err := gf.Create(service)
			if err != nil {
				return nil, fmt.Errorf("failed to create grep object: %w", err)
			}
			checks = append(checks, g)
		}
	}

	return checks, nil
}

// LoadConfiguration takes a configuration file and returns a Config struct.
func LoadConfiguration(file string) (Config, error) {
	var config Config
	configFile, err := os.Open(file)
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	if err := jsonParser.Decode(&config); err != nil {
		return config, err
	}
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
	config, err := LoadConfiguration(configPath)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	services, err := config.CreateFactories()
	if err != nil {
		log.Fatalf("create factories: %v", err)
	}

	down := make(map[string]int)
	var up []string

	for _, service := range services {
		err := service.Status()
		if err != nil {
			down[service.GetService().URL] = 60
			continue
		}
		up = append(up, service.GetService().URL)
	}

	p := status.Page{
		Title:  "My Status",
		Status: "danger",
		Up:     up,
		Down:   down,
		Time:   time.Now().Format("2006-01-02 15:04:05"),
	}

	// create and serve the page
	http.HandleFunc("/", status.Index(p))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
