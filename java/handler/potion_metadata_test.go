package handler

import (
	"bytes"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
)

func TestJavaThrownPotionMetadataCarriesItemPayload(t *testing.T) {
	stack := player.ItemStack{ItemID: "minecraft:splash_potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{
		"potion": "minecraft:strong_healing",
	}); err != nil {
		t.Fatal(err)
	}
	projectile := corentity.New(41, [16]byte{}, corentity.TypePotion, 0, 64, 0)
	projectile.ProjectileItem = stack
	pkt := buildMobMetadata(projectile)
	if pkt == nil || pkt.ID != packetIDSetEntityData {
		t.Fatalf("potion metadata packet = %+v", pkt)
	}
	if !bytes.Contains(pkt.Data, []byte("potion_contents")) ||
		!bytes.Contains(pkt.Data, []byte("strong_healing")) {
		t.Fatalf("potion metadata omitted payload: %x", pkt.Data)
	}
}
