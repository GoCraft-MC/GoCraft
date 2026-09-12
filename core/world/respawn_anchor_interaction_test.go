package world

import "testing"

func TestChargeRespawnAnchor(t *testing.T) {
	anchor := Block{Namespace: "minecraft", Name: "respawn_anchor", Properties: map[string]string{"charges": "3"}}
	charged, ok := ChargeRespawnAnchor(anchor, "minecraft:glowstone")
	if !ok || charged.Properties["charges"] != "4" {
		t.Fatalf("charged anchor = %+v, ok=%v", charged, ok)
	}
	if _, ok := ChargeRespawnAnchor(charged, "minecraft:glowstone"); ok {
		t.Fatal("fully charged anchor accepted glowstone")
	}
	if _, ok := ChargeRespawnAnchor(anchor, "minecraft:coal"); ok {
		t.Fatal("anchor accepted a non-glowstone item")
	}
}
