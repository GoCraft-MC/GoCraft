package cli

import (
	gen "GoCraft/internal/generators/genpumpkinblocks"

	"github.com/spf13/cobra"
)

func init() {
	gen := &cobra.Command{
		Use: "pumpkin-blocks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen.Generate()
		},
	}
	gen.Flags().String("input", "", "Pumpkin assets/bedrock/block_states.nbt")
	gen.Flags().String("output", "bedrock/world/block_states.nbt.gz", "compressed embedded block-state stream")
	genCommand.AddCommand(gen)
}
