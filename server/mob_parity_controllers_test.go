package server

import (
	"math"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func newMobParityTestServer(t *testing.T) *Server {
	t.Helper()
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = world.Close() })
	world.Chunk(0, 0)
	return &Server{
		world:  world,
		game:   game.New(),
		mobAIs: make(map[int32]*mobAI),
	}
}

func TestParityFlyingNavigationUsesThreeDimensions(t *testing.T) {
	s := newMobParityTestServer(t)
	phantom := corentity.New(1, [16]byte{}, corentity.TypePhantom, 1.5, 70, 1.5)
	ai := s.mobAIFor(phantom)
	if !s.navigateMob(phantom, ai, spatial.Vec3{X: 9.5, Y: 76, Z: 1.5}, 0.1) {
		t.Fatal("phantom navigation did not take ownership")
	}
	if phantom.VX <= 0 || phantom.VY <= 0 {
		t.Fatalf("phantom did not fly toward 3-D target: velocity=(%.3f, %.3f, %.3f)", phantom.VX, phantom.VY, phantom.VZ)
	}
	if ai.hasPathGoal {
		t.Fatal("flying navigation incorrectly allocated a ground A* path")
	}
}

func TestParityRabbitNavigationUsesHopCadence(t *testing.T) {
	s := newMobParityTestServer(t)
	rabbit := corentity.New(2, [16]byte{}, corentity.TypeRabbit, 1.5, 64, 1.5)
	rabbit.OnGround = true
	ai := s.mobAIFor(rabbit)
	if !s.navigateMob(rabbit, ai, spatial.Vec3{X: 6.5, Y: 64, Z: 1.5}, 0.1) {
		t.Fatal("rabbit navigation did not take ownership")
	}
	if rabbit.VY <= 0 {
		t.Fatalf("rabbit did not jump while beginning navigation: VY=%.3f", rabbit.VY)
	}
	if parityState(rabbit).jumpCooldown <= 0 {
		t.Fatal("rabbit jump cooldown was not armed")
	}
}

func TestParityShulkerDoesNotGroundPath(t *testing.T) {
	s := newMobParityTestServer(t)
	shulker := corentity.New(3, [16]byte{}, corentity.TypeShulker, 1.5, 64, 1.5)
	ai := s.mobAIFor(shulker)
	if s.navigateMob(shulker, ai, spatial.Vec3{X: 8.5, Y: 64, Z: 1.5}, 0.1) {
		t.Fatal("anchored shulker reported ground movement")
	}
	if shulker.VX != 0 || shulker.VY != 0 || shulker.VZ != 0 || ai.hasPathGoal {
		t.Fatalf("shulker moved or allocated a path: velocity=(%.3f, %.3f, %.3f) path=%v", shulker.VX, shulker.VY, shulker.VZ, ai.hasPathGoal)
	}
}

func TestParityPiglinRespectsGoldArmourUnlessProvoked(t *testing.T) {
	s := newMobParityTestServer(t)
	piglin := corentity.New(4, [16]byte{}, corentity.TypePiglin, 1.5, 64, 1.5)
	p := player.New([16]byte{9}, "gold", player.ClientEditionJava)
	p.Inventory[5] = player.ItemStack{ItemID: "minecraft:golden_helmet", Count: 1}
	if s.parityMobHostileToPlayer(piglin, p) {
		t.Fatal("unprovoked piglin targeted a gold-armoured player")
	}
	p.LastAttackedEntityID = piglin.EntityID
	if !s.parityMobHostileToPlayer(piglin, p) {
		t.Fatal("provoked piglin remained neutral")
	}
}

func TestParitySpiderNeutralInDirectDaylightButRetaliates(t *testing.T) {
	s := newMobParityTestServer(t)
	s.worldAge = 1000
	spider := corentity.New(5, [16]byte{}, corentity.TypeSpider, 1.5, 64, 1.5)
	p := player.New([16]byte{10}, "daylight", player.ClientEditionJava)
	if s.parityMobHostileToPlayer(spider, p) {
		t.Fatal("unprovoked spider targeted a player in direct daylight")
	}
	p.LastAttackedEntityID = spider.EntityID
	if !s.parityMobHostileToPlayer(spider, p) {
		t.Fatal("spider failed to retaliate after being attacked")
	}
}

func TestParityTameableTeleportsToFarOwner(t *testing.T) {
	s := newMobParityTestServer(t)
	owner := player.New([16]byte{11}, "owner", player.ClientEditionJava)
	owner.Position = spatial.Vec3{X: 30.5, Y: 64, Z: 30.5}
	if err := s.game.AddPlayer(owner); err != nil {
		t.Fatal(err)
	}
	wolf := corentity.New(6, [16]byte{}, corentity.TypeWolf, 1.5, 64, 1.5)
	wolf.Tamed = true
	wolf.HasTameOwner = true
	wolf.TameOwnerUUID = owner.UUID
	ai := s.mobAIFor(wolf)
	if !s.tickParityPassiveIdle(wolf, ai) {
		t.Fatal("tamed wolf did not run FollowOwner behavior")
	}
	if math.Hypot(wolf.Position.X-owner.Position.X, wolf.Position.Z-owner.Position.Z) > 2 {
		t.Fatalf("wolf did not teleport beside distant owner: wolf=%+v owner=%+v", wolf.Position, owner.Position)
	}
}

func TestParityZombifiedPiglinGroupAngerPropagates(t *testing.T) {
	s := newMobParityTestServer(t)
	first := corentity.New(7, [16]byte{}, corentity.TypeZombifiedPiglin, 1.5, 64, 1.5)
	second := corentity.New(8, [16]byte{}, corentity.TypeZombifiedPiglin, 6.5, 64, 1.5)
	s.world.Entities.Add(first)
	s.world.Entities.Add(second)
	attacker := player.New([16]byte{12}, "attacker", player.ClientEditionJava)
	attacker.LastAttackedEntityID = first.EntityID
	if err := s.game.AddPlayer(attacker); err != nil {
		t.Fatal(err)
	}
	s.refreshParityProvocation(first)
	if parityState(first).angerTicks <= 0 || parityState(second).angerTicks <= 0 {
		t.Fatalf("zombified piglin anger did not spread: first=%d second=%d", parityState(first).angerTicks, parityState(second).angerTicks)
	}
}

func TestOutOfBandEnderDragonReceivesFlightAI(t *testing.T) {
	s := newMobParityTestServer(t)
	dragon := corentity.New(9, [16]byte{}, corentity.TypeEnderDragon, 0.5, 80, 0.5)
	if !s.tickOutOfBandParityMob(dragon) {
		t.Fatal("ender dragon was not handled by out-of-band parity controller")
	}
	if math.Abs(dragon.VX)+math.Abs(dragon.VY)+math.Abs(dragon.VZ) == 0 {
		t.Fatal("ender dragon remained motionless after AI tick")
	}
}
