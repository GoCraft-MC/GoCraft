package world

import "math"

// populateBiomes writes the 4x4x4 biome cells used by modern Java and Bedrock
// chunks. Surface climate is sampled once per horizontal quart column and cave
// biomes are then selected from continuous 3-D noise.
func (g *OverworldGenerator) populateBiomes(c *Chunk) {
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	var samples [4][4]terrainSample
	for quartX := 0; quartX < 4; quartX++ {
		for quartZ := 0; quartZ < 4; quartZ++ {
			x := chunkMinX + quartX*4 + 2
			z := chunkMinZ + quartZ*4 + 2
			samples[quartX][quartZ] = g.sampleTerrain(x, z)
		}
	}

	for sectionIndex, section := range c.Sections {
		if section == nil {
			continue
		}
		sectionY := SectionMinY(sectionIndex)
		first := g.biomeAt3D(chunkMinX+2, sectionY+2, chunkMinZ+2, samples[0][0])
		section.SetUniformBiome(first)
		for quartY := 0; quartY < 4; quartY++ {
			y := sectionY + quartY*4 + 2
			for quartZ := 0; quartZ < 4; quartZ++ {
				z := chunkMinZ + quartZ*4 + 2
				for quartX := 0; quartX < 4; quartX++ {
					x := chunkMinX + quartX*4 + 2
					biome := g.biomeAt3D(x, y, z, samples[quartX][quartZ])
					section.SetBiomeCell(quartX, quartY, quartZ, biome)
				}
			}
		}
	}
}

func (g *OverworldGenerator) biomeAt3D(x, y, z int, surface terrainSample) string {
	// The surface biome owns the upper terrain and ocean column. Cave biomes
	// only replace cells sufficiently far below the terrain surface.
	if y >= surface.height-8 || y > 64 {
		return surface.biome
	}

	selector := g.fractal3D(float64(x), float64(y)*0.72, float64(z), 145, 3, 0.52, 0x6361766562696f6d)
	detail := g.gradNoise3D(float64(x)/58, float64(y)/43, float64(z)/58, 0x6361766564657461)

	// Deep dark occupies broad, low-erosion regions far below the surface.
	if y < 8 && surface.erosion < 0.18 && selector+detail*0.35 > 0.34 {
		return "minecraft:deep_dark"
	}
	if y < 58 && selector > 0.18 {
		lushness := surface.humidity*0.72 - surface.temperature*0.12 + detail*0.55
		if lushness > 0.16 {
			return "minecraft:lush_caves"
		}
		if lushness < -0.12 || surface.continental > 0.32 {
			return "minecraft:dripstone_caves"
		}
	}
	if y < 44 && selector < -0.38 && surface.humidity < 0.28 {
		return "minecraft:dripstone_caves"
	}
	return surface.biome
}

// carveNoiseCaves adds the cheese, spaghetti, noodle and canyon component of
// the modern density pipeline. Worm carvers are retained as a separate stage,
// just as Pumpkin runs configured carvers alongside density caves.
func (g *OverworldGenerator) carveNoiseCaves(c *Chunk, heights [SectionSize * SectionSize]int) {
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	for localX := 0; localX < SectionSize; localX++ {
		worldX := chunkMinX + localX
		for localZ := 0; localZ < SectionSize; localZ++ {
			worldZ := chunkMinZ + localZ
			surfaceY := heights[localZ*SectionSize+localX]
			maxY := minInt(104, surfaceY-8)
			for y := WorldMinY + 6; y <= maxY; y++ {
				existing := generatedBlock(c, localX, y, localZ).ResourceLocation()
				if existing != "minecraft:stone" && existing != "minecraft:deepslate" {
					continue
				}

				fx, fy, fz := float64(worldX), float64(y), float64(worldZ)
				cheese := g.gradNoise3D(fx/52, fy/39, fz/52, 0x6368656573653031)
				spaghettiA := g.gradNoise3D(fx/31, fy/24, fz/31, 0x7370616768657431)
				spaghettiB := g.gradNoise3D(fx/31, fy/24, fz/31, 0x7370616768657432)
				noodle := g.gradNoise3D(fx/18, fy/34, fz/18, 0x6e6f6f646c657331)
				barrier := float64(surfaceY-y) / 48
				cheeseThreshold := 0.57 + math.Max(0, 0.18-barrier)*0.8
				isCheese := cheese > cheeseThreshold && y < 54
				isSpaghetti := math.Abs(spaghettiA) < 0.052 && math.Abs(spaghettiB) < 0.075
				isNoodle := y < 22 && math.Abs(noodle) < 0.032 && cheese > -0.12
				if !isCheese && !isSpaghetti && !isNoodle {
					continue
				}
				setGeneratedBlock(c, localX, y, localZ, g.aquiferMaterial(worldX, y, worldZ))
			}
		}
	}
	g.carveCanyons(c, heights)
}

