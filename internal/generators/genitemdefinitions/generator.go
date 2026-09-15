// Command genitemdefinitions builds GoCraft's compact canonical item data.
package genitemdefinitions

import (
	"fmt"

	"github.com/spf13/viper"
)

func Generate() error {
	var paths options
	paths.itemsReport = viper.GetString("items-report")
	paths.itemIDs = viper.GetString("item-ids")
	paths.itemTags = viper.GetString("item-tags")
	paths.fuels = viper.GetString("fuels")
	paths.pumpkinItems = viper.GetString("pumpkin-items")
	paths.pumpkinTags = viper.GetString("pumpkin-tags")
	paths.output = viper.GetString("out")

	if paths.itemsReport == "" {
		return fmt.Errorf("items-report is empty")
	} else if paths.pumpkinItems == "" {
		return fmt.Errorf("pumpkin-items is empty")
	} else if paths.pumpkinTags == "" {
		return fmt.Errorf("pumpkin-tags is empty")
	}

	writeCatalogue(paths.output, generate(paths))
	return nil
}
