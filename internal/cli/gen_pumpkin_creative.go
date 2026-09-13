package cli

import (
	gen "GoCraft/internal/generators/genpumpkincreative"

	"github.com/spf13/cobra"
)

func init() {
	gen := &cobra.Command{
		Use: "pumpkin-creative",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen.Generate()
		},
	}
	gen.Flags().String("input", "", "Pumpkin assets/bedrock/creative_items.json")
	gen.Flags().String("runtime-items", "", "Pumpkin assets/bedrock/runtime_item_states.json")
	gen.Flags().String("output", "bedrock/creative_catalog_generated.go", "generated Go file")
	genCommand.AddCommand(gen)
}
