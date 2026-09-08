package handler

import (
	"bytes"
	"testing"

	"GoCraft/core/player"
	"GoCraft/java/protocol"
)

func TestJavaExtensionComponentsRoundTrip(t *testing.T) {
	want := player.ItemStack{ItemID: "minecraft:potion", Count: 2}
	if err := want.SetComponent("potion_contents", map[string]any{
		"potion":        "minecraft:strong_healing",
		"custom_colour": 0x55aaff,
	}); err != nil {
		t.Fatal(err)
	}
	b := protocol.NewBuilder(0)
	encodeSlot(b, want)
	got, err := readPlainSlot(bytes.NewReader(b.Build().Data))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Java component round trip = %#v, want %#v", got, want)
	}
}
