// Package blockloot evaluates Minecraft Java 1.21.4 block loot tables for the
// edition-neutral world model. Both protocol adapters use this package so
// drops and harvest-tool checks stay identical for Java and Bedrock players.
package blockloot

import (
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"sync"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/internal/gamedata"
)

type bundle struct {
	Version      string                     `json:"version"`
	LootTables   map[string]json.RawMessage `json:"loot_tables"`
	BlockTags    map[string][]string        `json:"block_tags"`
	ItemTags     map[string][]string        `json:"item_tags"`
	ToolRequired []string                   `json:"tool_required"`
}

type database struct {
	bundle
	toolRequiredSet map[string]bool
	blockTagSets    map[string]map[string]bool
	itemTagSets     map[string]map[string]bool
	tables          map[string]map[string]any
}

// Context contains the state that vanilla loot conditions may inspect.
// Enchantments is optional until GoCraft stores enchantments on ItemStack.
type Context struct {
	Block        coreworld.Block
	Tool         player.ItemStack
	Enchantments map[string]int
	Explosion    float64
	Random       *rand.Rand
	BlockAt      func(dx, dy, dz int) coreworld.Block
}

var (
	loadOnce sync.Once
	loaded   database
)

// Drops returns the exact normal-break drops selected by the embedded 1.21.4
// loot table. Blocks requiring a correct harvest tool return no drops when the
// held item has the wrong type or tier.
func Drops(ctx Context) []player.ItemStack {
	db := data()
	blockID := ctx.Block.ResourceLocation()
	if db.toolRequiredSet[blockID] && !db.correctTool(blockID, ctx.Tool.ItemID) {
		return nil
	}
	table, ok := db.tables[blockID]
	if !ok {
		return nil
	}
	var drops []player.ItemStack
	for _, poolValue := range list(table["pools"]) {
		pool := object(poolValue)
		if !conditionsPass(list(pool["conditions"]), ctx, db) {
			continue
		}
		rolls := int(math.Round(number(pool["rolls"], ctx)))
		for range rolls {
			entries := eligibleEntries(list(pool["entries"]), ctx, db)
			if len(entries) == 0 {
				continue
			}
			entry := weightedEntry(entries, ctx)
			stacks, _ := evaluateReadyEntry(entry, ctx, db)
			stacks = applyFunctions(stacks, list(pool["functions"]), ctx, db)
			drops = append(drops, stacks...)
		}
	}
	drops = applyFunctions(drops, list(table["functions"]), ctx, db)
	return merge(drops)
}

// CanHarvest reports whether toolID satisfies vanilla's requires-correct-tool
// flag for blockID. A block without that flag can always produce its loot.
func CanHarvest(blockID, toolID string) bool {
	db := data()
	return !db.toolRequiredSet[blockID] || db.correctTool(blockID, toolID)
}

// TableCount is exposed for registry completeness tests and diagnostics.
func TableCount() int { return len(data().LootTables) }

func data() *database {
	loadOnce.Do(func() {
		raw, err := gamedata.FS.ReadFile("java/1.21.4/block_loot.json")
		if err != nil || json.Unmarshal(raw, &loaded.bundle) != nil {
			panic("blockloot: invalid embedded Java 1.21.4 loot data")
		}
		loaded.toolRequiredSet = make(map[string]bool, len(loaded.ToolRequired))
		for _, name := range loaded.ToolRequired {
			loaded.toolRequiredSet[name] = true
		}
		loaded.blockTagSets = sets(loaded.BlockTags)
		loaded.itemTagSets = sets(loaded.ItemTags)
		loaded.tables = make(map[string]map[string]any, len(loaded.LootTables))
		for name, rawTable := range loaded.LootTables {
			var table map[string]any
			if json.Unmarshal(rawTable, &table) != nil {
				panic("blockloot: invalid loot table " + name)
			}
			loaded.tables[name] = table
		}
	})
	return &loaded
}

func sets(tags map[string][]string) map[string]map[string]bool {
	result := make(map[string]map[string]bool, len(tags))
	for tag, values := range tags {
		set := make(map[string]bool, len(values))
		for _, value := range values {
			set[value] = true
		}
		result[tag] = set
	}
	return result
}

func (db *database) correctTool(blockID, toolID string) bool {
	if toolID == "minecraft:shears" || strings.HasSuffix(toolID, "_sword") {
		return blockID == "minecraft:cobweb"
	}
	name := strings.TrimPrefix(toolID, "minecraft:")
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return false
	}
	material, kind := parts[0], parts[1]
	if kind != "pickaxe" && kind != "axe" && kind != "shovel" && kind != "hoe" {
		return false
	}
	if !db.blockTagSets["minecraft:mineable/"+kind][blockID] {
		return false
	}
	if material == "golden" {
		material = "gold"
	}
	return !db.blockTagSets["minecraft:incorrect_for_"+material+"_tool"][blockID]
}

