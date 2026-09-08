package handler

import (
	"bytes"
	"net"
	"testing"
	"time"

	"GoCraft/core/player"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

type boatPacketSink struct {
	net.Conn
	buffer bytes.Buffer
}

func (s *boatPacketSink) Write(data []byte) (int, error)   { return s.buffer.Write(data) }
func (s *boatPacketSink) SetWriteDeadline(time.Time) error { return nil }

func TestBoatInventorySendsChestScreenAndClosesWhenOutOfRange(t *testing.T) {
	p, boat, w := newBoatInventoryTest(t)
	sink := &boatPacketSink{}
	conn := network.NewClientConn(sink)
	if err := openBoatInventory(p, conn, boat); err != nil {
		t.Fatal(err)
	}
	screen, err := protocol.ReadPacket(&sink.buffer)
	if err != nil {
		t.Fatal(err)
	}
	r := screen.Reader()
	window, _ := protocol.ReadVarInt(r)
	menu, _ := protocol.ReadVarInt(r)
	if screen.ID != packetIDOpenScreen || window != 1 || menu != 2 {
		t.Fatal("incorrect chest screen")
	}
	contents, err := protocol.ReadPacket(&sink.buffer)
	if err != nil {
		t.Fatal(err)
	}
	r = contents.Reader()
	protocol.ReadVarInt(r)
	protocol.ReadVarInt(r)
	slots, _ := protocol.ReadVarInt(r)
	if contents.ID != packetIDSetContainerContent || slots != 63 {
		t.Fatalf("inventory slots=%d, want 27+36", slots)
	}
	p.Position.X += 20
	p.CarriedItem = player.ItemStack{ItemID: "minecraft:diamond", Count: 1}
	click := protocol.NewBuilder(packetIDContainerClick).VarInt(1).VarInt(p.ContainerStateID).
		Short(0).Byte(0).VarInt(0).VarInt(0).VarInt(0).Build()
	if err := handleContainerClick(click, p, conn, w, nil); err != nil {
		t.Fatal(err)
	}
	closed, err := protocol.ReadPacket(&sink.buffer)
	if err != nil {
		t.Fatal(err)
	}
	if closed.ID != 0x12 || p.OpenContainerKind != "" || p.CarriedItem.Count != 1 || !boat.Storage.Snapshot()[0].IsEmpty() {
		t.Fatal("out-of-range access did not close without transferring items")
	}
}

func TestDismountSendsEmptyPassengerList(t *testing.T) {
	p, boat, w := newBoatInventoryTest(t)
	boat.AddPassenger(p.EntityID)
	p.VehicleEntityID = boat.EntityID
	sink := &boatPacketSink{}
	conn := network.NewClientConn(sink)
	mgr := session.NewManager()
	mgr.Add(&session.Session{Player: p, Conn: conn})
	DismountPlayer(p, w, conn, mgr)
	packet, err := protocol.ReadPacket(&sink.buffer)
	if err != nil {
		t.Fatal(err)
	}
	r := packet.Reader()
	id, _ := protocol.ReadVarInt(r)
	count, _ := protocol.ReadVarInt(r)
	if packet.ID != packetIDSetPassengers || id != boat.EntityID || count != 0 || p.VehicleEntityID != 0 {
		t.Fatal("client still sees a mounted passenger")
	}
}
