// Package player defines the edition-agnostic Player model used by the
// GoCraft game core. Java- or Bedrock-specific fields (packet IDs, metadata
// formats, registry indices) must live in the respective adapter packages.
package player

import (
	"GoCraft/core/spatial"
)

// ClientEdition identifies which protocol edition the player is using.
type ClientEdition uint8

const (
	// ClientEditionJava is the Minecraft: Java Edition protocol.
	ClientEditionJava ClientEdition = iota
	// ClientEditionBedrock is the Minecraft: Bedrock Edition protocol.
	ClientEditionBedrock
)

// GameMode is the in-game play mode, identical across editions.
type GameMode uint8

const (
	GameModeSurvival  GameMode = 0
	GameModeCreative  GameMode = 1
	GameModeAdventure GameMode = 2
	GameModeSpectator GameMode = 3
)

// Player is the canonical server-side player representation.
// It is intentionally free of any Java- or Bedrock-specific types.
//
// Network-level concerns (packet sending, connection state, encryption) are
// owned by the edition adapter, which holds a *Player and updates it as
// packets arrive.
type Player struct {
	// UUID is the player's unique identifier (edition-agnostic).
	UUID [16]byte
	// Username is the player's display name.
	Username string
	// Edition indicates which protocol the player is connecting over.
	Edition ClientEdition

	// Position is the player's current world position.
	Position spatial.Vec3
	// Rotation holds the player's look direction.
	Rotation spatial.Rotation
	// OnGround reports whether the player last reported being on the ground.
	OnGround bool

	// GameMode is the current game mode.
	GameMode GameMode

	// EntityID is the server-assigned entity ID used in packets.
	// It is assigned by the game core when the player joins.
	EntityID int32
}

// New creates a Player with sensible defaults.
func New(uuid [16]byte, username string, edition ClientEdition) *Player {
	return &Player{
		UUID:     uuid,
		Username: username,
		Edition:  edition,
		Position: spatial.DefaultSpawnPos,
		GameMode: GameModeSurvival,
	}
}
