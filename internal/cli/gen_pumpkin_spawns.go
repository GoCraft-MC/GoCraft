package cli

import (
	gen "GoCraft/internal/generators/genpumpkinspawns"

	"github.com/spf13/cobra"
)

func init() {
	gen := &cobra.Command{
		Use: "pumpkin-spawns",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen.Generate()
		},
	}
	gen.Flags().String("source", "", "path to Pumpkin's generated biome.rs")
	gen.Flags().String("entity-source", "", "path to Pumpkin's generated entity_type.rs")
	gen.Flags().String("output", "server/natural_spawn_data_generated.go", "generated Go output")
	gen.Flags().String("experience-output", "server/entity_experience_data_generated.go", "generated entity XP output")
	genCommand.AddCommand(gen)
}
