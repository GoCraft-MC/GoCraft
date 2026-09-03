package world

import coreworld "GoCraft/core/world"

const blockLightRegionWidth = coreworld.SectionSize * 3

type blockLightRegion struct {
	chunks [3][3]*coreworld.Chunk
}

func localBlockLightRegion(chunk *coreworld.Chunk) blockLightRegion {
	var region blockLightRegion
	region.chunks[1][1] = chunk
	return region
}

func worldBlockLightRegion(world *coreworld.World, chunk *coreworld.Chunk) blockLightRegion {
	region := localBlockLightRegion(chunk)
	if world == nil || chunk == nil {
		return region
	}
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			if dx != 0 || dz != 0 {
				region.chunks[dz+1][dx+1], _ = world.ChunkIfLoaded(chunk.X+int32(dx), chunk.Z+int32(dz))
			}
		}
	}
	return region
}

func (r blockLightRegion) blockAt(x, worldY, z int) coreworld.Block {
	if x < 0 || x >= blockLightRegionWidth || z < 0 || z >= blockLightRegionWidth ||
		worldY < coreworld.WorldMinY || worldY > coreworld.WorldMaxY {
		return coreworld.Air
	}
	chunk := r.chunks[z/coreworld.SectionSize][x/coreworld.SectionSize]
	if chunk == nil {
		// Unknown chunk boundaries are opaque so light cannot leak out and back.
		return coreworld.Block{Namespace: "minecraft", Name: "bedrock"}
	}
	sectionIndex := (worldY - coreworld.WorldMinY) / coreworld.SectionSize
	return chunk.Sections[sectionIndex].At(x&15, (worldY-coreworld.WorldMinY)&15, z&15)
}

func blockLightRegionIndex(x, worldY, z int) int {
	return (worldY-coreworld.WorldMinY)*blockLightRegionWidth*blockLightRegionWidth + z*blockLightRegionWidth + x
}
