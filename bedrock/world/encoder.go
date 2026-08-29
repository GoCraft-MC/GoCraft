package world

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	dfworld "github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

// currentBlockVersion is the version stamped into Pumpkin's Bedrock 1.26.40
// block-state palette. Persistent sub-chunk entries must use the same version
// as the network hashes resolved below.
const currentBlockVersion int32 = 18168865

// pumpkinBlockStatesGZIP is Pumpkin's current concatenated LittleEndian NBT
// block-state stream. It includes the stable network hash for every vanilla
// state, including both halves and every open/hinge combination of doors.
//
//go:embed block_states.nbt.gz
var pumpkinBlockStatesGZIP []byte

// persistentBlockEntry is the NBT structure of one entry in a persistent sub-chunk palette.
// It uses the same field names as Dragonfly's blockEntry struct in server/world/chunk/encode.go.
type persistentBlockEntry struct {
	Name    string         `nbt:"name"`
	States  map[string]any `nbt:"states"`
	Version int32          `nbt:"version"`
}

// Encoder translates GoCraft's edition-neutral blocks into stable Bedrock
// network block hashes using the same current palette as the item and Creative
// registries sent during login.
type Encoder struct {
	mu        sync.RWMutex
	byName    map[string][]bedrockState
	cache     map[string]uint32
	airHash   uint32
	stoneHash uint32
}

type bedrockState struct {
	name      string
	networkID uint32
	props     map[string]any
}

// NewEncoder prepares the vanilla Bedrock state lookup. Dragonfly embeds the
// block state palette for the same gophertunnel protocol family used here.
func NewEncoder() *Encoder {
	// Item/block interaction helpers still use Dragonfly's independent vanilla
	// registry for hardness and item behaviour. Preserve its one-time
	// initialisation even though network state resolution uses Pumpkin below.
	dfworld.DefaultBlockRegistry.Finalize()
	e := &Encoder{
		byName: make(map[string][]bedrockState),
		cache:  make(map[string]uint32),
	}
	e.loadPumpkinBlockStates()
	e.airHash = e.resolve(coreworld.Air)
	e.stoneHash = e.resolve(coreworld.Block{Namespace: "minecraft", Name: "stone"})
	return e
}

func (e *Encoder) loadPumpkinBlockStates() {
	compressed, err := gzip.NewReader(bytes.NewReader(pumpkinBlockStatesGZIP))
	if err != nil {
		panic(fmt.Errorf("bedrock: open embedded Pumpkin block states: %w", err))
	}
	defer compressed.Close()

	raw, err := io.ReadAll(compressed)
	if err != nil {
		panic(fmt.Errorf("bedrock: decompress embedded Pumpkin block states: %w", err))
	}
	reader := bytes.NewReader(raw)
	decoder := nbt.NewDecoderWithEncoding(reader, nbt.NetworkLittleEndian)
	count := 0
	for reader.Len() > 0 {
		var entry map[string]any
		err := decoder.Decode(&entry)
		if err != nil {
			panic(fmt.Errorf("bedrock: decode embedded Pumpkin block state %d: %w", count, err))
		}
		name, nameOK := entry["name"].(string)
		networkID, networkOK := entry["network_id"].(int32)
		states, statesOK := entry["states"].(map[string]any)
		if !nameOK || !networkOK || !statesOK {
			panic(fmt.Sprintf("bedrock: malformed Pumpkin block state %d (%T/%T/%T)", count, entry["name"], entry["network_id"], entry["states"]))
		}
		if name == "" {
			panic(fmt.Sprintf("bedrock: Pumpkin block state %d has no name", count))
		}
		e.byName[name] = append(e.byName[name], bedrockState{
			name: name, networkID: uint32(networkID), props: states,
		})
		count++
	}
	if count == 0 {
		panic("bedrock: embedded Pumpkin block-state palette is empty")
	}
}

// BlockNetworkID returns the stable network hash of a Bedrock block state.
// StartGame advertises UseBlockNetworkIDHashes, so every block ID sent to the
// client must use this value instead of a palette index. Hashes remain valid
// when the vanilla palette gains or reorders states between protocol versions.
func (e *Encoder) BlockNetworkID(block coreworld.Block) uint32 {
	key := block.Key()
	e.mu.RLock()
	if networkID, ok := e.cache[key]; ok {
		e.mu.RUnlock()
		return networkID
	}
	e.mu.RUnlock()

	networkID := e.resolve(block)
	e.mu.Lock()
	e.cache[key] = networkID
	e.mu.Unlock()
	return networkID
}

