package registry

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"GoCraft/internal/gamedata"
	"GoCraft/internal/protocoldata"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

var (
	packetIDRegistryData = protocoldata.MustCB("configuration", "minecraft:registry_data")
	packetIDUpdateTags   = protocoldata.MustCB("configuration", "minecraft:update_tags")
)

const dimensionTypeRegistryName = "minecraft:dimension_type"

// RegistryDataLoader.SYNCHRONIZED_REGISTRIES in the Mojang-mapped 1.21.4
// server. Once a client receives any Registry Data packet it rebuilds this
// entire set, so sending only dimension_type leaves required registries empty.
var synchronizedRegistryOrder769 = []string{
	"minecraft:worldgen/biome",
	"minecraft:chat_type",
	"minecraft:trim_pattern",
	"minecraft:trim_material",
	"minecraft:wolf_variant",
	"minecraft:painting_variant",
	"minecraft:dimension_type",
	"minecraft:damage_type",
	"minecraft:banner_pattern",
	"minecraft:enchantment",
	"minecraft:jukebox_song",
	"minecraft:instrument",
}

// TagNetworkSerialization.networkSafeRegistries in 1.21.4: synchronized
// dynamic registries with tags followed by built-in registries with tags.
var networkTagRegistryOrder769 = []string{
	"minecraft:worldgen/biome",
	"minecraft:painting_variant",
	"minecraft:damage_type",
	"minecraft:banner_pattern",
	"minecraft:enchantment",
	"minecraft:instrument",
	"minecraft:block",
	"minecraft:cat_variant",
	"minecraft:entity_type",
	"minecraft:fluid",
	"minecraft:game_event",
	"minecraft:item",
	"minecraft:point_of_interest_type",
}

type networkRegistry struct {
	Name    string   `json:"name"`
	Entries []string `json:"entries"`
}

type networkRegistryFile struct {
	Registries []networkRegistry `json:"registries"`
}

type networkTag struct {
	Name    string  `json:"name"`
	Entries []int32 `json:"entries"`
}

type networkTagRegistry struct {
	Name         string       `json:"name"`
	RegistrySize int32        `json:"registry_size"`
	Tags         []networkTag `json:"tags"`
}

type networkTagFile struct {
	Registries []networkTagRegistry `json:"registries"`
}

var (
	vanillaNetworkRegistries = mustLoadNetworkRegistries()
	vanillaNetworkTags       = mustLoadNetworkTags()
)

// VanillaProvider implements Provider using the Known Packs shortcut for
// vanilla Minecraft 1.21.4.
//
// Registry Data packets and Update Tags are still required after the client
// selects the known pack. The shortcut permits each known entry's NBT value to
// be omitted, but the complete ordered registry and tag snapshots must be sent.
type VanillaProvider struct{}

// Packs returns the single "minecraft:core" pack for 1.21.4.
func (p *VanillaProvider) Packs() []Pack {
	return []Pack{{
		Namespace: "minecraft",
		ID:        "core",
		Version:   "1.21.4",
	}}
}

// SendRegistries establishes all protocol-769 synchronized network registries.
// Entry NBT is omitted because the client confirmed the matching core pack and
// resolves each value from its bundled 1.21.4 data.
func (p *VanillaProvider) SendRegistries(conn *network.ClientConn, selected []Pack) error {
	if !samePacks(selected, p.Packs()) {
		return fmt.Errorf("registry: client did not select the exact advertised minecraft:core 1.21.4 pack: got %+v", selected)
	}
	for _, registry := range vanillaNetworkRegistries {
		if err := conn.WritePacket(buildRegistryDataPacket(registry)); err != nil {
			return fmt.Errorf("registry: sending %s: %w", registry.Name, err)
		}
	}
	dimensionID, err := p.DimensionTypeID("minecraft:overworld")
	if err != nil {
		return err
	}
	slog.Info("java configuration registries sent",
		"registries", len(vanillaNetworkRegistries),
		"dimensionType", "minecraft:overworld",
		"dimensionTypeID", dimensionID,
	)
	return nil
}

// SendTags sends the complete 1.21.4 network-safe tag snapshot. Tag entries
// use the same IDs established by Registry Data and Mojang's built-in registry
// report, so the client can bind and freeze the rebuilt registry access.
func (p *VanillaProvider) SendTags(conn *network.ClientConn) error {
	pkt := buildUpdateTagsPacket(vanillaNetworkTags)
	if err := conn.WritePacket(pkt); err != nil {
		return fmt.Errorf("registry: sending Update Tags: %w", err)
	}
	tagCount := 0
	for _, registry := range vanillaNetworkTags {
		tagCount += len(registry.Tags)
	}
	slog.Info("java configuration tags sent",
		"registries", len(vanillaNetworkTags),
		"tags", tagCount,
		"payloadLength", len(pkt.Data),
	)
	return nil
}

// DimensionTypeID returns the ID assigned by the dimension_type Registry Data
// packet sent during this connection's Configuration state.
func (p *VanillaProvider) DimensionTypeID(name string) (int32, error) {
	for _, registry := range vanillaNetworkRegistries {
		if registry.Name != dimensionTypeRegistryName {
			continue
		}
		for id, entry := range registry.Entries {
			if entry == name {
				return int32(id), nil
			}
		}
		return 0, fmt.Errorf("registry: unknown dimension type %q", name)
	}
	return 0, fmt.Errorf("registry: %s was not synchronized", dimensionTypeRegistryName)
}

