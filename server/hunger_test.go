package server

import (
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/java/session"
)

func TestNaturalRegenerationWorksForJavaPlayers(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{9}, "hungry-java", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Health = 10
	p.Food = 20
	p.Saturation = 5
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	mgr := session.NewManager()
	mgr.Add(&session.Session{Player: p})
	s := &Server{game: g, sessions: mgr, worldAge: 80}

	s.tickPlayerHunger()
	health, food, saturation, _ := p.HealthSnapshot()
	if health != 11 || food != 20 || saturation != 4 {
		t.Fatalf("regeneration state = health %.1f food %d saturation %.1f", health, food, saturation)
	}
}

func TestSaturatedJavaPlayerUsesFastRegenerationInterval(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{10}, "saturated-java", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Health = 10
	p.Food = 20
	p.Saturation = 5
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	mgr := session.NewManager()
	mgr.Add(&session.Session{Player: p})
	s := &Server{game: g, sessions: mgr, worldAge: 10}

	s.tickPlayerHunger()
	health, _, saturation, _ := p.HealthSnapshot()
	if health != 11 || saturation != 4 {
		t.Fatalf("fast regeneration state = health %.1f saturation %.1f", health, saturation)
	}
}