func (e *Encoder) resolve(block coreworld.Block) uint32 {
	block = bedrockVisualBlock(block)
	name := block.ResourceLocation()
	if block.IsAir() || name == "" {
		name = "minecraft:air"
	}
	candidates := e.byName[name]
	if len(candidates) == 0 {
		for _, alternate := range alternateBlockNames(name) {
			if candidates = e.byName[alternate]; len(candidates) != 0 {
				break
			}
		}
	}
	if len(candidates) == 0 {
		// Unknown Java-only states are rendered as air rather than corrupting
		// the palette with a non-existent Bedrock network hash.
		if e.airHash != 0 {
			return e.airHash
		}
		return 0
	}

	wanted := translateBlockProperties(block)
	best, bestScore := candidates[0], -1<<30
	for _, candidate := range candidates {
		score := stateScore(candidate.props, wanted)
		if score > bestScore {
			best, bestScore = candidate, score
		}
	}
	return best.networkID
}

// resolveState returns the Bedrock block name and property map for a canonical block.
// Used when encoding persistent palette entries (NBT compounds) in SubChunk responses.
func (e *Encoder) resolveState(block coreworld.Block) (string, map[string]any) {
	block = bedrockVisualBlock(block)
	name := block.ResourceLocation()
	if block.IsAir() || name == "" {
		return "minecraft:air", map[string]any{}
	}

	candidates := e.byName[name]
	if len(candidates) == 0 {
		for _, alternate := range alternateBlockNames(name) {
			if candidates = e.byName[alternate]; len(candidates) != 0 {
				break
			}
		}
	}
	if len(candidates) == 0 {
		return "minecraft:air", map[string]any{}
	}

	wanted := translateBlockProperties(block)
	best, bestScore := candidates[0], -1<<30
	for _, candidate := range candidates {
		score := stateScore(candidate.props, wanted)
		if score > bestScore {
			best, bestScore = candidate, score
		}
	}
	props := best.props
	if props == nil {
		props = map[string]any{}
	}
	return best.name, props
}

// bedrockVisualBlock applies edition-specific visual fallbacks without
// changing the canonical block stored in the world. Current Bedrock clients
// render Java's imported two-block tall_grass pair as two complete plants
// stacked on top of each other. A single short_grass at the lower position is
// the closest equivalent that renders consistently; the upper half is hidden.
func bedrockVisualBlock(block coreworld.Block) coreworld.Block {
	switch block.ResourceLocation() {
	case "minecraft:tall_grass":
		if block.Properties["half"] == "upper" {
			return coreworld.Air
		}
		return coreworld.Block{Namespace: "minecraft", Name: "short_grass"}
	case "minecraft:light":
		level, _ := strconv.Atoi(block.Properties["level"])
		if level < 0 {
			level = 0
		} else if level > 15 {
			level = 15
		}
		return coreworld.Block{Namespace: "minecraft", Name: "light_block_" + strconv.Itoa(level)}
	case "minecraft:snow":
		block.Name = "snow_layer"
	case "minecraft:nether_portal":
		// Java names the block nether_portal, while Bedrock exposes the same
		// block as minecraft:portal with a portal_axis state.
		block.Name = "portal"
	case "minecraft:redstone_torch", "minecraft:redstone_wall_torch":
		if block.Properties["lit"] == "false" {
			block.Name = "unlit_redstone_torch"
		}
	case "minecraft:furnace", "minecraft:blast_furnace", "minecraft:smoker":
		// Java stores the burning state as a lit property, while Bedrock uses
		// separate block identifiers for the burning furnace variants.
		if block.Properties["lit"] == "true" {
			block.Name = "lit_" + block.Name
		}
	case "minecraft:beetroots":
		// Bedrock uses the singular identifier for the same canonical crop.
		block.Name = "beetroot"
	case "minecraft:attached_pumpkin_stem":
		// Bedrock represents attachment through a mature stem next to its fruit,
		// rather than exposing Java's separate attached-stem block identifier.
		block.Name = "pumpkin_stem"
		block.Properties = map[string]string{"age": "7"}
	case "minecraft:attached_melon_stem":
		block.Name = "melon_stem"
		block.Properties = map[string]string{"age": "7"}
	}
	return block
}

