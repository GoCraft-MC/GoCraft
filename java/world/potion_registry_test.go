package world

import "testing"

func TestPotionRegistryIDs(t *testing.T) {
	if id := PotionID("minecraft:strong_healing"); id != 25 {
		t.Fatalf("strong healing potion ID = %d", id)
	}
	if name := PotionName(45); name != "minecraft:infested" {
		t.Fatalf("potion ID 45 = %q", name)
	}
}
