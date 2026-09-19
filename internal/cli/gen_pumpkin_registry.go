package cli

import (
	gen "GoCraft/internal/generators/genpumpkinregistry"

	"github.com/spf13/cobra"
)

func init() {
	gen := &cobra.Command{
		Use: "pumpkin-registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen.Generate()
		},
	}
	gen.Flags().String("input", "", "Pumpkin assets/bedrock/runtime_item_states.json")
	gen.Flags().String("components", "", "Pumpkin assets/bedrock/item_components.nbt")
	gen.Flags().String("output", "bedrock/item_registry_generated.go", "generated Go file")
	gen.Flags().String("component-output", "bedrock/item_components.nbt.gz", "copied gzip-compressed component NBT")
	genCommand.AddCommand(gen)
}
