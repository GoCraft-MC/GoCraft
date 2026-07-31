package world

import "math"

const (
	villageCellSize = 384
	villageSalt     uint64 = 0x76696c6c61676531
)

// VillageCenter describes a deterministically-placed village.
type VillageCenter struct {
	WorldX, WorldZ int
	Biome          string
	Hash           uint64
}

func isVillageBiome(biome string) bool {
	switch biome {
	case "minecraft:plains", "minecraft:meadow",
		"minecraft:savanna", "minecraft:desert",
		"minecraft:taiga", "minecraft:snowy_plains":
		return true
	}
	return false
}

// VillageCentersNear returns every village center whose structures could
// overlap the given chunk (search radius = 100 blocks beyond chunk edge).
func (g *OverworldGenerator) VillageCentersNear(chunkX, chunkZ int32) []VillageCenter {
	const searchRadius = 100
	chunkMinX := int(chunkX) * SectionSize
	chunkMinZ := int(chunkZ) * SectionSize

	var result []VillageCenter
	for cellX := floorDiv(chunkMinX-searchRadius, villageCellSize); cellX <= floorDiv(chunkMinX+SectionSize+searchRadius, villageCellSize); cellX++ {
		for cellZ := floorDiv(chunkMinZ-searchRadius, villageCellSize); cellZ <= floorDiv(chunkMinZ+SectionSize+searchRadius, villageCellSize); cellZ++ {
			h := g.featureHash(int32(cellX), int32(cellZ), villageSalt)
			if h%5 != 0 {
				continue
			}
			state := h
			margin := 32
			cx := cellX*villageCellSize + margin + int(nextRandom(&state)*float64(villageCellSize-2*margin))
			cz := cellZ*villageCellSize + margin + int(nextRandom(&state)*float64(villageCellSize-2*margin))
			biome := g.BiomeAt(cx, cz)
			if !isVillageBiome(biome) {
				continue
			}
			result = append(result, VillageCenter{WorldX: cx, WorldZ: cz, Biome: biome, Hash: h})
		}
	}
	return result
}

// villageStyle holds biome-dependent building materials.
type villageStyle struct {
	wall   Block
	roof   Block
	pillar Block
	fence  Block
	log    Block
}

func villageStyleFor(biome string) villageStyle {
	switch biome {
	case "minecraft:desert":
		return villageStyle{
			wall:   block("sandstone"),
			roof:   block("cut_sandstone"),
			pillar: block("sandstone"),
			fence:  block("oak_fence"),
			log:    block("oak_log"),
		}
	case "minecraft:savanna":
		return villageStyle{
			wall:   block("acacia_planks"),
			roof:   block("acacia_planks"),
			pillar: block("cobblestone"),
			fence:  block("acacia_fence"),
			log:    block("acacia_log"),
		}
	case "minecraft:taiga", "minecraft:snowy_plains":
		return villageStyle{
			wall:   block("spruce_planks"),
			roof:   block("spruce_planks"),
			pillar: block("cobblestone"),
			fence:  block("spruce_fence"),
			log:    block("spruce_log"),
		}
	default: // plains, meadow
		return villageStyle{
			wall:   block("oak_planks"),
			roof:   block("oak_planks"),
			pillar: block("cobblestone"),
			fence:  block("oak_fence"),
			log:    block("oak_log"),
		}
	}
}

// addVillageStructures places all village fragments that fall inside chunk c.
func (g *OverworldGenerator) addVillageStructures(c *Chunk) {
	for _, v := range g.VillageCentersNear(c.X, c.Z) {
		wellY := g.SurfaceHeight(v.WorldX, v.WorldZ)
		if wellY <= SeaLevel || wellY > 210 {
			continue
		}
		style := villageStyleFor(v.Biome)
		state := v.Hash

		g.placeWell(c, v.WorldX, wellY, v.WorldZ, style)

		buildingCount := 4 + int(nextRandom(&state)*4)
		offsets := [][2]int{
			{14, 0}, {-14, 0}, {0, 14}, {0, -14},
			{12, 12}, {-12, 12}, {12, -12}, {-12, -12},
		}
		for i := 0; i < buildingCount && i < len(offsets); i++ {
			j := i + int(nextRandom(&state)*float64(len(offsets)-i))
			if j >= len(offsets) {
				j = len(offsets) - 1
			}
			offsets[i], offsets[j] = offsets[j], offsets[i]
			ox := v.WorldX + offsets[i][0]
			oz := v.WorldZ + offsets[i][1]
			gy := g.SurfaceHeight(ox, oz)
			if gy <= SeaLevel {
				continue
			}
			g.placeVillagePath(c, v.WorldX, v.WorldZ, ox, oz)
			switch i % 5 {
			case 0:
				g.placeVillageFarm(c, ox, gy, oz, style)
			case 4:
				g.placeVillageHouse(c, ox, gy, oz, 9, 7, style)
			default:
				g.placeVillageHouse(c, ox, gy, oz, 7, 5, style)
			}
		}
	}
}

// setVB places block b at world coordinates (wx, y, wz), skipping if outside c.
func setVB(c *Chunk, wx, y, wz int, b Block) {
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	lx := wx - chunkMinX
	lz := wz - chunkMinZ
	if lx < 0 || lx >= SectionSize || lz < 0 || lz >= SectionSize {
		return
	}
	setGeneratedBlock(c, lx, y, lz, b)
}