func alternateBlockNames(name string) []string {
	switch name {
	case "minecraft:wall_torch":
		return []string{"minecraft:torch"}
	case "minecraft:soul_wall_torch":
		return []string{"minecraft:soul_torch"}
	case "minecraft:redstone_wall_torch":
		return []string{"minecraft:redstone_torch"}
	case "minecraft:snow_block":
		return []string{"minecraft:snow"}
	}
	if strings.HasSuffix(name, `_bed`) {
		return []string{`minecraft:bed`}
	}
	switch name {
	case `minecraft:oak_door`:
		return []string{`minecraft:wooden_door`}
	case "minecraft:lily_pad":
		return []string{"minecraft:waterlily"}
	case "minecraft:sugar_cane":
		return []string{"minecraft:reeds"}
	case "minecraft:cobweb":
		return []string{"minecraft:web"}
	case "minecraft:dirt_path":
		return []string{"minecraft:grass_path"}
	case "minecraft:short_grass":
		return []string{"minecraft:tallgrass"}
	}
	if strings.HasSuffix(name, "_wall_banner") {
		return []string{"minecraft:wall_banner"}
	}
	if strings.HasSuffix(name, "_banner") {
		return []string{"minecraft:standing_banner"}
	}
	if strings.HasSuffix(name, "_wall_sign") {
		wood := strings.TrimSuffix(strings.TrimPrefix(name, "minecraft:"), "_wall_sign")
		return []string{"minecraft:" + bedrockSignWood(wood) + "wall_sign"}
	}
	if strings.HasSuffix(name, "_sign") {
		wood := strings.TrimSuffix(strings.TrimPrefix(name, "minecraft:"), "_sign")
		return []string{"minecraft:" + bedrockSignWood(wood) + "standing_sign"}
	}
	return nil
}

func bedrockSignWood(wood string) string {
	switch wood {
	case "oak":
		return ""
	case "dark_oak":
		return "darkoak_"
	default:
		return wood + "_"
	}
}

