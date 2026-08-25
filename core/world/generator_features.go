package world

import "math"

func (g *OverworldGenerator) carveCaves(c *Chunk, heights [SectionSize * SectionSize]int) {
	// Each neighbouring source chunk may start a short deterministic cave worm.
	// Looking three chunks out makes worms continuous across chunk boundaries.
	for sourceX := c.X - 3; sourceX <= c.X+3; sourceX++ {
		for sourceZ := c.Z - 3; sourceZ <= c.Z+3; sourceZ++ {
			state := g.featureHash(sourceX, sourceZ, 0x636176655f776f72)
			if state%4 != 0 {
				continue
			}
			worms := 1 + int((state>>8)%2)
			for worm := 0; worm < worms; worm++ {
				state = mix64(state + uint64(worm)*0x9e3779b97f4a7c15)
				px := float64(int(sourceX)*SectionSize) + nextRandom(&state)*SectionSize
				pz := float64(int(sourceZ)*SectionSize) + nextRandom(&state)*SectionSize
				py := -45 + nextRandom(&state)*150
				yaw := nextRandom(&state) * math.Pi * 2
				pitch := (nextRandom(&state) - 0.5) * 0.32
				steps := 18 + int(nextRandom(&state)*18)
				for step := 0; step < steps; step++ {
					yaw += (nextRandom(&state) - 0.5) * 0.38
					pitch = pitch*0.72 + (nextRandom(&state)-0.5)*0.19
					travel := 1.8 + nextRandom(&state)*0.9
					px += math.Cos(yaw) * math.Cos(pitch) * travel
					pz += math.Sin(yaw) * math.Cos(pitch) * travel
					py += math.Sin(pitch) * travel
					radius := 1.7 + nextRandom(&state)*2.4 + math.Sin(float64(step)*math.Pi/float64(steps))*1.1
					g.carveSphere(c, heights, px, py, pz, radius, radius*0.72)
				}
			}
		}
	}

	// Sparse large chambers keep the underground from being only narrow tubes.
	cellX := int32(floorDiv(int(c.X), 4))
	cellZ := int32(floorDiv(int(c.Z), 4))
	for sx := cellX - 1; sx <= cellX+1; sx++ {
		for sz := cellZ - 1; sz <= cellZ+1; sz++ {
			state := g.featureHash(sx, sz, 0x63617665726e3031)
			if state%3 != 0 {
				continue
			}
			px := float64(int(sx)*64) + nextRandom(&state)*64
			pz := float64(int(sz)*64) + nextRandom(&state)*64
			py := -38 + nextRandom(&state)*68
			radius := 5.5 + nextRandom(&state)*5.5
			g.carveSphere(c, heights, px, py, pz, radius, radius*0.55)
			g.carveSphere(c, heights, px+radius*0.7, py+2, pz-radius*0.35, radius*0.72, radius*0.46)
		}
	}
}

func (g *OverworldGenerator) carveSphere(c *Chunk, heights [SectionSize * SectionSize]int, cx, cy, cz, radiusX, radiusY float64) {
	chunkMinX, chunkMinZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	minX := maxInt(chunkMinX, int(math.Floor(cx-radiusX)))
	maxX := minInt(chunkMinX+SectionSize-1, int(math.Ceil(cx+radiusX)))
	minZ := maxInt(chunkMinZ, int(math.Floor(cz-radiusX)))
	maxZ := minInt(chunkMinZ+SectionSize-1, int(math.Ceil(cz+radiusX)))
	minY := maxInt(WorldMinY+6, int(math.Floor(cy-radiusY)))
	maxY := minInt(128, int(math.Ceil(cy+radiusY)))
	if minX > maxX || minZ > maxZ || minY > maxY {
		return
	}
	for worldX := minX; worldX <= maxX; worldX++ {
		localX := worldX - chunkMinX
		dx := (float64(worldX) + 0.5 - cx) / radiusX
		for worldZ := minZ; worldZ <= maxZ; worldZ++ {
			localZ := worldZ - chunkMinZ
			dz := (float64(worldZ) + 0.5 - cz) / radiusX
			surfaceLimit := heights[localZ*SectionSize+localX] - 6
			for y := minY; y <= maxY && y <= surfaceLimit; y++ {
				dy := (float64(y) + 0.5 - cy) / radiusY
				if dx*dx+dy*dy+dz*dz > 1 {
					continue
				}
				existing := generatedBlock(c, localX, y, localZ).ResourceLocation()
				if existing == "minecraft:stone" || existing == "minecraft:deepslate" {
					setGeneratedBlock(c, localX, y, localZ, g.aquiferMaterial(worldX, y, worldZ))
				}
			}
		}
	}
}

type oreDefinition struct {
	salt             uint64
	attempts, size   int
	minY, maxY       int
	stone, deepslate Block
	mountainsOnly    bool
}

