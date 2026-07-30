// Package entity defines the edition-agnostic entity model used by the
// GoCraft game core.  Java- or Bedrock-specific identifiers (numeric entity
// type registry IDs, metadata formats) must live in the respective adapter
// packages; this package uses only canonical resource-location strings.
package entity

import "GoCraft/core/spatial"

// EntityType is the canonical resource location for a Minecraft entity type,
// e.g. "minecraft:cow".
type EntityType string

// Common entity types.  Expand as needed; Milestone 13 (data-driven
// registries) will load the full list from Minecraft's generated output.
const (
	// Passive mobs
	TypeBat     EntityType = "minecraft:bat"
	TypeChicken EntityType = "minecraft:chicken"
	TypeCow     EntityType = "minecraft:cow"
	TypeDonkey  EntityType = "minecraft:donkey"
	TypeFox     EntityType = "minecraft:fox"
	TypeFrog    EntityType = "minecraft:frog"
	TypeGoat    EntityType = "minecraft:goat"
	TypeHorse   EntityType = "minecraft:horse"
	TypeMule    EntityType = "minecraft:mule"
	TypeOcelot  EntityType = "minecraft:ocelot"
	TypePanda   EntityType = "minecraft:panda"
	TypeParrot  EntityType = "minecraft:parrot"
	TypePig     EntityType = "minecraft:pig"
	TypeRabbit  EntityType = "minecraft:rabbit"
	TypeSheep   EntityType = "minecraft:sheep"
	TypeSniffer EntityType = "minecraft:sniffer"
	TypeSquid   EntityType = "minecraft:squid"
	TypeTurtle  EntityType = "minecraft:turtle"
	TypeWolf    EntityType = "minecraft:wolf"

	// Hostile mobs
	TypeBlaze      EntityType = "minecraft:blaze"
	TypeCaveSpider EntityType = "minecraft:cave_spider"
	TypeCreeper    EntityType = "minecraft:creeper"
	TypeDrowned    EntityType = "minecraft:drowned"
	TypeEnderman   EntityType = "minecraft:enderman"
	TypeGhast      EntityType = "minecraft:ghast"
	TypeGuardian   EntityType = "minecraft:guardian"
	TypeHusk       EntityType = "minecraft:husk"
	TypeMagmaCube  EntityType = "minecraft:magma_cube"
	TypePhantom    EntityType = "minecraft:phantom"
	TypePiglin     EntityType = "minecraft:piglin"
	TypePillager   EntityType = "minecraft:pillager"
	TypeRavager    EntityType = "minecraft:ravager"
	TypeSilverfish EntityType = "minecraft:silverfish"
	TypeSkeleton   EntityType = "minecraft:skeleton"
	TypeSlime      EntityType = "minecraft:slime"
	TypeSpider     EntityType = "minecraft:spider"
	TypeVex        EntityType = "minecraft:vex"
	TypeVindicator EntityType = "minecraft:vindicator"
	TypeWarden     EntityType = "minecraft:warden"
	TypeWitch      EntityType = "minecraft:witch"
	TypeZombie     EntityType = "minecraft:zombie"

	// Neutral mobs
	TypeIronGolem EntityType = "minecraft:iron_golem"
	TypePolarBear EntityType = "minecraft:polar_bear"
	TypeSnowGolem EntityType = "minecraft:snow_golem"
	TypeStrider   EntityType = "minecraft:strider"
)

// Entity is the canonical server-side representation of any non-player entity.
//
// # Concurrency ownership
//
// The entity tick goroutine in server.Server is the sole writer of the spatial
// and health fields (Position, VX/VY/VZ, OnGround, Health, Dead).  All other
// goroutines (player handlers, commands) must treat these fields as read-only
// unless they hold an external lock or go through a dedicated mutation API.
//
// The current design is safe because:
//   - Only the tick goroutine calls Damage/Heal and integrates velocity.
//   - Broadcast goroutines only read from already-built protocol.Packet bytes,
//     never from entity fields directly (see server.tickEntities).
//
// Per-entity locking will be added when combat or concurrent mutations are
// introduced.
type Entity struct {
	// Identity
	EntityID int32
	UUID     [16]byte
	Type     EntityType

	// Spatial state — written only by the entity tick goroutine.
	Position spatial.Vec3
	VX, VY, VZ float64 // velocity in blocks/tick
	Yaw, Pitch float32
	OnGround   bool

	// Health
	Health    float32
	MaxHealth float32
	Dead      bool
}

// New creates an entity of the given type at the given position with full health.
func New(id int32, uuid [16]byte, t EntityType, x, y, z float64) *Entity {
	maxHP := defaultMaxHealth(t)
	return &Entity{
		EntityID:  id,
		UUID:      uuid,
		Type:      t,
		Position:  spatial.Vec3{X: x, Y: y, Z: z},
		Health:    maxHP,
		MaxHealth: maxHP,
	}
}

// defaultMaxHealth returns the base max health for well-known entity types.
// Unknown types default to 20 (10 hearts).
func defaultMaxHealth(t EntityType) float32 {
	switch t {
	case TypeChicken:
		return 4
	case TypeRabbit:
		return 3
	case TypeBat:
		return 6
	case TypeCow, TypePig, TypeDonkey, TypeMule:
		return 10
	case TypeSheep, TypeOcelot, TypeParrot:
		return 8
	case TypeFox, TypeGoat, TypePanda, TypePolarBear, TypeStrider:
		return 20
	case TypeWolf:
		return 8
	case TypeHorse:
		return 30
	case TypeSniffer:
		return 14
	case TypeCaveSpider:
		return 12
	case TypeSilverfish, TypeSlime:
		return 8
	case TypeZombie, TypeSkeleton, TypeCreeper, TypeEnderman,
		TypeSpider, TypeHusk, TypeDrowned, TypePhantom, TypeVex,
		TypePiglin, TypePillager, TypeVindicator, TypeWitch, TypeBlaze:
		return 20
	case TypeGuardian:
		return 30
	case TypeGhast:
		return 10
	case TypeMagmaCube:
		return 16
	case TypeRavager:
		return 100
	case TypeWarden:
		return 500
	case TypeIronGolem:
		return 100
	case TypeSnowGolem:
		return 4
	default:
		return 20
	}
}

// Damage reduces the entity's health by amount.  If health reaches zero the
// entity is marked Dead.  Damage to an already-dead entity is a no-op.
func (e *Entity) Damage(amount float32) {
	if e.Dead || amount <= 0 {
		return
	}
	e.Health -= amount
	if e.Health <= 0 {
		e.Health = 0
		e.Dead = true
	}
}

// Heal restores health by amount, capped at MaxHealth.  Reviving a dead
// entity is not supported here; callers that want to respawn entities should
// create a new one.
func (e *Entity) Heal(amount float32) {
	if amount <= 0 {
		return
	}
	e.Health += amount
	if e.Health > e.MaxHealth {
		e.Health = e.MaxHealth
	}
}

// IsAlive reports whether the entity has not yet died.
func (e *Entity) IsAlive() bool { return !e.Dead }
