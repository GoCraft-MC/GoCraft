package cli

import (
	"GoCraft/server"
	"fmt"

	"github.com/spf13/cobra"
)

func handleVersion(cmd *cobra.Command, args []string) error {
	fmt.Printf("gocraft %s", server.GetVersion())
	fmt.Println()
	return nil
}

func init() {
	version := &cobra.Command{
		Use:   "version",
		Short: "Show the version",
		RunE:  handleVersion,
	}
	RootCommand.AddCommand(version)
}
