package server

import (
	"strings"
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	"GoCraft/java/handler"
)

func TestSetSpawnAndSpawnCommands(t *testing.T) {
	g := game.New()
	operator := player.New([16]byte{1}, "operator", player.ClientEditionJava)
	operator.Operator = true
	operator.GameMode = player.GameModeSurvival
	operator.Position = spatial.Vec3{X: 12.8, Y: 70.9, Z: -4.2}
	if err := g.AddPlayer(operator); err != nil {
		t.Fatal(err)
	}
	dispatcher := handler.NewDispatcher()
	s := &Server{
		game:       g,
		cmds:       dispatcher,
		spawnState: newWorldSpawnState(spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}),
	}
	s.registerSpawnCommands()

	var reply string
	dispatcher.Dispatch("/setspawn", handler.CommandContext{
		Player: operator,
		Reply:  func(message string) error { reply = message; return nil },
	})
	want := spatial.Vec3{X: 12.5, Y: 70, Z: -4.5}
	if got := s.currentWorldSpawn(); got != want {
		t.Fatalf("world spawn = %+v, want %+v", got, want)
	}
	if operator.WorldSpawn != want || !strings.Contains(reply, "World spawn set") {
		t.Fatalf("operator spawn=%+v reply=%q", operator.WorldSpawn, reply)
	}

	operator.Dimension = dimensionEnd
	var dimension int32 = -1
	var teleported spatial.Vec3
	dispatcher.Dispatch("/spawn", handler.CommandContext{
		Player: operator,
		ChangeWorld: func(target int32) error {
			dimension = target
			operator.Dimension = target
			return nil
		},
		TeleportTo: func(x, y, z float64) error {
			teleported = spatial.Vec3{X: x, Y: y, Z: z}
			return nil
		},
		Reply: func(message string) error { reply = message; return nil },
	})
	if dimension != dimensionOverworld || teleported != want || reply != "Teleported to spawn" {
		t.Fatalf("dimension=%d position=%+v reply=%q", dimension, teleported, reply)
	}
}

func TestSetSpawnRequiresOperator(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{2}, "member", player.ClientEditionJava)
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	dispatcher := handler.NewDispatcher()
	s := &Server{game: g, cmds: dispatcher, spawnState: newWorldSpawnState(spatial.Vec3{X: 0.5, Y: 64, Z: 0.5})}
	s.registerSpawnCommands()
	var reply string
	dispatcher.Dispatch("/setspawn", handler.CommandContext{
		Player: p,
		Reply:  func(message string) error { reply = message; return nil },
	})
	if !strings.Contains(reply, "permission") || s.currentWorldSpawn().X != 0.5 {
		t.Fatalf("reply=%q spawn=%+v", reply, s.currentWorldSpawn())
	}
}