func buildRegistryDataPacket(registry networkRegistry) *protocol.Packet {
	b := protocol.NewBuilder(packetIDRegistryData).
		String(registry.Name).
		VarInt(int32(len(registry.Entries)))
	for _, entry := range registry.Entries {
		b.String(entry).
			Bool(false) // no NBT follows: value comes from the selected known pack
	}
	return b.Build()
}

func buildUpdateTagsPacket(registries []networkTagRegistry) *protocol.Packet {
	b := protocol.NewBuilder(packetIDUpdateTags).
		VarInt(int32(len(registries)))
	for _, registry := range registries {
		b.String(registry.Name).
			VarInt(int32(len(registry.Tags)))
		for _, tag := range registry.Tags {
			b.String(tag.Name).
				VarInt(int32(len(tag.Entries)))
			for _, registryID := range tag.Entries {
				b.VarInt(registryID)
			}
		}
	}
	return b.Build()
}

func samePacks(got, want []Pack) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func mustLoadNetworkRegistries() []networkRegistry {
	data, err := gamedata.FS.ReadFile("java/1.21.4/network_registries.json")
	if err != nil {
		panic(fmt.Sprintf("registry: reading network_registries.json: %v", err))
	}

	var file networkRegistryFile
	if err := json.Unmarshal(data, &file); err != nil {
		panic(fmt.Sprintf("registry: parsing network_registries.json: %v", err))
	}
	if len(file.Registries) != len(synchronizedRegistryOrder769) {
		panic(fmt.Sprintf("registry: synchronized registry count is %d, want %d", len(file.Registries), len(synchronizedRegistryOrder769)))
	}

	for i, registry := range file.Registries {
		if registry.Name != synchronizedRegistryOrder769[i] {
			panic(fmt.Sprintf("registry: synchronized registry %d is %q, want %q", i, registry.Name, synchronizedRegistryOrder769[i]))
		}
		if len(registry.Entries) == 0 {
			panic(fmt.Sprintf("registry: synchronized registry %s is empty", registry.Name))
		}
		seen := make(map[string]struct{}, len(registry.Entries))
		for id, entry := range registry.Entries {
			if entry == "" {
				panic(fmt.Sprintf("registry: %s entry %d is empty", registry.Name, id))
			}
			if _, duplicate := seen[entry]; duplicate {
				panic(fmt.Sprintf("registry: %s contains duplicate entry %q", registry.Name, entry))
			}
			seen[entry] = struct{}{}
		}
	}
	return file.Registries
}

func mustLoadNetworkTags() []networkTagRegistry {
	data, err := gamedata.FS.ReadFile("java/1.21.4/network_tags.json")
	if err != nil {
		panic(fmt.Sprintf("registry: reading network_tags.json: %v", err))
	}

	var file networkTagFile
	if err := json.Unmarshal(data, &file); err != nil {
		panic(fmt.Sprintf("registry: parsing network_tags.json: %v", err))
	}
	if len(file.Registries) != len(networkTagRegistryOrder769) {
		panic(fmt.Sprintf("registry: network tag registry count is %d, want %d", len(file.Registries), len(networkTagRegistryOrder769)))
	}

	dynamicSizes := make(map[string]int32, len(vanillaNetworkRegistries))
	for _, registry := range vanillaNetworkRegistries {
		dynamicSizes[registry.Name] = int32(len(registry.Entries))
	}
	for registryIndex, registry := range file.Registries {
		if registry.Name != networkTagRegistryOrder769[registryIndex] {
			panic(fmt.Sprintf("registry: network tag registry %d is %q, want %q", registryIndex, registry.Name, networkTagRegistryOrder769[registryIndex]))
		}
		if registry.RegistrySize <= 0 {
			panic(fmt.Sprintf("registry: network tag registry %s has invalid size %d", registry.Name, registry.RegistrySize))
		}
		if size, dynamic := dynamicSizes[registry.Name]; dynamic && size != registry.RegistrySize {
			panic(fmt.Sprintf("registry: network tag registry %s size is %d, synchronized registry assigns %d", registry.Name, registry.RegistrySize, size))
		}
		seenTags := make(map[string]struct{}, len(registry.Tags))
		for _, tag := range registry.Tags {
			if tag.Name == "" {
				panic(fmt.Sprintf("registry: %s contains an empty tag name", registry.Name))
			}
			if _, duplicate := seenTags[tag.Name]; duplicate {
				panic(fmt.Sprintf("registry: %s contains duplicate tag %q", registry.Name, tag.Name))
			}
			seenTags[tag.Name] = struct{}{}

			seenIDs := make(map[int32]struct{}, len(tag.Entries))
			for _, id := range tag.Entries {
				if id < 0 || id >= registry.RegistrySize {
					panic(fmt.Sprintf("registry: %s tag %s contains out-of-range ID %d (size %d)", registry.Name, tag.Name, id, registry.RegistrySize))
				}
				if _, duplicate := seenIDs[id]; duplicate {
					panic(fmt.Sprintf("registry: %s tag %s contains duplicate ID %d", registry.Name, tag.Name, id))
				}
				seenIDs[id] = struct{}{}
			}
		}
	}
	return file.Registries
}
