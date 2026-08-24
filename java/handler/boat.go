package handler

// Boat packet handling for GoCraft.
//
// Boats are rideable entities: a player right-clicks to board, the client then
// sends move_vehicle packets (instead of move_player_*) to update the boat's
// position, and sneaking exits the boat.
//
// Relevant packets (Java 1.21.4 / protocol 769):
//   CB  set_passengers  (0x65 = 101) — tells client who is riding what
//   SB  move_vehicle    (0x1B = 27)  — client updates boat position while riding
//   SB  player_input    (0x20 = 32)  — input flags; shift = dismount
//   SB  player_command  (0x19 = 25)  — action 8 = start fall flying (elytra)

import (
	"fmt"
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/intent"
	coreplayer "GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

// ── Packet builders ───────────────────────────────────────────────────────────

// buildSetPassengers builds a CB Set Passengers packet.
//
// Wire layout (1.21.4):
//
//	VarInt  entity_id
//	VarInt  count
//	VarInt  passenger_id[0..count-1]
func buildSetPassengers(vehicleEntityID int32, passengerIDs []int32) *protocol.Packet {
	b := protocol.NewBuilder(packetIDSetPassengers).VarInt(vehicleEntityID).VarInt(int32(len(passengerIDs)))
	for _, id := range passengerIDs {
		b = b.VarInt(id)
	}
	return b.Build()
}

// BroadcastSetPassengers sends a Set Passengers packet to all sessions.
func BroadcastSetPassengers(vehicleEntityID int32, passengerIDs []int32, mgr *session.Manager) {
	pkt := buildSetPassengers(vehicleEntityID, passengerIDs)
	for _, s := range mgr.SnapshotAll() {
		_ = s.Conn.WritePacket(pkt)
	}
}

// ── Serverbound packet handlers ───────────────────────────────────────────────

// HandleMoveVehiclePacket parses a SB Move Vehicle packet and updates the
// vehicle position in the world, then broadcasts it to all other players.
func HandleMoveVehiclePacket(pkt *protocol.Packet, p *coreplayer.Player, w *coreworld.World, conn *network.ClientConn, mgr *session.Manager, buses ...*intent.Bus) error {
	r := pkt.Reader()

	x, err := protocol.ReadDouble(r)
	if err != nil {
		return fmt.Errorf("move_vehicle: x: %w", err)
	}
	y, err := protocol.ReadDouble(r)
	if err != nil {
		return fmt.Errorf("move_vehicle: y: %w", err)
	}
	z, err := protocol.ReadDouble(r)
	if err != nil {
		return fmt.Errorf("move_vehicle: z: %w", err)
	}
	yaw, err := protocol.ReadFloat(r)
	if err != nil {
		return fmt.Errorf("move_vehicle: yaw: %w", err)
	}
	if _, err = protocol.ReadFloat(r); err != nil { // pitch (unused for boats)
		return fmt.Errorf("move_vehicle: pitch: %w", err)
	}
	onGround, err := protocol.ReadBool(r)
	if err != nil {
		return fmt.Errorf("move_vehicle: on_ground: %w", err)
	}
	if len(buses) > 0 && buses[0] != nil {
		buses[0].PostVehicleMove(intent.VehicleMoveIntent{
			PlayerUUID: p.UUID,
			Position:   spatial.Vec3{X: x, Y: y, Z: z},
			Yaw:        yaw,
			OnGround:   onGround,
		})
		return nil
	}

	vehicleID := p.VehicleEntityID
	if vehicleID == 0 {
		return nil
	}
	vehicle, ok := w.Entities.Get(vehicleID)
	if !ok || !corentity.IsRideableVehicle(vehicle.Type) || vehicle.RiderEntityID != p.EntityID {
		p.VehicleEntityID = 0
		return nil
	}

	// Reject suspiciously large jumps (> 3 blocks) as lag-spike protection.
	dx := x - vehicle.Position.X
	dy := y - vehicle.Position.Y
	dz := z - vehicle.Position.Z
	if math.Abs(dx) > 3 || math.Abs(dy) > 3 || math.Abs(dz) > 3 {
		_ = sendSyncPosition(conn, p, 0)
		return nil
	}

	vehicle.Position.X = x
	vehicle.Position.Y = y
	vehicle.Position.Z = z
	vehicle.Yaw = yaw
	vehicle.OnGround = onGround

	p.Position.X = x
	p.Position.Y = y + 0.35
	p.Position.Z = z

	broadcastBoatPositionExcept(vehicle, p.EntityID, mgr)
	return nil
}

// HandlePlayerInputPacket parses a SB Player Input packet.
// Shift flag (bit 5) exits a vehicle.
func HandlePlayerInputPacket(pkt *protocol.Packet, p *coreplayer.Player, w *coreworld.World, conn *network.ClientConn, mgr *session.Manager, buses ...*intent.Bus) error {
	r := pkt.Reader()
	flags, err := protocol.ReadByte(r)
	if err != nil {
		return fmt.Errorf("player_input: flags: %w", err)
	}
	if flags&0x20 != 0 && p.VehicleEntityID != 0 {
		if len(buses) > 0 && buses[0] != nil {
			buses[0].PostEntityInteract(intent.EntityInteractIntent{PlayerUUID: p.UUID, TargetID: 0, HotbarSlot: int32(p.HeldSlot)})
			return nil
		}
		DismountPlayer(p, w, conn, mgr)
	}
	return nil
}

// HandlePlayerCommandPacket parses a SB Player Command packet.
//
// Modern Java action IDs used here:
//   0 = start sneaking
//   1 = stop sneaking
//   2 = leave bed
//   3 = start sprinting
//   4 = stop sprinting
//   8 = start fall flying (elytra)
//
// Previous GoCraft code incorrectly treated action 8 as LEAVE_VEHICLE. That
// made an elytra start packet hit the vehicle path and left fall-flying state
// effectively unsupported. Vehicle dismount remains handled by Player Input's
// shift flag, which is the packet path used by modern clients.
func HandlePlayerCommandPacket(pkt *protocol.Packet, p *coreplayer.Player, w *coreworld.World, conn *network.ClientConn, mgr *session.Manager, buses ...*intent.Bus) error {
	r := pkt.Reader()
	if _, err := protocol.ReadVarInt(r); err != nil {
		return fmt.Errorf("player_command: entity_id: %w", err)
	}
	action, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("player_command: action: %w", err)
	}
	// jump_boost follows action on the wire. It is unused by these actions but
	// consume it when present so malformed/truncated command packets are caught.
	if _, err := protocol.ReadVarInt(r); err != nil {
		return fmt.Errorf("player_command: jump_boost: %w", err)
	}

	switch action {
	case 0: // START_SNEAKING
		p.Sneaking = true
	case 1: // STOP_SNEAKING
		p.Sneaking = false
	case 2: // LEAVE_BED
		if p.Sleeping {
			p.Sleeping = false
			BroadcastPlayerWaking(p.EntityID, mgr)
			_ = sendSystemMessage(conn, "You left your bed.")
		}
	case 3: // START_SPRINTING
		p.Sprinting = true
	case 4: // STOP_SPRINTING
		p.Sprinting = false
	case 8: // START_FALL_FLYING
		// The client may only begin gliding while airborne with an elytra in the
		// chest slot. The actual movement remains client-driven like ordinary
		// Java movement; reset accumulated fall distance so starting a valid
		// glide cannot immediately apply the pre-glide fall as landing damage.
		if p.VehicleEntityID == 0 && !p.OnGround && p.Inventory[6].ItemID == "minecraft:elytra" {
			p.FallDistance = 0
		}
	}
	return nil
}

