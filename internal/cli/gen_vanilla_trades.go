package cli

import (
	gen "GoCraft/internal/generators/genvanillatrades"

	"github.com/spf13/cobra"
)

func init() {
	gen := &cobra.Command{
		Use: "vanilla-trades",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen.Generate()
		},
	}
	gen.Flags().String("source", "", "decompiled Mojang VillagerTrades.java")
	gen.Flags().String("output", "java/handler/trade_catalog_generated.go", "generated Go output")
	genCommand.AddCommand(gen)
}