func translateBlockProperties(block coreworld.Block) map[string]any {
	properties := block.Properties
	out := make(map[string]any, len(properties)*2)
	for key, raw := range properties {
		value := propertyValue(raw)
		out[key] = value
		switch key {
		case "axis":
			if block.ResourceLocation() == "minecraft:portal" {
				out["portal_axis"] = raw
			} else {
				out["pillar_axis"] = raw
			}
		case "eye":
			out["end_portal_eye_bit"] = boolByte(raw == "true")
		case "facing":
			out["minecraft:cardinal_direction"] = raw
			if direction, ok := cardinalDirection(raw); ok {
				out["direction"] = direction
			}
			if direction, ok := bedrockFacingDirection(raw); ok {
				out["facing_direction"] = direction
			}
			if direction, ok := stairDirection(raw); ok {
				out["weirdo_direction"] = direction
			}
			if raw == "up" {
				out["torch_facing_direction"] = "top"
			} else if isBedrockWallTorch(block) {
				out["torch_facing_direction"] = oppositeCardinal(raw)
			} else {
				out["torch_facing_direction"] = raw
			}
		case "part":
			out["head_piece_bit"] = boolByte(raw == "head")
		case "occupied":
			out["occupied_bit"] = boolByte(raw == "true")
		case "open":
			out["open_bit"] = boolByte(raw == "true")
		case "powered":
			out["powered_bit"] = boolByte(raw == "true")
			if block.ResourceLocation() == "minecraft:lever" {
				out["open_bit"] = boolByte(raw == "true")
			}
			out["button_pressed_bit"] = boolByte(raw == "true")
		case "half":
			out["upper_block_bit"] = boolByte(raw == "upper")
			out["vertical_half"] = raw
			out["upside_down_bit"] = boolByte(raw == "top")
		case "hinge":
			out["door_hinge_bit"] = boolByte(raw == "right")
		case "type":
			out["top_slot_bit"] = boolByte(raw == "top")
			if raw == "top" {
				out["vertical_half"] = "top"
			} else if raw == "bottom" {
				out["vertical_half"] = "bottom"
			}
		case "age":
			out["growth"] = intProperty(raw)
		case "moisture":
			out["moisturized_amount"] = intProperty(raw)
		case "level":
			out["liquid_depth"] = intProperty(raw)
			out["block_light_level"] = intProperty(raw)
			out["fill_level"] = intProperty(raw)
			out["composter_fill_level"] = intProperty(raw)
		case "power":
			out["redstone_signal"] = intProperty(raw)
		case "rotation":
			out["ground_sign_direction"] = intProperty(raw)
		case "delay":
			out["repeater_delay"] = intProperty(raw)
		case "lit":
			out["lit_bit"] = boolByte(raw == "true")
			out["output_lit_bit"] = boolByte(raw == "true")
			out["extinguished"] = boolByte(raw != "true")
		case "in_wall":
			out["in_wall_bit"] = boolByte(raw == "true")
		case "extended":
			out["extended_bit"] = boolByte(raw == "true")
		case "locked":
			out["repeater_locked"] = boolByte(raw == "true")
		case "mode":
			out["output_subtract_bit"] = boolByte(raw == "subtract")
		case "hanging":
			out["hanging"] = boolByte(raw == "true")
		case "layers":
			layers := intProperty(raw)
			if layers > 0 {
				layers--
			}
			out["height"] = layers
		case "charges":
			out["respawn_anchor_charge"] = intProperty(raw)
		case "shape":
			if direction, ok := bedrockRailDirection(raw); ok {
				out["rail_direction"] = direction
			}
		case "snowy":
			out["covered_bit"] = boolByte(raw == "true")
		case "persistent":
			out["persistent_bit"] = boolByte(raw == "true")
		case "drag":
			out["drag_down"] = boolByte(raw == "true")
		}
	}
	if block.ResourceLocation() == "minecraft:lever" {
		face, facing := properties["face"], properties["facing"]
		switch face {
		case "floor":
			if facing == "north" || facing == "south" {
				out["lever_direction"] = "up_north_south"
			} else {
				out["lever_direction"] = "up_east_west"
			}
		case "ceiling":
			if facing == "north" || facing == "south" {
				out["lever_direction"] = "down_north_south"
			} else {
				out["lever_direction"] = "down_east_west"
			}
		default:
			out["lever_direction"] = facing
		}
	}
	if block.ResourceLocation() == "minecraft:candle" || strings.HasSuffix(block.ResourceLocation(), "_candle") {
		candles, _ := strconv.Atoi(properties["candles"])
		if candles > 0 {
			out["candles"] = int32(candles - 1)
		}
	}
	if strings.HasSuffix(block.ResourceLocation(), "_door") && !strings.HasSuffix(block.ResourceLocation(), "_trapdoor") {
		// Bedrock door states encode their cardinal direction one clockwise
		// rotation from Java's facing property (as Dragonfly's WoodDoor does).
		out["minecraft:cardinal_direction"] = rotateCardinalRight(properties["facing"])
	}
	if strings.HasSuffix(block.ResourceLocation(), "_wall") {
		for _, direction := range []string{"north", "east", "south", "west"} {
			connection := properties[direction]
			if connection == "low" {
				connection = "short"
			}
			if connection == "" {
				connection = "none"
			}
			out["wall_connection_type_"+direction] = connection
		}
		out["wall_post_bit"] = boolByte(properties["up"] == "true")
	}
	return out
}

func isBedrockWallTorch(block coreworld.Block) bool {
	name := block.ResourceLocation()
	return name == "minecraft:wall_torch" || strings.HasSuffix(name, "_wall_torch") ||
		(name == "minecraft:unlit_redstone_torch" && block.Properties["facing"] != "")
}

func oppositeCardinal(facing string) string {
	switch facing {
	case "north":
		return "south"
	case "south":
		return "north"
	case "west":
		return "east"
	case "east":
		return "west"
	default:
		return facing
	}
}

func rotateCardinalRight(facing string) string {
	switch facing {
	case "north":
		return "east"
	case "east":
		return "south"
	case "south":
		return "west"
	case "west":
		return "north"
	default:
		return facing
	}
}

