from pathlib import Path


def read(path):
    return Path(path).read_text()


def write(path, value):
    Path(path).write_text(value)


def replace_once(path, old, new):
    value = read(path)
    if old not in value:
        raise SystemExit(f"missing patch target in {path}: {old[:160]!r}")
    write(path, value.replace(old, new, 1))


def append_once(path, marker, addition):
    value = read(path)
    if marker in value:
        return
    write(path, value.rstrip() + "\n\n" + addition.strip() + "\n")


# ---------------------------------------------------------------------------
# Core: publish block-entity mutations without coupling core to either adapter.
# ---------------------------------------------------------------------------
replace_once(
    "core/world/world.go",
    "\tblockObserverMu sync.RWMutex\n\tblockObserver   func(BlockChange)\n",
    "\tblockObserverMu       sync.RWMutex\n\tblockObserver         func(BlockChange)\n\tblockEntityObserverMu sync.RWMutex\n\tblockEntityObserver   func(BlockEntity)\n",
)

replace_once(
    "core/world/world.go",
    '''func (w *World) notifyBlockObserver(x, y, z int, block Block) {\n\tw.blockObserverMu.RLock()\n\tobserver := w.blockObserver\n\tw.blockObserverMu.RUnlock()\n\tif observer != nil {\n\t\tobserver(BlockChange{X: x, Y: y, Z: z, Block: block})\n\t}\n}\n''',
    '''func (w *World) notifyBlockObserver(x, y, z int, block Block) {\n\tw.blockObserverMu.RLock()\n\tobserver := w.blockObserver\n\tw.blockObserverMu.RUnlock()\n\tif observer != nil {\n\t\tobserver(BlockChange{X: x, Y: y, Z: z, Block: block})\n\t}\n}\n\n// SetBlockEntityObserver installs an adapter-neutral notification invoked after\n// canonical block-entity data changes. The snapshot passed to the observer owns\n// its Data and Items slices and may safely outlive the world mutation.\nfunc (w *World) SetBlockEntityObserver(observer func(BlockEntity)) {\n\tw.blockEntityObserverMu.Lock()\n\tw.blockEntityObserver = observer\n\tw.blockEntityObserverMu.Unlock()\n}\n\nfunc (w *World) notifyBlockEntityObserver(entity BlockEntity) {\n\tw.blockEntityObserverMu.RLock()\n\tobserver := w.blockEntityObserver\n\tw.blockEntityObserverMu.RUnlock()\n\tif observer == nil {\n\t\treturn\n\t}\n\tentity.Data = append([]byte(nil), entity.Data...)\n\tentity.Items = append([]ContainerItem(nil), entity.Items...)\n\tobserver(entity)\n}\n''',
)

replace_once(
    "core/world/world.go",
    '''\tw.containerMu.Lock()\n\tupdated := false\n\tfor index := range c.BlockEntities {\n\t\tentity := &c.BlockEntities[index]\n\t\tif entity.X != x || entity.Y != y || entity.Z != z {\n\t\t\tcontinue\n\t\t}\n\t\tif entity.Type == "" {\n\t\t\tentity.Type = "minecraft:decorated_pot"\n\t\t}\n\t\tif len(entity.Data) < 2 {\n\t\t\tentity.Data = []byte{10, 0}\n\t\t}\n\t\tentity.PotDecorations = decorations\n\t\tupdated = true\n\t\tbreak\n\t}\n\tif !updated {\n\t\tc.BlockEntities = append(c.BlockEntities, BlockEntity{\n\t\t\tX: x, Y: y, Z: z, Type: "minecraft:decorated_pot", Data: []byte{10, 0},\n\t\t\tPotDecorations: decorations,\n\t\t})\n\t}\n\tw.containerMu.Unlock()\n\n\tw.mu.Lock()\n\tkey := [2]int32{cx, cz}\n\tw.chunks[key] = c\n\tw.touchChunkLocked(key)\n\tw.dirty[key] = struct{}{}\n\tw.mu.Unlock()\n}\n''',
    '''\tw.containerMu.Lock()\n\tupdated := false\n\tvar snapshot BlockEntity\n\tfor index := range c.BlockEntities {\n\t\tentity := &c.BlockEntities[index]\n\t\tif entity.X != x || entity.Y != y || entity.Z != z {\n\t\t\tcontinue\n\t\t}\n\t\tif entity.Type == "" {\n\t\t\tentity.Type = "minecraft:decorated_pot"\n\t\t}\n\t\tif len(entity.Data) < 2 {\n\t\t\tentity.Data = []byte{10, 0}\n\t\t}\n\t\tentity.PotDecorations = decorations\n\t\tsnapshot = *entity\n\t\tupdated = true\n\t\tbreak\n\t}\n\tif !updated {\n\t\tsnapshot = BlockEntity{\n\t\t\tX: x, Y: y, Z: z, Type: "minecraft:decorated_pot", Data: []byte{10, 0},\n\t\t\tPotDecorations: decorations,\n\t\t}\n\t\tc.BlockEntities = append(c.BlockEntities, snapshot)\n\t}\n\tw.containerMu.Unlock()\n\n\tw.mu.Lock()\n\tkey := [2]int32{cx, cz}\n\tw.chunks[key] = c\n\tw.touchChunkLocked(key)\n\tw.dirty[key] = struct{}{}\n\tw.mu.Unlock()\n\tw.notifyBlockEntityObserver(snapshot)\n}\n''',
)

