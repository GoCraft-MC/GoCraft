// Command genitemdefinitions builds GoCraft's compact canonical item data.
package main

import (
	"flag"
	"os"
)

func main() {
	var paths options
	flag.StringVar(&paths.itemsReport, "items-report", "", "Mojang 1.21.4 reports/items.json")
	flag.StringVar(&paths.itemIDs, "item-ids", "internal/gamedata/java/1.21.4/items.json", "Java item IDs")
	flag.StringVar(&paths.itemTags, "item-tags", "internal/gamedata/java/1.21.4/network_tags.json", "Java item tags")
	flag.StringVar(&paths.fuels, "fuels", "internal/gamedata/java/1.21.4/fuels.json", "Java fuel data")
	flag.StringVar(&paths.pumpkinItems, "pumpkin-items", "", "Pumpkin assets/items.json")
	flag.StringVar(&paths.pumpkinTags, "pumpkin-tags", "", "matching Pumpkin item tags")
	flag.StringVar(&paths.output, "out", "internal/gamedata/vanilla/1.21.4/item_definitions.json", "output")
	flag.Parse()
	if paths.itemsReport == "" || paths.pumpkinItems == "" || paths.pumpkinTags == "" {
		flag.Usage()
		os.Exit(2)
	}
	writeCatalogue(paths.output, generate(paths))
}
