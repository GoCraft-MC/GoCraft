package world

import "testing"

func TestHarvestBeehive(t *testing.T) {
	hive := Block{Namespace: "minecraft", Name: "beehive", Properties: map[string]string{"honey_level": "5", "facing": "east"}}
	for _, test := range []struct {
		item  string
		want  string
		count int
	}{
		{"minecraft:shears", "minecraft:honeycomb", 3},
		{"minecraft:glass_bottle", "minecraft:honey_bottle", 1},
	} {
		harvested, output, ok := HarvestBeehive(hive, test.item)
		if !ok || output.ItemID != test.want || output.Count != test.count || harvested.Properties["honey_level"] != "0" || harvested.Properties["facing"] != "east" {
			t.Errorf("harvest with %s = block %+v output %+v", test.item, harvested, output)
		}
	}
	if _, _, ok := HarvestBeehive(hive, "minecraft:stick"); ok {
		t.Fatal("hive accepted an invalid harvesting item")
	}
}
