package world

import coreworld "GoCraft/core/world"

func seedBlockLightRegion(region blockLightRegion, levels []byte) []int {
	queue := make([]int, 0, 4096)
	for chunkZ, row := range region.chunks {
		for chunkX, chunk := range row {
			if chunk == nil {
				continue
			}
			for sectionIndex, section := range chunk.Sections {
				if section == nil || section.NonAir == 0 {
					continue
				}
				palette := section.BlockPalette()
				emission := make([]byte, len(palette))
				hasEmitter := false
				for index, block := range palette {
					emission[index] = blockLightEmission(block)
					hasEmitter = hasEmitter || emission[index] != 0
				}
				if !hasEmitter {
					continue
				}
				for localIndex, paletteIndex := range section.BlockData() {
					level := emission[paletteIndex]
					if level == 0 {
						continue
					}
					x := chunkX*16 + (localIndex & 15)
					z := chunkZ*16 + ((localIndex >> 4) & 15)
					worldY := coreworld.SectionMinY(sectionIndex) + localIndex/256
					index := blockLightRegionIndex(x, worldY, z)
					if levels[index] < level {
						levels[index] = level
						queue = append(queue, index)
					}
				}
			}
		}
	}
	return queue
}
