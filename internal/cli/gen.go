package cli

import "github.com/spf13/cobra"

var genCommand *cobra.Command

func init() {
	genCommand = &cobra.Command{
		Use: "gen",
	}
	RootCommand.AddCommand(genCommand)
}
