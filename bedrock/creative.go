package bedrock

import (
	"bytes"
	"encoding/base64"
	"log/slog"
	"strconv"
	"strings"

	"GoCraft/core/player"
	"GoCraft/internal/debuglog"
	javaworld "GoCraft/java/world"

	dfworld "github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// creativeKnownItem is the canonical identity associated with a creative
// network ID. The simulation uses this when a Bedrock player selects an item.
type creativeKnownItem struct {
	name         string
	meta         int16
	hasFireworks bool
	fireworks    player.FireworkData
}

type creativeBedrockIdentity struct {
	name string
	meta uint32
}

type creativeBedrockMapping struct {
	javaName string
	mapping  bedrockItemMapping
}

// creativeBedrockMappings reverses Pumpkin's generated Java-to-Bedrock table.
// The embedded creative catalogue supplies its ordering, grouping, aux values,
// and NBT variants, but its sequential item runtime IDs belong to a different
// Bedrock palette. Inventory stacks must use the IDs advertised by the current
// protocol's Pumpkin table.
var creativeBedrockMappings = func() map[creativeBedrockIdentity]creativeBedrockMapping {
	out := make(map[creativeBedrockIdentity]creativeBedrockMapping, len(javaToBedrockItemMappings))
	for javaName, mapping := range javaToBedrockItemMappings {
		key := creativeBedrockIdentity{name: mapping.name, meta: mapping.metadata}
		candidate := creativeBedrockMapping{javaName: javaName, mapping: mapping}
		current, exists := out[key]
		if !exists || preferCreativeJavaName(candidate.javaName, current.javaName, mapping.name) {
			out[key] = candidate
		}
	}
	return out
}()

// creativeBedrockMappingsByName mirrors Pumpkin's runtime-ID-only fallback in
// JavaToBedrockItemMapping::from_bedrock. Creative aux values are not always
// identical to the aux value used when the same Java item is placed in an
// inventory (planks are one example), but both stacks must still use the same
// current Bedrock runtime ID.
var creativeBedrockMappingsByName = func() map[string]creativeBedrockMapping {
	out := make(map[string]creativeBedrockMapping, len(javaToBedrockItemMappings))
	for javaName, mapping := range javaToBedrockItemMappings {
		candidate := creativeBedrockMapping{javaName: javaName, mapping: mapping}
		current, exists := out[mapping.name]
		if !exists || preferCreativeJavaName(candidate.javaName, current.javaName, mapping.name) {
			out[mapping.name] = candidate
		}
	}
	return out
}()

// pumpkinCreativeRuntimeIDsByName contains the complete current Bedrock item
// palette used by Pumpkin's creative catalogue. It covers newer Bedrock-only
// identities that do not yet have a Java mapping in GoCraft.
var pumpkinCreativeRuntimeIDsByName = func() map[string]int32 {
	out := make(map[string]int32, len(pumpkinCreativeItems))
	for _, entry := range pumpkinCreativeItems {
		out[entry.name] = entry.runtimeID
	}
	return out
}()

func pumpkinCreativeRuntimeID(name string) (int32, bool) {
	runtimeID, ok := pumpkinCreativeRuntimeIDsByName[name]
	return runtimeID, ok
}

func pumpkinInventoryRuntimeID(name string, mapping bedrockItemMapping, mapped bool) (int32, bool) {
	runtimeID, ok := pumpkinCreativeRuntimeID(name)
	if !ok {
		return 0, false
	}
	// Keep legitimate legacy-name conversions such as Java stone_stairs to
	// Bedrock normal_stone_stairs. Only bypass mappings that explicitly fell
	// back to unknown (plus the current bounce-disc compatibility fallback).
	return runtimeID, !mapped || mapping.name == "minecraft:unknown" || name == "minecraft:music_disc_bounce"
}

func preferCreativeJavaName(candidate, current, bedrockName string) bool {
	candidateKnown := javaworld.ItemID(candidate) >= 0
	currentKnown := javaworld.ItemID(current) >= 0
	if candidateKnown != currentKnown {
		return candidateKnown
	}
	candidateExact := candidate == bedrockName
	currentExact := current == bedrockName
	if candidateExact != currentExact {
		return candidateExact
	}
	return candidate < current
}

func creativeCurrentMapping(name string, meta int16) (creativeBedrockMapping, bool) {
	mapping, ok := creativeBedrockMappings[creativeBedrockIdentity{
		name: name,
		meta: uint32(uint16(meta)),
	}]
	return mapping, ok
}

func creativeCurrentMappingByName(name string) (creativeBedrockMapping, bool) {
	mapping, ok := creativeBedrockMappingsByName[name]
	return mapping, ok
}

// initCreativeContent publishes the catalogue generated from the same current
// Pumpkin creative JSON as the item palette. GoCraft retains the authoritative
// NBT variants and block network IDs from that file: dropping them turns 125
// enchanted books and many other variants into duplicate/incomplete entries,
// which makes current Bedrock clients crash while building the search index.
func (l *Listener) initCreativeContent() {
	l.creativeGroups = make([]protocol.CreativeGroup, 0, len(pumpkinCreativeGroups))
	for index, group := range pumpkinCreativeGroups {
		icon, ok := pumpkinCreativeItemStack(group.iconName, group.iconMeta, group.iconRuntimeID)
		if !ok {
			slog.Debug(
				"bedrock: creative group icon unavailable",
				"group_index", index,
				"group_name", creativeGroupDebugName(index, group.name),
				"icon", group.iconName,
				"meta", group.iconMeta,
			)
			icon = protocol.ItemStack{}
		}

		l.creativeGroups = append(l.creativeGroups, protocol.CreativeGroup{
			Category: byte(group.category),
			Name:     group.name,
			Icon:     icon,
		})
	}

	l.creativeItems = make([]protocol.CreativeItem, 0, len(pumpkinCreativeItems))
	l.creativeNames = make(map[uint32]creativeKnownItem, len(pumpkinCreativeItems))

	var skipped int
	for _, entry := range pumpkinCreativeItems {
		if entry.group >= uint32(len(l.creativeGroups)) {
			slog.Warn(
				"bedrock: creative item has invalid group index",
				"name", entry.name,
				"meta", entry.meta,
				"group_index", entry.group,
				"group_count", len(l.creativeGroups),
			)
			skipped++
			continue
		}

		stack, ok := pumpkinCreativeCatalogueStack(entry)
		if !ok {
			slog.Debug(
				"bedrock: creative item unavailable",
				"name", entry.name,
				"meta", entry.meta,
				"group_index", entry.group,
			)
			skipped++
			continue
		}

		creativeNetworkID := uint32(len(l.creativeItems) + 1)
		l.creativeItems = append(l.creativeItems, protocol.CreativeItem{
			CreativeItemNetworkID: creativeNetworkID,
			Item:                  stack,
			GroupIndex:            entry.group,
		})
		canonicalName, canonicalMeta := canonicalCreativeIdentity(entry.name, entry.meta)
		known := creativeKnownItem{name: canonicalName, meta: canonicalMeta}
		if canonicalName == "minecraft:firework_rocket" {
			known.fireworks, known.hasFireworks = bedrockFireworkDataFromNBT(stack.NBTData)
		}
		l.creativeNames[creativeNetworkID] = known
	}

	debuglog.Info(debuglog.BedrockCatalogues,
		"bedrock: creative catalogue ready",
		"groups", len(l.creativeGroups),
		"items", len(l.creativeItems),
		"skipped", skipped,
	)
}

func canonicalCreativeIdentity(name string, meta int16) (string, int16) {
	const lightPrefix = "minecraft:light_block_"
	if strings.HasPrefix(name, lightPrefix) {
		if level, err := strconv.Atoi(strings.TrimPrefix(name, lightPrefix)); err == nil && level >= 0 && level <= 15 {
			return "minecraft:light", int16(level)
		}
	}
	if name == "minecraft:light_block" {
		return "minecraft:light", meta
	}
	if mapping, ok := creativeCurrentMapping(name, meta); ok {
		// Pumpkin resolves the Bedrock ID/aux pair back to its Java item and then
		// constructs a fresh ItemStack. The auxiliary value selects the mapping;
		// it is not durability already consumed by the new stack.
		return mapping.javaName, 0
	}
	if mapping, ok := creativeCurrentMappingByName(name); ok {
		// Pumpkin falls back from the exact ID/aux pair to the Bedrock runtime
		// ID alone. It then creates a fresh Java stack, so the creative aux
		// value is a variant selector rather than consumed durability.
		return mapping.javaName, 0
	}
	if player.MaxDurability(name) > 0 {
		return name, 0
	}
	return name, meta
}

func pumpkinCreativeItemStack(name string, meta int16, runtimeID int32) (protocol.ItemStack, bool) {
	if name == "" || runtimeID == 0 {
		return protocol.ItemStack{}, false
	}
	return protocol.ItemStack{
		ItemType: protocol.ItemType{
			NetworkID:     runtimeID,
			MetadataValue: uint32(uint16(meta)),
		},
		Count: 1,
	}, true
}

func pumpkinCreativeCatalogueStack(entry pumpkinCreativeItemData) (protocol.ItemStack, bool) {
	stack, ok := pumpkinCreativeItemStack(entry.name, entry.meta, entry.runtimeID)
	if !ok {
		return protocol.ItemStack{}, false
	}
	stack.BlockRuntimeID = entry.blockRuntimeID
	if entry.nbtBase64 == "" {
		return stack, true
	}
	raw, err := base64.StdEncoding.DecodeString(entry.nbtBase64)
	if err != nil {
		panic("bedrock: decode generated Creative item NBT for " + entry.name + ": " + err.Error())
	}
	var data map[string]any
	if err := nbt.NewDecoderWithEncoding(bytes.NewReader(raw), nbt.LittleEndian).Decode(&data); err != nil {
		panic("bedrock: parse generated Creative item NBT for " + entry.name + ": " + err.Error())
	}
	stack.NBTData = data
	return stack, true
}

func creativeBlockNetworkID(block dfworld.Block) int32 {
	runtimeID := dfworld.BlockRuntimeID(block)
	networkID, ok := dfworld.DefaultBlockRegistry.RuntimeIDToHash(runtimeID)
	if !ok {
		return 0
	}
	return int32(networkID)
}

// creativePlayerStack resolves the creative network ID selected by the client
// back to the canonical item identity used by GoCraft.
func (l *Listener) creativePlayerStack(creativeNetID uint32, count int) (creativeKnownItem, bool) {
	item, ok := l.creativeNames[creativeNetID]
	return item, ok
}

// creativeGroupDebugName returns the same stable anonymous-group naming scheme
// used while building the protocol catalogue.
func creativeGroupDebugName(index int, name string) string {
	if name != "" {
		return name
	}
	return "anonymous#" + strconv.Itoa(index)
}
