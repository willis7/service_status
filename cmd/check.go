package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/willis7/service_status/config"
	"github.com/willis7/service_status/status"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run a single status check",
	Long:  `Loads the configuration, runs a single iteration of all configured checks (ping, http, etc.), and prints the result to standard out instead of starting an HTTP server. Useful for CI/CD or cron job execution. Exit with non-zero status if any check fails.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfigWithViper(viper.GetViper())
		if err != nil {
			log.Fatalf("failed to load configuration: %v", err)
		}

		services, err := cfg.CreateFactories()
		if err != nil {
			log.Fatalf("create factories: %v", err)
		}

		maintenanceMsg := cfg.GetMaintenanceMessage()

		if maintenanceMsg != "" {
			fmt.Printf("Maintenance mode active: %s\n", maintenanceMsg)
			os.Exit(0)
		}

		hasErrors := false

		for _, pinger := range services {
			result := pinger.StatusWithTiming()
			svc := pinger.GetService()
			displayName := svc.DisplayName()

			if result.Err != nil {
				if status.IsDegraded(result.Err) {
					fmt.Printf("[DEGRADED] %s (%v) - %v\n", displayName, result.ResponseTime, result.Err)
					hasErrors = true
				} else {
					fmt.Printf("[DOWN] %s (%v) - %v\n", displayName, result.ResponseTime, result.Err)
					hasErrors = true
				}
			} else {
				fmt.Printf("[UP] %s (%v)\n", displayName, result.ResponseTime)
			}
		}

		if hasErrors {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