func eligibleEntries(values []any, ctx Context, db *database) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		entry := object(value)
		if conditionsPass(list(entry["conditions"]), ctx, db) {
			result = append(result, entry)
		}
	}
	return result
}

func weightedEntry(entries []map[string]any, ctx Context) map[string]any {
	total := 0
	for _, entry := range entries {
		weight := intValue(entry["weight"], 1)
		if weight > 0 {
			total += weight
		}
	}
	if total <= 1 {
		return entries[0]
	}
	choice := randomInt(ctx, total)
	for _, entry := range entries {
		choice -= intValue(entry["weight"], 1)
		if choice < 0 {
			return entry
		}
	}
	return entries[len(entries)-1]
}

func evaluateEntry(entry map[string]any, ctx Context, db *database) ([]player.ItemStack, bool) {
	if !conditionsPass(list(entry["conditions"]), ctx, db) {
		return nil, false
	}
	return evaluateReadyEntry(entry, ctx, db)
}

// evaluateReadyEntry evaluates an entry whose own conditions have already
// passed. Pool expansion performs this check before weighted selection, so it
// must not be sampled a second time for random conditions.
func evaluateReadyEntry(entry map[string]any, ctx Context, db *database) ([]player.ItemStack, bool) {
	switch stringValue(entry["type"]) {
	case "minecraft:item":
		name := stringValue(entry["name"])
		if name == "" {
			return nil, false
		}
		stacks := []player.ItemStack{{ItemID: name, Count: 1}}
		return applyFunctions(stacks, list(entry["functions"]), ctx, db), true
	case "minecraft:alternatives":
		for _, child := range list(entry["children"]) {
			if stacks, success := evaluateEntry(object(child), ctx, db); success {
				return applyFunctions(stacks, list(entry["functions"]), ctx, db), true
			}
		}
		return nil, false
	case "minecraft:dynamic":
		// Dynamic container contents are handled by the world container store.
		return nil, true
	default:
		return nil, false
	}
}

func conditionsPass(values []any, ctx Context, db *database) bool {
	for _, value := range values {
		if !conditionPass(object(value), ctx, db) {
			return false
		}
	}
	return true
}

func conditionPass(condition map[string]any, ctx Context, db *database) bool {
	switch stringValue(condition["condition"]) {
	case "minecraft:any_of":
		for _, term := range list(condition["terms"]) {
			if conditionPass(object(term), ctx, db) {
				return true
			}
		}
		return false
	case "minecraft:inverted":
		return !conditionPass(object(condition["term"]), ctx, db)
	case "minecraft:block_state_property":
		if stringValue(condition["block"]) != ctx.Block.ResourceLocation() {
			return false
		}
		return propertiesMatch(ctx.Block, object(condition["properties"]))
	case "minecraft:match_tool":
		return itemPredicate(object(condition["predicate"]), ctx, db)
	case "minecraft:random_chance":
		return randomFloat(ctx) < floatValue(condition["chance"])
	case "minecraft:table_bonus":
		chances := list(condition["chances"])
		if len(chances) == 0 {
			return false
		}
		level := ctx.Enchantments[stringValue(condition["enchantment"])]
		if level >= len(chances) {
			level = len(chances) - 1
		}
		return randomFloat(ctx) < floatValue(chances[level])
	case "minecraft:survives_explosion":
		return ctx.Explosion <= 0 || randomFloat(ctx) <= 1/ctx.Explosion
	case "minecraft:entity_properties":
		return true // Block breaking always supplies the breaking player as "this".
	case "minecraft:location_check":
		if ctx.BlockAt == nil {
			return false
		}
		near := ctx.BlockAt(intValue(condition["offsetX"], 0), intValue(condition["offsetY"], 0), intValue(condition["offsetZ"], 0))
		blockPredicate := object(object(condition["predicate"])["block"])
		if !matchesID(near.ResourceLocation(), blockPredicate["blocks"], db.blockTagSets) {
			return false
		}
		return propertiesMatch(near, object(blockPredicate["state"]))
	default:
		return false
	}
}

func itemPredicate(predicate map[string]any, ctx Context, db *database) bool {
	if items, ok := predicate["items"]; ok && !matchesID(ctx.Tool.ItemID, items, db.itemTagSets) {
		return false
	}
	predicates := object(predicate["predicates"])
	for key, value := range predicates {
		if key != "minecraft:enchantments" {
			return false
		}
		for _, required := range list(value) {
			requirement := object(required)
			level := ctx.Enchantments[stringValue(requirement["enchantments"])]
			limits := object(requirement["levels"])
			if min, ok := limits["min"]; ok && level < int(floatValue(min)) {
				return false
			}
			if max, ok := limits["max"]; ok && level > int(floatValue(max)) {
				return false
			}
		}
	}
	return true
}

