package entity

// Panda gene identifiers, matching vanilla Panda.Gene ordinals.
const (
	PandaGeneNormal     int8 = 0
	PandaGeneLazy       int8 = 1
	PandaGeneWorried    int8 = 2
	PandaGenePlayful    int8 = 3
	PandaGeneBrown      int8 = 4
	PandaGeneWeak       int8 = 5
	PandaGeneAggressive int8 = 6
)

// pandaGeneRecessive reports whether a gene only expresses when both genes match
// it (vanilla Gene.isRecessive: BROWN and WEAK).
func pandaGeneRecessive(gene int8) bool {
	return gene == PandaGeneBrown || gene == PandaGeneWeak
}

// RandomPandaGene maps a 0–15 roll to a gene using vanilla's weighted table
// (Gene.getRandom): LAZY/WORRIED/PLAYFUL/AGGRESSIVE at 1/16 each, WEAK 5/16,
// BROWN 2/16, NORMAL 5/16.
func RandomPandaGene(roll int) int8 {
	switch {
	case roll == 0:
		return PandaGeneLazy
	case roll == 1:
		return PandaGeneWorried
	case roll == 2:
		return PandaGenePlayful
	case roll == 4:
		return PandaGeneAggressive
	case roll < 9:
		return PandaGeneWeak
	case roll < 11:
		return PandaGeneBrown
	default:
		return PandaGeneNormal
	}
}

// PandaVariant resolves the expressed variant from a panda's two genes
// (Gene.getVariantFromGenes): a recessive main gene shows only when the hidden
// gene matches, otherwise the main gene (or NORMAL) is expressed.
func (e *Entity) PandaVariant() int8 {
	main, hidden := e.PandaMainGene, e.PandaHiddenGene
	if pandaGeneRecessive(main) {
		if main == hidden {
			return main
		}
		return PandaGeneNormal
	}
	return main
}