func (g *OverworldGenerator) addOres(c *Chunk) {
	ores := []oreDefinition{
		{0x6f72655f636f616c, 18, 14, 0, 192, coalOreBlock, deepslateCoalOreBlock, false},
		{0x6f72655f69726f6e, 18, 10, -48, 160, ironOreBlock, deepslateIronOreBlock, false},
		{0x6f72655f636f7070, 12, 11, -16, 112, copperOreBlock, deepslateCopperOreBlock, false},
		{0x6f72655f676f6c64, 5, 9, -48, 32, goldOreBlock, deepslateGoldOreBlock, false},
		{0x6f72655f72656473, 8, 8, WorldMinY + 4, 16, redstoneOreBlock, deepslateRedstoneOreBlock, false},
		{0x6f72655f6c617069, 4, 7, WorldMinY + 4, 64, lapisOreBlock, deepslateLapisOreBlock, false},
		{0x6f72655f6469616d, 5, 8, WorldMinY + 4, 16, diamondOreBlock, deepslateDiamondOreBlock, false},
		{0x6f72655f656d6572, 4, 5, -16, 176, emeraldOreBlock, deepslateEmeraldOreBlock, true},
	}
	for _, ore := range ores {
		for sourceX := c.X - 1; sourceX <= c.X+1; sourceX++ {
			for sourceZ := c.Z - 1; sourceZ <= c.Z+1; sourceZ++ {
				state := g.featureHash(sourceX, sourceZ, ore.salt)
				for attempt := 0; attempt < ore.attempts; attempt++ {
					state = mix64(state + uint64(attempt)*0x9e3779b97f4a7c15)
					worldX := int(sourceX)*SectionSize + int(nextRandom(&state)*SectionSize)
					worldZ := int(sourceZ)*SectionSize + int(nextRandom(&state)*SectionSize)
					// Two samples form a triangular distribution, avoiding hard bands.
					yFactor := (nextRandom(&state) + nextRandom(&state)) * 0.5
					worldY := ore.minY + int(yFactor*float64(ore.maxY-ore.minY+1))
					if ore.mountainsOnly && !isMountainBiome(g.BiomeAt(worldX, worldZ)) {
						continue
					}
					g.placeOreVein(c, worldX, worldY, worldZ, ore, &state)
				}
			}
		}
	}
}

func (g *OverworldGenerator) placeOreVein(c *Chunk, startX, startY, startZ int, ore oreDefinition, state *uint64) {
	px, py, pz := float64(startX), float64(startY), float64(startZ)
	for step := 0; step < ore.size; step++ {
		px += (nextRandom(state) - 0.5) * 2.2
		py += (nextRandom(state) - 0.5) * 1.5
		pz += (nextRandom(state) - 0.5) * 2.2
		radius := 0.75 + nextRandom(state)*0.8
		g.replaceOreSphere(c, px, py, pz, radius, ore)
	}
}

func (g *OverworldGenerator) replaceOreSphere(c *Chunk, cx, cy, cz, radius float64, ore oreDefinition) {
	chunkMinX, chunkMinZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	minX := maxInt(chunkMinX, int(math.Floor(cx-radius)))
	maxX := minInt(chunkMinX+15, int(math.Ceil(cx+radius)))
	minZ := maxInt(chunkMinZ, int(math.Floor(cz-radius)))
	maxZ := minInt(chunkMinZ+15, int(math.Ceil(cz+radius)))
	for worldX := minX; worldX <= maxX; worldX++ {
		for worldZ := minZ; worldZ <= maxZ; worldZ++ {
			for y := maxInt(WorldMinY+1, int(math.Floor(cy-radius))); y <= minInt(WorldMaxY, int(math.Ceil(cy+radius))); y++ {
				dx, dy, dz := float64(worldX)-cx, float64(y)-cy, float64(worldZ)-cz
				if dx*dx+dy*dy+dz*dz > radius*radius {
					continue
				}
				localX, localZ := worldX-chunkMinX, worldZ-chunkMinZ
				existing := generatedBlock(c, localX, y, localZ).ResourceLocation()
				switch existing {
				case "minecraft:stone":
					setGeneratedBlock(c, localX, y, localZ, ore.stone)
				case "minecraft:deepslate":
					setGeneratedBlock(c, localX, y, localZ, ore.deepslate)
				}
			}
		}
	}
}

