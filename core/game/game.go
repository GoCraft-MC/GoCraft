// Package game is the protocol-independent GoCraft game core.
// It owns the canonical state of all online players and provides the
// interfaces that Java and Bedrock adapters use to register/deregister players.
//
// Neither this package nor any package under core/ may import Java- or
// Bedrock-specific packages (java/, bedrock/).
package game

import (
	"fmt"
	"sync"
	"sync/atomic"

	"GoCraft/core/player"
)

// Game is the central game engine. It manages online players and assigns
// entity IDs. World management will be added in a later milestone.
type Game struct {
	players sync.Map // [16]byte UUID → *player.Player

	// nextEntityID is the atomic counter for server-side entity IDs.
	// Entity ID 0 is reserved; player IDs start at 1.
	nextEntityID atomic.Int32
}

// New creates a ready-to-use Game instance.
func New() *Game {
	g := &Game{}
	g.nextEntityID.Store(1)
	return g
}

// AddPlayer registers a player with the game core, assigns an entity ID,
// and stores them in the online player map.
// Returns an error if a player with the same UUID is already online.
func (g *Game) AddPlayer(p *player.Player) error {
	p.EntityID = g.nextEntityID.Add(1) - 1
	if _, loaded := g.players.LoadOrStore(p.UUID, p); loaded {
		return fmt.Errorf("game: player %s (%s) is already online", p.Username, fmtUUID(p.UUID))
	}
	return nil
}

// RemovePlayer deregisters the player identified by uuid.
func (g *Game) RemovePlayer(uuid [16]byte) {
	g.players.Delete(uuid)
}

// GetPlayer returns the player with uuid, or nil if not online.
func (g *Game) GetPlayer(uuid [16]byte) *player.Player {
	v, ok := g.players.Load(uuid)
	if !ok {
		return nil
	}
	return v.(*player.Player)
}

// OnlinePlayers calls fn for every currently-online player.
// fn must not call AddPlayer or RemovePlayer (deadlock risk with sync.Map).
func (g *Game) OnlinePlayers(fn func(*player.Player)) {
	g.players.Range(func(_, v any) bool {
		fn(v.(*player.Player))
		return true
	})
}

// OnlineCount returns the number of currently-online players.
func (g *Game) OnlineCount() int {
	n := 0
	g.players.Range(func(_, _ any) bool { n++; return true })
	return n
}

// fmtUUID formats a [16]byte UUID as the standard 8-4-4-4-12 string.
func fmtUUID(u [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}
