package bedrock

import (
	"testing"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

func TestBedrockFireworkNBTRoundTrip(t *testing.T) {
	want := player.FireworkData{Flight: 3, ExplosionCount: 1}
	want.Explosions[0] = player.FireworkExplosion{
		Shape: 4, Colors: [player.MaxFireworkColors]int32{0xb02e26, 0x3c44aa}, ColorCount: 2,
		FadeColors: [player.MaxFireworkColors]int32{0xfed83d}, FadeColorCount: 1,
		Trail: true, Twinkle: true,
	}
	got, ok := bedrockFireworkDataFromNBT(bedrockFireworkNBT(want))
	if !ok {
		t.Fatal("firework NBT was not recognised")
	}
	if got != want {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestBedrockSignAndBannerBlockActorData(t *testing.T) {
	sign := bedrockBlockEntityData(coreworld.BlockEntity{X: 1, Y: 64, Z: 2, Type: "minecraft:sign"}, coreworld.Block{})
	if sign["id"] != "Sign" || sign["FrontText"] == nil || sign["BackText"] == nil {
		t.Fatalf("sign actor data = %#v", sign)
	}
	banner := bedrockBlockEntityData(coreworld.BlockEntity{Type: "minecraft:banner"}, coreworld.Block{
		Namespace: "minecraft", Name: "red_wall_banner",
	})
	if banner["id"] != "Banner" || banner["Base"] != int32(1) {
		t.Fatalf("red banner actor data = %#v", banner)
	}
}

func TestBedrockFireworkNBTBounds(t *testing.T) {
	got, ok := bedrockFireworkDataFromNBT(map[string]any{"Fireworks": map[string]any{
		"Flight": uint8(9),
		"Explosions": []any{map[string]any{
			"FireworkType": uint8(9), "FireworkColor": []uint8{15},
		}},
	}})
	if !ok || got.Flight != 9 || got.ExplosionCount != 1 || got.Explosions[0].Shape != 4 {
		t.Fatalf("unexpected bounded data: %+v, ok=%v", got, ok)
	}
}

func TestCreativeFireworkVariantsKeepCanonicalComponents(t *testing.T) {
	l := &Listener{}
	l.initCreativeContent()
	var rockets, effectRockets int
	for _, item := range l.creativeNames {
		if item.name != "minecraft:firework_rocket" {
			continue
		}
		rockets++
		if item.hasFireworks && item.fireworks.ExplosionCount > 0 {
			effectRockets++
		}
	}
	if rockets < 2 || effectRockets == 0 {
		t.Fatalf("creative rockets = %d total/%d with effects", rockets, effectRockets)
	}
}
