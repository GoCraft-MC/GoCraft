package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func newFireworkTestServer(t *testing.T, mode player.GameMode) (*Server, *player.Player) {
	t.Helper()
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	g := game.New()
	p := player.New([16]byte{52}, "rocket-user", player.ClientEditionBedrock)
	p.GameMode = mode
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	return &Server{game: g, world: w, sessions: session.NewManager()}, p
}

func TestFireworkUseConsumesSurvivalButNotCreative(t *testing.T) {
	for _, test := range []struct {
		name string
		mode player.GameMode
		want int
	}{{"survival", player.GameModeSurvival, 1}, {"creative", player.GameModeCreative, 2}} {
		t.Run(test.name, func(t *testing.T) {
			s, p := newFireworkTestServer(t, test.mode)
			data := player.FireworkData{Flight: 2, ExplosionCount: 1}
			p.Inventory[player.HotbarStart] = player.ItemStack{
				ItemID: "minecraft:firework_rocket", Count: 2, HasFireworks: true, Fireworks: data,
			}
			rocket := s.applyFireworkUse(intent.FireworkUseIntent{
				PlayerUUID: p.UUID, HotbarSlot: 0, Position: spatial.Vec3{X: 1, Y: 64.5, Z: 0.5},
			})
			if rocket == nil || rocket.Type != corentity.TypeFireworkRocket || rocket.FireworkData != data {
				t.Fatalf("rocket = %+v", rocket)
			}
			if got := p.Inventory[player.HotbarStart].Count; got != test.want {
				t.Fatalf("remaining rockets = %d, want %d", got, test.want)
			}
			if _, ok := s.world.Entities.Get(rocket.EntityID); !ok {
				t.Fatal("canonical rocket was not added to the world")
			}
		})
	}
}

func TestFireworkLifetimeAndMotion(t *testing.T) {
	if got := fireworkLifetimeTicks(2, 5, 6); got != 41 {
		t.Fatalf("lifetime = %d, want 41", got)
	}
	s, _ := newFireworkTestServer(t, player.GameModeSurvival)
	rocket := corentity.New(80, [16]byte{8}, corentity.TypeFireworkRocket, 0, 70, 0)
	rocket.FireworkLifetime = 1
	rocket.VX, rocket.VY, rocket.VZ = 0.01, 0.05, -0.01
	if s.tickFireworkRocket(rocket) {
		t.Fatal("rocket expired one tick too early")
	}
	if rocket.VX != 0.0115 || rocket.VY != 0.09 || rocket.VZ != -0.0115 {
		t.Fatalf("velocity = %.4f/%.4f/%.4f", rocket.VX, rocket.VY, rocket.VZ)
	}
	if !s.tickFireworkRocket(rocket) {
		t.Fatal("rocket did not expire after its lifetime")
	}
}

func TestFireworkExplosionRequiresEffectsAndVisibility(t *testing.T) {
	if got := fireworkDamage(spatial.Vec3{}, spatial.Vec3{X: 5.1}, 7); got != 0 {
		t.Fatalf("out-of-range damage = %v", got)
	}
	if got := fireworkDamage(spatial.Vec3{}, spatial.Vec3{}, 7); got != 7 {
		t.Fatalf("point-blank damage = %v, want 7", got)
	}
}
