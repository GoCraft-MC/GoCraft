package bedrock

import (
	_ "embed"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"GoCraft/core/player"
	javaworld "GoCraft/java/world"

	dfworld "github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

//go:embed creative_items.nbt
var creativeItemsNBT []byte

// creativeNBTItem mirrors one item entry in Dragonfly's creative_items.nbt.
type creativeNBTItem struct {
	Name            string         `nbt:"name"`
	Meta            int16          `nbt:"meta"`
	NBTData         map[string]any `nbt:"nbt,omitempty"`
	BlockProperties map[string]any `nbt:"block_properties,omitempty"`
	GroupIndex      int32          `nbt:"group_index,omitempty"`
}

// creativeNBTGroup mirrors one creative group entry.
type creativeNBTGroup struct {
	Category int32           `nbt:"category"`
	Name     string          `nbt:"name"`
	Icon     creativeNBTItem `nbt:"icon"`
}

// creativeKnownItem is the canonical identity associated with a creative
// network ID. The simulation uses this when a Bedrock player selects an item.
type creativeKnownItem struct {
	name string
	meta int16
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

// initCreativeContent loads the embedded vanilla creative catalogue and
// re-encodes it with Pumpkin's current Bedrock item palette.
//
// This intentionally mirrors Pumpkin's behaviour:
//   - anonymous groups receive stable internal names such as "anon0";
//   - catalogue ordering, group indices, aux values, and NBT are preserved;
//   - exact Bedrock name/aux mappings are preferred;
//   - Bedrock-name fallback mirrors Pumpkin's runtime-ID fallback;
//   - entries absent from Pumpkin's mappings fall back to Dragonfly resolution;
//   - genuinely unavailable entries are skipped.
func (l *Listener) initCreativeContent() {
	// Item stacks that represent blocks need the registry's stable network
	// hashes. Finalize is idempotent and makes this initializer safe on its own.
	dfworld.DefaultBlockRegistry.Finalize()
	var root struct {
		Groups []creativeNBTGroup `nbt:"groups"`
		Items  []creativeNBTItem  `nbt:"items"`
	}
	if err := nbt.Unmarshal(creativeItemsNBT, &root); err != nil {
		slog.Warn(
			"bedrock: could not parse creative_items.nbt; creative menu will be empty",
			"err", err,
		)
		return
	}

	l.creativeGroups = make([]protocol.CreativeGroup, 0, len(root.Groups))
	for index, group := range root.Groups {
		name := group.Name
		if name == "" {
			name = fmt.Sprintf("anon%d", index)
		}

		icon, ok := creativeItemStack(group.Icon)
		if !ok {
			slog.Debug(
				"bedrock: creative group icon unavailable",
				"group_index", index,
				"group_name", name,
				"icon", group.Icon.Name,
				"meta", group.Icon.Meta,
			)
			icon = protocol.ItemStack{}
		}

		l.creativeGroups = append(l.creativeGroups, protocol.CreativeGroup{
			Category: byte(group.Category),
			Name:     name,
			Icon:     icon,
		})
	}

	l.creativeItems = make([]protocol.CreativeItem, 0, len(root.Items))
	l.creativeNames = make(map[uint32]creativeKnownItem, len(root.Items))

	var skipped int
	for _, entry := range root.Items {
		if entry.GroupIndex < 0 || entry.GroupIndex >= int32(len(l.creativeGroups)) {
			slog.Warn(
				"bedrock: creative item has invalid group index",
				"name", entry.Name,
				"meta", entry.Meta,
				"group_index", entry.GroupIndex,
				"group_count", len(l.creativeGroups),
			)
			skipped++
			continue
		}

		stack, ok := creativeItemStack(entry)
		if !ok {
			slog.Debug(
				"bedrock: creative item unavailable",
				"name", entry.Name,
				"meta", entry.Meta,
				"group_index", entry.GroupIndex,
			)
			skipped++
			continue
		}

		creativeNetworkID := uint32(len(l.creativeItems) + 1)
		l.creativeItems = append(l.creativeItems, protocol.CreativeItem{
			CreativeItemNetworkID: creativeNetworkID,
			Item:                  stack,
			GroupIndex:            uint32(entry.GroupIndex),
		})
		canonicalName, canonicalMeta := canonicalCreativeIdentity(entry.Name, entry.Meta)
		l.creativeNames[creativeNetworkID] = creativeKnownItem{name: canonicalName, meta: canonicalMeta}
	}

	slog.Info(
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

// creativeItemStack converts one NBT catalogue entry into a protocol stack.
func creativeItemStack(data creativeNBTItem) (protocol.ItemStack, bool) {
	if data.Name == "" {
		return protocol.ItemStack{}, false
	}
	if current, found := creativeCurrentMapping(data.Name, data.Meta); found {
		return currentCreativeItemStack(data, current), true
	}
	if current, found := creativeCurrentMappingByName(data.Name); found {
		return currentCreativeItemStack(data, current), true
	}

	var (
		resolvedItem dfworld.Item
		ok           bool
		blockRID     int32
	)

	if len(data.BlockProperties) > 0 {
		block, found := dfworld.BlockByName(data.Name, data.BlockProperties)
		if !found {
			return protocol.ItemStack{}, false
		}

		resolvedItem, ok = block.(dfworld.Item)
		if !ok {
			return protocol.ItemStack{}, false
		}
		blockRID = creativeBlockNetworkID(block)
	} else {
		resolvedItem, ok = dfworld.ItemByName(data.Name, data.Meta)
		if !ok {
			return protocol.ItemStack{}, false
		}

		_, resultingMeta := resolvedItem.EncodeItem()
		if resultingMeta != data.Meta {
			return protocol.ItemStack{}, false
		}
	}

	if decoder, ok := resolvedItem.(dfworld.NBTer); ok && len(data.NBTData) > 0 {
		decoded := decoder.DecodeNBT(cloneCreativeNBT(data.NBTData))
		item, valid := decoded.(dfworld.Item)
		if !valid {
			return protocol.ItemStack{}, false
		}
		resolvedItem = item
	}

	runtimeID, metadata, ok := dfworld.ItemRuntimeID(resolvedItem)
	if !ok {
		return protocol.ItemStack{}, false
	}
	if blockRID == 0 {
		if block, ok := resolvedItem.(dfworld.Block); ok {
			blockRID = creativeBlockNetworkID(block)
		}
	}

	return protocol.ItemStack{
		ItemType: protocol.ItemType{
			NetworkID:     runtimeID,
			MetadataValue: uint32(uint16(metadata)),
		},
		Count:          1,
		NBTData:        cloneCreativeNBT(data.NBTData),
		BlockRuntimeID: blockRID,
	}, true
}

func currentCreativeItemStack(data creativeNBTItem, current creativeBedrockMapping) protocol.ItemStack {
	// This follows Pumpkin's CCreativeContent construction: the runtime ID
	// comes from Pumpkin's current palette, while the catalogue's aux value is
	// preserved. In particular, creative block variants commonly advertise aux
	// zero even though their normal Java-to-Bedrock inventory mapping does not.
	return protocol.ItemStack{
		ItemType: protocol.ItemType{
			NetworkID:     current.mapping.runtimeID,
			MetadataValue: uint32(uint16(data.Meta)),
		},
		Count:   1,
		NBTData: cloneCreativeNBT(data.NBTData),
	}
}

func creativeBlockNetworkID(block dfworld.Block) int32 {
	runtimeID := dfworld.BlockRuntimeID(block)
	networkID, ok := dfworld.DefaultBlockRegistry.RuntimeIDToHash(runtimeID)
	if !ok {
		return 0
	}
	return int32(networkID)
}

func cloneCreativeNBT(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	out := make(map[string]any, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
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
	return fmt.Sprintf("anon%d", index)
}
