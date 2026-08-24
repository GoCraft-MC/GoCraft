package server

import (
	"math/rand"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func newAnimalTestServer(t *testing.T) (*Server, *player.Player) {
	t.Helper()
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	g := game.New()
	p := player.New([16]byte{1}, "breeder", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0, Y: 64, Z: 0}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	return &Server{
		game: g, world: w, sessions: session.NewManager(), mobAIs: make(map[int32]*mobAI),
		spawnRNG: rand.New(rand.NewSource(1)),
	}, p
}

func TestMinecartMountsThroughCanonicalInteraction(t *testing.T) {
	s, p := newAnimalTestServer(t)
	cart := corentity.New(90, [16]byte{90}, corentity.TypeMinecart, p.Position.X, p.Position.Y, p.Position.Z)
	s.world.Entities.Add(cart)
	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: p.UUID, TargetID: cart.EntityID})
	if p.VehicleEntityID != cart.EntityID || cart.RiderEntityID != p.EntityID {
		t.Fatalf("minecart mount player=%d rider=%d", p.VehicleEntityID, cart.RiderEntityID)
	}
}

func putHeld(p *player.Player, item string, count int) {
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: item, Count: count}
}

func TestFeedingCowEntersLoveAndConsumesExactlyOne(t *testing.T) {
	s, p := newAnimalTestServer(t)
	cow := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeCow, 1, 64, 0)
	s.world.Entities.Add(cow)
	putHeld(p, "minecraft:wheat", 2)

	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: p.UUID, TargetID: cow.EntityID, HotbarSlot: 0})
	if cow.LoveTicks != corentity.LoveDurationTicks || !cow.HasLoveCause || cow.LoveCauseUUID != p.UUID {
		t.Fatalf("cow love state = ticks %d cause %v", cow.LoveTicks, cow.HasLoveCause)
	}
	if got := p.HeldItem().Count; got != 1 {
		t.Fatalf("held wheat = %d, want 1", got)
	}

	// Already-in-love adults do not consume another breeding item.
	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: p.UUID, TargetID: cow.EntityID, HotbarSlot: 0})
	if got := p.HeldItem().Count; got != 1 {
		t.Fatalf("second interaction consumed wheat: count %d", got)
	}
}

func TestBabyFeedingAcceleratesTenPercentOfRemainingGrowth(t *testing.T) {
	s, p := newAnimalTestServer(t)
	calf := corentity.New(s.game.NextEntityID(), [16]byte{3}, corentity.TypeCow, 1, 64, 0)
	calf.IsBaby = true
	calf.BabyAgeTicks = 4_000
	s.world.Entities.Add(calf)
	putHeld(p, "minecraft:wheat", 1)

	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: p.UUID, TargetID: calf.EntityID, HotbarSlot: 0})
	if calf.BabyAgeTicks != 6_000 {
		t.Fatalf("baby age = %d, want 6000", calf.BabyAgeTicks)
	}
	if !p.HeldItem().IsEmpty() {
		t.Fatalf("baby feeding did not consume item: %+v", p.HeldItem())
	}
}

func TestPumpkinBreedGoalSpawnsBabyAndSetsCooldown(t *testing.T) {
	s, _ := newAnimalTestServer(t)
	first := corentity.New(s.game.NextEntityID(), [16]byte{4}, corentity.TypeCow, 0, 64, 0)
	second := corentity.New(s.game.NextEntityID(), [16]byte{5}, corentity.TypeCow, 2, 64, 0)
	first.LoveTicks, second.LoveTicks = 600, 600
	s.world.Entities.Add(first)
	s.world.Entities.Add(second)

	for range corentity.BreedingDelayTicks {
		s.tickAnimalLifecycle(s.world.Entities.Snapshot())
	}
	if got := s.world.Entities.Count(); got != 3 {
		t.Fatalf("entity count after breeding = %d, want 3", got)
	}
	if first.BreedingCooldownTicks != corentity.BreedingCooldownTicks || second.BreedingCooldownTicks != corentity.BreedingCooldownTicks {
		t.Fatalf("parent cooldowns = %d/%d", first.BreedingCooldownTicks, second.BreedingCooldownTicks)
	}
	var child *corentity.Entity
	for _, candidate := range s.world.Entities.Snapshot() {
		if candidate != first && candidate != second {
			child = candidate
		}
	}
	if child == nil || child.Type != corentity.TypeCow || !child.IsBaby {
		t.Fatalf("child = %+v, want baby cow", child)
	}
}