func (g *OverworldGenerator) addVegetation(c *Chunk) {
	const cellSize = 10
	chunkMinX, chunkMinZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	for cellX := floorDiv(chunkMinX-5, cellSize); cellX <= floorDiv(chunkMinX+20, cellSize); cellX++ {
		for cellZ := floorDiv(chunkMinZ-5, cellSize); cellZ <= floorDiv(chunkMinZ+20, cellSize); cellZ++ {
			state := mix64(uint64(g.seed) ^ uint64(int64(cellX))*0x8da6b343 ^ uint64(int64(cellZ))*0xd8163841 ^ 0x7472656573303032)
			worldX := cellX*cellSize + 1 + int(nextRandom(&state)*8)
			worldZ := cellZ*cellSize + 1 + int(nextRandom(&state)*8)
			if worldX*worldX+worldZ*worldZ < 24*24 {
				continue
			}
			biome := g.BiomeAt(worldX, worldZ)
			denominator := vegetationDenominator(biome)
			if denominator == 0 || int(state%uint64(denominator)) != 0 {
				continue
			}
			groundY := g.SurfaceHeight(worldX, worldZ)
			if groundY <= SeaLevel+1 || !g.treeSlopeSafe(worldX, worldZ, groundY) {
				continue
			}
			if biome == "minecraft:desert" || biome == "minecraft:badlands" || biome == "minecraft:eroded_badlands" {
				g.placeCactus(c, worldX, groundY, worldZ, 2+int((state>>20)%3))
				continue
			}
			log, leaves, style := treeBlocksForBiome(biome, state)
			height := 4 + int((state>>24)%3)
			if biome == "minecraft:jungle" {
				height += 3
			}
			g.placeTree(c, worldX, groundY, worldZ, height, log, leaves, style)
		}
	}
}

func vegetationDenominator(biome string) int {
	switch biome {
	case "minecraft:jungle", "minecraft:bamboo_jungle":
		return 1
	case "minecraft:sparse_jungle":
		return 3
	case "minecraft:dark_forest", "minecraft:pale_garden":
		return 1
	case "minecraft:forest", "minecraft:flower_forest", "minecraft:windswept_forest":
		return 2
	case "minecraft:birch_forest", "minecraft:old_growth_birch_forest":
		return 2
	case "minecraft:cherry_grove":
		return 3
	case "minecraft:taiga", "minecraft:snowy_taiga", "minecraft:grove":
		return 3
	case "minecraft:old_growth_pine_taiga", "minecraft:old_growth_spruce_taiga":
		return 1
	case "minecraft:swamp", "minecraft:mangrove_swamp":
		return 4
	case "minecraft:savanna", "minecraft:savanna_plateau", "minecraft:windswept_savanna":
		return 4
	case "minecraft:plains", "minecraft:sunflower_plains":
		return 8
	case "minecraft:meadow":
		return 5
	case "minecraft:desert", "minecraft:badlands", "minecraft:eroded_badlands":
		return 4
	case "minecraft:wooded_badlands":
		return 5
	case "minecraft:windswept_hills":
		return 12
	default:
		return 0
	}
}

func treeBlocksForBiome(biome string, state uint64) (Block, Block, string) {
	switch biome {
	case "minecraft:birch_forest":
		return birchLogBlock, birchLeafBlock, "round"
	case "minecraft:old_growth_birch_forest":
		return birchLogBlock, birchLeafBlock, "tallbirch"
	case "minecraft:taiga", "minecraft:snowy_taiga", "minecraft:grove":
		if state%3 == 0 {
			return spruceLogBlock, spruceLeafBlock, "largespruce"
		}
		return spruceLogBlock, spruceLeafBlock, "spruce"
	case "minecraft:old_growth_pine_taiga":
		return spruceLogBlock, spruceLeafBlock, "spruce"
	case "minecraft:old_growth_spruce_taiga":
		return spruceLogBlock, spruceLeafBlock, "largespruce"
	case "minecraft:savanna", "minecraft:savanna_plateau", "minecraft:windswept_savanna":
		return acaciaLogBlock, acaciaLeafBlock, "acacia"
	case "minecraft:jungle", "minecraft:sparse_jungle":
		if state%5 == 0 {
			return jungleLogBlock, jungleLeafBlock, "largejungle"
		}
		return jungleLogBlock, jungleLeafBlock, "round"
	case "minecraft:bamboo_jungle":
		if state%4 == 0 {
			return jungleLogBlock, jungleLeafBlock, "largejungle"
		}
		return jungleLogBlock, jungleLeafBlock, "round"
	case "minecraft:dark_forest":
		return darkOakLogBlock, darkOakLeafBlock, "darkoak"
	case "minecraft:pale_garden":
		return paleOakLogBlock, paleOakLeafBlock, "darkoak"
	case "minecraft:mangrove_swamp":
		return mangroveLogBlock, mangroveLeafBlock, "round"
	case "minecraft:windswept_forest", "minecraft:wooded_badlands":
		return oakLogBlock, oakLeafBlock, "round"
	case "minecraft:cherry_grove":
		return cherryLogBlock, cherryLeafBlock, "cherry"
	case "minecraft:flower_forest":
		if state&3 == 0 {
			return birchLogBlock, birchLeafBlock, "round"
		}
		return oakLogBlock, oakLeafBlock, "round"
	case "minecraft:forest":
		if state&3 == 0 {
			return birchLogBlock, birchLeafBlock, "round"
		}
	}
	return oakLogBlock, oakLeafBlock, "round"
}

