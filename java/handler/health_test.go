package handler

import (
	"bytes"
	"testing"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

func TestEnteringWaterCancelsAccumulatedFallDamage(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(3, 70, 4, coreworld.Block{Namespace: "minecraft", Name: "water", Properties: map[string]string{"level": "0"}})
	p := player.New([16]byte{}, "diver", player.ClientEditionJava)
	p.Position = spatial.Vec3{X: 3.5, Y: 70, Z: 4.5}
	p.OnGround = true
	p.FallDistance = 18
	sess := &session.Session{Player: p}

	applyPlayerFallDamage(sess, 72, false, w, nil)
	if p.FallDistance != 0 || p.Health != p.MaxHealth {
		t.Fatalf("water landing state: fall=%.1f health=%.1f", p.FallDistance, p.Health)
	}
}

func TestJavaSprintingConsumesHunger(t *testing.T) {
	p := player.New([16]byte{}, "runner", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Sprinting = true
	p.Saturation = 0
	p.Exhaustion = 3.95
	p.Position = spatial.Vec3{X: 1, Y: 64, Z: 0}

	applyJavaMovementExhaustion(p, 0, 0, nil)
	food, _, exhaustion := p.HungerSnapshot()
	if food != 19 || exhaustion <= 0 || exhaustion >= 1 {
		t.Fatalf("sprint hunger = food %d exhaustion %.2f, want 19 and rollover remainder", food, exhaustion)
	}
}

func TestReviveProvidesRespawnInvulnerability(t *testing.T) {
	p := &player.Player{
		Health: 0, MaxHealth: 20, Dead: true, GameMode: player.GameModeSurvival,
		OnGround: true, Sleeping: true, Sprinting: true, Flying: true,
		FallDistance: 12, UsingItemID: "minecraft:bow", VehicleEntityID: 42,
	}
	p.Revive()
	target := &session.Session{Player: p}
	if DamagePlayer(target, 10, "was tested", nil) {
		t.Fatal("damage was accepted during respawn invulnerability")
	}
	if p.Health != 20 || p.Dead {
		t.Fatalf("respawn state = health %.1f dead %v", p.Health, p.Dead)
	}
	if p.OnGround || p.Sleeping || p.Sprinting || p.Flying || p.FallDistance != 0 || p.UsingItemID != "" {
		t.Fatalf("transient movement state survived respawn: %+v", p)
	}
	// VehicleEntityID is deliberately cleared by the world-aware respawn path,
	// not Revive, so that the boat's reverse rider link is also removed.
	if p.VehicleEntityID != 42 {
		t.Fatalf("Revive changed vehicle without clearing the vehicle rider link")
	}
}

func TestUpdateHealthPacketProtocol769(t *testing.T) {
	p := player.New([16]byte{}, "health", player.ClientEditionJava)
	p.ApplyDamage(3.5, "test")
	pkt := buildUpdateHealth(p)
	if pkt.ID != packetIDUpdateHealth {
		t.Fatalf("packet ID = %d, want %d", pkt.ID, packetIDUpdateHealth)
	}
	r := bytes.NewReader(pkt.Data)
	health, _ := protocol.ReadFloat(r)
	food, _ := protocol.ReadVarInt(r)
	saturation, _ := protocol.ReadFloat(r)
	if health != 16.5 || food != 20 || saturation != 5 || r.Len() != 0 {
		t.Fatalf("health payload = (%v,%d,%v), trailing=%d", health, food, saturation, r.Len())
	}
}

func TestLegacyArmorReduction(t *testing.T) {
	p := player.New([16]byte{}, "tank", player.ClientEditionJava)
	p.Inventory[5] = player.ItemStack{ItemID: "minecraft:diamond_helmet", Count: 1}
	p.Inventory[6] = player.ItemStack{ItemID: "minecraft:diamond_chestplate", Count: 1}
	p.Inventory[7] = player.ItemStack{ItemID: "minecraft:diamond_leggings", Count: 1}
	p.Inventory[8] = player.ItemStack{ItemID: "minecraft:diamond_boots", Count: 1}
	if got := reducedDamage(p, 10, true); got != 2 {
		t.Fatalf("legacy full diamond damage = %v, want 2", got)
	}
}
