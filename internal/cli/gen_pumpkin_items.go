package cli

import (
	gen "GoCraft/internal/generators/genpumpkinitems"

	"github.com/spf13/cobra"
)

func init() {
	gen := &cobra.Command{
		Use: "pumpkin-items",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen.Generate()
		},
	}
	gen.Flags().String("source", "", "Pumpkin crates/pumpkin-data/src/generated/item.rs")
	gen.Flags().String("output", "bedrock/item_mappings_generated.go", "generated Go output")
	genCommand.AddCommand(gen)
}
