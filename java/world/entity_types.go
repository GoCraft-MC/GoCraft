package world

// entityTypeIDs maps canonical entity resource locations to their numeric
// registry index for Minecraft Java Edition 1.21.4 (protocol 769).
//
// Registry indices are assigned alphabetically in vanilla.  The values below
// are estimates derived from counting the full 1.21.4 entity type list
// (including entries added in 1.20–1.21: bogged, breeze, breeze_wind_charge,
// camel, interaction, ominous_item_spawner, sniffer, wind_charge).
//
// Milestone 13 (data-driven registries) will replace this table by loading
// the real mapping from Minecraft's generated reports/registries.json.
// Verify against a 1.21.4 client capture if an entity spawns as the wrong
// type or causes the client to disconnect.
var entityTypeIDs = map[string]int32{
	// ── Passive mobs ─────────────────────────────────────────────────────────
	"minecraft:allay":   0,
	"minecraft:axolotl": 4,
	"minecraft:bat":     5,
	"minecraft:camel":   13,
	"minecraft:cat":     14,
	"minecraft:chicken": 18,
	"minecraft:cod":     19,
	"minecraft:cow":     21,
	"minecraft:donkey":  25,
	"minecraft:fox":     43,
	"minecraft:frog":    44,
	"minecraft:goat":    50,
	"minecraft:horse":   54,
	"minecraft:mooshroom": 69,
	"minecraft:mule":    70,
	"minecraft:ocelot":  71,
	"minecraft:panda":   74,
	"minecraft:parrot":  75,
	"minecraft:pig":     77,
	"minecraft:polar_bear": 81,
	"minecraft:pufferfish": 83,
	"minecraft:rabbit":  84,
	"minecraft:salmon":  86,
	"minecraft:sheep":   87,
	"minecraft:sniffer": 95,
	"minecraft:squid":   101,
	"minecraft:strider": 103,
	"minecraft:tadpole": 104,
	"minecraft:tropical_fish": 110,
	"minecraft:turtle":  111,
	"minecraft:wolf":    122,

	// ── Hostile mobs ─────────────────────────────────────────────────────────
	"minecraft:blaze":         7,
	"minecraft:bogged":        10,
	"minecraft:breeze":        11,
	"minecraft:cave_spider":   15,
	"minecraft:creeper":       22,
	"minecraft:drowned":       27,
	"minecraft:elder_guardian": 29,
	"minecraft:enderman":      33,
	"minecraft:endermite":     34,
	"minecraft:evoker":        35,
	"minecraft:ghast":         46,
	"minecraft:guardian":      51,
	"minecraft:hoglin":        52,
	"minecraft:husk":          55,
	"minecraft:magma_cube":    66,
	"minecraft:phantom":       76,
	"minecraft:piglin":        78,
	"minecraft:piglin_brute":  79,
	"minecraft:pillager":      80,
	"minecraft:ravager":       85,
	"minecraft:shulker":       88,
	"minecraft:silverfish":    90,
	"minecraft:skeleton":      91,
	"minecraft:slime":         93,
	"minecraft:spider":        100,
	"minecraft:stray":         102,
	"minecraft:vex":           112,
	"minecraft:vindicator":    114,
	"minecraft:warden":        116,
	"minecraft:witch":         118,
	"minecraft:wither_skeleton": 120,
	"minecraft:zoglin":        123,
	"minecraft:zombie":        124,
	"minecraft:zombie_villager": 126,
	"minecraft:zombified_piglin": 127,

	// ── Neutral mobs ─────────────────────────────────────────────────────────
	"minecraft:bee":        6,
	"minecraft:iron_golem": 58,
	"minecraft:snow_golem": 96,

	// ── Bosses ───────────────────────────────────────────────────────────────
	"minecraft:ender_dragon": 31,
	"minecraft:wither":       119,

	// ── Villagers / traders ───────────────────────────────────────────────────
	"minecraft:villager":         113,
	"minecraft:wandering_trader": 115,

	// ── Players (for completeness; spawned via dedicated player packet) ───────
	"minecraft:player": 130,
}

// EntityTypeID returns the numeric registry index for the given canonical
// entity resource location, or -1 if the entity type is not in the table.
func EntityTypeID(resourceLocation string) int32 {
	id, ok := entityTypeIDs[resourceLocation]
	if !ok {
		return -1
	}
	return id
}