func (g *OverworldGenerator) treeSlopeSafe(x, z, groundY int) bool {
	for _, offset := range [][2]int{{-2, 0}, {2, 0}, {0, -2}, {0, 2}} {
		if absInt(g.SurfaceHeight(x+offset[0], z+offset[1])-groundY) > 3 {
			return false
		}
	}
	return true
}

func (g *OverworldGenerator) placeTree(c *Chunk, worldX, groundY, worldZ, height int, log, leaves Block, style string) {
	topY := groundY + height

	switch style {
	case "darkoak":
		// 2×2 trunk, wide flat canopy
		topY2 := groundY + 4 + height%3
		for y := groundY + 1; y <= topY2; y++ {
			for dx := 0; dx <= 1; dx++ {
				for dz := 0; dz <= 1; dz++ {
					setFeatureBlock(c, worldX+dx, y, worldZ+dz, log, true)
				}
			}
		}
		for dy := -3; dy <= 1; dy++ {
			radius := 3
			if dy == 1 {
				radius = 2
			}
			if dy == -3 {
				radius = 2
			}
			for dx := -radius; dx <= radius+1; dx++ {
				for dz := -radius; dz <= radius+1; dz++ {
					if dy == 1 && absInt(dx) == radius && absInt(dz) == radius {
						continue
					}
					setFeatureBlock(c, worldX+dx, topY2+dy, worldZ+dz, leaves, false)
				}
			}
		}

	case "acacia":
		roll := generatedHash(g.seed, worldX, groundY, worldZ)
		lean := horizontalCropDirections[int(roll&3)]
		leanHeight := height - int(roll>>2&3) - 1
		leanSteps := 3 - int(roll>>4%3)
		trunkX, trunkZ := worldX, worldZ
		for dy := 0; dy < height; dy++ {
			if dy >= leanHeight && leanSteps > 0 {
				trunkX, trunkZ = trunkX+lean.dx, trunkZ+lean.dz
				leanSteps--
			}
			setFeatureBlock(c, trunkX, groundY+1+dy, trunkZ, log, true)
		}
		g.placeAcaciaFeatureFoliage(c, trunkX, groundY+height+1, trunkZ, 1, leaves)
		branch := horizontalCropDirections[int(roll>>8&3)]
		if branch != lean {
			branchY := max(1, leanHeight-int(roll>>10&1)-1)
			branchX, branchZ, branchTop := worldX, worldZ, groundY+1+branchY
			steps := 1 + int(roll>>12%3)
			for step := 0; step < steps && branchY+step < height; step++ {
				branchX, branchZ = branchX+branch.dx, branchZ+branch.dz
				branchTop = groundY + 1 + branchY + step
				setFeatureBlock(c, branchX, branchTop, branchZ, log, true)
			}
			g.placeAcaciaFeatureFoliage(c, branchX, branchTop+1, branchZ, 0, leaves)
		}

	case "largejungle":
		// 2×2 tall trunk
		topY2 := groundY + height + 4
		for y := groundY + 1; y <= topY2; y++ {
			for dx := 0; dx <= 1; dx++ {
				for dz := 0; dz <= 1; dz++ {
					setFeatureBlock(c, worldX+dx, y, worldZ+dz, log, true)
				}
			}
		}
		for dy := -3; dy <= 1; dy++ {
			radius := 3
			if dy >= 0 {
				radius = 2
			}
			for dx := -radius; dx <= radius+1; dx++ {
				for dz := -radius; dz <= radius+1; dz++ {
					if absInt(dx) == radius && absInt(dz) == radius {
						continue
					}
					setFeatureBlock(c, worldX+dx, topY2+dy, worldZ+dz, leaves, false)
				}
			}
		}

	case "largespruce":
		// Extra-tall spruce with tiered canopy
		topY2 := groundY + height + 3
		for y := groundY + 1; y <= topY2; y++ {
			setFeatureBlock(c, worldX, y, worldZ, log, true)
		}
		for dy := -6; dy <= 1; dy++ {
			var radius int
			switch {
			case dy >= 0:
				radius = 0
			case dy >= -1:
				radius = 1
			case dy >= -3:
				radius = 2
			default:
				radius = 3
			}
			for dx := -radius; dx <= radius; dx++ {
				for dz := -radius; dz <= radius; dz++ {
					if absInt(dx)+absInt(dz) <= radius+1 {
						setFeatureBlock(c, worldX+dx, topY2+dy, worldZ+dz, leaves, false)
					}
				}
			}
		}

	case "spruce":
		for y := groundY + 1; y <= topY; y++ {
			setFeatureBlock(c, worldX, y, worldZ, log, true)
		}
		for dy := -4; dy <= 1; dy++ {
			radius := 1
			if dy == -3 || dy == -1 {
				radius = 2
			}
			for dx := -radius; dx <= radius; dx++ {
				for dz := -radius; dz <= radius; dz++ {
					if absInt(dx)+absInt(dz) <= radius+1 {
						setFeatureBlock(c, worldX+dx, topY+dy, worldZ+dz, leaves, false)
					}
				}
			}
		}

	case "cherry":
		for y := groundY + 1; y <= topY; y++ {
			setFeatureBlock(c, worldX, y, worldZ, log, true)
		}
		roll := generatedHash(g.seed, worldX, groundY, worldZ)
		direction := horizontalCropDirections[int(roll&3)]
		for branch := 0; branch < 2; branch++ {
			if branch == 1 {
				direction.dx, direction.dz = -direction.dx, -direction.dz
			}
			axis := "z"
			if direction.dx != 0 {
				axis = "x"
			}
			branchLog := Block{Namespace: log.Namespace, Name: log.Name, Properties: map[string]string{"axis": axis}}
			branchY := topY - 1 - branch
			endX, endZ := worldX, worldZ
			for step := 1; step <= 2; step++ {
				endX, endZ = worldX+direction.dx*step, worldZ+direction.dz*step
				setFeatureBlock(c, endX, branchY, endZ, branchLog, true)
			}
			endY := branchY + int(roll>>uint(4+branch)&1)
			setFeatureBlock(c, endX, endY, endZ, log, true)
			g.placeCherryFeatureFoliage(c, endX, endY+1, endZ, leaves)
		}
		if roll>>8&1 == 1 {
			g.placeCherryFeatureFoliage(c, worldX, topY+1, worldZ, leaves)
		}

	case "tallbirch":
		// Old-growth birch: taller trunk (6-7 blocks) with same round canopy
		topY2 := groundY + height + 2
		for y := groundY + 1; y <= topY2; y++ {
			setFeatureBlock(c, worldX, y, worldZ, log, true)
		}
		for dy := -2; dy <= 1; dy++ {
			radius := 2
			if dy == 1 {
				radius = 1
			}
			for dx := -radius; dx <= radius; dx++ {
				for dz := -radius; dz <= radius; dz++ {
					if radius == 2 && absInt(dx) == 2 && absInt(dz) == 2 {
						continue
					}
					setFeatureBlock(c, worldX+dx, topY2+dy, worldZ+dz, leaves, false)
				}
			}
		}

	default: // "round" — oak, birch, small jungle
		for y := groundY + 1; y <= topY; y++ {
			setFeatureBlock(c, worldX, y, worldZ, log, true)
		}
		for dy := -2; dy <= 1; dy++ {
			radius := 2
			if dy == 1 {
				radius = 1
			}
			for dx := -radius; dx <= radius; dx++ {
				for dz := -radius; dz <= radius; dz++ {
					if radius == 2 && absInt(dx) == 2 && absInt(dz) == 2 {
						continue
					}
					setFeatureBlock(c, worldX+dx, topY+dy, worldZ+dz, leaves, false)
				}
			}
		}
	}
}

