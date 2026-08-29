package handler

import (
	"bytes"
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

func TestJavaFireworkComponentRoundTrip(t *testing.T) {
	stack := player.ItemStack{ItemID: "minecraft:firework_rocket", Count: 3, HasFireworks: true}
	stack.Fireworks.Flight = 2
	stack.Fireworks.ExplosionCount = 1
	stack.Fireworks.Explosions[0] = player.FireworkExplosion{
		Shape: 2, ColorCount: 2, Colors: [player.MaxFireworkColors]int32{0xb02e26, 0x3c44aa},
		FadeColorCount: 1, FadeColors: [player.MaxFireworkColors]int32{0xfed83d}, Trail: true, Twinkle: true,
	}
	b := protocol.NewBuilder(0)
	encodeSlot(b, stack)
	decoded, err := readPlainSlot(bytes.NewReader(b.Build().Data))
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.HasFireworks || decoded.EffectiveFireworks() != stack.EffectiveFireworks() || decoded.Count != 3 {
		t.Fatalf("decoded=%+v, want %+v", decoded, stack)
	}
}

func TestJavaFireworkUsePostsCanonicalIntent(t *testing.T) {
	p := player.New([16]byte{44}, "rocket-user", player.ClientEditionJava)
	p.HeldSlot = 2
	p.Inventory[player.HotbarStart+2] = player.ItemStack{ItemID: "minecraft:firework_rocket", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	bus := intent.NewBus(1, 4)
	if err := handleUseItemOnWithIntents(useItemOnPacket(4, 63, -2, 1, 900), p, w, session.NewManager(), nil, nil, bus); err != nil {
		t.Fatal(err)
	}
	drained := bus.Drain()
	if len(drained.Gameplay) != 1 {
		t.Fatalf("gameplay intents = %#v", drained.Gameplay)
	}
	got, ok := drained.Gameplay[0].(intent.FireworkUseIntent)
	if !ok || got.PlayerUUID != p.UUID || got.HotbarSlot != 2 ||
		got.Position != (spatial.Vec3{X: 4.5, Y: 63.5, Z: -1.5}) {
		t.Fatalf("firework intent = %#v", drained.Gameplay[0])
	}
}
