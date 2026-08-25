package server

import (
	"testing"

	"GoCraft/config"
	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

// BenchmarkNightHostileAI measures one server tick at vanilla's 70-monster
// cap, including GoCraft's two-tick hostile staggering and Pumpkin navigation.
func BenchmarkNightHostileAI(b *testing.B) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	for cx := int32(-1); cx <= 2; cx++ {
		for cz := int32(-1); cz <= 2; cz++ {
			w.Chunk(cx, cz)
		}
	}
	g := game.New()
	target := player.New([16]byte{1}, "benchmark", player.ClientEditionBedrock)
	target.GameMode = player.GameModeSurvival
	target.Position = spatial.Vec3{X: 16.5, Y: 64, Z: 16.5}
	if err := g.AddPlayer(target); err != nil {
		b.Fatal(err)
	}
	s := &Server{
		cfg: &config.Config{Difficulty: "normal"}, game: g, world: w,
		sessions: session.NewManager(), mobAIs: make(map[int32]*mobAI),
	}
	mobs := make([]*corentity.Entity, 70)
	for index := range mobs {
		mobs[index] = corentity.New(int32(index+2), [16]byte{}, corentity.TypeZombie,
			float64(index%10)+0.5, 64, float64(index/10)+0.5)
		mobs[index].AgeTicks = int64(index % 2)
	}
	b.ReportMetric(70, "mobs/tick")
	b.ResetTimer()
	for range b.N {
		for _, mob := range mobs {
			mob.AgeTicks++
			if mob.AgeTicks%2 == 0 {
				s.tickHostileMobAI(mob)
			}
		}
	}
}
