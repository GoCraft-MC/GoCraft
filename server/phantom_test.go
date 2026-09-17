package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	"GoCraft/java/session"
)

func newPhantomTarget(dim int32) *session.Session {
	p := player.New([16]byte{9}, "prey", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Dimension = dim
	p.MaxHealth, p.Health = 20, 20
	p.Position = spatial.Vec3{X: 20, Y: 64, Z: 20}
	return &session.Session{Player: p}
}

func TestPhantomSwoopHitsThenReturnsToCircle(t *testing.T) {
	s := newGolemTestServer(t)
	sess := newPhantomTarget(s.simulationDimension)
	phantom := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypePhantom, 20, 65, 20)
	s.world.Entities.Add(phantom)
	ai := s.mobAIFor(phantom)
	state := parityState(phantom)
	state.phantomSwooping = true

	s.tickPhantomSwoop(phantom, ai, sess, 1.0, true)

	if sess.Player.Health >= 20 {
		t.Fatalf("phantom swoop dealt no damage: health=%.1f", sess.Player.Health)
	}
	if state.phantomSwooping {
		t.Fatal("phantom did not peel out of its dive after hitting")
	}
	if state.phantomCircleTicks <= 0 {
		t.Fatal("phantom did not schedule its next circle before swooping again")
	}
}

func TestPhantomBeginsSwoopAfterCircling(t *testing.T) {
	s := newGolemTestServer(t)
	sess := newPhantomTarget(s.simulationDimension)
	phantom := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypePhantom, 20, 71, 20)
	s.world.Entities.Add(phantom)
	ai := s.mobAIFor(phantom)
	state := parityState(phantom)
	state.phantomSwooping = false
	state.phantomCircleTicks = 0

	s.tickPhantomSwoop(phantom, ai, sess, 5.0, true)

	if !state.phantomSwooping {
		t.Fatal("phantom did not begin a swoop once its circle timer elapsed")
	}
}

func TestPhantomKeepsCirclingWhileTimerRuns(t *testing.T) {
	s := newGolemTestServer(t)
	sess := newPhantomTarget(s.simulationDimension)
	phantom := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypePhantom, 20, 71, 20)
	s.world.Entities.Add(phantom)
	ai := s.mobAIFor(phantom)
	state := parityState(phantom)
	state.phantomSwooping = false
	state.phantomCircleTicks = 10

	s.tickPhantomSwoop(phantom, ai, sess, 5.0, true)

	if state.phantomSwooping {
		t.Fatal("phantom swooped before its circle timer elapsed")
	}
	if state.phantomCircleTicks != 9 {
		t.Fatalf("circle timer did not tick down: %d", state.phantomCircleTicks)
	}
}