func matchesID(id string, value any, tags map[string]map[string]bool) bool {
	for _, candidate := range stringsValue(value) {
		if strings.HasPrefix(candidate, "#") {
			if tags[strings.TrimPrefix(candidate, "#")][id] {
				return true
			}
		} else if candidate == id {
			return true
		}
	}
	return false
}

func propertiesMatch(block coreworld.Block, wanted map[string]any) bool {
	for key, value := range wanted {
		if block.Properties[key] != stringValue(value) {
			return false
		}
	}
	return true
}

func applyFunctions(stacks []player.ItemStack, functions []any, ctx Context, db *database) []player.ItemStack {
	for _, value := range functions {
		function := object(value)
		if !conditionsPass(list(function["conditions"]), ctx, db) {
			continue
		}
		switch stringValue(function["function"]) {
		case "minecraft:set_count":
			count := int(math.Round(number(function["count"], ctx)))
			for index := range stacks {
				if boolValue(function["add"]) {
					stacks[index].Count += count
				} else {
					stacks[index].Count = count
				}
			}
		case "minecraft:limit_count":
			limit := object(function["limit"])
			for index := range stacks {
				if min, ok := limit["min"]; ok && stacks[index].Count < int(math.Round(number(min, ctx))) {
					stacks[index].Count = int(math.Round(number(min, ctx)))
				}
				if max, ok := limit["max"]; ok && stacks[index].Count > int(math.Round(number(max, ctx))) {
					stacks[index].Count = int(math.Round(number(max, ctx)))
				}
			}
		case "minecraft:explosion_decay":
			if ctx.Explosion > 0 {
				chance := 1 / ctx.Explosion
				for index := range stacks {
					kept := 0
					for range stacks[index].Count {
						if randomFloat(ctx) <= chance {
							kept++
						}
					}
					stacks[index].Count = kept
				}
			}
		case "minecraft:apply_bonus":
			level := ctx.Enchantments[stringValue(function["enchantment"])]
			if level <= 0 {
				continue
			}
			for index := range stacks {
				switch stringValue(function["formula"]) {
				case "minecraft:ore_drops":
					multiplier := randomInt(ctx, level+2) - 1
					if multiplier > 0 {
						stacks[index].Count *= multiplier + 1
					}
				case "minecraft:uniform_bonus_count":
					parameters := object(function["parameters"])
					stacks[index].Count += randomInt(ctx, int(floatValue(parameters["bonusMultiplier"]))*level+1)
				case "minecraft:binomial_with_bonus_count":
					parameters := object(function["parameters"])
					trials := level + int(floatValue(parameters["extra"]))
					probability := floatValue(parameters["probability"])
					for range trials {
						if randomFloat(ctx) < probability {
							stacks[index].Count++
						}
					}
				}
			}
		case "minecraft:copy_components", "minecraft:copy_state":
			// ItemStack currently stores identity/count/damage only. These
			// functions do not alter which item or how many items are dropped.
		}
	}
	return stacks
}

func number(value any, ctx Context) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case map[string]any:
		switch stringValue(typed["type"]) {
		case "minecraft:uniform":
			min, max := number(typed["min"], ctx), number(typed["max"], ctx)
			if min == math.Trunc(min) && max == math.Trunc(max) && max >= min {
				return min + float64(randomInt(ctx, int(max-min)+1))
			}
			return min + randomFloat(ctx)*(max-min)
		case "minecraft:binomial":
			n := int(math.Round(number(typed["n"], ctx)))
			p, result := number(typed["p"], ctx), 0
			for range n {
				if randomFloat(ctx) < p {
					result++
				}
			}
			return float64(result)
		}
	}
	return 0
}

func merge(stacks []player.ItemStack) []player.ItemStack {
	result := make([]player.ItemStack, 0, len(stacks))
	for _, stack := range stacks {
		if stack.IsEmpty() {
			continue
		}
		found := false
		for index := range result {
			if result[index].ItemID == stack.ItemID && result[index].Damage == stack.Damage {
				result[index].Count += stack.Count
				found = true
				break
			}
		}
		if !found {
			result = append(result, stack)
		}
	}
	return result
}

func randomFloat(ctx Context) float64 {
	if ctx.Random != nil {
		return ctx.Random.Float64()
	}
	return rand.Float64()
}

func randomInt(ctx Context, n int) int {
	if n <= 1 {
		return 0
	}
	if ctx.Random != nil {
		return ctx.Random.Intn(n)
	}
	return rand.Intn(n)
}

func object(value any) map[string]any {
	if value == nil {
		return nil
	}
	result, _ := value.(map[string]any)
	return result
}

func list(value any) []any {
	result, _ := value.([]any)
	return result
}

func stringsValue(value any) []string {
	if single, ok := value.(string); ok {
		return []string{single}
	}
	values := list(value)
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, stringValue(value))
	}
	return result
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}

func floatValue(value any) float64 {
	result, _ := value.(float64)
	return result
}

func intValue(value any, fallback int) int {
	if value == nil {
		return fallback
	}
	return int(floatValue(value))
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}