# ---------------------------------------------------------------------------
# Java: synthesize decorated-pot network NBT and send live block entity updates.
# ---------------------------------------------------------------------------
replace_once(
    "java/world/nbt.go",
    '''import (\n\t"encoding/binary"\n\t"io"\n)\n''',
    '''import (\n\t"bytes"\n\t"encoding/binary"\n\t"io"\n\n\t"GoCraft/core/player"\n\tcoreworld "GoCraft/core/world"\n)\n''',
)
replace_once(
    "java/world/nbt.go",
    '''const (\n\tnbtTagEnd      byte = 0x00\n\tnbtTagLongArr  byte = 0x0C\n\tnbtTagCompound byte = 0x0A\n)\n''',
    '''const (\n\tnbtTagEnd      byte = 0x00\n\tnbtTagString   byte = 0x08\n\tnbtTagList     byte = 0x09\n\tnbtTagCompound byte = 0x0A\n\tnbtTagLongArr  byte = 0x0C\n)\n''',
)
append_once(
    "java/world/nbt.go",
    "func BlockEntityNetworkData",
    r'''
// BlockEntityNetworkData returns the network-NBT payload for a canonical block
// entity. Decorated-pot sherds are generated from canonical state so pots placed
// during this server session render correctly before the chunk is persisted.
func BlockEntityNetworkData(entity coreworld.BlockEntity) []byte {
	if entity.Type != "minecraft:decorated_pot" && entity.Type != "decorated_pot" {
		return entity.Data
	}
	decorations := player.NormalizePotDecorations(entity.PotDecorations)
	var buf bytes.Buffer
	writeNetworkNBTCompound(&buf, func(w io.Writer) {
		writeNBTStringList(w, "sherds", decorations[:])
	})
	return buf.Bytes()
}

func writeNBTStringList(w io.Writer, name string, values []string) {
	_, _ = w.Write([]byte{nbtTagList})
	writeNBTStringPayload(w, name)
	_, _ = w.Write([]byte{nbtTagString})
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(values)))
	_, _ = w.Write(length[:])
	for _, value := range values {
		writeNBTStringPayload(w, value)
	}
}

func writeNBTStringPayload(w io.Writer, value string) {
	data := []byte(value)
	var length [2]byte
	binary.BigEndian.PutUint16(length[:], uint16(len(data)))
	_, _ = w.Write(length[:])
	_, _ = w.Write(data)
}
''',
)

