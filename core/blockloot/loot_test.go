package blockloot

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

func block(name string, properties ...map[string]string) coreworld.Block {
	result := coreworld.Block{Namespace: "minecraft", Name: name}
	if len(properties) != 0 {
		result.Properties = properties[0]
	}
	return result
}

func drops(name, tool string, properties ...map[string]string) []player.ItemStack {
	return Drops(Context{
		Block:  block(name, properties...),
		Tool:   player.ItemStack{ItemID: tool, Count: 1},
		Random: rand.New(rand.NewSource(1)),
	})
}

func TestEmbeddedRegistryIsComplete(t *testing.T) {
	if got := TableCount(); got != 1015 {
		t.Fatalf("loot table count = %d, want 1015 from Java 1.21.4", got)
	}
}

func TestEveryLootTableUsesSupportedSchema(t *testing.T) {
	supportedTypes := map[string]bool{
		"minecraft:block": true, "minecraft:item": true, "minecraft:alternatives": true,
		"minecraft:dynamic": true, "minecraft:uniform": true, "minecraft:binomial": true,
	}
	supportedConditions := map[string]bool{
		"minecraft:any_of": true, "minecraft:block_state_property": true,
		"minecraft:entity_properties": true, "minecraft:inverted": true,
		"minecraft:location_check": true, "minecraft:match_tool": true,
		"minecraft:random_chance": true, "minecraft:survives_explosion": true,
		"minecraft:table_bonus": true,
	}
	supportedFunctions := map[string]bool{
		"minecraft:apply_bonus": true, "minecraft:copy_components": true,
		"minecraft:copy_state": true, "minecraft:explosion_decay": true,
		"minecraft:limit_count": true, "minecraft:set_count": true,
	}
	var inspect func(path string, value any)
	inspect = func(path string, value any) {
		switch value := value.(type) {
		case map[string]any:
			if kind := stringValue(value["type"]); strings.HasPrefix(kind, "minecraft:") && !supportedTypes[kind] {
				t.Errorf("%s uses unsupported type %q", path, kind)
			}
			if condition := stringValue(value["condition"]); condition != "" && !supportedConditions[condition] {
				t.Errorf("%s uses unsupported condition %q", path, condition)
			}
			if function := stringValue(value["function"]); function != "" && !supportedFunctions[function] {
				t.Errorf("%s uses unsupported function %q", path, function)
			}
			for key, child := range value {
				inspect(path+"."+key, child)
			}
		case []any:
			for index, child := range value {
				inspect(fmt.Sprintf("%s[%d]", path, index), child)
			}
		}
	}
	for name, table := range data().tables {
		inspect(name, table)
	}
}

func TestStoneRequiresPickaxeAndDropsCobblestone(t *testing.T) {
	if got := drops("stone", ""); len(got) != 0 {
		t.Fatalf("stone broken by hand dropped %+v", got)
	}
	wantDrop(t, drops("stone", "minecraft:wooden_pickaxe"), "minecraft:cobblestone", 1)
}

func TestOreHarvestTiers(t *testing.T) {
	if got := drops("diamond_ore", "minecraft:wooden_pickaxe"); len(got) != 0 {
		t.Fatalf("diamond ore broken by wooden pickaxe dropped %+v", got)
	}
	wantDrop(t, drops("diamond_ore", "minecraft:iron_pickaxe"), "minecraft:diamond", 1)
	if got := drops("obsidian", "minecraft:iron_pickaxe"); len(got) != 0 {
		t.Fatalf("obsidian broken by iron pickaxe dropped %+v", got)
	}
	wantDrop(t, drops("obsidian", "minecraft:diamond_pickaxe"), "minecraft:obsidian", 1)
}

func TestBlocksWithoutToolRequirementStillDrop(t *testing.T) {
	wantDrop(t, drops("dirt", ""), "minecraft:dirt", 1)
	wantDrop(t, drops("oak_log", ""), "minecraft:oak_log", 1)
}

func TestSlabCountUsesBlockState(t *testing.T) {
	wantDrop(t, drops("stone_slab", "minecraft:wooden_pickaxe", map[string]string{"type": "bottom"}), "minecraft:stone_slab", 1)
	wantDrop(t, drops("stone_slab", "minecraft:wooden_pickaxe", map[string]string{"type": "double"}), "minecraft:stone_slab", 2)
}