func propertyValue(raw string) any {
	if raw == "true" {
		return uint8(1)
	}
	if raw == "false" {
		return uint8(0)
	}
	if n, err := strconv.ParseInt(raw, 10, 32); err == nil {
		return int32(n)
	}
	return raw
}

func intProperty(raw string) int32 {
	n, _ := strconv.ParseInt(raw, 10, 32)
	return int32(n)
}

func boolByte(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}

func cardinalDirection(value string) (int32, bool) {
	switch value {
	case "south":
		return 0, true
	case "west":
		return 1, true
	case "north":
		return 2, true
	case "east":
		return 3, true
	default:
		return 0, false
	}
}

func bedrockFacingDirection(value string) (int32, bool) {
	switch value {
	case "down":
		return 0, true
	case "up":
		return 1, true
	case "north":
		return 2, true
	case "south":
		return 3, true
	case "west":
		return 4, true
	case "east":
		return 5, true
	default:
		return 0, false
	}
}

func bedrockRailDirection(shape string) (int32, bool) {
	switch shape {
	case "north_south":
		return 0, true
	case "east_west":
		return 1, true
	case "ascending_east":
		return 2, true
	case "ascending_west":
		return 3, true
	case "ascending_north":
		return 4, true
	case "ascending_south":
		return 5, true
	case "south_east":
		return 6, true
	case "south_west":
		return 7, true
	case "north_west":
		return 8, true
	case "north_east":
		return 9, true
	default:
		return 0, false
	}
}

func stairDirection(value string) (int32, bool) {
	switch value {
	case "east":
		return 0, true
	case "west":
		return 1, true
	case "south":
		return 2, true
	case "north":
		return 3, true
	default:
		return 0, false
	}
}

func stateScore(candidate, wanted map[string]any) int {
	score := -len(candidate)
	for key, value := range wanted {
		candidateValue, ok := candidate[key]
		if !ok {
			continue
		}
		if fmt.Sprint(candidateValue) == fmt.Sprint(value) {
			score += 12
		} else {
			score -= 4
		}
	}
	return score
}

// EncodeSubChunk encodes one canonical section using Bedrock sub-chunk v9
// with a persistent (NBT) palette. Persistent palette entries contain the
// Bedrock block name and property map rather than version-specific runtime IDs,
// making the encoding valid regardless of which Bedrock protocol version the
// client uses. subY is the absolute sub-chunk coordinate (-4..19 in the overworld).
func (e *Encoder) EncodeSubChunk(section *coreworld.Section, subY int32) ([]byte, error) {
	if section == nil || section.NonAir == 0 {
		return nil, nil
	}
	palette := section.BlockPalette()
	data := section.BlockData()
	bits := paletteBits(len(palette))

	var buf bytes.Buffer
	buf.WriteByte(9) // sub-chunk version
	buf.WriteByte(1) // one block storage layer
	buf.WriteByte(byte(int8(subY)))
	buf.WriteByte(bits << 1) // persistent palette: flag bit 0 is clear

	if bits != 0 {
		valuesPerWord := 32 / int(bits)
		wordCount := (4096 + valuesPerWord - 1) / valuesPerWord
		for wordIndex := 0; wordIndex < wordCount; wordIndex++ {
			var word uint32
			for valueIndex := 0; valueIndex < valuesPerWord; valueIndex++ {
				bedrockIndex := wordIndex*valuesPerWord + valueIndex
				if bedrockIndex >= len(data) {
					break
				}
				// Bedrock stores blocks in X-Z-Y order, whereas the canonical
				// section uses Y-Z-X order for Java compatibility.
				x := bedrockIndex >> 8
				z := (bedrockIndex >> 4) & 15
				y := bedrockIndex & 15
				canonicalIndex := y*256 + z*16 + x
				word |= uint32(data[canonicalIndex]) << (valueIndex * int(bits))
			}
			writeUint32LE(&buf, word)
		}
		// Persistent palette: count is LE uint32 (not varint).
		writeUint32LE(&buf, uint32(len(palette)))
	}

	// Write each palette entry as an NBT compound (LittleEndian encoding).
	// This matches Dragonfly's BlockPaletteEncoding.encode / diskEncoding path.
	for _, block := range palette {
		name, props := e.resolveState(block)
		entry := persistentBlockEntry{Name: name, States: props, Version: currentBlockVersion}
		entryBytes, err := nbt.MarshalEncoding(entry, nbt.LittleEndian)
		if err != nil {
			return nil, err
		}
		buf.Write(entryBytes)
	}
	return buf.Bytes(), nil
}

