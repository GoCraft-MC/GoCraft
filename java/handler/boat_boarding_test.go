package handler

import (
	"testing"

	"GoCraft/core/intent"
	"GoCraft/java/protocol"
)

func TestJavaBoatBoardingPostsCanonicalIntent(t *testing.T) {
	p, boat, w := newBoatInventoryTest(t)
	bus := intent.NewBus(1, 4)
	pkt := protocol.NewBuilder(packetIDInteract).VarInt(boat.EntityID).VarInt(0).VarInt(0).Bool(false).Build()
	if err := handleInteractPacket(pkt, p, w, nil, nil, bus); err != nil {
		t.Fatal(err)
	}
	if p.VehicleEntityID != 0 || len(boat.PassengerIDs()) != 0 {
		t.Fatal("handler changed passengers outside the tick")
	}
	events := bus.Drain().Gameplay
	if len(events) != 1 {
		t.Fatalf("boarding intents = %d", len(events))
	}
	if event, ok := events[0].(intent.EntityInteractIntent); !ok || event.TargetID != boat.EntityID || event.PlayerUUID != p.UUID {
		t.Fatalf("boarding intent = %+v", events[0])
	}
}