// fillDown fills air/water downward from (wx, gy-1, wz) with b until solid ground.
func fillDown(c *Chunk, wx, gy, wz int, b Block) {
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	lx := wx - chunkMinX
	lz := wz - chunkMinZ
	if lx < 0 || lx >= SectionSize || lz < 0 || lz >= SectionSize {
		return
	}
	for y := gy - 1; y >= gy-6; y-- {
		existing := generatedBlock(c, lx, y, lz)
		if !existing.IsAir() && existing.ResourceLocation() != "minecraft:water" {
			break
		}
		setGeneratedBlock(c, lx, y, lz, b)
	}
}

func (g *OverworldGenerator) placeWell(c *Chunk, cx, gy, cz int, style villageStyle) {
	stone := block("stone_bricks")

	// 5×5 cobblestone base; center 3×3 is water.
	for x := cx - 2; x <= cx+2; x++ {
		for z := cz - 2; z <= cz+2; z++ {
			setVB(c, x, gy, z, style.pillar)
			fillDown(c, x, gy, z, style.pillar)
		}
	}
	for x := cx - 1; x <= cx+1; x++ {
		for z := cz - 1; z <= cz+1; z++ {
			setVB(c, x, gy, z, waterBlock)
		}
	}

	// Stone brick ring at gy+1 and gy+2.
	for x := cx - 2; x <= cx+2; x++ {
		for z := cz - 2; z <= cz+2; z++ {
			if x != cx-2 && x != cx+2 && z != cz-2 && z != cz+2 {
				continue
			}
			setVB(c, x, gy+1, z, stone)
			setVB(c, x, gy+2, z, stone)
		}
	}

	// Fence posts at inner corners (gy+3 and gy+4).
	for _, off := range [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}} {
		setVB(c, cx+off[0], gy+3, cz+off[1], style.fence)
		setVB(c, cx+off[0], gy+4, cz+off[1], style.fence)
	}

	// Log cross-beam at gy+4.
	for x := cx - 1; x <= cx+1; x++ {
		setVB(c, x, gy+4, cz, style.log)
	}
	for z := cz - 1; z <= cz+1; z++ {
		setVB(c, cx, gy+4, z, style.log)
	}
}

func (g *OverworldGenerator) placeVillageHouse(c *Chunk, cx, gy, cz, width, depth int, style villageStyle) {
	doorLower := blockProps("oak_door", "facing", "south", "half", "lower", "hinge", "left", "open", "false", "powered", "false")
	doorUpper := blockProps("oak_door", "facing", "south", "half", "upper", "hinge", "left", "open", "false", "powered", "false")
	glass := block("glass_pane")

	hw := width / 2
	hd := depth / 2

	for x := cx - hw; x <= cx+hw; x++ {
		for z := cz - hd; z <= cz+hd; z++ {
			fillDown(c, x, gy, z, style.pillar)
		}
	}

	for x := cx - hw; x <= cx+hw; x++ {
		for z := cz - hd; z <= cz+hd; z++ {
			onEdge := x == cx-hw || x == cx+hw || z == cz-hd || z == cz+hd
			if !onEdge {
				continue
			}
			isDoor := z == cz+hd && x == cx
			isWindow := !isDoor && (
				(z == cz-hd && x == cx) ||
					(x == cx-hw && z == cz) ||
					(x == cx+hw && z == cz))
			for y := gy + 1; y <= gy+4; y++ {
				if isDoor && (y == gy+1 || y == gy+2) {
					continue
				}
				if isWindow && y == gy+2 {
					setVB(c, x, y, z, glass)
					continue
				}
				setVB(c, x, y, z, style.wall)
			}
		}
	}

	setVB(c, cx, gy+1, cz+hd, doorLower)
	setVB(c, cx, gy+2, cz+hd, doorUpper)

	// Roof at gy+5: log frame with plank fill.
	for x := cx - hw; x <= cx+hw; x++ {
		for z := cz - hd; z <= cz+hd; z++ {
			onEdge := x == cx-hw || x == cx+hw || z == cz-hd || z == cz+hd
			if onEdge {
				setVB(c, x, gy+5, z, style.log)
			} else {
				setVB(c, x, gy+5, z, style.roof)
			}
		}
	}
}

func (g *OverworldGenerator) placeVillageFarm(c *Chunk, cx, gy, cz int, style villageStyle) {
	farmland := block("farmland")
	wheat := blockProps("wheat", "age", "7")

	for x := cx - 3; x <= cx+3; x++ {
		for z := cz - 3; z <= cz+3; z++ {
			onEdge := x == cx-3 || x == cx+3 || z == cz-3 || z == cz+3
			if onEdge {
				if !(x == cx && z == cz+3) {
					setVB(c, x, gy, z, style.fence)
				}
				continue
			}
			if x == cx && z == cz {
				setVB(c, x, gy, z, waterBlock)
				continue
			}
			setVB(c, x, gy, z, farmland)
			setVB(c, x, gy+1, z, wheat)
		}
	}
}

func (g *OverworldGenerator) placeVillagePath(c *Chunk, x1, z1, x2, z2 int) {
	dx := x2 - x1
	dz := z2 - z1
	steps := absInt(dx)
	if absInt(dz) > steps {
		steps = absInt(dz)
	}
	if steps == 0 {
		return
	}
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	pathBlock := block("gravel")
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		wx := x1 + int(math.Round(float64(dx)*t))
		wz := z1 + int(math.Round(float64(dz)*t))
		lx := wx - chunkMinX
		lz := wz - chunkMinZ
		if lx < 0 || lx >= SectionSize || lz < 0 || lz >= SectionSize {
			continue
		}
		surfY := g.SurfaceHeight(wx, wz)
		if surfY <= SeaLevel {
			continue
		}
		setGeneratedBlock(c, lx, surfY, lz, pathBlock)
	}
}
