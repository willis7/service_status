package main

import (
	"encoding/json"
	"errors"
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

// Config holds a list of services to be checked
type Config struct {
	Services []status.Service `json:"services"`
}

// CreateServices will return a slice of concrete services which
// conform to the Pinger interface
func (c *Config) CreateServices() ([]status.Pinger, error) {
	var checks []status.Pinger

	for _, service := range c.Services {
		switch service.Type {
		case "ping":
			pf := status.PingFactory{}
			p, err := pf.Create(service)
			if err != nil {
				return nil, errors.New("failed to create ping object")
			}
			checks = append(checks, p)
		case "grep":
			gf := status.GrepFactory{}
			g, err := gf.Create(service)
			if err != nil {
				return nil, errors.New("failed to create ping object")
			}
			checks = append(checks, g)
		}
	}

	return checks, nil
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

	services, err := config.CreateServices()
	if err != nil {
		log.Fatalf("create factories: %v", err)
	}

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

	p := status.Page{
		Title:  "My Status",
		Status: "danger",
		Up:     up,
		Down:   down,
		Time:   time.Now().Format("2006-01-02 15:04:05"),
	}

	// create and serve the page
	http.HandleFunc("/", status.Index(p))
	http.ListenAndServe(":8080", nil)
}