func (g *OverworldGenerator) carveCanyons(c *Chunk, heights [SectionSize * SectionSize]int) {
	for sourceX := c.X - 5; sourceX <= c.X+5; sourceX++ {
		for sourceZ := c.Z - 5; sourceZ <= c.Z+5; sourceZ++ {
			state := g.featureHash(sourceX, sourceZ, 0x63616e796f6e3031)
			if state%29 != 0 {
				continue
			}
			px := float64(int(sourceX)*SectionSize) + nextRandom(&state)*SectionSize
			pz := float64(int(sourceZ)*SectionSize) + nextRandom(&state)*SectionSize
			py := -28 + nextRandom(&state)*96
			yaw := nextRandom(&state) * math.Pi * 2
			pitch := (nextRandom(&state) - 0.5) * 0.12
			steps := 42 + int(nextRandom(&state)*28)
			for step := 0; step < steps; step++ {
				yaw += (nextRandom(&state) - 0.5) * 0.16
				pitch = pitch*0.82 + (nextRandom(&state)-0.5)*0.07
				px += math.Cos(yaw) * 2.3
				pz += math.Sin(yaw) * 2.3
				py += math.Sin(pitch) * 1.7
				width := 2.2 + math.Sin(float64(step)*math.Pi/float64(steps))*3.4
				height := 6.5 + width*1.7
				g.carveSphere(c, heights, px, py, pz, width, height)
			}
		}
	}
}

// aquiferMaterial chooses the fluid for a carved density cell. Vanilla keeps
// a global lava table below -54 and derives local water tables from low
// frequency pressure/barrier noise.
func (g *OverworldGenerator) aquiferMaterial(x, y, z int) Block {
	if y <= -55 {
		return lavaBlock
	}
	if y >= 48 {
		return Air
	}
	levelNoise := g.gradNoise(float64(x)/96, float64(z)/96, 0x61717569666c766c)
	waterLevel := -8 + int(math.Round(levelNoise*36))
	waterLevel = maxInt(-38, minInt(46, waterLevel))
	if y > waterLevel {
		return Air
	}
	pressure := g.gradNoise3D(float64(x)/64, float64(y)/46, float64(z)/64, 0x6171756970726573)
	barrier := g.gradNoise3D(float64(x)/28, float64(y)/22, float64(z)/28, 0x6171756962617272)
	if pressure > -0.18 && barrier < 0.43 {
		return waterBlock
	}
	return Air
}

