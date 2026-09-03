package world

import coreworld "GoCraft/core/world"

func computeBlockLightRegion(region blockLightRegion) []byte {
	levels := make([]byte, coreworld.SectionCount*coreworld.SectionSize*blockLightRegionWidth*blockLightRegionWidth)
	queue := seedBlockLightRegion(region, levels)
	const layerSize = blockLightRegionWidth * blockLightRegionWidth
	for head := 0; head < len(queue); head++ {
		index := queue[head]
		level := levels[index]
		if level <= 1 {
			continue
		}
		yOffset := index / layerSize
		rem := index % layerSize
		z, x := rem/blockLightRegionWidth, rem%blockLightRegionWidth
		worldY := coreworld.WorldMinY + yOffset
		for _, delta := range [6][3]int{{-1, 0, 0}, {1, 0, 0}, {0, -1, 0}, {0, 1, 0}, {0, 0, -1}, {0, 0, 1}} {
			nx, ny, nz := x+delta[0], worldY+delta[1], z+delta[2]
			if nx < 0 || nx >= blockLightRegionWidth || nz < 0 || nz >= blockLightRegionWidth ||
				ny < coreworld.WorldMinY || ny > coreworld.WorldMaxY {
				continue
			}
			neighbor := blockLightRegionIndex(nx, ny, nz)
			next := level - 1
			if levels[neighbor] >= next || !blockLightPasses(region.blockAt(nx, ny, nz)) {
				continue
			}
			levels[neighbor] = next
			queue = append(queue, neighbor)
		}
	}
	return levels
}