replace_once(
    "java/world/sender.go",
    '''\ttype encodedBlockEntity struct {\n\t\tentity coreworld.BlockEntity\n\t\ttypeID int32\n\t}\n\tblockEntities := make([]encodedBlockEntity, 0, len(c.BlockEntities))\n\tfor _, entity := range c.BlockEntities {\n\t\tif typeID, ok := BlockEntityTypeID(entity.Type); ok && len(entity.Data) > 0 {\n\t\t\tblockEntities = append(blockEntities, encodedBlockEntity{entity: entity, typeID: typeID})\n\t\t}\n\t}\n''',
    '''\ttype encodedBlockEntity struct {\n\t\tentity coreworld.BlockEntity\n\t\ttypeID int32\n\t\tdata   []byte\n\t}\n\tblockEntities := make([]encodedBlockEntity, 0, len(c.BlockEntities))\n\tfor _, entity := range c.BlockEntities {\n\t\ttypeID, ok := BlockEntityTypeID(entity.Type)\n\t\tdata := BlockEntityNetworkData(entity)\n\t\tif ok && len(data) > 0 {\n\t\t\tblockEntities = append(blockEntities, encodedBlockEntity{entity: entity, typeID: typeID, data: data})\n\t\t}\n\t}\n''',
)
replace_once(
    "java/world/sender.go",
    '''\t\tb.Byte(packedXZ).Short(int16(entity.Y)).VarInt(encoded.typeID).Bytes(entity.Data)\n''',
    '''\t\tb.Byte(packedXZ).Short(int16(entity.Y)).VarInt(encoded.typeID).Bytes(encoded.data)\n''',
)

replace_once(
    "internal/protocoldata/java/1.21.4/play.json",
    '''    "minecraft:acknowledge_block_change": 5,\n    "minecraft:block_update": 9,\n''',
    '''    "minecraft:acknowledge_block_change": 5,\n    "minecraft:block_entity_data": 7,\n    "minecraft:block_update": 9,\n''',
)
replace_once(
    "java/handler/packets.go",
    '''\tpacketIDBlockUpdate            = protocoldata.MustCB("play", "minecraft:block_update")\n''',
    '''\tpacketIDBlockEntityData        = protocoldata.MustCB("play", "minecraft:block_entity_data")\n\tpacketIDBlockUpdate            = protocoldata.MustCB("play", "minecraft:block_update")\n''',
)

Path("java/handler/block_entity.go").write_text(r'''package handler

import (
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

func buildBlockEntityData(entity coreworld.BlockEntity) *protocol.Packet {
	typeID, ok := javaworld.BlockEntityTypeID(entity.Type)
	data := javaworld.BlockEntityNetworkData(entity)
	if !ok || len(data) == 0 {
		return nil
	}
	position := spatial.BlockPos{X: int32(entity.X), Y: int32(entity.Y), Z: int32(entity.Z)}
	return protocol.NewBuilder(packetIDBlockEntityData).
		Long(position.Encode()).
		VarInt(typeID).
		Bytes(data).
		Build()
}

// BroadcastBlockEntityDataInDimension mirrors one canonical block-entity
// mutation to Java viewers in the same dimension.
func BroadcastBlockEntityDataInDimension(entity coreworld.BlockEntity, mgr *session.Manager, dimension int32) {
	if mgr == nil {
		return
	}
	pkt := buildBlockEntityData(entity)
	if pkt == nil {
		return
	}
	for _, current := range mgr.SnapshotAll() {
		if current.Player == nil || current.Player.Dimension != dimension {
			continue
		}
		_ = current.Conn.WritePacket(pkt)
	}
}
''')

# Preserve components for an item spilled from a decorated pot/container.
replace_once(
    "java/handler/block.go",
    '''player.ItemStack{ItemID: item.ItemID, Count: item.Count, Damage: item.Damage}, ordinal, mgr, p.Dimension)''',
    '''player.ItemStack{ItemID: item.ItemID, Count: item.Count, Damage: item.Damage, Enchantments: item.Enchantments, PotDecorations: item.PotDecorations}, ordinal, mgr, p.Dimension)''',
)

