package server

import (
	"testing"

	corentity "GoCraft/core/entity"
)

func TestPandaVariantResolvesDominantAndRecessiveGenes(t *testing.T) {
	// A dominant main gene always expresses.
	worried := &corentity.Entity{PandaMainGene: corentity.PandaGeneWorried, PandaHiddenGene: corentity.PandaGeneNormal}
	if worried.PandaVariant() != corentity.PandaGeneWorried {
		t.Fatal("dominant WORRIED gene should express")
	}
	// A recessive main gene expresses only when both genes match.
	brownCarrier := &corentity.Entity{PandaMainGene: corentity.PandaGeneBrown, PandaHiddenGene: corentity.PandaGeneNormal}
	if brownCarrier.PandaVariant() != corentity.PandaGeneNormal {
		t.Fatal("single recessive BROWN gene should express as NORMAL")
	}
	brownPure := &corentity.Entity{PandaMainGene: corentity.PandaGeneBrown, PandaHiddenGene: corentity.PandaGeneBrown}
	if brownPure.PandaVariant() != corentity.PandaGeneBrown {
		t.Fatal("matched recessive BROWN genes should express BROWN")
	}
}

func TestRandomPandaGeneMatchesVanillaTable(t *testing.T) {
	cases := map[int]int8{
		0: corentity.PandaGeneLazy, 1: corentity.PandaGeneWorried, 2: corentity.PandaGenePlayful,
		4: corentity.PandaGeneAggressive, 3: corentity.PandaGeneWeak, 8: corentity.PandaGeneWeak,
		9: corentity.PandaGeneBrown, 10: corentity.PandaGeneBrown, 11: corentity.PandaGeneNormal,
		15: corentity.PandaGeneNormal,
	}
	for roll, want := range cases {
		if got := corentity.RandomPandaGene(roll); got != want {
			t.Fatalf("roll %d: got gene %d, want %d", roll, got, want)
		}
	}
}

func TestWorriedPandaHidesDuringThunder(t *testing.T) {
	s := newGolemTestServer(t)
	panda := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypePanda, 20, 64, 20)
	panda.PandaMainGene, panda.PandaHiddenGene = corentity.PandaGeneWorried, corentity.PandaGeneNormal
	panda.VX, panda.VZ = 0.5, 0.5
	s.world.Entities.Add(panda)
	ai := s.mobAIFor(panda)

	// No storm: the panda goes about its business (idle wander handles it).
	if s.tickPandaParity(panda, ai) {
		t.Fatal("worried panda cowered without a thunderstorm")
	}

	s.weather.Store(2) // thundering
	if !s.tickPandaParity(panda, ai) {
		t.Fatal("worried panda did not cower during the thunderstorm")
	}
	if panda.VX != 0 || panda.VZ != 0 {
		t.Fatalf("cowering panda kept moving: vx=%.3f vz=%.3f", panda.VX, panda.VZ)
	}
}

func TestNonWorriedPandaIgnoresThunder(t *testing.T) {
	s := newGolemTestServer(t)
	panda := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypePanda, 20, 64, 20)
	panda.PandaMainGene, panda.PandaHiddenGene = corentity.PandaGenePlayful, corentity.PandaGeneNormal
	s.world.Entities.Add(panda)
	ai := s.mobAIFor(panda)
	s.weather.Store(2)

	if s.tickPandaParity(panda, ai) {
		t.Fatal("a playful panda should not cower in a storm")
	}
}
