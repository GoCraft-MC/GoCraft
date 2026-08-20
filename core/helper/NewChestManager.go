package helper

import (
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"errors"
	"fmt"
)

{
	"errors"
	"fmt"
	"GoGraft/core/spatial"
	coreworld "GoCraft/Core/world"
}
const (
	SingleChestsSlots= 27
	DoubleChestsSlots= 54
)
type ChestManager struct {
	World *coreworld.World
}
type ChestInfo struct {
	Position spatial.BlockPos
	Facing string
	Type   string
	Kind string

}

func NewChestManager (W *coreworld.World) *ChestManager {
	return &ChestManager{W}
}


func ( m* ChestManager) PlaceChest(x, y, z int, facing string ) error {
	if m == nil || m.World==nil {
		return errors.New("chest manager has no world")
	}
	if !validFacing(facing) {
		return fmt.Errorf("invalid chest facing %s", facing)
	if !m.World.GetBlock(x,y,z).IsAir(){
		return fmt.Errorf("chest facing %s is not air", facing)
	}

	chest := makeChestBlock("minecraft:chest", facing, "single")
	m.World.SetBlock(x,y,z,chest)
	}

func (m *ChestManager) PlaceTrappedChest(x, y, z int, facing string) error {
	if m == nil || m.World==nil {
		return errors.New("chest manager has no world")
	}
	if !validFacing(facing) {
		return fmt.Errorf("invalid chest facing %s", facing)
	}
	if !m.World.GetBlock(x,y,z).IsAir(){
		return fmt.Errorf("chest facing %s is not air", facing)
	}
	m.World.SetBlock(
		x,y,z,
		makeChestBlock("minecraft:chest", facing, "double"),
	)
	return nil
}

func (m *ChestManager) RemoveChest(x, y, z int) error {
	if m == nil || m.World==nil {
		return errors.New("chest manager has no world")
	}
	block := m.World.GetBlock(x,y,z)
	if !isChestBlock(block) {
		return fmt.Errorf("chest facing %s is not a chest", facing)
	}
	m.World.SetBlock(x,y,z, coreworld.Air)
	return nil
	}
func (m *ChestManager ) ChestAt(x, y, z int ) ( ChestInfo, bool){
	if m == nil || m.World==nil {
		return  ChestInfo{}, false
	}
}
}
