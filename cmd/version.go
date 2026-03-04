package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the application version, injected at build time.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the application version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("service_status version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
