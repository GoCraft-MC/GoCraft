package bedrock

import (
	"GoCraft/core/player"
	"GoCraft/core/spatial"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

var fireworkDyeColours = [...]int32{
	0xf0f0f0, 0xf9801d, 0xc74ebd, 0x3ab3da,
	0xfed83d, 0x80c71f, 0xf38baa, 0x474f52,
	0x9d9d97, 0x169c9c, 0x8932b8, 0x3c44aa,
	0x835432, 0x5e7c16, 0xb02e26, 0x1d1d21,
}

func bedrockFireworkDataFromNBT(data map[string]any) (player.FireworkData, bool) {
	root, ok := data["Fireworks"].(map[string]any)
	if !ok {
		return player.FireworkData{}, false
	}
	result := player.FireworkData{Flight: uint8(clampInt(nbtInt(root["Flight"]), 0, 255))}
	for _, value := range nbtList(root["Explosions"]) {
		entry, ok := value.(map[string]any)
		if !ok || result.ExplosionCount >= player.MaxFireworkExplosions {
			continue
		}
		explosion := &result.Explosions[result.ExplosionCount]
		explosion.Shape = uint8(clampInt(nbtInt(entry["FireworkType"]), 0, 4))
		explosion.Twinkle = nbtInt(entry["FireworkFlicker"]) != 0
		explosion.Trail = nbtInt(entry["FireworkTrail"]) != 0
		readBedrockFireworkColours(entry["FireworkColor"], &explosion.Colors, &explosion.ColorCount)
		readBedrockFireworkColours(entry["FireworkFade"], &explosion.FadeColors, &explosion.FadeColorCount)
		result.ExplosionCount++
	}
	return result, true
}

func bedrockFireworkNBT(data player.FireworkData) map[string]any {
	data = player.ItemStack{ItemID: "minecraft:firework_rocket", HasFireworks: true, Fireworks: data}.EffectiveFireworks()
	explosions := make([]any, 0, data.ExplosionCount)
	for index := 0; index < int(data.ExplosionCount); index++ {
		explosion := data.Explosions[index]
		explosions = append(explosions, map[string]any{
			"FireworkType":    uint8(explosion.Shape),
			"FireworkColor":   bedrockFireworkColours(explosion.Colors, explosion.ColorCount),
			"FireworkFade":    bedrockFireworkColours(explosion.FadeColors, explosion.FadeColorCount),
			"FireworkFlicker": boolByte(explosion.Twinkle),
			"FireworkTrail":   boolByte(explosion.Trail),
		})
	}
	return map[string]any{"Fireworks": map[string]any{
		"Flight": data.Flight, "Explosions": explosions,
	}}
}

func readBedrockFireworkColours(value any, colours *[player.MaxFireworkColors]int32, count *uint8) {
	for _, raw := range nbtList(value) {
		if *count >= player.MaxFireworkColors {
			return
		}
		index := uint8(nbtInt(raw)) ^ 15
		if int(index) >= len(fireworkDyeColours) {
			continue
		}
		colours[*count] = fireworkDyeColours[index]
		*count++
	}
}

func bedrockFireworkColours(colours [player.MaxFireworkColors]int32, count uint8) []uint8 {
	out := make([]uint8, 0, count)
	for index := 0; index < int(count) && index < len(colours); index++ {
		out = append(out, nearestFireworkDye(colours[index])^15)
	}
	return out
}

func nearestFireworkDye(rgb int32) uint8 {
	best, bestDistance := uint8(0), int64(1<<63-1)
	for index, candidate := range fireworkDyeColours {
		dr := int64((rgb>>16)&0xff) - int64((candidate>>16)&0xff)
		dg := int64((rgb>>8)&0xff) - int64((candidate>>8)&0xff)
		db := int64(rgb&0xff) - int64(candidate&0xff)
		if distance := dr*dr + dg*dg + db*db; distance < bestDistance {
			best, bestDistance = uint8(index), distance
		}
	}
	return best
}

func nbtList(value any) []any {
	switch values := value.(type) {
	case []any:
		return values
	case []uint8:
		out := make([]any, len(values))
		for index, value := range values {
			out[index] = value
		}
		return out
	case [1]uint8:
		return []any{values[0]}
	default:
		return nil
	}
}

func nbtInt(value any) int {
	switch number := value.(type) {
	case uint8:
		return int(number)
	case int8:
		return int(number)
	case int16:
		return int(number)
	case int32:
		return int(number)
	case int:
		return number
	default:
		return 0
	}
}

func clampInt(value, low, high int) int { return min(max(value, low), high) }
func boolByte(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}

// BroadcastFireworkExplosion translates a canonical rocket expiry for Bedrock viewers.
func (l *Listener) BroadcastFireworkExplosion(dimension, entityID int32) {
	l.broadcastDimension(dimension, &packet.ActorEvent{
		EntityRuntimeID: bedrockRemoteRuntimeID(entityID), EventType: packet.ActorEventFireworksExplode,
	})
}

// BroadcastFireworkLaunch translates the shared launch sound for Bedrock viewers.
func (l *Listener) BroadcastFireworkLaunch(dimension int32, position spatial.Vec3) {
	l.broadcastDimension(dimension, &packet.LevelSoundEvent{
		SoundType: packet.SoundEventLaunch,
		Position:  mgl32.Vec3{float32(position.X), float32(position.Y), float32(position.Z)},
		ExtraData: -1,
	})
}

func (l *Listener) broadcastDimension(dimension int32, value packet.Packet) {
	if l == nil {
		return
	}
	l.sessionsMu.RLock()
	sessions := make([]*bedrockSession, 0, len(l.sessions))
	for _, current := range l.sessions {
		if current.dimension.Load() == dimension {
			sessions = append(sessions, current)
		}
	}
	l.sessionsMu.RUnlock()
	for _, current := range sessions {
		_ = current.conn.WritePacket(value)
	}
}
