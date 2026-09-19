package cli

import (
	gen "GoCraft/internal/generators/genfurnacefuels"

	"github.com/spf13/cobra"
)

func init() {
	gen := &cobra.Command{
		Use: "furnace-fuels",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen.Generate()
		},
	}
	gen.Flags().String("pumpkin-items", "", "path to Pumpkin assets/items.json")
	gen.Flags().String("pumpkin-fuels", "", "path to Pumpkin generated/fuels.rs")
	gen.Flags().String("java-items", "internal/gamedata/java/1.21.4/items.json", "path to GoCraft Java items.json")
	gen.Flags().String("out", "internal/gamedata/java/1.21.4/fuels.json", "output path")
	genCommand.AddCommand(gen)
}
