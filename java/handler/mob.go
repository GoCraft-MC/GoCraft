package handler

// Mob (non-player entity) packet handling for Milestone 11.
//
// Builds and broadcasts the Java Edition packets needed to spawn, move, and
// despawn non-player entities.  The Spawn Entity packet (0x01) is reused for
// all entity types in 1.18+; this file handles mob-specific wiring.
//
// All packet IDs are estimates for 1.21.4 (protocol 769); verify against a
// 1.21.4 packet capture if an entity spawns incorrectly.

import (
	corentity "GoCraft/core/entity"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

// ── Packet builders ───────────────────────────────────────────────────────────

// buildSpawnMob constructs a Spawn Entity (0x01 S→C) packet for a non-player
// entity.  Returns (packet, true) on success, or (nil, false) when the entity
// type is not in the Java registry table (unknown types must not be sent
// because the client would disconnect on an invalid registry index).
//
// Wire layout (1.21.4, same as buildSpawnPlayer):
//
//	VarInt  entity_id
//	UUID    entity_uuid
//	VarInt  entity_type   (numeric registry index)
//	Double  x, y, z
//	Angle   pitch, yaw    (Byte, 256 units = 360°)
//	Angle   head_yaw      (Byte)
//	VarInt  data          (0 for most mobs)
//	Short   velocity_x/y/z (fixed-point blocks/tick × 8000)
func buildSpawnMob(e *corentity.Entity) (*protocol.Packet, bool) {
	typeID := javaworld.EntityTypeID(string(e.Type))
	if typeID < 0 {
		return nil, false
	}
	// Velocity wire format: blocks/tick × 8000, clamped to short range.
	vx := velToShort(e.VX)
	vy := velToShort(e.VY)
	vz := velToShort(e.VZ)
	return protocol.NewBuilder(packetIDSpawnEntity).
		VarInt(e.EntityID).
		UUID(protocol.UUID(e.UUID)).
		VarInt(typeID).
		Double(e.Position.X).
		Double(e.Position.Y).
		Double(e.Position.Z).
		Byte(degToAngle(e.Pitch)).
		Byte(degToAngle(e.Yaw)).
		Byte(degToAngle(e.Yaw)). // head yaw = body yaw at spawn
		VarInt(0).               // data (type-specific; 0 = default state)
		Short(vx).Short(vy).Short(vz).
		Build(), true
}

// buildTeleportMob constructs a Teleport Entity packet for a non-player entity.
// Uses the same packet ID and layout as buildTeleportEntity (players).
func buildTeleportMob(e *corentity.Entity) *protocol.Packet {
	return protocol.NewBuilder(packetIDTeleportEntity).
		VarInt(e.EntityID).
		Double(e.Position.X).
		Double(e.Position.Y).
		Double(e.Position.Z).
		Double(e.VX).
		Double(e.VY).
		Double(e.VZ).
		Float(e.Yaw).
		Float(e.Pitch).
		Bool(e.OnGround).
		Build()
}

// velToShort converts a velocity in blocks/tick to the Minecraft short wire
// format (blocks/tick × 8000, clamped to int16 range).
func velToShort(v float64) int16 {
	s := int64(v * 8000)
	if s > 32767 {
		return 32767
	}
	if s < -32768 {
		return -32768
	}
	return int16(s)
}

// ── Session-level helpers ─────────────────────────────────────────────────────

// sendExistingMobsTo sends a Spawn Entity packet for each entity in mobs to
// the given connection.  Called during player join so newly-connected clients
// see all currently-loaded non-player entities.
func sendExistingMobsTo(conn *network.ClientConn, mobs []*corentity.Entity) {
	for _, e := range mobs {
		pkt, ok := buildSpawnMob(e)
		if !ok {
			continue // unknown entity type — skip rather than disconnect client
		}
		_ = conn.WritePacket(pkt)
	}
}

// BroadcastSpawnMob sends a Spawn Entity packet for e to every connected
// session.  Call this after adding the entity to the world's entity manager.
func BroadcastSpawnMob(e *corentity.Entity, mgr *session.Manager) {
	pkt, ok := buildSpawnMob(e)
	if !ok {
		return
	}
	for _, s := range mgr.SnapshotAll() {
		_ = s.Conn.WritePacket(pkt)
	}
}

// BroadcastEntityPosition sends a Teleport Entity packet for e to every
// connected session.  Called by the entity tick goroutine after position
// integration.
func BroadcastEntityPosition(e *corentity.Entity, mgr *session.Manager) {
	pkt := buildTeleportMob(e)
	for _, s := range mgr.SnapshotAll() {
		_ = s.Conn.WritePacket(pkt)
	}
}

// BroadcastRemoveEntity sends a Remove Entities packet for entityID to every
// connected session.  Called when an entity dies or is otherwise despawned.
func BroadcastRemoveEntity(entityID int32, mgr *session.Manager) {
	pkt := buildRemoveEntities(entityID)
	for _, s := range mgr.SnapshotAll() {
		_ = s.Conn.WritePacket(pkt)
	}
}