// decorateCaves applies the visible floor and ceiling portions of the cave
// biome feature lists. It deliberately replaces only base stone so ore veins
// and generated structures remain intact.
func (g *OverworldGenerator) decorateCaves(c *Chunk, heights [SectionSize * SectionSize]int) {
	chunkMinX := int(c.X) * SectionSize
	chunkMinZ := int(c.Z) * SectionSize
	for localX := 0; localX < SectionSize; localX++ {
		worldX := chunkMinX + localX
		for localZ := 0; localZ < SectionSize; localZ++ {
			worldZ := chunkMinZ + localZ
			surface := g.sampleTerrain(worldX, worldZ)
			maxY := minInt(72, heights[localZ*SectionSize+localX]-8)
			for y := WorldMinY + 7; y <= maxY; y++ {
				current := generatedBlock(c, localX, y, localZ)
				if !current.IsAir() && current.ResourceLocation() != "minecraft:water" {
					continue
				}
				biome := g.biomeAt3D(worldX, y, worldZ, surface)
				state := g.columnHash(worldX, worldZ, uint64(int64(y))^0x636176656465636f)
				below := generatedBlock(c, localX, y-1, localZ).ResourceLocation()
				above := generatedBlock(c, localX, y+1, localZ).ResourceLocation()
				solidBelow := below == "minecraft:stone" || below == "minecraft:deepslate"
				solidAbove := above == "minecraft:stone" || above == "minecraft:deepslate"

				switch biome {
				case "minecraft:lush_caves":
					if solidBelow {
						material := mossBlock
						if state%17 == 0 {
							material = clayBlock
						}
						setGeneratedBlock(c, localX, y-1, localZ, material)
					}
				case "minecraft:dripstone_caves":
					if solidBelow && state%3 != 0 {
						setGeneratedBlock(c, localX, y-1, localZ, dripstoneBlock)
						if current.IsAir() && state%13 == 0 {
							setGeneratedBlock(c, localX, y, localZ, pointedDripstoneUpBlock)
						}
					}
					if solidAbove && current.IsAir() && state%19 == 1 {
						setGeneratedBlock(c, localX, y+1, localZ, dripstoneBlock)
						setGeneratedBlock(c, localX, y, localZ, pointedDripstoneDownBlock)
					}
				case "minecraft:deep_dark":
					if solidBelow && state%3 != 0 {
						setGeneratedBlock(c, localX, y-1, localZ, sculkBlock)
					}
				}
			}
		}
	}
}

func (g *OverworldGenerator) fractal3D(x, y, z, scale float64, octaves int, persistence float64, salt uint64) float64 {
	amplitude, totalAmplitude, value := 1.0, 0.0, 0.0
	for octave := 0; octave < octaves; octave++ {
		value += g.gradNoise3D(x/scale, y/scale, z/scale, salt+uint64(octave)*0x9e3779b97f4a7c15) * amplitude
		totalAmplitude += amplitude
		amplitude *= persistence
		scale *= 0.5
	}
	return value / totalAmplitude
}

func (g *OverworldGenerator) gradNoise3D(x, y, z float64, salt uint64) float64 {
	x0, y0, z0 := int64(math.Floor(x)), int64(math.Floor(y)), int64(math.Floor(z))
	dx, dy, dz := x-float64(x0), y-float64(y0), z-float64(z0)
	tx, ty, tz := smoothstep(dx), smoothstep(dy), smoothstep(dz)

	c000 := g.grad3D(x0, y0, z0, salt, dx, dy, dz)
	c100 := g.grad3D(x0+1, y0, z0, salt, dx-1, dy, dz)
	c010 := g.grad3D(x0, y0+1, z0, salt, dx, dy-1, dz)
	c110 := g.grad3D(x0+1, y0+1, z0, salt, dx-1, dy-1, dz)
	c001 := g.grad3D(x0, y0, z0+1, salt, dx, dy, dz-1)
	c101 := g.grad3D(x0+1, y0, z0+1, salt, dx-1, dy, dz-1)
	c011 := g.grad3D(x0, y0+1, z0+1, salt, dx, dy-1, dz-1)
	c111 := g.grad3D(x0+1, y0+1, z0+1, salt, dx-1, dy-1, dz-1)

	x00 := lerp(c000, c100, tx)
	x10 := lerp(c010, c110, tx)
	x01 := lerp(c001, c101, tx)
	x11 := lerp(c011, c111, tx)
	return lerp(lerp(x00, x10, ty), lerp(x01, x11, ty), tz) * 1.15
}

func (g *OverworldGenerator) grad3D(x, y, z int64, salt uint64, dx, dy, dz float64) float64 {
	h := mix64(uint64(g.seed) ^ uint64(x)*0x9e3779b185ebca87 ^ uint64(y)*0xc2b2ae3d27d4eb4f ^ uint64(z)*0x165667b19e3779f9 ^ salt)
	switch h % 12 {
	case 0:
		return dx + dy
	case 1:
		return -dx + dy
	case 2:
		return dx - dy
	case 3:
		return -dx - dy
	case 4:
		return dx + dz
	case 5:
		return -dx + dz
	case 6:
		return dx - dz
	case 7:
		return -dx - dz
	case 8:
		return dy + dz
	case 9:
		return -dy + dz
	case 10:
		return dy - dz
	default:
		return -dy - dz
	}
}
