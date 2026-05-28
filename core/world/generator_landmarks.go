package world

// addBiomeLandmarks supplies the large, biome-defining placed features that
// materially change the silhouette or underwater appearance of a chunk.
func (g *OverworldGenerator) addBiomeLandmarks(c *Chunk) {
	g.addIceSpikes(c)
	g.addIcebergs(c)
	g.addCoralReefs(c)
}

func (g *OverworldGenerator) addIceSpikes(c *Chunk) {
	const cellSize = 18
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	for cellX := floorDiv(chunkMinX-4, cellSize); cellX <= floorDiv(chunkMinX+SectionSize+4, cellSize); cellX++ {
		for cellZ := floorDiv(chunkMinZ-4, cellSize); cellZ <= floorDiv(chunkMinZ+SectionSize+4, cellSize); cellZ++ {
			state := mix64(uint64(g.seed) ^ uint64(int64(cellX))*0x8da6b343 ^ uint64(int64(cellZ))*0xd8163841 ^ 0x6963657370696b65)
			if state%3 != 0 {
				continue
			}
			worldX := cellX*cellSize + 4 + int(nextRandom(&state)*10)
			worldZ := cellZ*cellSize + 4 + int(nextRandom(&state)*10)
			if g.BiomeAt(worldX, worldZ) != "minecraft:ice_spikes" {
				continue
			}
			baseY := g.SurfaceHeight(worldX, worldZ)
			height := 7 + int(nextRandom(&state)*15)
			for dy := 1; dy <= height; dy++ {
				radius := 0
				if dy <= height/3 {
					radius = 2
				} else if dy <= height*2/3 {
					radius = 1
				}
				for dx := -radius; dx <= radius; dx++ {
					for dz := -radius; dz <= radius; dz++ {
						if dx*dx+dz*dz > radius*radius+1 {
							continue
						}
						material := packedIceBlock
						if dy < height/2 && (state+uint64(dx*31+dz*17+dy))%23 == 0 {
							material = blueIceBlock
						}
						setLandmarkBlock(c, worldX+dx, baseY+dy, worldZ+dz, material)
					}
				}
			}
		}
	}
}

func (g *OverworldGenerator) addIcebergs(c *Chunk) {
	const cellSize = 52
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	for cellX := floorDiv(chunkMinX-12, cellSize); cellX <= floorDiv(chunkMinX+SectionSize+12, cellSize); cellX++ {
		for cellZ := floorDiv(chunkMinZ-12, cellSize); cellZ <= floorDiv(chunkMinZ+SectionSize+12, cellSize); cellZ++ {
			state := mix64(uint64(g.seed) ^ uint64(int64(cellX))*0x8da6b343 ^ uint64(int64(cellZ))*0xd8163841 ^ 0x6963656265726773)
			if state%4 != 0 {
				continue
			}
			worldX := cellX*cellSize + 10 + int(nextRandom(&state)*32)
			worldZ := cellZ*cellSize + 10 + int(nextRandom(&state)*32)
			biome := g.BiomeAt(worldX, worldZ)
			if biome != "minecraft:frozen_ocean" && biome != "minecraft:deep_frozen_ocean" {
				continue
			}
			radius := 4 + int(nextRandom(&state)*5)
			height := 5 + int(nextRandom(&state)*13)
			for dy := -4; dy <= height; dy++ {
				vertical := float64(dy) / float64(height+2)
				layerRadius := int(float64(radius) * (1 - vertical*vertical))
				if layerRadius < 1 {
					layerRadius = 1
				}
				for dx := -layerRadius; dx <= layerRadius; dx++ {
					for dz := -layerRadius; dz <= layerRadius; dz++ {
						if dx*dx+dz*dz > layerRadius*layerRadius {
							continue
						}
						material := packedIceBlock
						if dy < 1 && (state+uint64((dx+16)*31+(dz+16)*17+dy+8))%11 == 0 {
							material = blueIceBlock
						}
						setLandmarkBlock(c, worldX+dx, SeaLevel+dy, worldZ+dz, material)
					}
				}
			}
		}
	}
}

func (g *OverworldGenerator) addCoralReefs(c *Chunk) {
	coral := [...]Block{tubeCoralBlock, brainCoralBlock, bubbleCoralBlock, fireCoralBlock, hornCoralBlock}
	const cellSize = 7
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	for cellX := floorDiv(chunkMinX, cellSize); cellX <= floorDiv(chunkMinX+SectionSize-1, cellSize); cellX++ {
		for cellZ := floorDiv(chunkMinZ, cellSize); cellZ <= floorDiv(chunkMinZ+SectionSize-1, cellSize); cellZ++ {
			state := mix64(uint64(g.seed) ^ uint64(int64(cellX))*0x8da6b343 ^ uint64(int64(cellZ))*0xd8163841 ^ 0x636f72616c726565)
			if state%3 != 0 {
				continue
			}
			worldX := cellX*cellSize + int(nextRandom(&state)*cellSize)
			worldZ := cellZ*cellSize + int(nextRandom(&state)*cellSize)
			biome := g.BiomeAt(worldX, worldZ)
			if biome != "minecraft:warm_ocean" && biome != "minecraft:lukewarm_ocean" {
				continue
			}
			floorY := g.SurfaceHeight(worldX, worldZ)
			if floorY >= SeaLevel-2 {
				continue
			}
			height := 1 + int(nextRandom(&state)*3)
			material := coral[state>>16%uint64(len(coral))]
			for dy := 1; dy <= height; dy++ {
				setLandmarkBlock(c, worldX, floorY+dy, worldZ, material)
			}
			if state%4 == 0 {
				setLandmarkBlock(c, worldX+1, floorY+1, worldZ, coral[(state>>24)%uint64(len(coral))])
			}
		}
	}
}

func setLandmarkBlock(c *Chunk, worldX, y, worldZ int, material Block) {
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	localX := worldX - chunkMinX
	localZ := worldZ - chunkMinZ
	if localX < 0 || localX >= SectionSize || localZ < 0 || localZ >= SectionSize || y < WorldMinY || y > WorldMaxY {
		return
	}
	existing := generatedBlock(c, localX, y, localZ).ResourceLocation()
	if existing != "minecraft:air" && existing != "minecraft:water" && existing != "minecraft:ice" && existing != "minecraft:snow_block" {
		return
	}
	setGeneratedBlock(c, localX, y, localZ, material)
}
