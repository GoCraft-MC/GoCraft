package entity

import (
	"testing"

	"GoCraft/core/player"
)

func TestProjectileItemPreservesPotionPayload(t *testing.T) {
	projectile := New(7, [16]byte{}, TypePotion, 0, 64, 0)
	projectile.ProjectileItem = player.ItemStack{
		ItemID: "minecraft:splash_potion", Count: 1,
		Components: `{"minecraft:potion_contents":{"potion":"minecraft:poison"}}`,
	}
	if projectile.ProjectileItem.Components == "" || projectile.ProjectileItem.ItemID != "minecraft:splash_potion" {
		t.Fatalf("projectile item = %+v", projectile.ProjectileItem)
	}
}
