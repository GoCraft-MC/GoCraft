package cli

import (
	gen "GoCraft/internal/generators/genitemdefinitions"

	"github.com/spf13/cobra"
)

func init() {
	gen := &cobra.Command{
		Use: "item-definitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen.Generate()
		},
	}
	gen.Flags().String("items-report", "", "Mojang 1.21.4 reports/items.json")
	gen.Flags().String("item-ids", "internal/gamedata/java/1.21.4/items.json", "Java item IDs")
	gen.Flags().String("item-tags", "internal/gamedata/java/1.21.4/network_tags.json", "Java item tags")
	gen.Flags().String("fuels", "internal/gamedata/java/1.21.4/fuels.json", "Java fuel data")
	gen.Flags().String("pumpkin-items", "", "Pumpkin assets/items.json")
	gen.Flags().String("pumpkin-tags", "", "matching Pumpkin item tags")
	gen.Flags().String("out", "internal/gamedata/vanilla/1.21.4/item_definitions.json", "output")
	genCommand.AddCommand(gen)
}
