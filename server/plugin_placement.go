package server

import (
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func (s *Server) allowBedrockPlacement(p *player.Player, x, y, z int, placed coreworld.Block) bool {
	w := s.bedrockWorld()
	if s.plugins == nil || s.plugins.EmitBlockPlace(p, spatial.BlockPos{X: int32(x), Y: int32(y), Z: int32(z)},
		placed, w.GetBlock(x, y, z), int64(p.Dimension)) {
		return true
	}
	s.resyncBedrockEventBlocks(p, x, y, z)
	return false
}

func (s *Server) resyncBedrockEventBlocks(p *player.Player, x, y, z int) {
	if s.bedrockListener == nil {
		return
	}
	observe := s.bedrockListener.DimensionBlockObserver(p.Dimension)
	for _, offset := range [][3]int{{}, {0, 1, 0}, {0, -1, 0}, {1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}} {
		bx, by, bz := x+offset[0], y+offset[1], z+offset[2]
		observe(coreworld.BlockChange{X: bx, Y: by, Z: bz, Block: s.bedrockWorld().GetBlock(bx, by, bz)})
	}
	// The adapter's regular canonical inventory sync restores predicted stacks.
}