# ---------------------------------------------------------------------------
# Bedrock: append block actors to LevelChunk and publish live BlockActorData.
# ---------------------------------------------------------------------------
replace_once(
    "bedrock/world/encoder.go",
    '''\tcoreworld "GoCraft/core/world"\n)\n''',
    '''\t"GoCraft/core/player"\n\tcoreworld "GoCraft/core/world"\n)\n''',
)
replace_once(
    "bedrock/world/encoder.go",
    '''\tbuf.WriteByte(0x00) // border block count varint (0 = none)\n\treturn buf.Bytes(), nil\n}\n''',
    '''\tbuf.WriteByte(0x00) // border block count varint (0 = none)\n\tif chunk != nil {\n\t\tencoder := nbt.NewEncoderWithEncoding(&buf, nbt.NetworkLittleEndian)\n\t\tfor _, entity := range chunk.BlockEntities {\n\t\t\tdata, ok := bedrockBlockEntityData(entity)\n\t\t\tif !ok {\n\t\t\t\tcontinue\n\t\t\t}\n\t\t\tif err := encoder.Encode(data); err != nil {\n\t\t\t\treturn nil, fmt.Errorf("bedrock: encode block actor at %d,%d,%d: %w", entity.X, entity.Y, entity.Z, err)\n\t\t\t}\n\t\t}\n\t}\n\treturn buf.Bytes(), nil\n}\n\nfunc bedrockBlockEntityData(entity coreworld.BlockEntity) (map[string]any, bool) {\n\tif entity.Type != "minecraft:decorated_pot" && entity.Type != "decorated_pot" {\n\t\treturn nil, false\n\t}\n\tdecorations := player.NormalizePotDecorations(entity.PotDecorations)\n\treturn map[string]any{\n\t\t"id": "DecoratedPot",\n\t\t"x": int32(entity.X), "y": int32(entity.Y), "z": int32(entity.Z),\n\t\t"sherds": []string{decorations[0], decorations[1], decorations[2], decorations[3]},\n\t}, true\n}\n''',
)

append_once(
    "bedrock/sync.go",
    "func (l *Listener) BroadcastBlockEntityData",
    r'''
// BroadcastBlockEntityData mirrors canonical decorated-pot block actor data to
// Bedrock viewers in the affected dimension.
func (l *Listener) BroadcastBlockEntityData(dimension int32, entity coreworld.BlockEntity) {
	if l == nil || (entity.Type != "minecraft:decorated_pot" && entity.Type != "decorated_pot") {
		return
	}
	decorations := player.NormalizePotDecorations(entity.PotDecorations)
	data := map[string]any{
		"id": "DecoratedPot",
		"x": int32(entity.X), "y": int32(entity.Y), "z": int32(entity.Z),
		"sherds": []string{decorations[0], decorations[1], decorations[2], decorations[3]},
	}
	l.sessionsMu.RLock()
	sessions := make([]*bedrockSession, 0, len(l.sessions))
	for _, current := range l.sessions {
		if current.dimension.Load() == dimension {
			sessions = append(sessions, current)
		}
	}
	l.sessionsMu.RUnlock()
	for _, current := range sessions {
		_ = current.conn.WritePacket(&packet.BlockActorData{
			Position: protocol.BlockPos{int32(entity.X), int32(entity.Y), int32(entity.Z)},
			NBTData:  data,
		})
	}
}
''',
)

# ---------------------------------------------------------------------------
# Server wiring: one canonical mutation fans out to both adapters.
# ---------------------------------------------------------------------------
replace_once(
    "server/server.go",
    '''\tif cfg.Bedrock.Enabled {\n\t\ts.bedrockListener = bedrock.NewListener(\n''',
    '''\tif cfg.Bedrock.Enabled {\n\t\ts.bedrockListener = bedrock.NewListener(\n''',
)
replace_once(
    "server/server.go",
    '''\t}\n\tcmds.SetPlayerTeleporter(s.teleportPlayer)\n''',
    '''\t}\n\tfor dimension, dimensionWorld := range map[int32]*coreworld.World{\n\t\tdimensionOverworld: worldInstance,\n\t\tdimensionNether: netherWorld,\n\t\tdimensionEnd: endWorld,\n\t} {\n\t\tdimension := dimension\n\t\tdimensionWorld.SetBlockEntityObserver(func(entity coreworld.BlockEntity) {\n\t\t\thandler.BroadcastBlockEntityDataInDimension(entity, s.sessions, dimension)\n\t\t\tif s.bedrockListener != nil {\n\t\t\t\ts.bedrockListener.BroadcastBlockEntityData(dimension, entity)\n\t\t\t}\n\t\t})\n\t}\n\tcmds.SetPlayerTeleporter(s.teleportPlayer)\n''',
)

# Bedrock-side container spilling may occur in more than one server file. Keep
# all canonical components when converting ContainerItem back to ItemStack.
for path in Path("server").glob("*.go"):
    value = path.read_text()
    old = "player.ItemStack{ItemID: item.ItemID, Count: item.Count, Damage: item.Damage}"
    new = "player.ItemStack{ItemID: item.ItemID, Count: item.Count, Damage: item.Damage, Enchantments: item.Enchantments, PotDecorations: item.PotDecorations}"
    if old in value:
        path.write_text(value.replace(old, new))

