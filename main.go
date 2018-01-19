package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/willis7/status/commands"
)

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

	// check tool availibility

	fmt.Println("Starting the application...")
	// read the config file to determine which services need to be checked
	config, _ := LoadConfiguration(configPath)
	// :debug
	fmt.Println(config)

	// For each of the services, get their status
	var (
		out []byte
		err error
	)
	// PING
	// google := commands.Ping{URL: "google.com"}
	// if out, err = google.Command().Output(); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }
	// fmt.Println(string(out))

	// CURL
	google2 := commands.Curl{URL: "google.com"}
	if out, err = google2.Command().Output(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(out))

	// NC
	google3 := commands.NC{URL: "google.com", Port: "80"}
	if out, err = google3.Command().Output(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(out))

	// create and serve the page
}