func paletteBits(entries int) byte {
	switch {
	case entries <= 1:
		return 0
	case entries <= 2:
		return 1
	case entries <= 4:
		return 2
	case entries <= 8:
		return 3
	case entries <= 16:
		return 4
	case entries <= 32:
		return 5
	case entries <= 64:
		return 6
	case entries <= 256:
		return 8
	default:
		return 16
	}
}

func writeUint32LE(buf *bytes.Buffer, value uint32) {
	buf.WriteByte(byte(value))
	buf.WriteByte(byte(value >> 8))
	buf.WriteByte(byte(value >> 16))
	buf.WriteByte(byte(value >> 24))
}

// EncodeV9AirSubChunk returns a minimal version-9 persistent-palette sub-chunk
// containing only air.  subY is the absolute Bedrock sub-chunk Y coordinate
// (range −4 to 19 in the overworld).
//
// Format (matches EncodeSubChunk for bitsPerBlock=0):
//
//	byte(9)         sub-chunk version
//	byte(1)         one block storage layer
//	byte(subY)      absolute sub-chunk Y (signed)
//	byte(0)         bitsPerBlock=0 | persistent flag (bit0=0) → 0<<1 = 0x00
//	NBT compound    minecraft:air with currentBlockVersion
func EncodeV9AirSubChunk(subY int8) ([]byte, error) {
	entry := persistentBlockEntry{
		Name:    "minecraft:air",
		States:  map[string]any{},
		Version: currentBlockVersion,
	}
	entryBytes, err := nbt.MarshalEncoding(entry, nbt.LittleEndian)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteByte(9)          // version 9
	buf.WriteByte(1)          // 1 layer
	buf.WriteByte(byte(subY)) // sub-chunk Y (signed byte)
	buf.WriteByte(0)          // bitsPerBlock=0, persistent (bit0 clear)
	// bitsPerBlock=0: no word data, no palette count — just the single entry
	buf.Write(entryBytes)
	return buf.Bytes(), nil
}

// EncodeFullChunkPayload builds the complete RawPayload for a LevelChunk packet
// sent with SubChunkCount = SectionCount (24).  Block data is included directly
// so the client does not need to send SubChunkRequest packets.
//
// Sub-chunks use the V9 network format (flag bit0=1, network hashes as zigzag
// varints) — matching Dragonfly's NetworkEncoding for inline LevelChunk payloads.
//
// Payload layout:
//  1. 24 sub-chunk blobs (V9 network-hash palette), one per section
//  2. 24 × single-value plains biome storage (network palette)
//  3. Border block count varint (0x00)
func (e *Encoder) EncodeFullChunkPayload(chunk *coreworld.Chunk) ([]byte, error) {
	var buf bytes.Buffer
	for i := 0; i < coreworld.SectionCount; i++ {
		subY := int8(i - 4) // section 0 → subY=-4, section 4 → subY=0, etc.
		var sectionBytes []byte
		if chunk != nil && chunk.Sections[i] != nil && chunk.Sections[i].NonAir > 0 {
			sectionBytes = e.encodeNetworkSubChunk(chunk.Sections[i], subY)
		}
		if len(sectionBytes) == 0 {
			sectionBytes = e.encodeNetworkAirSubChunk(subY)
		}
		buf.Write(sectionBytes)
	}
	// Biome data: translate Java's quart-resolution cells to Bedrock's
	// 16x16x16 network palette for every sub-chunk.
	for i := 0; i < coreworld.SectionCount; i++ {
		var section *coreworld.Section
		if chunk != nil {
			section = chunk.Sections[i]
		}
		buf.Write(e.encodeBiomeStorage(section))
	}
	buf.WriteByte(0x00) // border block count varint (0 = none)
	if chunk != nil {
		encoder := nbt.NewEncoderWithEncoding(&buf, nbt.NetworkLittleEndian)
		for _, entity := range chunk.BlockEntities {
			data, ok := bedrockBlockEntityData(entity)
			if !ok {
				continue
			}
			if err := encoder.Encode(data); err != nil {
				return nil, fmt.Errorf("bedrock: encode block actor at %d,%d,%d: %w", entity.X, entity.Y, entity.Z, err)
			}
		}
	}
	return buf.Bytes(), nil
}