func (g *OverworldGenerator) placeAcaciaFeatureFoliage(c *Chunk, x, y, z, nodeRadius int, leaves Block) {
	for dy, radius := range map[int]int{-2: 1, -1: 2 + nodeRadius, 0: 1 + nodeRadius} {
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				ax, az := absInt(dx), absInt(dz)
				invalid := ax == radius && az == radius && radius > 0
				if dy == 0 {
					invalid = (ax > 1 || az > 1) && ax != 0 && az != 0
				}
				if !invalid {
					setFeatureBlock(c, x+dx, y+dy, z+dz, leaves, false)
				}
			}
		}
	}
}

func (g *OverworldGenerator) placeCherryFeatureFoliage(c *Chunk, x, y, z int, leaves Block) {
	layers := [...]struct{ dy, radius int }{{2, 1}, {1, 2}, {0, 3}, {-1, 3}, {-2, 2}}
	for _, layer := range layers {
		for dx := -layer.radius; dx <= layer.radius; dx++ {
			for dz := -layer.radius; dz <= layer.radius; dz++ {
				corner := absInt(dx) == layer.radius && absInt(dz) == layer.radius
				hash := generatedHash(g.seed, x+dx, y+layer.dy, z+dz)
				if corner || layer.dy == -1 && hash%5 == 0 {
					continue
				}
				setFeatureBlock(c, x+dx, y+layer.dy, z+dz, leaves, false)
				perimeter := absInt(dx) == layer.radius || absInt(dz) == layer.radius
				if layer.dy == -1 && perimeter && hash>>8%4 == 0 {
					setFeatureBlock(c, x+dx, y-2, z+dz, leaves, false)
					if hash>>12%3 == 0 {
						setFeatureBlock(c, x+dx, y-3, z+dz, leaves, false)
					}
				}
			}
		}
	}
}

func (g *OverworldGenerator) placeCactus(c *Chunk, worldX, groundY, worldZ, height int) {
	for y := groundY + 1; y <= groundY+height; y++ {
		setFeatureBlock(c, worldX, y, worldZ, cactusBlock, true)
	}
}

