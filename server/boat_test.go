package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
	"GoCraft/java/protocol"
)

func TestJavaBoatDismountInEveryDimension(t *testing.T) {
	for _, boatType := range []corentity.EntityType{corentity.TypeOakBoat, corentity.TypeOakChestBoat} {
		for dimension := int32(0); dimension <= 2; dimension++ {
			for _, command := range []bool{false, true} {
				s, p := newAnimalTestServer(t)
				p.Edition, p.Dimension = player.ClientEditionJava, dimension
				if dimension != 0 {
					w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
					t.Cleanup(func() { _ = w.Close() })
					if dimension == 1 {
						s.netherWorld = w
					} else {
						s.endWorld = w
					}
				}
				boat := corentity.New(90, [16]byte{}, boatType, 0, 64, 0)
				s.worldForPlayer(p).Entities.Add(boat)
				boat.AddPassenger(p.EntityID)
				p.VehicleEntityID = boat.EntityID
				bus := intent.NewBus(1, 4)
				var err error
				if command {
					packet := protocol.NewBuilder(0).VarInt(p.EntityID).VarInt(0).VarInt(0).Build()
					err = handler.HandlePlayerCommandPacket(packet, p, s.worldForPlayer(p), nil, s.sessions, bus)
				} else {
					err = handler.HandlePlayerInputPacket(&protocol.Packet{Data: []byte{0x20}}, p, s.worldForPlayer(p), nil, s.sessions, bus)
				}
				if err != nil {
					t.Fatal(err)
				}
				for _, event := range bus.Drain().Gameplay {
					s.applyEntityInteract(event.(intent.EntityInteractIntent))
				}
				if p.VehicleEntityID != 0 || boat.HasPassenger(p.EntityID) {
					t.Errorf("type=%s dimension=%d command=%v: vehicle=%d passengers=%v", boatType, dimension, command, p.VehicleEntityID, boat.PassengerIDs())
				}
			}
		}
	}
}
