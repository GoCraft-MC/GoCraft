// Package experience contains edition-neutral experience-orb rules shared by
// the Java and Bedrock protocol adapters.
package experience

import (
	"encoding/binary"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

// RoundToOrbSize matches Pumpkin's ExperienceOrbEntity thresholds.
func RoundToOrbSize(value int32) int32 {
	switch {
	case value >= 2477:
		return 2477
	case value >= 1237:
		return 1237
	case value >= 617:
		return 617
	case value >= 307:
		return 307
	case value >= 149:
		return 149
	case value >= 73:
		return 73
	case value >= 37:
		return 37
	case value >= 17:
		return 17
	case value >= 7:
		return 7
	case value >= 3:
		return 3
	case value > 0:
		return 1
	default:
		return 0
	}
}

// SpawnOrbs splits points using Pumpkin's thresholds and inserts every orb
// into the canonical world. Entity IDs are globally supplied by the game.
func SpawnOrbs(world *coreworld.World, nextEntityID func() int32, position spatial.Vec3, points int32) []*corentity.Entity {
	if world == nil || nextEntityID == nil || points <= 0 {
		return nil
	}
	orbs := make([]*corentity.Entity, 0, 4)
	for points > 0 {
		amount := RoundToOrbSize(points)
		id := nextEntityID()
		orb := corentity.New(id, orbUUID(id), corentity.TypeExperienceOrb, position.X, position.Y, position.Z)
		orb.ExperienceAmount = amount
		world.Entities.Add(orb)
		orbs = append(orbs, orb)
		points -= amount
	}
	return orbs
}

func orbUUID(id int32) [16]byte {
	var result [16]byte
	binary.BigEndian.PutUint32(result[12:], uint32(id))
	result[6] = 0x40
	result[8] = 0x80
	return result
}
