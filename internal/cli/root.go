package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "genbu",
	Short: "Environment variable manager for cloud environments",
	Long:  "Genbu sets and validates environment variables from YAML configs with cloud provider integration (AWS SSM, Secrets Manager).",
}

var configFile string
var logLevel string
var providerFlag string

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "genbu.yaml", "path to config file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&providerFlag, "provider", "", "cloud provider: aws, gcp, azure (overrides config file)")

	rootCmd.AddCommand(newVersionCmd())
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("genbu %s\n", version)
		},
	}
}