func bedrockBlockEntityData(entity coreworld.BlockEntity) (map[string]any, bool) {
	if entity.Type != "minecraft:decorated_pot" && entity.Type != "decorated_pot" {
		return nil, false
	}
	decorations := player.NormalizePotDecorations(entity.PotDecorations)
	return map[string]any{
		"id": "DecoratedPot",
		"x":  int32(entity.X), "y": int32(entity.Y), "z": int32(entity.Z),
		"sherds": []string{decorations[0], decorations[1], decorations[2], decorations[3]},
	}, true
}

// encodeBiomeStorage converts a Java quart-resolution biome container to the
// Bedrock 16x16x16 paletted storage used in LevelChunk. Every quart value is
// expanded to its corresponding 4x4x4 block cube.
func (e *Encoder) encodeBiomeStorage(section *coreworld.Section) []byte {
	if section == nil {
		return makeSingleValueBiomeStorage(biomePlains)
	}
	javaPalette := section.BiomePalette()
	javaData := section.BiomeData()
	bedrockPalette := make([]uint32, 0, len(javaPalette))
	bedrockIndex := make(map[uint32]uint16, len(javaPalette))
	translated := make([]uint16, len(javaPalette))
	for i, biome := range javaPalette {
		id := bedrockBiomeRuntimeID(biome)
		index, ok := bedrockIndex[id]
		if !ok {
			index = uint16(len(bedrockPalette))
			bedrockIndex[id] = index
			bedrockPalette = append(bedrockPalette, id)
		}
		translated[i] = index
	}
	if len(bedrockPalette) <= 1 {
		id := biomePlains
		if len(bedrockPalette) == 1 {
			id = bedrockPalette[0]
		}
		return makeSingleValueBiomeStorage(id)
	}

	bits := paletteBits(len(bedrockPalette))
	valuesPerWord := 32 / int(bits)
	wordCount := (4096 + valuesPerWord - 1) / valuesPerWord
	var buf bytes.Buffer
	buf.WriteByte(bits<<1 | 1)
	for wordIndex := 0; wordIndex < wordCount; wordIndex++ {
		var word uint32
		for valueIndex := 0; valueIndex < valuesPerWord; valueIndex++ {
			linear := wordIndex*valuesPerWord + valueIndex
			if linear >= 4096 {
				break
			}
			x := linear >> 8
			z := (linear >> 4) & 15
			y := linear & 15
			quartIndex := (y>>2)*16 + (z>>2)*4 + (x >> 2)
			javaIndex := int(javaData[quartIndex])
			if javaIndex >= len(translated) {
				javaIndex = 0
			}
			word |= uint32(translated[javaIndex]) << (valueIndex * int(bits))
		}
		writeUint32LE(&buf, word)
	}
	_ = protocol.WriteVarint32(&buf, int32(len(bedrockPalette)))
	for _, id := range bedrockPalette {
		_ = protocol.WriteVarint32(&buf, int32(id))
	}
	return buf.Bytes()
}

