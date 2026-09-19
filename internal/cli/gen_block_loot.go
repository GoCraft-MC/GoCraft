package cli

import (
	gen "GoCraft/internal/generators/genblockloot"
	"github.com/spf13/cobra"
)

func init() {
	gen := &cobra.Command{
		Use: "block-loot",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen.Generate()
		},
	}
	gen.Flags().String("jar", "", "official Minecraft jar containing data/minecraft")
	gen.MarkFlagRequired("jar")

	gen.Flags().String("pumpkin-blocks", "", "Pumpkin assets/blocks.json")
	gen.MarkFlagRequired("pumpkin-blocks")

	gen.Flags().String("output", "internal/gamedata/java/1.21.4/block_loot.json", "output bundle")
	gen.Flags().String("version", "1.21.4", "Minecraft Java version")
	genCommand.AddCommand(gen)
}
