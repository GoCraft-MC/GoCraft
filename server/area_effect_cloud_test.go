package server

import (
	"math"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	"GoCraft/java/session"
)

func newLingeringCloudTest(t *testing.T) (*Server, *player.Player, *corentity.Entity) {
	t.Helper()
	g := game.New()
	p := player.New([16]byte{61}, "cloud-target", player.ClientEditionJava)
	p.Position = spatial.Vec3{X: 1, Y: 64, Z: 0}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	stack := player.ItemStack{ItemID: "minecraft:lingering_potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:poison"}); err != nil {
		t.Fatal(err)
	}
	cloud := corentity.NewAreaEffectCloud(62, [16]byte{62}, 0, 64, 0, stack)
	return &Server{game: g, sessions: session.NewManager(), simulationDimension: dimensionOverworld}, p, cloud
}

func TestAreaEffectCloudWaitsThenAppliesPotion(t *testing.T) {
	s, p, cloud := newLingeringCloudTest(t)
	cloud.AgeTicks = 9
	if s.tickAreaEffectCloud(cloud) {
		t.Fatal("cloud expired during warmup")
	}
	if _, ok := p.StatusEffect("poison"); ok {
		t.Fatal("cloud applied before warmup ended")
	}
	cloud.AgeTicks = 10
	if s.tickAreaEffectCloud(cloud) {
		t.Fatal("cloud expired on its first application")
	}
	if effect, ok := p.StatusEffect("poison"); !ok || effect.Duration != 225 {
		t.Fatalf("cloud poison = %+v, ok=%v", effect, ok)
	}
	if math.Abs(cloud.CloudRadius-2.495) > 0.000001 || cloud.CloudTargets[p.EntityID] != 50 {
		t.Fatalf("cloud state radius=%v targets=%v", cloud.CloudRadius, cloud.CloudTargets)
	}
}

func TestAreaEffectCloudExpiresAtDuration(t *testing.T) {
	s, _, cloud := newLingeringCloudTest(t)
	cloud.AgeTicks = cloud.CloudDurationTicks
	if !s.tickAreaEffectCloud(cloud) {
		t.Fatal("cloud survived its configured duration")
	}
}
