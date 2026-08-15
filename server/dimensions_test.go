package server

import (
	"math"
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestBedrockIgnitesCornerlessNetherPortal(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	const baseX, bottom, baseZ = 2, 64, 3
	for horizontal := 0; horizontal < 4; horizontal++ {
		for vertical := 0; vertical < 5; vertical++ {
			border := (horizontal == 0 || horizontal == 3) && vertical >= 1 && vertical <= 3 ||
				(vertical == 0 || vertical == 4) && horizontal >= 1 && horizontal <= 2
			if border {
				s.world.SetBlock(baseX+horizontal, bottom+vertical, baseZ, bedrockBlock("obsidian", nil))
			}
		}
	}
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:flint_and_steel", Count: 1}
	if !s.applyBedrockItemAction(p, intent.BlockInteractIntent{
		Position: spatial.BlockPos{X: baseX, Y: bottom + 2, Z: baseZ}, Face: 5,
	}, s.world.GetBlock(baseX, bottom+2, baseZ)) {
		t.Fatal("flint and steel did not handle valid obsidian frame")
	}
	for dx := 1; dx <= 2; dx++ {
		for dy := 1; dy <= 3; dy++ {
			portal := s.world.GetBlock(baseX+dx, bottom+dy, baseZ)
			if portal.ResourceLocation() != "minecraft:nether_portal" || portal.Properties["axis"] != "x" {
				t.Fatalf("portal interior (%d,%d) = %+v", dx, dy, portal)
			}
		}
	}
	if got := p.Inventory[player.HotbarStart].Damage; got != 1 {
		t.Fatalf("flint and steel damage = %d, want 1", got)
	}
}

func TestBedrockEndPortalRequiresEyesFacingCentre(t *testing.T) {
	s, _ := newBedrockActionTestServer(t)
	const originX, y, originZ = 5, 64, 7
	for offset := 1; offset <= 3; offset++ {
		for _, frame := range []struct {
			x, z   int
			facing string
		}{
			{originX + offset, originZ, "south"},
			{originX + offset, originZ + 4, "north"},
			{originX, originZ + offset, "east"},
			{originX + 4, originZ + offset, "west"},
		} {
			s.world.SetBlock(frame.x, y, frame.z, bedrockBlock("end_portal_frame", map[string]string{"facing": frame.facing, "eye": "true"}))
		}
	}
	if !s.tryActivateEndPortal(originX+1, y, originZ) {
		t.Fatal("complete inward-facing End frame did not activate")
	}
	for dx := 1; dx <= 3; dx++ {
		for dz := 1; dz <= 3; dz++ {
			if got := s.world.GetBlock(originX+dx, y, originZ+dz).ResourceLocation(); got != "minecraft:end_portal" {
				t.Fatalf("portal interior (%d,%d) = %q", dx, dz, got)
			}
		}
	}
}

func TestBedrockNetherPortalTravelScalesCoordinatesAndChangesWorld(t *testing.T) {
	overworld := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	nether := coreworld.New(coreworld.NewNetherGenerator(7), nil, false)
	end := coreworld.New(coreworld.NewEndGenerator(7), nil, false)
	t.Cleanup(func() {
		overworld.Close()
		nether.Close()
		end.Close()
	})
	p := player.New([16]byte{55}, "traveller", player.ClientEditionBedrock)
	p.Position = spatial.Vec3{X: 80.5, Y: 64, Z: -40.5}
	overworld.SetBlock(80, 64, -41, bedrockBlock("nether_portal", map[string]string{"axis": "x"}))
	s := &Server{world: overworld, netherWorld: nether, endWorld: end}
	if !s.tryBedrockPortalTravel(p) {
		t.Fatal("player standing in Nether portal did not travel")
	}
	if p.Dimension != dimensionNether {
		t.Fatalf("dimension = %d, want Nether", p.Dimension)
	}
	if p.Position.X < 9 || p.Position.X > 12 || p.Position.Z < -6 || p.Position.Z > -4 {
		t.Fatalf("scaled Nether position = %+v", p.Position)
	}
	if got := nether.GetBlock(int(math.Floor(p.Position.X)), int(math.Floor(p.Position.Y)), int(math.Floor(p.Position.Z))).ResourceLocation(); got != "minecraft:nether_portal" {
		t.Fatalf("destination block = %q, want nether_portal", got)
	}
}

func TestBedrockNetherDeathDropsAndRespawnsInOverworld(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	nether := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = nether.Close() })
	s.netherWorld = nether
	p.Dimension = dimensionNether
	p.Position = spatial.Vec3{X: 4.5, Y: 64, Z: 6.5}
	p.WorldSpawn = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:diamond", Count: 2}

	s.dropPlayerInventory(p)
	if got := len(s.world.Entities.Snapshot()); got != 0 {
		t.Fatalf("Overworld received %d Nether death drops", got)
	}
	if got := nether.Entities.Snapshot(); len(got) != 1 || got[0].ItemID != "minecraft:diamond" {
		t.Fatalf("Nether death drops = %+v", got)
	}

	p.ApplyDamage(p.MaxHealth, "test")
	s.applyBedrockRespawn(intent.RespawnIntent{PlayerUUID: p.UUID})
	if p.Dimension != dimensionOverworld || p.Position != p.WorldSpawn {
		t.Fatalf("respawn state: dimension=%d position=%+v spawn=%+v", p.Dimension, p.Position, p.WorldSpawn)
	}
}