# ---------------------------------------------------------------------------
# Regression tests for canonical observer + Java/Bedrock visual payloads.
# ---------------------------------------------------------------------------
Path("core/world/decorated_pot_observer_test.go").write_text(r'''package world

import "testing"

func TestDecoratedPotDecorationMutationNotifiesObserver(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	var observed BlockEntity
	w.SetBlockEntityObserver(func(entity BlockEntity) { observed = entity })
	want := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:flow_pottery_sherd", "minecraft:brick"}
	w.SetDecoratedPotDecorations(3, 64, -2, want)
	if observed.X != 3 || observed.Y != 64 || observed.Z != -2 || observed.Type != "minecraft:decorated_pot" {
		t.Fatalf("unexpected block entity observer snapshot: %+v", observed)
	}
	if observed.PotDecorations != want {
		t.Fatalf("observed decorations = %#v, want %#v", observed.PotDecorations, want)
	}
}
''')

append_once(
    "java/world/sender_block_entity_test.go",
    "func TestDecoratedPotNetworkDataUsesCanonicalSherds",
    r'''
func TestDecoratedPotNetworkDataUsesCanonicalSherds(t *testing.T) {
	entity := coreworld.BlockEntity{
		Type: "minecraft:decorated_pot",
		PotDecorations: [4]string{
			"minecraft:angler_pottery_sherd", "minecraft:brick",
			"minecraft:flow_pottery_sherd", "minecraft:miner_pottery_sherd",
		},
	}
	data := BlockEntityNetworkData(entity)
	for _, decoration := range entity.PotDecorations {
		if !bytes.Contains(data, []byte(decoration)) {
			t.Fatalf("network NBT does not contain %q: %x", decoration, data)
		}
	}
}
''',
)
# Add bytes import to sender test.
replace_once(
    "java/world/sender_block_entity_test.go",
    '''import (\n\t"testing"\n''',
    '''import (\n\t"bytes"\n\t"testing"\n''',
)

Path("bedrock/world/encoder_pot_test.go").write_text(r'''package world

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestDecoratedPotBlockActorDataUsesCanonicalSherds(t *testing.T) {
	entity := coreworld.BlockEntity{
		X: 7, Y: 65, Z: -4, Type: "minecraft:decorated_pot",
		PotDecorations: [4]string{
			"minecraft:angler_pottery_sherd", "minecraft:brick",
			"minecraft:flow_pottery_sherd", "minecraft:miner_pottery_sherd",
		},
	}
	data, ok := bedrockBlockEntityData(entity)
	if !ok {
		t.Fatal("decorated pot was not encoded as a Bedrock block actor")
	}
	if data["id"] != "DecoratedPot" || data["x"] != int32(7) || data["y"] != int32(65) || data["z"] != int32(-4) {
		t.Fatalf("unexpected block actor identity/position: %#v", data)
	}
	sherds, ok := data["sherds"].([]string)
	if !ok || len(sherds) != 4 {
		t.Fatalf("unexpected sherd payload: %#v", data["sherds"])
	}
	for index, want := range entity.PotDecorations {
		if sherds[index] != want {
			t.Fatalf("sherd %d = %q, want %q", index, sherds[index], want)
		}
	}
}
''')

Path("java/handler/block_entity_test.go").write_text(r'''package handler

import (
	"bytes"
	"testing"

	coreworld "GoCraft/core/world"
)

func TestDecoratedPotBlockEntityPacketCarriesSherds(t *testing.T) {
	entity := coreworld.BlockEntity{
		X: 2, Y: 70, Z: 4, Type: "minecraft:decorated_pot",
		PotDecorations: [4]string{
			"minecraft:angler_pottery_sherd", "minecraft:brick",
			"minecraft:flow_pottery_sherd", "minecraft:miner_pottery_sherd",
		},
	}
	pkt := buildBlockEntityData(entity)
	if pkt == nil {
		t.Fatal("decorated pot block entity packet was nil")
	}
	if pkt.ID != packetIDBlockEntityData {
		t.Fatalf("packet ID = %d, want %d", pkt.ID, packetIDBlockEntityData)
	}
	if !bytes.Contains(pkt.Data, []byte("minecraft:flow_pottery_sherd")) {
		t.Fatalf("packet did not contain decorated pot sherd NBT: %x", pkt.Data)
	}
}
''')
