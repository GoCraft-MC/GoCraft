package cli

import (
	"GoCraft/healthcheck"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func createCheckRunner(endpoint string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		verbose := viper.GetBool("verbose")

		status := healthcheck.CheckServiceEndpointAndExit(endpoint)
		if status != 0 {
			if verbose {
				fmt.Printf("status code: %d", status)
			}

			fmt.Println()
			os.Exit(1)
		}

		if verbose {
			fmt.Println("ok")
		}

		return nil
	}
}

func createCheckCommand() *cobra.Command {
	check := &cobra.Command{
		Use:   "check",
		Short: "Check the local environment",
	}
	check.PersistentFlags().Bool("verbose", false, "Enable verbose logs")
	check.AddCommand(&cobra.Command{
		Use:   "health",
		Short: "Check the /health endpoint of the healthcheck service",
		RunE:  createCheckRunner("/health"),
	})
	check.AddCommand(&cobra.Command{
		Use:   "ready",
		Short: "Check the /readyz endpoint of the healthcheck service",
		RunE:  createCheckRunner("/readyz"),
	})
	check.AddCommand(&cobra.Command{
		Use:   "live",
		Short: "Check the /livez endpoint of the healthcheck service",
		RunE:  createCheckRunner("/livez"),
	})
	return check
}

func init() {
	RootCommand.AddCommand(createCheckCommand())
}
