package world

import coreworld "GoCraft/core/world"

func buildBlockLight(region blockLightRegion) (mask, emptyMask int64, arrays [][]byte) {
	levels := computeBlockLightRegion(region)
	emptyMask = 1 | int64(1)<<25
	for sectionIndex := 0; sectionIndex < coreworld.SectionCount; sectionIndex++ {
		light := acquireSkyLightPage()
		nonZero := false
		for localIndex := 0; localIndex < 4096; localIndex++ {
			x := coreworld.SectionSize + (localIndex & 15)
			z := coreworld.SectionSize + ((localIndex >> 4) & 15)
			worldY := coreworld.SectionMinY(sectionIndex) + localIndex/256
			level := levels[blockLightRegionIndex(x, worldY, z)]
			if level == 0 {
				continue
			}
			if localIndex&1 == 0 {
				light[localIndex>>1] |= level
			} else {
				light[localIndex>>1] |= level << 4
			}
			nonZero = true
		}
		bit := int64(1) << (sectionIndex + 1)
		if nonZero {
			mask |= bit
			arrays = append(arrays, light)
		} else {
			emptyMask |= bit
			releaseSkyLightPage(light)
		}
	}
	return mask, emptyMask, arrays
}
