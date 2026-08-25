package world

const strongholdRoomRadius = 7

// addStrongholdStructures places the underground portal room at the same
// concentric-ring position returned to Eye-of-Ender launches.
func (g *OverworldGenerator) addStrongholdStructures(chunk *Chunk) {
	minX, minZ := int(chunk.X)*SectionSize, int(chunk.Z)*SectionSize
	maxX, maxZ := minX+SectionSize-1, minZ+SectionSize-1
	for _, stronghold := range g.strongholdRingChunks() {
		centerX, centerZ := stronghold[0]*16+8, stronghold[1]*16+8
		if centerX+strongholdRoomRadius < minX || centerX-strongholdRoomRadius > maxX ||
			centerZ+strongholdRoomRadius < minZ || centerZ-strongholdRoomRadius > maxZ {
			continue
		}
		roomY := max(WorldMinY+10, g.SurfaceHeight(centerX, centerZ)-30)
		g.placeStrongholdPortalRoom(chunk, centerX, roomY, centerZ)
	}
}

func (g *OverworldGenerator) placeStrongholdPortalRoom(chunk *Chunk, centerX, floorY, centerZ int) {
	stone := Block{Namespace: "minecraft", Name: "stone_bricks"}
	air := Air
	for dx := -6; dx <= 6; dx++ {
		for dz := -7; dz <= 5; dz++ {
			for dy := 0; dy <= 6; dy++ {
				material := air
				if dy == 0 || dy == 6 || dx == -6 || dx == 6 || dz == -7 || dz == 5 {
					material = stone
					hash := generatedHash(g.seed, centerX+dx, floorY+dy, centerZ+dz)
					if hash%20 == 0 {
						material.Name = "cracked_stone_bricks"
					} else if hash%13 == 0 {
						material.Name = "mossy_stone_bricks"
					}
				}
				setStrongholdBlock(chunk, centerX+dx, floorY+dy, centerZ+dz, material)
			}
		}
	}
	frame := func(x, z int, facing string) {
		setStrongholdBlock(chunk, x, floorY+1, z, Block{
			Namespace: "minecraft", Name: "end_portal_frame",
			Properties: map[string]string{"eye": "false", "facing": facing},
		})
	}
	for offset := -1; offset <= 1; offset++ {
		frame(centerX+offset, centerZ-2, "south")
		frame(centerX+offset, centerZ+2, "north")
		frame(centerX-2, centerZ+offset, "east")
		frame(centerX+2, centerZ+offset, "west")
	}
}

func setStrongholdBlock(chunk *Chunk, worldX, y, worldZ int, block Block) {
	localX := worldX - int(chunk.X)*SectionSize
	localZ := worldZ - int(chunk.Z)*SectionSize
	if localX < 0 || localX >= SectionSize || localZ < 0 || localZ >= SectionSize || y < WorldMinY || y > WorldMaxY {
		return
	}
	setGeneratedBlock(chunk, localX, y, localZ, block)
}
