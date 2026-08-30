package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
	"GoCraft/java/session"
)

func fireworkLifetimeTicks(flight uint8, firstRandom, secondRandom int) int32 {
	firstRandom = min(max(firstRandom, 0), 5)
	secondRandom = min(max(secondRandom, 0), 6)
	return int32((int(flight)+1)*10 + firstRandom + secondRandom)
}

func (s *Server) applyFireworkUse(i intent.FireworkUseIntent) *corentity.Entity {
	if s == nil || s.game == nil || i.HotbarSlot < 0 || i.HotbarSlot >= 9 {
		return nil
	}
	p := s.game.GetPlayer(i.PlayerUUID)
	if p == nil || p.Dead || p.GameMode == player.GameModeSpectator || p.Position.Distance(i.Position) > 6.5 {
		return nil
	}
	slot := player.HotbarStart + int(i.HotbarSlot)
	stack := p.Inventory[slot]
	if stack.ItemID != "minecraft:firework_rocket" || stack.Count < 1 {
		return nil
	}
	dimensionWorld := s.worldForPlayer(p)
	if dimensionWorld == nil {
		return nil
	}
	id := s.game.NextEntityID()
	rocket := corentity.New(id, newRandomUUID(), corentity.TypeFireworkRocket,
		i.Position.X, i.Position.Y, i.Position.Z)
	rocket.OwnerEntityID = p.EntityID
	rocket.FireworkData = stack.EffectiveFireworks()
	seed := uint32(id)*1664525 + 1013904223
	first, second := int(seed%6), int((seed>>8)%7)
	rocket.FireworkLifetime = fireworkLifetimeTicks(rocket.FireworkData.Flight, first, second)
	rocket.VX = triangularFireworkVelocity(seed)
	rocket.VY = 0.05
	rocket.VZ = triangularFireworkVelocity(seed*1103515245 + 12345)
	rocket.Pitch = -90
	dimensionWorld.Entities.Add(rocket)

	if p.GameMode != player.GameModeCreative {
		p.Inventory[slot].Count--
		if p.Inventory[slot].Count <= 0 {
			p.Inventory[slot] = player.ItemStack{}
		}
		if p.Edition == player.ClientEditionJava {
			if target, ok := s.sessions.Get(p.UUID); ok {
				_ = handler.SyncPlayerInventory(target.Conn, p)
			}
		}
	}
	handler.BroadcastSpawnMobInDimension(rocket, s.sessions, p.Dimension)
	handler.BroadcastSoundAtDimension(s.sessions, p.Dimension, "minecraft:entity.firework_rocket.launch",
		handler.SoundCategoryAmbient, i.Position.X, i.Position.Y, i.Position.Z, 3, 1)
	if s.bedrockListener != nil {
		s.bedrockListener.BroadcastFireworkLaunch(p.Dimension, i.Position)
	}
	return rocket
}

func triangularFireworkVelocity(seed uint32) float64 {
	first := float64(seed&0xffff) / 65535
	second := float64((seed>>16)&0xffff) / 65535
	return (first - second) * 0.002297
}

// tickFireworkRocket advances the canonical entity. It reports true after the
// explosion event is emitted and the caller should remove the entity.
func (s *Server) tickFireworkRocket(rocket *corentity.Entity) bool {
	if rocket == nil || rocket.Type != corentity.TypeFireworkRocket {
		return false
	}
	rocket.VX *= 1.15
	rocket.VY += 0.04
	rocket.VZ *= 1.15
	rocket.Position.X += rocket.VX
	rocket.Position.Y += rocket.VY
	rocket.Position.Z += rocket.VZ
	rocket.FireworkLifeTicks++
	if rocket.FireworkLifeTicks <= rocket.FireworkLifetime {
		return false
	}
	handler.BroadcastEntityStatusInDimension(rocket.EntityID, 17, s.sessions, s.simulationDimension)
	if s.bedrockListener != nil {
		s.bedrockListener.BroadcastFireworkExplosion(s.simulationDimension, rocket.EntityID)
	}
	s.damageFireworkTargets(rocket)
	return true
}

func (s *Server) damageFireworkTargets(rocket *corentity.Entity) {
	count := int(rocket.FireworkData.ExplosionCount)
	if s == nil || s.world == nil || count == 0 {
		return
	}
	force := float64(5 + count*2)
	s.game.OnlinePlayers(func(p *player.Player) {
		if p == nil || p.Dimension != s.simulationDimension || p.Dead {
			return
		}
		if damage := fireworkDamage(rocket.Position, p.Position, force); damage > 0 &&
			s.fireworkHasLineOfSight(rocket.Position, spatial.Vec3{X: p.Position.X, Y: p.Position.Y + 1.62, Z: p.Position.Z}) {
			target := &session.Session{Player: p}
			if p.Edition == player.ClientEditionJava {
				if current, ok := s.sessions.Get(p.UUID); ok {
					target = current
				}
			}
			handler.DamagePlayer(target, damage, "was blasted by a firework", s.sessions)
		}
	})
	for _, target := range s.world.Entities.Snapshot() {
		if target == nil || target == rocket || target.Dead || target.MaxHealth <= 0 {
			continue
		}
		if damage := fireworkDamage(rocket.Position, target.Position, force); damage > 0 &&
			s.fireworkHasLineOfSight(rocket.Position, spatial.Vec3{X: target.Position.X, Y: target.Position.Y + 1, Z: target.Position.Z}) {
			s.world.QueueEntityDamageFrom(target.EntityID, damage, rocket.Position.X, rocket.Position.Z)
		}
	}
}

func fireworkDamage(source, target spatial.Vec3, force float64) float32 {
	distance := source.Distance(target)
	if distance > 5 {
		return 0
	}
	return float32(force * math.Sqrt((5-distance)/5))
}

func (s *Server) fireworkHasLineOfSight(start, end spatial.Vec3) bool {
	distance := start.Distance(end)
	steps := max(2, int(math.Ceil(distance*5)))
	for step := 1; step < steps; step++ {
		t := float64(step) / float64(steps)
		block := s.world.GetBlock(
			int(math.Floor(start.X+(end.X-start.X)*t)),
			int(math.Floor(start.Y+(end.Y-start.Y)*t)),
			int(math.Floor(start.Z+(end.Z-start.Z)*t)),
		)
		if coreworld.IsEntitySupportBlock(block.ResourceLocation()) {
			return false
		}
	}
	return true
}
