package helper

import (
	"errors"
	"fmt"

	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

const (
	SingleChestSlots = 27
	DoubleChestSlots = 54

	// Backward-compatible names kept for callers of the first implementation.
	SingleChestsSlots = SingleChestSlots
	DoubleChestsSlots = DoubleChestSlots
)

type ChestManager struct {
	World *coreworld.World
}

type ChestInfo struct {
	Position spatial.BlockPos
	Facing   string
	Type     string
	Kind     string
}

func NewChestManager(w *coreworld.World) *ChestManager {
	return &ChestManager{World: w}
}

func (m *ChestManager) PlaceChest(x, y, z int, facing string) error {
	return m.place(x, y, z, "minecraft:chest", facing)
}

func (m *ChestManager) PlaceTrappedChest(x, y, z int, facing string) error {
	return m.place(x, y, z, "minecraft:trapped_chest", facing)
}

func (m *ChestManager) place(x, y, z int, kind, facing string) error {
	if m == nil || m.World == nil {
		return errors.New("chest manager has no world")
	}
	if !validFacing(facing) {
		return fmt.Errorf("invalid chest facing %q", facing)
	}
	if !m.World.GetBlock(x, y, z).IsAir() {
		return fmt.Errorf("chest position %d %d %d is not air", x, y, z)
	}
	m.World.SetBlock(x, y, z, makeChestBlock(kind, facing, "single"))
	return nil
}

func (m *ChestManager) RemoveChest(x, y, z int) error {
	if m == nil || m.World == nil {
		return errors.New("chest manager has no world")
	}
	block := m.World.GetBlock(x, y, z)
	if !isChestBlock(block) {
		return fmt.Errorf("block at %d %d %d is not a chest", x, y, z)
	}
	m.World.SetBlock(x, y, z, coreworld.Air)
	return nil
}

func (m *ChestManager) ChestAt(x, y, z int) (ChestInfo, bool) {
	if m == nil || m.World == nil {
		return ChestInfo{}, false
	}
	block := m.World.GetBlock(x, y, z)
	if !isChestBlock(block) {
		return ChestInfo{}, false
	}
	return ChestInfo{
		Position: spatial.BlockPos{X: int32(x), Y: int32(y), Z: int32(z)},
		Facing:   block.Properties["facing"],
		Type:     block.Properties["type"],
		Kind:     block.ResourceLocation(),
	}, true
}

func validFacing(facing string) bool {
	switch facing {
	case "north", "south", "east", "west":
		return true
	default:
		return false
	}
}

func makeChestBlock(kind, facing, chestType string) coreworld.Block {
	namespace, name := "minecraft", kind
	for i := 0; i < len(kind); i++ {
		if kind[i] == ':' {
			namespace, name = kind[:i], kind[i+1:]
			break
		}
	}
	return coreworld.Block{
		Namespace: namespace,
		Name:      name,
		Properties: map[string]string{
			"facing":      facing,
			"type":        chestType,
			"waterlogged": "false",
		},
	}
}

func isChestBlock(block coreworld.Block) bool {
	switch block.ResourceLocation() {
	case "minecraft:chest", "minecraft:trapped_chest":
		return true
	default:
		return false
	}
}