var javaToBedrockBiomeID = map[string]uint32{
	"minecraft:badlands":                 37,
	"minecraft:bamboo_jungle":            48,
	"minecraft:beach":                    16,
	"minecraft:birch_forest":             27,
	"minecraft:cherry_grove":             192,
	"minecraft:cold_ocean":               44,
	"minecraft:dark_forest":              29,
	"minecraft:deep_cold_ocean":          45,
	"minecraft:deep_dark":                190,
	"minecraft:deep_frozen_ocean":        47,
	"minecraft:deep_lukewarm_ocean":      43,
	"minecraft:deep_ocean":               24,
	"minecraft:desert":                   2,
	"minecraft:dripstone_caves":          188,
	"minecraft:eroded_badlands":          165,
	"minecraft:flower_forest":            132,
	"minecraft:forest":                   4,
	"minecraft:frozen_ocean":             46,
	"minecraft:frozen_peaks":             183,
	"minecraft:frozen_river":             11,
	"minecraft:grove":                    185,
	"minecraft:ice_spikes":               140,
	"minecraft:jagged_peaks":             182,
	"minecraft:jungle":                   21,
	"minecraft:lukewarm_ocean":           42,
	"minecraft:lush_caves":               187,
	"minecraft:mangrove_swamp":           191,
	"minecraft:meadow":                   186,
	"minecraft:mushroom_fields":          14,
	"minecraft:ocean":                    0,
	"minecraft:old_growth_birch_forest":  155,
	"minecraft:old_growth_pine_taiga":    32,
	"minecraft:old_growth_spruce_taiga":  160,
	"minecraft:pale_garden":              193,
	"minecraft:plains":                   1,
	"minecraft:river":                    7,
	"minecraft:savanna":                  35,
	"minecraft:savanna_plateau":          36,
	"minecraft:snowy_beach":              26,
	"minecraft:snowy_plains":             12,
	"minecraft:snowy_slopes":             184,
	"minecraft:snowy_taiga":              30,
	"minecraft:sparse_jungle":            23,
	"minecraft:stony_peaks":              189,
	"minecraft:stony_shore":              25,
	"minecraft:sunflower_plains":         129,
	"minecraft:swamp":                    6,
	"minecraft:taiga":                    5,
	"minecraft:warm_ocean":               40,
	"minecraft:windswept_forest":         34,
	"minecraft:windswept_gravelly_hills": 131,
	"minecraft:windswept_hills":          3,
	"minecraft:windswept_savanna":        163,
	"minecraft:wooded_badlands":          38,
}

func bedrockBiomeRuntimeID(name string) uint32 {
	if id, ok := javaToBedrockBiomeID[name]; ok {
		return id
	}
	return biomePlains
}

// encodeNetworkSubChunk encodes one section using V9 network format (network hashes
// as zigzag varints, flag bit0=1).  This is the format Bedrock expects inside a
// LevelChunk RawPayload when SubChunkCount equals the actual section count.
func (e *Encoder) encodeNetworkSubChunk(section *coreworld.Section, subY int8) []byte {
	palette := section.BlockPalette()
	data := section.BlockData()
	bits := paletteBits(len(palette))

	var buf bytes.Buffer
	buf.WriteByte(9)           // sub-chunk version 9
	buf.WriteByte(1)           // one block storage layer
	buf.WriteByte(byte(subY))  // absolute sub-chunk Y (signed)
	buf.WriteByte(bits<<1 | 1) // network palette flag: bit0 = 1

	if bits != 0 {
		valuesPerWord := 32 / int(bits)
		wordCount := (4096 + valuesPerWord - 1) / valuesPerWord
		for wordIndex := 0; wordIndex < wordCount; wordIndex++ {
			var word uint32
			for valueIndex := 0; valueIndex < valuesPerWord; valueIndex++ {
				bedrockIndex := wordIndex*valuesPerWord + valueIndex
				if bedrockIndex >= len(data) {
					break
				}
				// Bedrock X-Z-Y vs canonical Y-Z-X
				x := bedrockIndex >> 8
				z := (bedrockIndex >> 4) & 15
				y := bedrockIndex & 15
				canonicalIndex := y*256 + z*16 + x
				word |= uint32(data[canonicalIndex]) << (valueIndex * int(bits))
			}
			writeUint32LE(&buf, word)
		}
		// Network palette: count as zigzag varint32
		_ = protocol.WriteVarint32(&buf, int32(len(palette)))
	}
	// Write each palette entry as its stable Bedrock network hash (zigzag varint32).
	for _, block := range palette {
		_ = protocol.WriteVarint32(&buf, int32(e.BlockNetworkID(block)))
	}
	return buf.Bytes()
}

// encodeNetworkAirSubChunk returns a minimal V9 network-format sub-chunk
// containing only air.  Used for nil/empty sections inside EncodeFullChunkPayload.
func (e *Encoder) encodeNetworkAirSubChunk(subY int8) []byte {
	var buf bytes.Buffer
	buf.WriteByte(9)          // version 9
	buf.WriteByte(1)          // 1 layer
	buf.WriteByte(byte(subY)) // sub-chunk Y
	buf.WriteByte(0x01)       // bitsPerBlock=0, network flag = 1
	// bitsPerBlock=0: no count, single entry as a network hash.
	_ = protocol.WriteVarint32(&buf, int32(e.airHash))
	return buf.Bytes()
}
