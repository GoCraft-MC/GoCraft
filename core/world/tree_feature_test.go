package world

import "testing"

func TestGeneratedAcaciaUsesForkingTrunkNodes(t *testing.T) {
	generator := NewOverworldGenerator(7)
	chunk := &Chunk{X: 0, Z: 0}
	generator.placeTree(chunk, 8, 63, 8, 6, acaciaLogBlock, acaciaLeafBlock, "acacia")
	logColumns := make(map[[2]int]struct{})
	leaves := 0
	for x := 0; x < SectionSize; x++ {
		for z := 0; z < SectionSize; z++ {
			for y := 64; y < 76; y++ {
				switch chunkBlock(chunk, x, y, z).ResourceLocation() {
				case "minecraft:acacia_log":
					logColumns[[2]int{x, z}] = struct{}{}
				case "minecraft:acacia_leaves":
					leaves++
				}
			}
		}
	}
	if len(logColumns) < 2 || leaves < 20 {
		t.Fatalf("generated acacia columns=%d leaves=%d", len(logColumns), leaves)
	}
}
