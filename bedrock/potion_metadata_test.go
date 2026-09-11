package bedrock

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestBedrockThrownPotionMetadataCarriesVariant(t *testing.T) {
	stack := player.ItemStack{ItemID: "minecraft:splash_potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:strong_healing"}); err != nil {
		t.Fatal(err)
	}
	projectile := corentity.New(81, [16]byte{}, corentity.TypePotion, 0, 64, 0)
	projectile.ProjectileItem = stack
	metadata := (&Listener{}).bedrockEntityMetadata(nil, projectile)
	if got := metadata[protocol.EntityDataKeyAuxValueData]; got != int16(22) {
		t.Fatalf("thrown potion auxiliary metadata = %v", got)
	}
	if got := metadata[protocol.EntityDataKeyCustomDisplay]; got != byte(23) {
		t.Fatalf("thrown potion display metadata = %v", got)
	}
}
