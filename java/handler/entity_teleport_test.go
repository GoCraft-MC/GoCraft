package handler

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/java/protocol"
)

func TestTeleportEntityPacketsIncludeRelativeFlags(t *testing.T) {
	t.Run("player", func(t *testing.T) {
		p := player.New([16]byte{1}, "player", player.ClientEditionJava)
		p.EntityID = 42
		p.Position.X, p.Position.Y, p.Position.Z = 1.25, 64, -3.5
		p.Rotation.Yaw, p.Rotation.Pitch = 90, -15
		p.OnGround = true

		assertTeleportEntityPayload(t, buildTeleportEntity(p), 42, true)
	})

	t.Run("mob", func(t *testing.T) {
		e := corentity.New(7, [16]byte{2}, corentity.TypeCow, 4, 70, -8)
		e.VX, e.VY, e.VZ = 0.1, -0.2, 0.3
		e.Yaw, e.Pitch = 45, 10
		e.OnGround = false

		assertTeleportEntityPayload(t, buildTeleportMob(e), 7, false)
	})
}

func assertTeleportEntityPayload(t *testing.T, pkt *protocol.Packet, wantEntityID int32, wantOnGround bool) {
	t.Helper()

	if pkt.ID != packetIDTeleportEntity {
		t.Fatalf("packet ID = %d, want %d", pkt.ID, packetIDTeleportEntity)
	}

	r := pkt.Reader()
	entityID, err := protocol.ReadVarInt(r)
	if err != nil {
		t.Fatalf("read entity ID: %v", err)
	}
	if entityID != wantEntityID {
		t.Fatalf("entity ID = %d, want %d", entityID, wantEntityID)
	}

	for i := 0; i < 6; i++ {
		if _, err := protocol.ReadDouble(r); err != nil {
			t.Fatalf("read double field %d: %v", i, err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := protocol.ReadFloat(r); err != nil {
			t.Fatalf("read float field %d: %v", i, err)
		}
	}

	flags, err := protocol.ReadInt(r)
	if err != nil {
		t.Fatalf("read relative flags: %v", err)
	}
	if flags != 0 {
		t.Fatalf("relative flags = %d, want 0", flags)
	}

	onGround, err := protocol.ReadBool(r)
	if err != nil {
		t.Fatalf("read on-ground flag: %v", err)
	}
	if onGround != wantOnGround {
		t.Fatalf("on-ground = %v, want %v", onGround, wantOnGround)
	}
	if r.Len() != 0 {
		t.Fatalf("unexpected trailing payload: %d bytes", r.Len())
	}
}