func setFeatureBlock(c *Chunk, worldX, y, worldZ int, material Block, trunk bool) {
	chunkMinX, chunkMinZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	localX, localZ := worldX-chunkMinX, worldZ-chunkMinZ
	if localX < 0 || localX >= SectionSize || localZ < 0 || localZ >= SectionSize || y < WorldMinY || y > WorldMaxY {
		return
	}
	existing := generatedBlock(c, localX, y, localZ)
	name := existing.ResourceLocation()
	isLeaf := name == "minecraft:oak_leaves" || name == "minecraft:birch_leaves" ||
		name == "minecraft:spruce_leaves" || name == "minecraft:acacia_leaves" ||
		name == "minecraft:jungle_leaves" || name == "minecraft:dark_oak_leaves" ||
		name == "minecraft:cherry_leaves" || name == "minecraft:mangrove_leaves" ||
		name == "minecraft:pale_oak_leaves"
	replaceable := existing.IsAir() || name == "minecraft:water" || isLeaf
	if !replaceable {
		return
	}
	setGeneratedBlock(c, localX, y, localZ, material)
}

// ── Ground cover ──────────────────────────────────────────────────────────────

// addGroundCover places short grass, tall grass, ferns, flowers, dead bushes,
// sugar cane, bamboo, lily pads, mushrooms, and seagrass across the chunk.
func (g *OverworldGenerator) addGroundCover(c *Chunk, heights [SectionSize * SectionSize]int) {
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize

	for localX := 0; localX < SectionSize; localX++ {
		for localZ := 0; localZ < SectionSize; localZ++ {
			worldX := chunkMinX + localX
			worldZ := chunkMinZ + localZ
			surfaceY := heights[localZ*SectionSize+localX]
			biome := g.BiomeAt(worldX, worldZ)

			aboveY := surfaceY + 1
			if aboveY > WorldMaxY {
				continue
			}

			surfBlock := generatedBlock(c, localX, surfaceY, localZ)
			surfName := surfBlock.ResourceLocation()
			aboveBlock := generatedBlock(c, localX, aboveY, localZ)

			// Seagrass on ocean floor (block above is water)
			if aboveBlock.ResourceLocation() == "minecraft:water" && surfaceY < SeaLevel-1 {
				state := g.columnHash(worldX, worldZ, 0x736561677261737)
				if state%3 == 0 {
					setGeneratedBlock(c, localX, aboveY, localZ, seagrassBlock)
				}
				continue
			}

			if !aboveBlock.IsAir() {
				continue // tree trunk or feature already placed here
			}

			switch surfName {
			case "minecraft:grass_block":
				g.placeGrassColumnDecor(c, localX, surfaceY, localZ, aboveY, worldX, worldZ, biome)

			case "minecraft:sand", "minecraft:red_sand":
				// Dead bushes in desert/badlands
				if biome == "minecraft:desert" || biome == "minecraft:badlands" || biome == "minecraft:eroded_badlands" || biome == "minecraft:wooded_badlands" {
					state := g.columnHash(worldX, worldZ, 0x6465616462757368)
					if state%7 == 0 {
						setGeneratedBlock(c, localX, aboveY, localZ, deadBushBlock)
					}
				}
				// Sugar cane on coastline sand
				if g.isNearWater(worldX, worldZ, surfaceY) {
					state := g.columnHash(worldX, worldZ, 0x7375676172636e73)
					if state%6 == 0 {
						stalks := 1 + int(state>>8%3)
						for h := 1; h <= stalks; h++ {
							if surfaceY+h > WorldMaxY {
								break
							}
							setGeneratedBlock(c, localX, surfaceY+h, localZ, sugarCaneBlock)
						}
					}
				}

			case "minecraft:water":
				// Lily pads on swamp water surface
				if surfaceY == SeaLevel-1 && (biome == "minecraft:swamp" || biome == "minecraft:mangrove_swamp") {
					state := g.columnHash(worldX, worldZ, 0x6c696c797061643)
					if state%5 == 0 {
						setGeneratedBlock(c, localX, SeaLevel, localZ, lilyPadBlock)
					}
				}
			}
		}
	}

	g.addBamboo(c, heights)
	g.addMushrooms(c, heights)
}