// ── Mount / dismount ──────────────────────────────────────────────────────────

func MountPlayer(p *coreplayer.Player, boatEntityID int32, w *coreworld.World, mgr *session.Manager) bool {
	boat, ok := w.Entities.Get(boatEntityID)
	if !ok || !corentity.IsBoat(boat.Type) {
		return false
	}
	if !boat.AddPassenger(p.EntityID) {
		return false
	}
	p.VehicleEntityID = boatEntityID
	BroadcastSetPassengers(boatEntityID, boat.PassengerIDs(), mgr)
	return true
}

func DismountPlayer(p *coreplayer.Player, w *coreworld.World, conn *network.ClientConn, mgr *session.Manager) {
	boatID := p.VehicleEntityID
	if boatID == 0 {
		return
	}
	p.VehicleEntityID = 0

	if boat, ok := w.Entities.Get(boatID); ok {
		boat.RemovePassenger(p.EntityID)
		p.Position.X = boat.Position.X + 1.5
		p.Position.Y = boat.Position.Y
		p.Position.Z = boat.Position.Z
	}

	if boat, ok := w.Entities.Get(boatID); ok {
		BroadcastSetPassengers(boatID, boat.PassengerIDs(), mgr)
	} else {
		BroadcastSetPassengers(boatID, nil, mgr)
	}
	_ = sendSyncPosition(conn, p, 0)
}

// ── Broadcast helpers ─────────────────────────────────────────────────────────

func broadcastBoatPositionExcept(boat *corentity.Entity, excludeEntityID int32, mgr *session.Manager) {
	pkt := buildTeleportMob(boat)
	for _, s := range mgr.SnapshotAll() {
		if s.Player != nil && s.Player.EntityID == excludeEntityID {
			continue
		}
		_ = s.Conn.WritePacket(pkt)
	}
}