func TestStateDrivenCropDrops(t *testing.T) {
	wantDrop(t, drops("wheat", "", map[string]string{"age": "0"}), "minecraft:wheat_seeds", 1)
	mature := drops("wheat", "", map[string]string{"age": "7"})
	if count(mature, "minecraft:wheat") != 1 || count(mature, "minecraft:wheat_seeds") < 1 {
		t.Fatalf("mature wheat drops = %+v", mature)
	}
}

func TestMatureCropLootTablesRemainAgeAware(t *testing.T) {
	tests := []struct {
		block, item string
		age         int
		minimum     int
	}{
		{"carrots", "minecraft:carrot", 7, 2},
		{"potatoes", "minecraft:potato", 7, 2},
		{"beetroots", "minecraft:beetroot", 3, 1},
		{"nether_wart", "minecraft:nether_wart", 3, 2},
		{"pumpkin_stem", "minecraft:pumpkin_seeds", 7, 1},
		{"melon_stem", "minecraft:melon_seeds", 7, 1},
	}
	for _, test := range tests {
		t.Run(test.block, func(t *testing.T) {
			total := 0
			for seed := int64(0); seed < 32; seed++ {
				got := Drops(Context{
					Block:  block(test.block, map[string]string{"age": strconv.Itoa(test.age)}),
					Random: rand.New(rand.NewSource(seed)),
				})
				total += count(got, test.item)
			}
			if total < test.minimum {
				t.Fatalf("32 mature harvests gave %d %s, want at least %d", total, test.item, test.minimum)
			}
		})
	}
	wantDrop(t, drops("torchflower_crop", "", map[string]string{"age": "1"}), "minecraft:torchflower_seeds", 1)
}

func TestShearsAndSilkTouchConditions(t *testing.T) {
	wantDrop(t, drops("oak_leaves", "minecraft:shears"), "minecraft:oak_leaves", 1)
	stone := Drops(Context{
		Block:        block("stone"),
		Tool:         player.ItemStack{ItemID: "minecraft:diamond_pickaxe", Count: 1},
		Enchantments: map[string]int{"minecraft:silk_touch": 1},
		Random:       rand.New(rand.NewSource(2)),
	})
	wantDrop(t, stone, "minecraft:stone", 1)
}

func TestBlockExperienceRequiresToolAndRejectsSilkTouch(t *testing.T) {
	ctx := Context{Block: block("diamond_ore"), Tool: player.ItemStack{ItemID: "minecraft:iron_pickaxe", Count: 1}, Random: rand.New(rand.NewSource(1))}
	if got := Experience(ctx); got < 3 || got > 7 {
		t.Fatalf("diamond XP = %d, want 3..7", got)
	}
	ctx.Tool.ItemID = "minecraft:wooden_pickaxe"
	if got := Experience(ctx); got != 0 {
		t.Fatalf("wrong-tool diamond XP = %d, want 0", got)
	}
	ctx.Tool.ItemID = "minecraft:iron_pickaxe"
	ctx.Enchantments = map[string]int{"minecraft:silk_touch": 1}
	if got := Experience(ctx); got != 0 {
		t.Fatalf("Silk Touch diamond XP = %d, want 0", got)
	}
}

func TestDoublePlantChecksOtherHalf(t *testing.T) {
	lower := block("large_fern", map[string]string{"half": "lower"})
	got := Drops(Context{
		Block:  lower,
		Tool:   player.ItemStack{ItemID: "minecraft:shears", Count: 1},
		Random: rand.New(rand.NewSource(3)),
		BlockAt: func(_, dy, _ int) coreworld.Block {
			if dy == 1 {
				return block("large_fern", map[string]string{"half": "upper"})
			}
			return coreworld.Air
		},
	})
	wantDrop(t, got, "minecraft:fern", 2)
}

func wantDrop(t *testing.T, got []player.ItemStack, item string, amount int) {
	t.Helper()
	if len(got) != 1 || got[0].ItemID != item || got[0].Count != amount {
		t.Fatalf("drops = %+v, want %s x%d", got, item, amount)
	}
}

func count(stacks []player.ItemStack, item string) int {
	total := 0
	for _, stack := range stacks {
		if stack.ItemID == item {
			total += stack.Count
		}
	}
	return total
}