// placeGrassColumnDecor places short/tall grass, ferns, or flowers on a grass_block column.
func (g *OverworldGenerator) placeGrassColumnDecor(c *Chunk, localX, surfaceY, localZ, aboveY, worldX, worldZ int, biome string) {
	r := g.columnHash(worldX, worldZ, 0x6772617373646563)

	var lower, upper Block
	hasUpper := false

	switch biome {
	case "minecraft:taiga", "minecraft:snowy_taiga", "minecraft:old_growth_pine_taiga", "minecraft:old_growth_spruce_taiga", "minecraft:grove":
		switch r % 10 {
		case 0, 1, 2, 3:
			lower = fernBlock
		case 4:
			lower, upper, hasUpper = largeFernLowerBlock, largeFernUpperBlock, true
		case 5, 6:
			lower = shortGrassBlock
		}

	case "minecraft:jungle", "minecraft:sparse_jungle":
		switch r % 10 {
		case 0, 1, 2:
			lower = shortGrassBlock
		case 3:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		case 4:
			lower = fernBlock
		case 5:
			lower, upper, hasUpper = largeFernLowerBlock, largeFernUpperBlock, true
		}

	case "minecraft:swamp", "minecraft:mangrove_swamp":
		switch r % 10 {
		case 0, 1, 2:
			lower = shortGrassBlock
		case 3, 4:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		case 5:
			lower = blueOrchidBlock
		}

	case "minecraft:dark_forest", "minecraft:pale_garden":
		switch r % 12 {
		case 0, 1, 2, 3:
			lower = shortGrassBlock
		case 4, 5:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		}

	case "minecraft:forest", "minecraft:windswept_forest":
		switch r % 20 {
		case 0, 1, 2, 3, 4, 5, 6, 7:
			lower = shortGrassBlock
		case 8, 9, 10:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		case 11:
			lower = dandelionBlock
		case 12:
			lower = poppyBlock
		}

	case "minecraft:birch_forest":
		switch r % 20 {
		case 0, 1, 2, 3, 4, 5, 6, 7:
			lower = shortGrassBlock
		case 8, 9, 10:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		case 11:
			lower = lilyOfTheValleyBlock
		case 12:
			lower = dandelionBlock
		}

	case "minecraft:meadow":
		switch r % 10 {
		case 0:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		case 1, 2:
			lower = shortGrassBlock
		default:
			lower = biomeFlower("minecraft:meadow", r)
		}

	case "minecraft:plains", "minecraft:sunflower_plains":
		switch r % 32 {
		case 0, 1, 2, 3, 4, 5, 6, 7:
			lower = shortGrassBlock
		case 8, 9, 10:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		case 11:
			if biome == "minecraft:sunflower_plains" {
				lower, upper, hasUpper = sunflowerLowerBlock, sunflowerUpperBlock, true
			} else {
				lower = biomeFlower("minecraft:plains", r)
			}
		}

	case "minecraft:savanna", "minecraft:savanna_plateau", "minecraft:windswept_savanna":
		switch r % 8 {
		case 0, 1, 2:
			lower = shortGrassBlock
		case 3, 4:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		}

	case "minecraft:cherry_grove":
		switch r % 10 {
		case 0, 1, 2, 3:
			lower = shortGrassBlock
		case 4, 5:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		case 6:
			lower = pinkTulipBlock
		case 7:
			lower = lilyOfTheValleyBlock
		case 8:
			lower = alliumBlock
		}

	case "minecraft:flower_forest":
		switch r % 8 {
		case 0, 1:
			lower = shortGrassBlock
		case 2, 3:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		default:
			lower = biomeFlower("minecraft:plains", r)
		}

	case "minecraft:old_growth_birch_forest":
		switch r % 20 {
		case 0, 1, 2, 3, 4, 5, 6, 7:
			lower = shortGrassBlock
		case 8, 9, 10:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		case 11:
			lower = lilyOfTheValleyBlock
		case 12:
			lower = dandelionBlock
		}

	case "minecraft:bamboo_jungle":
		switch r % 10 {
		case 0, 1, 2:
			lower = shortGrassBlock
		case 3:
			lower, upper, hasUpper = tallGrassLowerBlock, tallGrassUpperBlock, true
		case 4:
			lower = fernBlock
		case 5:
			lower, upper, hasUpper = largeFernLowerBlock, largeFernUpperBlock, true
		}

	case "minecraft:windswept_hills", "minecraft:windswept_gravelly_hills":
		if r%6 == 0 {
			lower = shortGrassBlock
		}

	case "minecraft:snowy_plains", "minecraft:snowy_slopes", "minecraft:ice_spikes":
		if r%20 == 0 {
			lower = shortGrassBlock
		}
	}

	if lower.IsAir() {
		return
	}

	setGeneratedBlock(c, localX, aboveY, localZ, lower)

	if hasUpper && aboveY+1 <= WorldMaxY {
		if generatedBlock(c, localX, aboveY+1, localZ).IsAir() {
			setGeneratedBlock(c, localX, aboveY+1, localZ, upper)
		} else {
			// Can't place upper half — remove the lower to avoid broken state
			setGeneratedBlock(c, localX, aboveY, localZ, Air)
		}
	}
}

// biomeFlower returns a biome-appropriate single-block flower chosen by state.
func biomeFlower(biome string, state uint64) Block {
	switch biome {
	case "minecraft:plains":
		flowers := []Block{
			dandelionBlock, poppyBlock, azureBluetBlock,
			redTulipBlock, orangeTulipBlock, whiteTulipBlock, pinkTulipBlock,
			oxeyeDaisyBlock, cornflowerBlock, lilyOfTheValleyBlock,
		}
		return flowers[state>>8%uint64(len(flowers))]
	case "minecraft:meadow":
		flowers := []Block{
			dandelionBlock, poppyBlock, alliumBlock, azureBluetBlock,
			oxeyeDaisyBlock, cornflowerBlock, lilyOfTheValleyBlock,
		}
		return flowers[state>>8%uint64(len(flowers))]
	case "minecraft:forest", "minecraft:birch_forest":
		flowers := []Block{dandelionBlock, poppyBlock, lilyOfTheValleyBlock}
		return flowers[state>>8%uint64(len(flowers))]
	}
	return Air
}