func TestHorseTamingSaddlingAndCamelTwoSeats(t *testing.T) {
	s, first := newAnimalTestServer(t)
	horse := corentity.New(s.game.NextEntityID(), [16]byte{6}, corentity.TypeHorse, 1, 64, 0)
	horse.Temper = 100
	s.world.Entities.Add(horse)
	putHeld(first, "", 0)
	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: first.UUID, TargetID: horse.EntityID, HotbarSlot: 0})
	if !horse.Tamed || horse.TameOwnerUUID != first.UUID || first.VehicleEntityID != horse.EntityID || !horse.HasPassenger(first.EntityID) {
		t.Fatalf("horse tame/mount state = tamed %v owner %v vehicle %d passengers %v", horse.Tamed, horse.HasTameOwner, first.VehicleEntityID, horse.PassengerIDs())
	}
	s.dismountPlayer(first)
	putHeld(first, "minecraft:saddle", 1)
	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: first.UUID, TargetID: horse.EntityID, HotbarSlot: 0})
	if !horse.Saddled || !first.HeldItem().IsEmpty() {
		t.Fatalf("horse saddle state = %v, held=%+v", horse.Saddled, first.HeldItem())
	}

	camel := corentity.New(s.game.NextEntityID(), [16]byte{7}, corentity.TypeCamel, 2, 64, 0)
	s.world.Entities.Add(camel)
	second := player.New([16]byte{8}, "rear", player.ClientEditionJava)
	second.Position = first.Position
	if err := s.game.AddPlayer(second); err != nil {
		t.Fatal(err)
	}
	putHeld(first, "", 0)
	putHeld(second, "", 0)
	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: first.UUID, TargetID: camel.EntityID, HotbarSlot: 0})
	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: second.UUID, TargetID: camel.EntityID, HotbarSlot: 0})
	if got := camel.PassengerIDs(); len(got) != 2 || first.VehicleEntityID != camel.EntityID || second.VehicleEntityID != camel.EntityID {
		t.Fatalf("camel passengers = %v, vehicles=%d/%d", got, first.VehicleEntityID, second.VehicleEntityID)
	}
}

func TestVanillaItemTamingAndParrotCookie(t *testing.T) {
	tests := []struct {
		name   string
		typeID corentity.EntityType
		item   string
	}{
		{"wolf", corentity.TypeWolf, "minecraft:bone"},
		{"cat", corentity.TypeCat, "minecraft:cod"},
		{"parrot", corentity.TypeParrot, "minecraft:wheat_seeds"},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, p := newAnimalTestServer(t)
			animal := corentity.New(s.game.NextEntityID(), [16]byte{byte(20 + index)}, test.typeID, 1, 64, 0)
			s.world.Entities.Add(animal)
			putHeld(p, test.item, 100)
			for attempt := 0; attempt < 100 && !animal.Tamed; attempt++ {
				s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: p.UUID, TargetID: animal.EntityID, HotbarSlot: 0})
			}
			if !animal.Tamed || !animal.HasTameOwner || animal.TameOwnerUUID != p.UUID {
				t.Fatalf("animal did not tame with correct owner: %+v", animal)
			}
			if p.HeldItem().Count >= 100 {
				t.Fatal("taming item was not consumed")
			}
		})
	}

	s, p := newAnimalTestServer(t)
	parrot := corentity.New(s.game.NextEntityID(), [16]byte{30}, corentity.TypeParrot, 1, 64, 0)
	s.world.Entities.Add(parrot)
	putHeld(p, "minecraft:cookie", 1)
	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: p.UUID, TargetID: parrot.EntityID, HotbarSlot: 0})
	if parrot.PoisonTicks != 900 || !p.HeldItem().IsEmpty() {
		t.Fatalf("parrot poison/consumption = %d/%+v", parrot.PoisonTicks, p.HeldItem())
	}
	for range 900 {
		s.tickAnimalLifecycle(s.world.Entities.Snapshot())
	}
	if !parrot.Dead {
		t.Fatal("Pumpkin cookie poison did not become lethal after 900 ticks")
	}
}