// addBamboo places bamboo stalks in jungle chunks.
func (g *OverworldGenerator) addBamboo(c *Chunk, heights [SectionSize * SectionSize]int) {
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize

	const cellSize = 5
	for cellX := floorDiv(chunkMinX, cellSize); cellX <= floorDiv(chunkMinX+SectionSize-1, cellSize); cellX++ {
		for cellZ := floorDiv(chunkMinZ, cellSize); cellZ <= floorDiv(chunkMinZ+SectionSize-1, cellSize); cellZ++ {
			state := g.featureHash(int32(cellX), int32(cellZ), 0x62616d626f6f3031)
			if state%3 != 0 {
				continue
			}
			worldX := cellX*cellSize + int(nextRandom(&state)*float64(cellSize))
			worldZ := cellZ*cellSize + int(nextRandom(&state)*float64(cellSize))
			biome := g.BiomeAt(worldX, worldZ)
			if biome != "minecraft:jungle" && biome != "minecraft:bamboo_jungle" && biome != "minecraft:sparse_jungle" {
				continue
			}
			// Bamboo jungle has much denser bamboo
			if biome == "minecraft:bamboo_jungle" && state%2 != 0 {
				continue
			}
			localX := worldX - chunkMinX
			localZ := worldZ - chunkMinZ
			if localX < 0 || localX >= SectionSize || localZ < 0 || localZ >= SectionSize {
				continue
			}
			surfaceY := heights[localZ*SectionSize+localX]
			if surfaceY <= SeaLevel {
				continue
			}
			stalks := 2 + int(nextRandom(&state)*5)
			for h := 1; h <= stalks; h++ {
				y := surfaceY + h
				if y > WorldMaxY {
					break
				}
				if !generatedBlock(c, localX, y, localZ).IsAir() {
					break
				}
				setGeneratedBlock(c, localX, y, localZ, bambooBlock)
			}
		}
	}
}

// addMushrooms places brown/red mushrooms in dark forest and swamp chunks.
func (g *OverworldGenerator) addMushrooms(c *Chunk, heights [SectionSize * SectionSize]int) {
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize

	const cellSize = 8
	for cellX := floorDiv(chunkMinX, cellSize); cellX <= floorDiv(chunkMinX+SectionSize-1, cellSize); cellX++ {
		for cellZ := floorDiv(chunkMinZ, cellSize); cellZ <= floorDiv(chunkMinZ+SectionSize-1, cellSize); cellZ++ {
			state := g.featureHash(int32(cellX), int32(cellZ), 0x6d757368726f6f6d)
			if state%4 != 0 {
				continue
			}
			worldX := cellX*cellSize + int(nextRandom(&state)*float64(cellSize))
			worldZ := cellZ*cellSize + int(nextRandom(&state)*float64(cellSize))
			biome := g.BiomeAt(worldX, worldZ)
			if biome != "minecraft:dark_forest" && biome != "minecraft:swamp" && biome != "minecraft:mushroom_fields" {
				continue
			}
			localX := worldX - chunkMinX
			localZ := worldZ - chunkMinZ
			if localX < 0 || localX >= SectionSize || localZ < 0 || localZ >= SectionSize {
				continue
			}
			surfaceY := heights[localZ*SectionSize+localX]
			if surfaceY <= SeaLevel {
				continue
			}
			if !generatedBlock(c, localX, surfaceY+1, localZ).IsAir() {
				continue
			}
			mushroom := brownMushroomBlock
			if state>>8%2 == 0 {
				mushroom = redMushroomBlock
			}
			setGeneratedBlock(c, localX, surfaceY+1, localZ, mushroom)
		}
	}
}

// isNearWater reports whether (worldX, worldZ) with given surfaceY is at the
// coastline — close enough to sea level and adjacent to an underwater column.
func (g *OverworldGenerator) isNearWater(worldX, worldZ, surfaceY int) bool {
	if surfaceY > SeaLevel+2 {
		return false
	}
	for _, off := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		if g.SurfaceHeight(worldX+off[0], worldZ+off[1]) < SeaLevel {
			return true
		}
	}
	return false
}

func isMountainBiome(biome string) bool {
	switch biome {
	case "minecraft:windswept_hills", "minecraft:windswept_gravelly_hills", "minecraft:windswept_forest", "minecraft:meadow", "minecraft:grove", "minecraft:snowy_slopes", "minecraft:frozen_peaks", "minecraft:jagged_peaks", "minecraft:stony_peaks":
		return true
	default:
		return false
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
