// Package bedrock implements the Minecraft Bedrock Edition network adapter for
// GoCraft.  It accepts UDP/RakNet connections via gophertunnel, authenticates
// players through Xbox Live, and translates between the Bedrock protocol and
// the edition-agnostic core simulation through the intent bus.
//
// Supported Bedrock protocol: determined by the pinned gophertunnel release.
//   - gophertunnel fork (HashimTheArab/gophertunnel@218ac3ff) → Bedrock protocol 2168 (Minecraft BE 1.26.40)
//
// Architecture (sole-writer invariant):
//
//	Bedrock client ──RakNet/UDP──> Listener.Listen()
//	                                     │
//	                               handleConn() goroutine per client
//	                                     │
//	                      post Intents to core/intent.Bus
//	                      (never mutate core state directly)
//	                                     │
//	                          core simulation tick goroutine
//	                          applies intents, sends JoinResult
package bedrock

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	dfblock "github.com/df-mc/dragonfly/server/block"
	dfitem "github.com/df-mc/dragonfly/server/item"
	dfworld "github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sandertv/gophertunnel/minecraft/text"

	bedrockworld "GoCraft/bedrock/world"
	"GoCraft/config"
	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/internal/debuglog"
	"GoCraft/java/handler"
	"github.com/GoCraft-MC/gocraft-abi/command"
)

const bedrockChunkRadius int32 = 4

// Listener wraps a gophertunnel minecraft.Listener and manages Bedrock client
// connections.
type Listener struct {
	cfg       config.BedrockConfig
	bus       *intent.Bus
	world     *coreworld.World
	worlds    map[int32]*coreworld.World
	game      *game.Game
	encoder   *bedrockworld.Encoder
	worldSeed int64
	spawnX    int
	spawnY    int
	spawnZ    int
	spawnMu   sync.RWMutex

	// commandTree reports what one player may use, built-ins and plugins in one
	// tree. Nil until the server installs it, which is what keeps a listener
	// built in a test from needing a command registry.
	commandMu   sync.RWMutex
	commandTree func(*player.Player) command.Root
	gameMode    atomic.Uint32
	difficulty  int32
	weather     atomic.Uint32
	sessionsMu  sync.RWMutex
	sessions    map[[16]byte]*bedrockSession
	screenID    atomic.Uint32

	// spawnNotify maps a client remote address to a channel that is closed/sent
	// when gophertunnel sends PlayStatus(PlayerSpawn) for that connection.
	// The chunk-streaming goroutine waits on this channel instead of sleeping.
	spawnNotifyMu sync.Mutex
	spawnNotify   map[string]chan struct{}

	// Creative inventory catalogue — built once in NewListener.
	creativeGroups []protocol.CreativeGroup
	creativeItems  []protocol.CreativeItem
	craftingData   []*packet.CraftingData
	creativeNames  map[uint32]creativeKnownItem // creative network ID → item name/meta

	// resourcePacks holds optional Bedrock-format packs sent to every client.
	resourcePacks []*resource.Pack

	// customItemEntries are component-based item definitions appended to the
	// vanilla item registry in every StartGame packet.
	customItemEntries []protocol.ItemEntry
}

type bedrockSession struct {
	conn                *minecraft.Conn
	uuid                [16]byte
	entityID            int32
	displayName         string
	xuid                string
	buildPlatform       int32
	skin                protocol.Skin
	listedPlayers       map[[16]byte]struct{}
	knownPlayers        map[[16]byte]bedrockPlayerView
	knownEntities       map[int32]bedrockEntityView
	lastHealth          float32
	lastFood            int32
	lastSaturation      float32
	lastExhaustion      float32
	hungerSent          bool
	lastExperienceLevel int32
	lastExperience      float32
	experienceSent      bool
	wasDead             bool
	inventorySent       bool
	lastInventory       [player.InventorySize]player.ItemStack
	lastHeldSlot        int
	abilitiesSent       bool
	lastGameMode        player.GameMode
	lastAllowFly        bool
	lastFlying          bool
	lastFlySpeed        float32
	lastWalkSpeed       float32
	lastOperator        bool
	lastGodMode         bool
	teleportMu          sync.Mutex
	teleportPos         *spatial.Vec3
	stackMu             sync.Mutex
	stackNetworkIDs     [player.InventorySize]int32
	craftingNetworkIDs  [10]int32
	furnaceNetworkIDs   [3]int32
	containerNetworkIDs [54]int32
	lastFurnaceSlots    [3]player.ItemStack
	lastFurnaceData     [4]int32
	lastFurnaceKind     string
	furnaceSent         bool
	cursorStackID       int32
	nextStackNetworkID  int32
	lastCarriedItem     player.ItemStack
	lastHeldItem        player.ItemStack
	clientHeldSlot      int
	clientHeldSlotSeen  bool
	invOpened           bool // true while the player's own inventory/creative screen is open
	breakingPos         protocol.BlockPos
	breaking            bool
	lastBlockUsePos     protocol.BlockPos
	lastBlockUseFace    int32
	lastBlockUseSlot    int32
	lastBlockUseAt      time.Time
	dimension           atomic.Int32
}

func (s *bedrockSession) expectTeleport(position spatial.Vec3) {
	s.teleportMu.Lock()
	s.teleportPos = &position
	s.teleportMu.Unlock()
}

func (s *bedrockSession) acceptMovement(position spatial.Vec3) bool {
	s.teleportMu.Lock()
	defer s.teleportMu.Unlock()
	if s.teleportPos == nil {
		return true
	}
	if position.Distance(*s.teleportPos) > 1 {
		return false
	}
	s.teleportPos = nil
	return true
}

func (l *Listener) addSession(s *bedrockSession) {
	// Publish the complete roster before the tick loop can expose this session.
	// This matches Dragonfly's join ordering and makes the pause-menu Social
	// list available immediately, including for a player who joins between ticks.
	players := make([]*player.Player, 0, l.game.OnlineCount())
	l.game.OnlinePlayers(func(p *player.Player) { players = append(players, p) })
	l.sessionsMu.RLock()
	bedrockByUUID := make(map[[16]byte]*bedrockSession, len(l.sessions)+1)
	for id, current := range l.sessions {
		bedrockByUUID[id] = current
	}
	l.sessionsMu.RUnlock()
	bedrockByUUID[s.uuid] = s
	l.syncPlayerList(s, players, bedrockByUUID)

	l.sessionsMu.Lock()
	l.sessions[s.uuid] = s
	l.sessionsMu.Unlock()
	l.sendWeather(s, l.weather.Load() >= 1, l.weather.Load() >= 2)
	debuglog.Info(debuglog.BedrockLogin, "bedrock: session added", "displayName", s.displayName)
}

func (l *Listener) removeSession(uuid [16]byte) {
	l.sessionsMu.Lock()
	delete(l.sessions, uuid)
	l.sessionsMu.Unlock()
}

// DisconnectPlayer closes a Bedrock session with a client-visible reason.
func (l *Listener) DisconnectPlayer(uuid [16]byte, reason string) bool {
	l.sessionsMu.RLock()
	s, ok := l.sessions[uuid]
	l.sessionsMu.RUnlock()
	if !ok {
		return false
	}
	_ = s.conn.WritePacket(&packet.Disconnect{
		Reason: packet.DisconnectReasonKicked, Message: reason,
	})
	_ = s.conn.Close()
	return true
}

// NewListener creates a Listener from the Bedrock section of the server config.
// The intent bus is used to submit player lifecycle and gameplay events to the
// core simulation tick goroutine.
func NewListener(
	cfg config.BedrockConfig,
	bus *intent.Bus,
	world *coreworld.World,
	netherWorld *coreworld.World,
	endWorld *coreworld.World,
	game *game.Game,
	worldSeed int64,
	spawnX, spawnZ int,
	gameMode player.GameMode,
	difficulty int32,
) *Listener {
	l := &Listener{
		cfg:         cfg,
		bus:         bus,
		world:       world,
		worlds:      map[int32]*coreworld.World{0: world, 1: netherWorld, 2: endWorld},
		game:        game,
		encoder:     bedrockworld.NewEncoder(),
		worldSeed:   worldSeed,
		spawnX:      spawnX,
		spawnZ:      spawnZ,
		difficulty:  difficulty,
		sessions:    make(map[[16]byte]*bedrockSession),
		spawnNotify: make(map[string]chan struct{}),
	}
	l.gameMode.Store(uint32(gameMode))
	l.initCreativeContent()
	l.craftingData = bedrockCraftingData()
	shapedRecipes, shapelessRecipes := 0, 0
	for _, data := range l.craftingData {
		shapedRecipes += len(data.ShapedRecipes)
		shapelessRecipes += len(data.ShapelessRecipes)
	}
	debuglog.Info(debuglog.BedrockCatalogues,
		"bedrock: crafting catalogue ready",
		"java_version", bedrockRecipeCompatibilityVersion,
		"packets", len(l.craftingData),
		"shaped", shapedRecipes,
		"shapeless", shapelessRecipes,
		"total", shapedRecipes+shapelessRecipes,
	)
	return l
}

// SetResourcePack adds a single Bedrock-format resource pack that is sent to
// every connecting Bedrock client. Call this before Listen.
func (l *Listener) SetResourcePack(pack *resource.Pack) {
	l.resourcePacks = append(l.resourcePacks, pack)
}

// SetResourcePacks registers multiple Bedrock-format packs at once. This is
// equivalent to calling SetResourcePack for each element. Call before Listen.
func (l *Listener) SetResourcePacks(packs []*resource.Pack) {
	l.resourcePacks = append(l.resourcePacks, packs...)
}

// SetCustomItemEntries registers component-based custom item entries that are
// appended to the vanilla item table in every StartGame packet. Call before Listen.
func (l *Listener) SetCustomItemEntries(entries []protocol.ItemEntry) {
	l.customItemEntries = entries
}

func (l *Listener) worldForDimension(dimension int32) *coreworld.World {
	if dimensionWorld := l.worlds[dimension]; dimensionWorld != nil {
		return dimensionWorld
	}
	return l.world
}

// Listen starts the RakNet UDP listener and blocks until ctx is cancelled or a
// fatal error occurs.  Each accepted connection is handled in its own goroutine.
func (l *Listener) Listen(ctx context.Context) error {
	if !l.cfg.OnlineMode {
		slog.Warn("⚠ BEDROCK AUTHENTICATION DISABLED — server is running in offline mode",
			"risk", "unauthenticated XUIDs must NOT be treated as trusted global identities",
			"address", l.cfg.Address,
		)
	}

	gt, err := minecraft.ListenConfig{
		AuthenticationDisabled: !l.cfg.OnlineMode,
		ErrorLog:               slog.Default(),
		ResourcePacks:          l.resourcePacks,
		// AllowUnknownPackets prevents gophertunnel from closing the conn when
		// the client sends a packet whose ID we do not recognise. Without this,
		// any novel 1.26.40 packet ID causes an immediate server-side close and
		// our ReadPacket returns "context canceled" — masking the real packet ID.
		AllowUnknownPackets: true,
		PacketFunc: func(h packet.Header, payload []byte, src, dst net.Addr) {
			// PacketID 0x02 = PlayStatus; status value 3 = PlayerSpawn.
			// This fires on outbound writes (src=server, dst=client).
			// Signal the chunk-streaming goroutine so it sends chunks
			// immediately after the client enters its loading screen.
			if h.PacketID == 0x02 && len(payload) >= 4 &&
				binary.BigEndian.Uint32(payload[:4]) == 3 {
				l.spawnNotifyMu.Lock()
				ch := l.spawnNotify[dst.String()]
				l.spawnNotifyMu.Unlock()
				if ch != nil {
					select {
					case ch <- struct{}{}:
					default:
					}
				}
			}
		},
	}.Listen("raknet", l.cfg.Address)
	if err != nil {
		return fmt.Errorf("bedrock: starting RakNet listener on %s: %w", l.cfg.Address, err)
	}

	slog.Info("bedrock listener started",
		"address", l.cfg.Address,
		"onlineMode", l.cfg.OnlineMode,
	)

	// Close the gophertunnel listener when the server context is cancelled.
	go func() {
		<-ctx.Done()
		_ = gt.Close()
	}()

	for {
		conn, err := gt.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil // clean shutdown
			}
			slog.Error("bedrock: Accept error", "err", err)
			return fmt.Errorf("bedrock: Accept: %w", err)
		}
		go l.handleConn(ctx, gt, conn.(*minecraft.Conn))
	}
}

// handleConn runs in its own goroutine for every accepted Bedrock connection.
//
// M14.1 flow:
//  1. gophertunnel completes the RakNet + login sequence
//  2. Post JoinIntent, wait for JoinResult from the simulation tick (≤10 s)
//  3. Call conn.StartGame with the assigned entity ID and spawn position
//  4. Send initial LevelChunk packets for the chunk view radius
//  5. Enter the play loop: route packets to intents, handle SubChunkRequests
//  6. On disconnect, post DisconnectIntent
func (l *Listener) handleConn(ctx context.Context, gt *minecraft.Listener, conn *minecraft.Conn) {
	remote := conn.RemoteAddr()

	// ── Step 1: resolve player identity ──────────────────────────────────────
	identity := conn.IdentityData()
	authenticated := conn.Authenticated()

	if !authenticated && l.cfg.OnlineMode {
		// Defensive: gophertunnel enforces auth when AuthenticationDisabled=false.
		slog.Warn("bedrock: unauthenticated connection despite online_mode=true; dropping",
			"remote", remote, "displayName", identity.DisplayName)
		_ = gt.Disconnect(conn, text.Colourf("<red>Authentication required.</red>"))
		return
	}

	// Derive a stable UUID for the session.
	// Online mode: parse identity.Identity (Xbox-issued UUID, trusted).
	// Offline mode: generate a deterministic offline UUID from the display
	//               name so it never collides with an Xbox UUID.
	playerUUID, err := resolveUUID(identity.Identity, identity.DisplayName, authenticated)
	if err != nil {
		slog.Warn("bedrock: could not parse player UUID; dropping",
			"remote", remote, "identity", identity.Identity, "err", err)
		_ = gt.Disconnect(conn, text.Colourf("<red>Internal server error.</red>"))
		return
	}

	slog.Info("bedrock: player connecting",
		"remote", remote,
		"displayName", identity.DisplayName,
		"uuid", playerUUID,
		"xuid", xuidLog(identity.XUID, authenticated),
		"authenticated", authenticated,
	)

	// ── Step 2: request world entry via the simulation ────────────────────────
	done := make(chan intent.JoinResult, 1)
	joinCtx, joinCancel := context.WithTimeout(ctx, 10*time.Second)
	defer joinCancel()

	if err := l.bus.PostJoin(joinCtx, intent.JoinIntent{
		PlayerUUID:      playerUUID,
		Username:        identity.DisplayName,
		Edition:         "bedrock",
		RemoteAddress:   remote.String(),
		TrustedIdentity: authenticated,
		Done:            done,
	}); err != nil {
		// ctx cancelled (server shutting down) or 10 s posting timeout.
		slog.Warn("bedrock: PostJoin failed; dropping connection",
			"remote", remote, "displayName", identity.DisplayName, "err", err)
		_ = gt.Disconnect(conn, text.Colourf("<yellow>Server timed out. Please reconnect.</yellow>"))
		return
	}

	var result intent.JoinResult
	select {
	case result = <-done:
		// Join was processed by the tick goroutine.
	case <-time.After(10 * time.Second):
		// The intent was queued but the tick goroutine did not respond in time.
		// Post a DisconnectIntent so the tick cleans up the player if it was
		// already added (lifecycle channel is FIFO, so disconnect follows join).
		slog.Warn("bedrock: JoinResult timed out; posting cleanup disconnect",
			"remote", remote, "displayName", identity.DisplayName)
		_ = l.bus.PostDisconnect(ctx, intent.DisconnectIntent{
			PlayerUUID: playerUUID,
			Reason:     "join response timeout",
		})
		_ = gt.Disconnect(conn, text.Colourf("<yellow>Server timed out. Please reconnect.</yellow>"))
		return
	case <-ctx.Done():
		return
	}
	if result.Err != nil {
		slog.Warn("bedrock: join rejected by simulation",
			"remote", remote, "displayName", identity.DisplayName, "err", result.Err)
		_ = gt.Disconnect(conn, text.Colourf("<red>Could not join: %v</red>", result.Err))
		return
	}

	defer func() {
		_ = l.bus.PostDisconnect(ctx, intent.DisconnectIntent{
			PlayerUUID: playerUUID,
			Reason:     "connection closed",
		})
		slog.Info("bedrock: player disconnected",
			"displayName", identity.DisplayName, "uuid", playerUUID)
	}()

	// ── Step 3: prepare session state ────────────────────────────────────────
	// All fields come from result, which is already resolved.
	const chunkRadius = bedrockChunkRadius
	spawnPos := playerNetworkPosition(result.Position)
	spawnCX := chunkCoordinate(result.Position.X)
	spawnCZ := chunkCoordinate(result.Position.Z)

	bedrockSess := &bedrockSession{
		conn:               conn,
		uuid:               playerUUID,
		entityID:           result.EntityID,
		displayName:        identity.DisplayName,
		xuid:               identity.XUID,
		buildPlatform:      int32(conn.ClientData().DeviceOS),
		skin:               skinFromClientData(conn.ClientData()),
		listedPlayers:      make(map[[16]byte]struct{}),
		knownPlayers:       make(map[[16]byte]bedrockPlayerView),
		knownEntities:      make(map[int32]bedrockEntityView),
		lastHealth:         -1,
		nextStackNetworkID: 1,
	}
	bedrockSess.dimension.Store(result.Dimension)
	defer func() {
		l.removeSession(playerUUID)
		debuglog.Info(debuglog.BedrockLogin, "bedrock: session removed", "displayName", identity.DisplayName)
	}()

	// ── Step 4: stream chunks concurrently with StartGame ─────────────────────
	//
	// conn.StartGame() sends the initial protocol handshake (StartGame packet,
	// ItemRegistry, ChunkRadiusUpdated, PlayStatus=PlayerSpawn) and then BLOCKS
	// until the Bedrock client sends SetLocalPlayerAsInitialised.  The client
	// only sends that packet after it finishes loading the world and exits the
	// loading screen.
	//
	// If we wait for StartGame() to return before sending NCPU and LevelChunks,
	// the client gets no chunk data while in the loading screen. After ~2 seconds
	// it times out, exits the loading screen prematurely (sending
	// SetLocalPlayerAsInitialised to unblock us), and is then in a broken state
	// where it no longer sends SubChunkRequests for the LevelChunks it receives
	// too late.
	//
	// Fix: send UpdateAttributes + NCPU + LevelChunks from a goroutine while
	// StartGame() is blocked. We wait for PlayStatus(PlayerSpawn) to be written
	// to the wire (detected via PacketFunc) before sending our own packets,
	// so the client is guaranteed to be inside the loading screen when chunks arrive.

	// Register the spawn-notify channel BEFORE launching the goroutine so that
	// the PacketFunc can signal it the moment PlayStatus(PlayerSpawn) is written.
	spawnReady := make(chan struct{}, 1)
	remoteAddr := conn.RemoteAddr().String()
	l.spawnNotifyMu.Lock()
	l.spawnNotify[remoteAddr] = spawnReady
	l.spawnNotifyMu.Unlock()
	defer func() {
		l.spawnNotifyMu.Lock()
		delete(l.spawnNotify, remoteAddr)
		l.spawnNotifyMu.Unlock()
	}()

	chunkStreamErr := make(chan error, 1)
	go func() {
		// Wait for gophertunnel to write PlayStatus(PlayerSpawn) before we
		// write our own packets. The PacketFunc signals spawnReady the instant
		// that packet hits the wire, so this has zero unnecessary delay.
		select {
		case <-spawnReady:
			debuglog.Info(debuglog.BedrockLogin, "bedrock: spawn goroutine: PlayStatus(PlayerSpawn) confirmed, sending chunks",
				"displayName", identity.DisplayName)
		case <-time.After(10 * time.Second):
			chunkStreamErr <- fmt.Errorf("timed out waiting for PlayStatus(PlayerSpawn)")
			return
		}

		// UpdateAttributes: health → XP → food (Dragonfly Spawn() order).
		if p := l.game.GetPlayer(playerUUID); p != nil {
			health, food, saturation, _ := p.HealthSnapshot()
			_, _, exhaustion := p.HungerSnapshot()
			experienceLevel, _, experienceProgress := p.ExperienceSnapshot()
			maxHealth := p.MaxHealth
			if maxHealth <= 0 {
				maxHealth = 20
			}
			initialHealth := float32(math.Ceil(float64(health)))
			initialMaxHealth := float32(math.Ceil(float64(maxHealth)))

			healthPk := &packet.UpdateAttributes{
				EntityRuntimeID: bedrockSelfRuntimeID,
				Attributes: []protocol.Attribute{{
					AttributeValue: protocol.AttributeValue{
						Name: "minecraft:health", Value: initialHealth, Max: initialMaxHealth,
					},
					DefaultMax: 20, Default: 20,
				}, {
					AttributeValue: protocol.AttributeValue{
						Name: "minecraft:absorption", Value: 0, Max: math.MaxFloat32,
					},
					DefaultMax: math.MaxFloat32,
				}},
			}
			debuglog.Info(debuglog.BedrockLogin, "bedrock: spawn goroutine: UpdateAttributes health", "health", initialHealth)
			if err := conn.WritePacket(healthPk); err != nil {
				chunkStreamErr <- err
				return
			}

			xpPk := &packet.UpdateAttributes{
				EntityRuntimeID: bedrockSelfRuntimeID,
				Attributes: []protocol.Attribute{{
					AttributeValue: protocol.AttributeValue{
						Name: "minecraft:player.level", Value: float32(experienceLevel), Max: math.MaxInt32,
					},
					DefaultMax: math.MaxInt32,
				}, {
					AttributeValue: protocol.AttributeValue{
						Name: "minecraft:player.experience", Value: experienceProgress, Max: 1,
					},
					DefaultMax: 1,
				}},
			}
			if err := conn.WritePacket(xpPk); err != nil {
				chunkStreamErr <- err
				return
			}
			bedrockSess.lastExperienceLevel = experienceLevel
			bedrockSess.lastExperience = experienceProgress
			bedrockSess.experienceSent = true

			foodPk := &packet.UpdateAttributes{
				EntityRuntimeID: bedrockSelfRuntimeID,
				Attributes: []protocol.Attribute{{
					AttributeValue: protocol.AttributeValue{
						Name: "minecraft:player.hunger", Value: float32(food), Max: 20,
					},
					DefaultMax: 20, Default: 20,
				}, {
					AttributeValue: protocol.AttributeValue{
						Name: "minecraft:player.saturation", Value: saturation, Max: 20,
					},
					DefaultMax: 20, Default: 20,
				}, {
					AttributeValue: protocol.AttributeValue{
						Name: "minecraft:player.exhaustion", Value: exhaustion, Max: 5,
					},
					DefaultMax: 5,
				}},
			}
			if err := conn.WritePacket(foodPk); err != nil {
				chunkStreamErr <- err
				return
			}

			// Suppress duplicate health update on the first Sync() tick.
			bedrockSess.lastHealth = health
			bedrockSess.lastFood = food
			bedrockSess.lastSaturation = saturation
			bedrockSess.lastExhaustion = exhaustion
			bedrockSess.hungerSent = true
		}

		// NCPU — client ignores LevelChunks received before this.
		debuglog.Info(debuglog.BedrockLogin, "bedrock: spawn goroutine: sending NCPU")
		if err := conn.WritePacket(initialChunkPublisher(result.Position, chunkRadius)); err != nil {
			chunkStreamErr <- err
			return
		}

		// AvailableActorIdentifiers (Dragonfly sends this right after NCPU).
		debuglog.Info(debuglog.BedrockLogin, "bedrock: spawn goroutine: sending AvailableActorIdentifiers")
		if err := conn.WritePacket(&packet.AvailableActorIdentifiers{
			SerialisedEntityIdentifiers: availableActorIdentifiersPayload,
		}); err != nil {
			chunkStreamErr <- err
			return
		}

		// 81 LevelChunks (biome stubs; block data arrives via SubChunkRequest).
		debuglog.Info(debuglog.BedrockChunks, "bedrock: spawn goroutine: sending LevelChunks", "displayName", identity.DisplayName)
		if err := l.sendInitialChunks(conn, spawnCX, spawnCZ, chunkRadius, result.Dimension); err != nil {
			chunkStreamErr <- err
			return
		}
		debuglog.Info(debuglog.BedrockChunks, "bedrock: spawn goroutine: LevelChunks done", "displayName", identity.DisplayName)

		chunkStreamErr <- nil
	}()

	// conn.StartGame() blocks here until the client sends SetLocalPlayerAsInitialised.
	// The goroutine above provides the chunk data the client needs to do so.
	l.spawnMu.RLock()
	worldSpawn := protocol.BlockPos{int32(l.spawnX), int32(l.spawnY), int32(l.spawnZ)}
	l.spawnMu.RUnlock()
	if err := conn.StartGame(minecraft.GameData{
		WorldName:         "GoCraft",
		EntityUniqueID:    int64(bedrockSelfRuntimeID),
		EntityRuntimeID:   bedrockSelfRuntimeID,
		PlayerPosition:    spawnPos,
		PlayerGameMode:    bedrockGameType(player.GameMode(l.gameMode.Load())),
		WorldGameMode:     bedrockGameType(player.GameMode(l.gameMode.Load())),
		Difficulty:        l.difficulty,
		PlayerPermissions: packet.PermissionLevelMember,
		PlayerMovementSettings: protocol.PlayerMovementSettings{
			ServerAuthoritativeBlockBreaking: true,
		},
		ServerAuthoritativeInventory: true,
		// The creative catalogue and every normal inventory stack reference this
		// exact Pumpkin/BDS 1.26.40 runtime table. Omitting it leaves the client
		// indexing unknown IDs when Creative search or scrolling is opened and
		// drops data-driven behaviour such as consumable item components.
		Items:           append(bedrockItemRegistry(), l.customItemEntries...),
		BaseGameVersion: protocol.CurrentVersion,
		WorldSeed:       l.worldSeed,
		Dimension:       result.Dimension,
		WorldSpawn:      worldSpawn,
		ChunkRadius:     bedrockChunkRadius,
		// Network block hashes are stable across Bedrock palette revisions. The
		// Dragonfly registry and the 1.26.40 protocol fork do not necessarily use
		// the same sequential runtime IDs, so palette indices corrupt terrain.
		UseBlockNetworkIDHashes: true,
	}); err != nil {
		slog.Debug("bedrock: StartGame failed",
			"remote", remote, "displayName", identity.DisplayName, "err", err)
		// Drain the channel so the goroutine can always write without blocking.
		// The goroutine will terminate on its own when WritePacket returns an
		// error on the closed connection.
		return
	}

	// Wait for the chunk-streaming goroutine to finish (usually already done
	// by the time StartGame() returns, since the client waits for chunks before
	// sending SetLocalPlayerAsInitialised).
	if err := <-chunkStreamErr; err != nil {
		slog.Debug("bedrock: initial chunk stream failed",
			"displayName", identity.DisplayName, "err", err)
		return
	}
	debuglog.Info(debuglog.BedrockLogin, "bedrock: StartGame complete, chunk stream done", "displayName", identity.DisplayName)

	// The generic connection login sends an empty CreativeContent packet.
	// Replace it with the actual catalogue before the player can open the menu.
	if err := conn.WritePacket(&packet.CreativeContent{Groups: l.creativeGroups, Items: l.creativeItems}); err != nil {
		slog.Debug("bedrock: creative content send failed", "displayName", identity.DisplayName, "err", err)
		return
	}
	for _, craftingData := range l.craftingData {
		if err := conn.WritePacket(craftingData); err != nil {
			slog.Debug("bedrock: crafting data send failed", "displayName", identity.DisplayName, "err", err)
			return
		}
	}

	// Send local player state (SetPlayerGameType, UpdateAbilities, UpdateAttributes
	// for movement speed, SetActorData) after chunks — matching Dragonfly ordering.
	if p := l.game.GetPlayer(playerUUID); p != nil {
		l.sendLocalPlayerState(bedrockSess, p)
	}

	// What this player may type. Sent once, because AvailableCommands replaces
	// the client's whole list and there is no way for it to ask again — a later
	// change reaches them through RefreshCommands.
	//
	// Sent to this connection rather than looked up by uuid: addSession below
	// is what publishes the roster, so a lookup here finds nothing and the
	// player joins with no commands at all.
	l.sendCommandsTo(bedrockSess.conn, playerUUID)

	// ── Step 5: play loop ─────────────────────────────────────────────────────
	l.addSession(bedrockSess)
	l.playLoop(ctx, conn, bedrockSess, spawnCX, spawnCZ, chunkRadius)
}

// sendInitialChunks sends full LevelChunk packets (SubChunkCount=24, all block
// data included) for a square of chunks around the spawn position.  Sending
// complete block data avoids the SubChunkRequest/Response round-trip that would
// deadlock during conn.StartGame().
func (l *Listener) sendInitialChunks(conn *minecraft.Conn, cx, cz, radius, dimension int32) error {
	return l.sendChunkSquare(conn, cx, cz, radius, dimension, false)
}

func (l *Listener) sendSurroundingChunks(conn *minecraft.Conn, cx, cz, radius, dimension int32) error {
	return l.sendChunkSquare(conn, cx, cz, radius, dimension, true)
}

func (l *Listener) sendChunkSquare(conn *minecraft.Conn, cx, cz, radius, dimension int32, skipCenter bool) error {
	dimensionWorld := l.worldForDimension(dimension)
	first := true
	for ring := int32(0); ring <= radius; ring++ {
		for dx := -ring; dx <= ring; dx++ {
			for dz := -ring; dz <= ring; dz++ {
				if (ring > 0 && bedrockAbs32(dx) != ring && bedrockAbs32(dz) != ring) || (skipCenter && dx == 0 && dz == 0) {
					continue
				}
				chunkX, chunkZ := cx+dx, cz+dz
				chunk := dimensionWorld.Chunk(chunkX, chunkZ)
				payload, err := l.encoder.EncodeFullChunkPayload(chunk)
				if err != nil {
					return fmt.Errorf("sendInitialChunks encode (%d,%d): %w", chunkX, chunkZ, err)
				}
				pk := &packet.LevelChunk{
					Position:      protocol.ChunkPos{chunkX, chunkZ},
					Dimension:     dimension,
					SubChunkCount: uint32(coreworld.SectionCount),
					CacheEnabled:  false,
					RawPayload:    payload,
				}
				if first {
					first = false
					debuglog.Info(debuglog.BedrockChunks, "bedrock: LevelChunk sample (full)",
						"chunkX", pk.Position[0],
						"chunkZ", pk.Position[1],
						"subChunkCount", pk.SubChunkCount,
						"payloadLen", len(pk.RawPayload),
					)
				}
				if err := conn.WritePacket(pk); err != nil {
					return fmt.Errorf("sendInitialChunks: %w", err)
				}
			}
		}
	}
	return nil
}

func bedrockAbs32(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

// sendEnteredChunks announces only columns newly covered by a moved view
// window. The client unloads columns outside the publisher radius itself.
func (l *Listener) sendEnteredChunks(conn *minecraft.Conn, oldCX, oldCZ, newCX, newCZ, radius, dimension int32) error {
	dimensionWorld := l.worldForDimension(dimension)
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			cx, cz := newCX+dx, newCZ+dz
			if chunkInsideWindow(cx, cz, oldCX, oldCZ, radius) {
				continue
			}
			chunk := dimensionWorld.Chunk(cx, cz)
			payload, err := l.encoder.EncodeFullChunkPayload(chunk)
			if err != nil {
				return fmt.Errorf("sendEnteredChunks encode (%d,%d): %w", cx, cz, err)
			}
			if err := conn.WritePacket(&packet.LevelChunk{
				Position:      protocol.ChunkPos{cx, cz},
				Dimension:     dimension,
				SubChunkCount: uint32(coreworld.SectionCount),
				CacheEnabled:  false,
				RawPayload:    payload,
			}); err != nil {
				return fmt.Errorf("sendEnteredChunks: %w", err)
			}
		}
	}
	return nil
}

func (l *Listener) updateChunkStream(conn *minecraft.Conn, position spatial.Vec3, cx, cz, streamDimension *int32, radius, dimension int32) error {
	newCX, newCZ := chunkCoordinate(position.X), chunkCoordinate(position.Z)
	if newCX == *cx && newCZ == *cz && dimension == *streamDimension {
		// Player has not crossed a chunk boundary — nothing to do.
		// Dragonfly's sendChunks() also skips NCPU when lastChunkPos == chunkPos.
		return nil
	}
	if err := conn.WritePacket(initialChunkPublisher(position, radius)); err != nil {
		return fmt.Errorf("publisher update: %w", err)
	}
	if dimension != *streamDimension {
		if err := l.sendInitialChunks(conn, newCX, newCZ, radius, dimension); err != nil {
			return err
		}
	} else if err := l.sendEnteredChunks(conn, *cx, *cz, newCX, newCZ, radius, dimension); err != nil {
		return err
	}
	*cx, *cz = newCX, newCZ
	*streamDimension = dimension
	return nil
}

// playLoop reads packets from a connected Bedrock client and routes them to
// the appropriate intent or response handler.
//
// Returns when the connection closes or ctx is cancelled.
func (l *Listener) playLoop(ctx context.Context, conn *minecraft.Conn, bedrockSess *bedrockSession, streamCX, streamCZ, streamRadius int32) {
	playerUUID, displayName := bedrockSess.uuid, bedrockSess.displayName
	readyForWorldSync := false
	streamDimension := bedrockSess.dimension.Load()
	debuglog.Info(debuglog.BedrockLogin, "bedrock: playLoop entered", "displayName", displayName)

	// Close the connection when the server context is cancelled.
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			slog.Warn("bedrock: playLoop ReadPacket error",
				"displayName", displayName,
				"err", err,
				"ctxErr", ctx.Err(),
				"readyForWorldSync", readyForWorldSync,
			)
			return
		}
		if online := l.game.GetPlayer(bedrockSess.uuid); online != nil {
			online.TouchActivity()
		}

		switch p := pk.(type) {
		case *packet.SubChunkRequest:
			debuglog.Info(debuglog.BedrockChunks, "bedrock: SubChunkRequest received", "display", displayName, "dim", p.Dimension, "pos", p.Position, "offsets", len(p.Offsets))
			l.handleSubChunkRequest(conn, p)
			debuglog.Info(debuglog.BedrockChunks, "bedrock: SubChunkRequest handled", "display", displayName)
			if !readyForWorldSync {
				readyForWorldSync = true
			}

		case *packet.MovePlayer:
			position := canonicalPlayerPosition(p.Position)
			if !l.acceptBedrockMovement(bedrockSess, position) {
				continue
			}
			if readyForWorldSync {
				if err := l.updateChunkStream(conn, position, &streamCX, &streamCZ, &streamDimension, streamRadius, bedrockSess.dimension.Load()); err != nil {
					slog.Debug("bedrock: updating chunk stream failed", "displayName", displayName, "err", err)
					return
				}
			}
			l.bus.UpdateMove(intent.MoveIntent{
				PlayerUUID: playerUUID,
				Position:   position,
				Rotation:   spatial.Rotation{Yaw: p.Yaw, Pitch: p.Pitch},
				OnGround:   p.OnGround,
			})

		case *packet.PlayerAuthInput:
			position := canonicalPlayerPosition(p.Position)
			if !l.acceptBedrockMovement(bedrockSess, position) {
				continue
			}
			if readyForWorldSync {
				if err := l.updateChunkStream(conn, position, &streamCX, &streamCZ, &streamDimension, streamRadius, bedrockSess.dimension.Load()); err != nil {
					slog.Debug("bedrock: updating chunk stream failed", "displayName", displayName, "err", err)
					return
				}
			}
			l.bus.UpdateMove(intent.MoveIntent{
				PlayerUUID: playerUUID,
				Position:   position,
				Rotation:   spatial.Rotation{Yaw: p.Yaw, Pitch: p.Pitch},
				OnGround:   inputHasFlag(p.InputData, packet.InputFlagVerticalCollision),
			})
			if inputHasFlag(p.InputData, packet.InputFlagPerformBlockActions) {
				if blockActions, ok := p.BlockActions.Value(); ok {
					for _, action := range blockActions {
						l.handlePlayerBlockAction(bedrockSess, playerUUID, action.Action, action.BlockPos, action.Face)
					}
				}
			}
			if inputHasFlag(p.InputData, packet.InputFlagPerformItemInteraction) {
				if itemData, ok := p.ItemInteractionData.Value(); ok {
					l.handleUseItemTransaction(bedrockSess, playerUUID, &itemData)
				}
			}
			if inputHasFlag(p.InputData, packet.InputFlagPerformItemStackRequest) {
				if sr, ok := p.ItemStackRequest.Value(); ok {
					l.handleStackRequests(ctx, conn, playerUUID, []protocol.ItemStackRequest{sr})
				}
			}
			l.postInputState(playerUUID, p.InputData)
			if inputHasFlag(p.InputData, packet.InputFlagStartUsingItem) {
				l.bus.PostStartUseItem(intent.StartUseItemIntent{PlayerUUID: playerUUID, HotbarSlot: -1})
			}

		case *packet.PlayerAction:
			l.handlePlayerBlockAction(bedrockSess, playerUUID, p.ActionType, p.BlockPosition, p.BlockFace)

		case *packet.Animate:
			if p.ActionType == packet.AnimateActionSwingArm && p.EntityRuntimeID == bedrockSelfRuntimeID {
				l.bus.PostArmSwing(intent.ArmSwingIntent{PlayerUUID: playerUUID, Hand: 0})
			}

		case *packet.RequestAbility:
			if p.Ability == packet.AbilityFlying {
				if enabled, ok := p.Value.(bool); ok {
					l.bus.PostPlayerState(intent.PlayerStateIntent{
						PlayerUUID: playerUUID,
						State:      intent.PlayerStateFlying,
						Enabled:    enabled,
					})
				}
			}

		case *packet.Respawn:
			// Modern Bedrock completes the death-screen handshake with a
			// Respawn packet, not PlayerActionRespawn.
			if p.State == packet.RespawnStateClientReadyToSpawn &&
				p.EntityRuntimeID == bedrockSelfRuntimeID {
				l.bus.PostRespawn(intent.RespawnIntent{PlayerUUID: playerUUID})
			}

		case *packet.InventoryTransaction:
			switch data := p.TransactionData.(type) {
			case *protocol.NormalTransactionData:
				l.handleNormalInventoryTransaction(ctx, conn, bedrockSess, playerUUID, p)
			case *protocol.UseItemTransactionData:
				l.handleUseItemTransaction(bedrockSess, playerUUID, data)
			case *protocol.UseItemOnEntityTransactionData:
				targetID, ok := canonicalEntityID(data.TargetEntityRuntimeID)
				if !ok {
					continue
				}
				if !validActionHotbarSlot(data.HotBarSlot, "InventoryTransaction/UseItemOnEntity") {
					continue
				}
				l.bus.PostEntityInteract(intent.EntityInteractIntent{
					PlayerUUID: playerUUID,
					TargetID:   targetID,
					Attack:     data.ActionType == protocol.UseItemOnEntityActionAttack,
					HotbarSlot: data.HotBarSlot,
				})
			case *protocol.ReleaseItemTransactionData:
				if !validActionHotbarSlot(data.HotBarSlot, "InventoryTransaction/ReleaseItem") {
					continue
				}
				l.bus.PostConsumeFood(intent.ConsumeFoodIntent{PlayerUUID: playerUUID, HotbarSlot: data.HotBarSlot})
			}

		case *packet.ItemStackRequest:
			l.handleStackRequests(ctx, conn, playerUUID, p.Requests)

		case *packet.MobEquipment:
			l.acceptClientHotbarSlot(bedrockSess, playerUUID, int32(p.HotBarSlot), "MobEquipment")

		case *packet.Text:
			if strings.TrimSpace(p.Message) != "" {
				l.bus.PostChat(intent.ChatIntent{
					PlayerUUID:  playerUUID,
					DisplayName: displayName,
					Message:     p.Message,
				})
			}

		case *packet.CommandRequest:
			if commandLine := strings.TrimSpace(p.CommandLine); commandLine != `` {
				if !strings.HasPrefix(commandLine, `/`) {
					commandLine = `/` + commandLine
				}
				l.bus.PostChat(intent.ChatIntent{
					PlayerUUID:  playerUUID,
					DisplayName: displayName,
					Message:     commandLine,
				})
			}

		case *packet.Interact:
			// The client sends InteractActionOpenInventory when the player presses
			// 'E' (open inventory / creative menu).  The server must respond with
			// ContainerOpen so the client renders the correct screen.
			if p.ActionType == packet.InteractActionOpenInventory && !bedrockSess.invOpened {
				player := l.game.GetPlayer(playerUUID)
				var pos protocol.BlockPos
				if player != nil {
					pos = protocol.BlockPos{
						int32(player.Position.X),
						int32(player.Position.Y),
						int32(player.Position.Z),
					}
				}
				_ = conn.WritePacket(&packet.ContainerOpen{
					WindowID:                0,
					ContainerType:           0xff, // special value: player inventory / creative screen
					ContainerPosition:       pos,
					ContainerEntityUniqueID: -1,
				})
				bedrockSess.invOpened = true
				bedrockSess.inventorySent = false
				if player != nil {
					l.sendPersonalCraftingSlots(conn, bedrockSess, player)
				}
			}

		case *packet.ContainerClose:
			// Echo the close back so the client confirms the screen is dismissed.
			bedrockSess.invOpened = false
			bedrockSess.inventorySent = false
			l.bus.PostContainerClose(intent.ContainerCloseIntent{PlayerUUID: playerUUID, WindowID: p.WindowID})
			_ = conn.WritePacket(&packet.ContainerClose{
				WindowID:      p.WindowID,
				ContainerType: p.ContainerType,
				ServerSide:    false,
			})

		case *packet.RequestChunkRadius:
			// GoCraft currently streams a four-chunk Bedrock radius.
			_ = conn.WritePacket(&packet.ChunkRadiusUpdated{
				ChunkRadius: bedrockChunkRadius,
			})

		case *packet.ServerBoundLoadingScreen:
			screenType := "Unknown"
			switch p.Type {
			case packet.LoadingScreenTypeStart:
				screenType = "Start"
			case packet.LoadingScreenTypeEnd:
				screenType = "End"
				if !readyForWorldSync {
					readyForWorldSync = true
					debuglog.Info(debuglog.BedrockLogin, "bedrock: readyForWorldSync set true via LoadingScreenEnd", "display", displayName)
				}
			}
			id, hasID := p.LoadingScreenID.Value()
			debuglog.Info(debuglog.BedrockLogin, "bedrock: ServerBoundLoadingScreen",
				"display", displayName,
				"type", screenType,
				"typeRaw", p.Type,
				"hasLoadingScreenID", hasID,
				"loadingScreenID", id,
				"readyForWorldSync", readyForWorldSync,
			)

		default:
			slog.Debug("bedrock: unhandled packet",
				"display", displayName,
				"type", fmt.Sprintf("%T", pk),
				"readyForWorldSync", readyForWorldSync,
			)
		}
	}
}

func (l *Listener) acceptBedrockMovement(session *bedrockSession, position spatial.Vec3) bool {
	p := l.game.GetPlayer(session.uuid)
	if p == nil {
		return false
	}
	_, _, _, dead := p.HealthSnapshot()
	return !dead && session.acceptMovement(position)
}

func (l *Listener) handleStackRequests(
	ctx context.Context,
	conn *minecraft.Conn,
	playerUUID [16]byte,
	requests []protocol.ItemStackRequest,
) {
	responses := make([]protocol.ItemStackResponse, 0, len(requests))
	craftingTouched := false

	debuglog.Info(debuglog.BedrockInventory,
		"bedrock: received stack request packet",
		"requests", len(requests),
	)

	for _, request := range requests {
		debuglog.Info(debuglog.BedrockInventory,
			"bedrock: processing stack request",
			"request_id", request.RequestID,
			"actions", len(request.Actions),
		)

		for index, action := range request.Actions {
			debuglog.Info(debuglog.BedrockInventory,
				"bedrock: stack request action",
				"request_id", request.RequestID,
				"index", index,
				"type", fmt.Sprintf("%T", action),
				"value", fmt.Sprintf("%+v", action),
			)
		}

		response := protocol.ItemStackResponse{
			Status:    protocol.ItemStackResponseStatusError,
			RequestID: request.RequestID,
		}

		playerState := l.game.GetPlayer(playerUUID)
		actions, valid := l.canonicalInventoryActions(playerState, request.Actions)
		if !valid {
			slog.Warn(
				"bedrock: stack request translation failed",
				"request_id", request.RequestID,
			)
			responses = append(responses, response)
			continue
		}

		debuglog.Info(debuglog.BedrockInventory,
			"bedrock: stack request translated",
			"request_id", request.RequestID,
			"canonical_actions", len(actions),
		)
		if len(actions) == 0 {
			// Crafting prediction packets may contain only CraftRecipe/Create
			// style actions. A successful empty response is required to keep the
			// Bedrock UI from rolling the local grid back.
			response.Status = protocol.ItemStackResponseStatusOK
			responses = append(responses, response)
			continue
		}

		done := make(chan intent.InventoryResult, 1)

		posted := l.bus.PostInventory(intent.InventoryIntent{
			PlayerUUID: playerUUID,
			Actions:    actions,
			Done:       done,
		})
		if !posted {
			slog.Warn(
				"bedrock: inventory intent could not be posted",
				"request_id", request.RequestID,
			)
			responses = append(responses, response)
			continue
		}

		timer := time.NewTimer(2 * time.Second)
		accepted := false

		select {
		case result := <-done:
			accepted = result.Accepted

			debuglog.Info(debuglog.BedrockInventory,
				"bedrock: simulation inventory result",
				"request_id", request.RequestID,
				"accepted", accepted,
			)

		case <-timer.C:
			slog.Warn(
				"bedrock: inventory intent timed out",
				"request_id", request.RequestID,
			)

		case <-ctx.Done():
			slog.Warn(
				"bedrock: inventory request context cancelled",
				"request_id", request.RequestID,
			)
		}

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}

		if !accepted {
			p := l.game.GetPlayer(playerUUID)

			if p == nil {
				slog.Warn(
					"bedrock: inventory rejected and player missing",
					"request_id", request.RequestID,
				)
			} else {
				slog.Warn(
					"bedrock: inventory rejected by simulation",
					"request_id", request.RequestID,
					"player", p.Username,
					"game_mode", p.GameMode,
					"carried_item", p.CarriedItem,
				)
			}

			responses = append(responses, response)
			continue
		}

		p := l.game.GetPlayer(playerUUID)
		if p == nil {
			slog.Warn(
				"bedrock: accepted inventory request but player missing",
				"request_id", request.RequestID,
			)
			responses = append(responses, response)
			continue
		}

		session := l.sessionForPlayer(playerUUID)
		if session == nil {
			slog.Warn(
				"bedrock: accepted inventory request but session missing",
				"request_id", request.RequestID,
				"player", p.Username,
			)
			responses = append(responses, response)
			continue
		}

		l.applyStackNetworkIDChanges(session, p, request.Actions)

		containerInfo := l.stackResponseContainerInfo(
			session,
			p,
			request.Actions,
		)
		for _, container := range containerInfo {
			if container.Container.ContainerID == protocol.ContainerCraftingInput ||
				container.Container.ContainerID == protocol.ContainerCraftingOutputPreview ||
				container.Container.ContainerID == protocol.ContainerCreatedOutput {
				craftingTouched = true
				break
			}
		}

		response.Status = protocol.ItemStackResponseStatusOK
		response.ContainerInfo = containerInfo

		debuglog.Info(debuglog.BedrockInventory,
			"bedrock: inventory request accepted",
			"request_id", request.RequestID,
			"player", p.Username,
			"game_mode", p.GameMode,
			"carried_item", p.CarriedItem,
			"cursor_stack_id", session.cursorStackID,
			"containers", len(containerInfo),
		)

		responses = append(responses, response)
	}

	if len(responses) == 0 {
		return
	}

	debuglog.Info(debuglog.BedrockInventory,
		"bedrock: sending stack responses",
		"responses", len(responses),
	)

	if err := conn.WritePacket(&packet.ItemStackResponse{
		Responses: responses,
	}); err != nil {
		slog.Warn(
			"bedrock: failed to send stack response",
			"err", err,
		)
	}
	if craftingTouched {
		if p := l.game.GetPlayer(playerUUID); p != nil {
			if session := l.sessionForPlayer(playerUUID); session != nil {
				l.sendPersonalCraftingSlots(conn, session, p)
			}
		}
	}
}

// handleNormalInventoryTransaction handles the legacy transaction path that
// Bedrock still uses for Q/Ctrl+Q drops, even with server-authoritative
// inventories enabled. The client predicts the slot change, so always send the
// resulting authoritative slot back after the simulation accepts or rejects it.
func (l *Listener) handleNormalInventoryTransaction(
	ctx context.Context,
	conn *minecraft.Conn,
	session *bedrockSession,
	playerUUID [16]byte,
	pk *packet.InventoryTransaction,
) {
	p := l.game.GetPlayer(playerUUID)
	actions, slots, valid := canonicalNormalDropActions(p, pk)
	accepted := false
	if valid {
		done := make(chan intent.InventoryResult, 1)
		if l.bus.PostInventory(intent.InventoryIntent{
			PlayerUUID: playerUUID,
			Actions:    actions,
			Done:       done,
		}) {
			timer := time.NewTimer(2 * time.Second)
			select {
			case result := <-done:
				accepted = result.Accepted
			case <-timer.C:
				slog.Warn("bedrock: normal inventory drop timed out", "player", playerUUID)
			case <-ctx.Done():
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}
	} else {
		slog.Debug("bedrock: rejected unsupported normal inventory transaction",
			"player", playerUUID, "actions", len(pk.Actions), "legacy_request_id", pk.LegacyRequestID)
	}

	// LegacyRequestID is the old request/response bridge retained specifically
	// for transactions such as hotbar drops. A bare success response is enough;
	// the authoritative InventorySlot packets below carry the final contents.
	if pk.LegacyRequestID != 0 && conn != nil {
		status := uint8(protocol.ItemStackResponseStatusError)
		if accepted {
			status = protocol.ItemStackResponseStatusOK
		}
		_ = conn.WritePacket(&packet.ItemStackResponse{Responses: []protocol.ItemStackResponse{{
			Status: status, RequestID: pk.LegacyRequestID,
		}}})
	}

	// On rejection the player's canonical inventory is unchanged, but the
	// client has already removed the item locally. Sending the slots in both
	// cases also makes successful Q drops visible immediately instead of waiting
	// for the next world sync tick.
	if p = l.game.GetPlayer(playerUUID); p != nil {
		l.sendNormalDropSlots(conn, session, p, slots)
	}
}

// canonicalNormalDropActions translates both Bedrock Q-drop encodings into
// canonical, server-authoritative inventory actions. It deliberately ignores
// the client-provided item as the server drops the item actually present in the
// source slot, but verifies that the two modern actions are balanced.
func canonicalNormalDropActions(p *player.Player, pk *packet.InventoryTransaction) ([]intent.InventoryAction, []byte, bool) {
	if p == nil || pk == nil {
		return nil, nil, false
	}

	if len(pk.Actions) == 0 {
		// Pumpkin also accepts this compatibility form: A non-zero legacy
		// request containing the affected inventory slots represents dropping
		// the complete stack from each slot.
		if pk.LegacyRequestID == 0 || len(pk.LegacySetItemSlots) == 0 || len(pk.LegacySetItemSlots) > 2 {
			return nil, nil, false
		}
		actions := make([]intent.InventoryAction, 0, len(pk.LegacySetItemSlots))
		slots := make([]byte, 0, len(pk.LegacySetItemSlots))
		seen := make(map[byte]struct{})
		for _, legacy := range pk.LegacySetItemSlots {
			if legacy.ContainerID != protocol.ContainerHotBar &&
				legacy.ContainerID != protocol.ContainerInventory &&
				legacy.ContainerID != protocol.ContainerCombinedHotBarAndInventory {
				return nil, nil, false
			}
			if len(legacy.Slots) == 0 || len(legacy.Slots) > 2 {
				return nil, nil, false
			}
			for _, slot := range legacy.Slots {
				if _, duplicate := seen[slot]; duplicate {
					continue
				}
				canonical := bedrockInventoryCanonicalSlot(int(slot))
				if canonical < 0 || p.Inventory[canonical].IsEmpty() {
					return nil, nil, false
				}
				seen[slot] = struct{}{}
				actions = append(actions, intent.InventoryAction{
					Kind: intent.InventoryActionDrop, Source: int16(canonical), Count: p.Inventory[canonical].Count,
				})
				slots = append(slots, slot)
			}
		}
		return actions, slots, len(actions) != 0
	}

	if len(pk.Actions) != 2 {
		return nil, nil, false
	}
	var containerAction, worldAction *protocol.InventoryAction
	for index := range pk.Actions {
		action := &pk.Actions[index]
		switch action.SourceType {
		case protocol.InventoryActionSourceContainer:
			windowID, present := action.WindowID.Value()
			if containerAction != nil || !present || windowID != int8(protocol.WindowIDInventory) || action.InventorySlot > 35 {
				return nil, nil, false
			}
			containerAction = action
		case protocol.InventoryActionSourceWorld:
			if worldAction != nil || action.InventorySlot != 0 {
				return nil, nil, false
			}
			worldAction = action
		default:
			return nil, nil, false
		}
	}
	if containerAction == nil || worldAction == nil ||
		!networkItemEmpty(worldAction.OldItem.Stack) || networkItemEmpty(worldAction.NewItem.Stack) ||
		networkItemEmpty(containerAction.OldItem.Stack) {
		return nil, nil, false
	}

	slot := byte(containerAction.InventorySlot)
	canonical := bedrockInventoryCanonicalSlot(int(slot))
	if canonical < 0 || p.Inventory[canonical].IsEmpty() {
		return nil, nil, false
	}
	dropped := int(worldAction.NewItem.Stack.Count)
	oldCount := int(containerAction.OldItem.Stack.Count)
	remaining := oldCount - dropped
	if dropped <= 0 || oldCount != p.Inventory[canonical].Count || remaining < 0 ||
		!sameNetworkItem(containerAction.OldItem.Stack, worldAction.NewItem.Stack) {
		return nil, nil, false
	}
	if remaining == 0 {
		if !networkItemEmpty(containerAction.NewItem.Stack) {
			return nil, nil, false
		}
	} else if networkItemEmpty(containerAction.NewItem.Stack) ||
		int(containerAction.NewItem.Stack.Count) != remaining ||
		!sameNetworkItem(containerAction.OldItem.Stack, containerAction.NewItem.Stack) {
		return nil, nil, false
	}

	return []intent.InventoryAction{{
		Kind: intent.InventoryActionDrop, Source: int16(canonical), Count: dropped,
	}}, []byte{slot}, true
}

func networkItemEmpty(stack protocol.ItemStack) bool {
	return stack.NetworkID == 0 && stack.Count == 0
}

func sameNetworkItem(a, b protocol.ItemStack) bool {
	return a.NetworkID == b.NetworkID &&
		a.MetadataValue == b.MetadataValue &&
		a.BlockRuntimeID == b.BlockRuntimeID
}

func (l *Listener) sendNormalDropSlots(conn *minecraft.Conn, session *bedrockSession, p *player.Player, slots []byte) {
	if conn == nil || session == nil || p == nil {
		return
	}
	seen := make(map[byte]struct{}, len(slots))
	for _, slot := range slots {
		if _, duplicate := seen[slot]; duplicate {
			continue
		}
		seen[slot] = struct{}{}
		canonical := bedrockInventoryCanonicalSlot(int(slot))
		if canonical < 0 {
			continue
		}
		stack := p.Inventory[canonical]
		session.stackMu.Lock()
		stackID := session.stackNetworkIDs[canonical]
		if stack.IsEmpty() {
			stackID = 0
			session.stackNetworkIDs[canonical] = 0
		} else if stackID <= 0 {
			stackID = session.allocateStackNetworkID()
			session.stackNetworkIDs[canonical] = stackID
		}
		item := l.itemInstance(stack, stackID)
		session.stackMu.Unlock()

		containerID := byte(protocol.ContainerInventory)
		if slot < 9 {
			containerID = protocol.ContainerHotBar
		}
		_ = conn.WritePacket(&packet.InventorySlot{
			WindowID: protocol.WindowIDInventory,
			Slot:     uint32(slot),
			Container: protocol.Option(protocol.FullContainerName{
				ContainerID: containerID,
			}),
			NewItem: item,
		})
	}
}

func (l *Listener) sendPersonalCraftingSlots(conn *minecraft.Conn, session *bedrockSession, p *player.Player) {
	if conn == nil || session == nil || p == nil {
		return
	}
	for _, pk := range l.personalCraftingSlotPackets(session, p) {
		_ = conn.WritePacket(pk)
	}
}

// personalCraftingSlotPackets keeps both crafting screens in Bedrock's shared
// UI inventory: personal input 28-31, workbench input 32-40, and result 50.
// This is the same UI layout used by Dragonfly's crafting handler and prevents
// crafting updates from overwriting normal inventory/hotbar slots.
func (l *Listener) personalCraftingSlotPackets(session *bedrockSession, p *player.Player) []*packet.InventorySlot {
	if session == nil || p == nil {
		return nil
	}
	updates := craftingSlotUpdates(p)

	session.stackMu.Lock()
	packets := make([]*packet.InventorySlot, 0, len(updates))
	for _, update := range updates {
		stack := canonicalStackAt(p, update.canonical)
		stackID := session.stackNetworkIDAt(update.canonical)
		if stack.IsEmpty() {
			stackID = 0
			session.setStackNetworkID(update.canonical, 0)
		} else if stackID == 0 {
			stackID = session.allocateStackNetworkID()
			session.setStackNetworkID(update.canonical, stackID)
		}
		packets = append(packets, &packet.InventorySlot{
			WindowID: update.windowID,
			Slot:     update.slot,
			Container: protocol.Option(protocol.FullContainerName{
				ContainerID: update.container,
			}),
			NewItem: l.itemInstance(stack, stackID),
		})
	}
	session.stackMu.Unlock()
	return packets
}

type craftingSlotUpdate struct {
	windowID  uint32
	slot      uint32
	container byte
	canonical int16
}

func craftingSlotUpdates(p *player.Player) []craftingSlotUpdate {
	updates := make([]craftingSlotUpdate, 0, 10)
	if p.OpenContainerKind == "minecraft:crafting_table" {
		for slot := int16(0); slot < 9; slot++ {
			updates = append(updates, craftingSlotUpdate{
				windowID: protocol.WindowIDUI, slot: uint32(32 + slot), container: protocol.ContainerCraftingInput,
				canonical: intent.InventoryCraftingTableStart + slot,
			})
		}
		updates = append(updates, craftingSlotUpdate{
			windowID: protocol.WindowIDUI, slot: 50, container: protocol.ContainerCreatedOutput,
			canonical: intent.InventoryCraftingTableOutput,
		})
	} else {
		for slot := int16(0); slot < 4; slot++ {
			updates = append(updates, craftingSlotUpdate{
				windowID: protocol.WindowIDUI, slot: uint32(28 + slot), container: protocol.ContainerCraftingInput,
				canonical: 1 + slot,
			})
		}
		updates = append(updates, craftingSlotUpdate{
			windowID: protocol.WindowIDUI, slot: 50, container: protocol.ContainerCreatedOutput, canonical: 0,
		})
	}
	return updates
}

func (l *Listener) canonicalInventoryActions(
	p *player.Player,
	actions []protocol.StackRequestAction,
) ([]intent.InventoryAction, bool) {
	out := make([]intent.InventoryAction, 0, len(actions))
	recognized := false
	creativeSelected := false
	creativeCount := creativeRequestCount(actions)
	craftCount := 1
	craftRequest := false
	for _, raw := range actions {
		switch raw.(type) {
		case *protocol.CraftRecipeStackRequestAction, *protocol.AutoCraftRecipeStackRequestAction,
			*protocol.CraftRecipeOptionalStackRequestAction:
			craftRequest = true
		}
	}

	for _, raw := range actions {
		switch action := raw.(type) {
		case *protocol.CraftCreativeStackRequestAction:
			recognized = true
			ki, ok := l.creativePlayerStack(
				action.CreativeItemNetworkID,
				int(action.NumberOfCrafts),
			)
			if !ok {
				slog.Warn(
					"bedrock: unknown creative item",
					"creative_network_id", action.CreativeItemNetworkID,
				)
				return nil, false
			}

			count := creativeCount
			if count < 1 {
				count = 1
			}
			maximum := player.MaxStackSize(ki.name)
			if maximum > 0 && count > maximum {
				count = maximum
			}

			out = append(out, intent.InventoryAction{
				Kind:        intent.InventoryActionCreativeGive,
				Destination: intent.InventoryCursorSlot,
				Count:       count,
				Item: player.ItemStack{
					ItemID:       ki.name,
					Count:        count,
					Damage:       int(ki.meta),
					HasFireworks: ki.hasFireworks,
					Fireworks:    ki.fireworks,
					Components:   ki.components,
				},
			})
			creativeSelected = true

		case *protocol.CraftResultsDeprecatedStackRequestAction:
			recognized = true
			// Client-side preview of the result. The authoritative item was
			// already resolved using CraftCreativeStackRequestAction.
			continue

		case *protocol.CreateStackRequestAction:
			recognized = true
			// CreatedOutput is virtual. Creative selection is handled above and
			// survival crafting is handled when the result is taken.
			continue

		case *protocol.TakeStackRequestAction:
			recognized = true
			// Modern Bedrock creative selection sends the temporary created
			// output (container 60, slot 50) to the cursor (container 59).
			// CraftCreativeStackRequestAction already created that authoritative
			// cursor stack, so this virtual transfer must be ignored.
			if creativeSelected &&
				action.Source.Container.ContainerID == protocol.ContainerCreatedOutput &&
				action.Destination.Container.ContainerID == protocol.ContainerCursor {
				continue
			}

			source, sourceOK := canonicalInventorySlotFor(p, action.Source)
			destination, destinationOK := canonicalInventorySlotFor(p, action.Destination)
			if !sourceOK || !destinationOK || action.Count == 0 {
				slog.Warn(
					"bedrock: invalid take stack slots",
					"source_container", action.Source.Container.ContainerID,
					"source_slot", action.Source.Slot,
					"destination_container", action.Destination.Container.ContainerID,
					"destination_slot", action.Destination.Slot,
				)
				return nil, false
			}

			out = append(out, intent.InventoryAction{
				Kind:        intent.InventoryActionMove,
				Source:      source,
				Destination: destination,
				Count:       int(action.Count),
				CraftCount:  craftCount,
			})

		case *protocol.PlaceStackRequestAction:
			recognized = true
			source, sourceOK := canonicalInventorySlotFor(p, action.Source)
			destination, destinationOK := canonicalInventorySlotFor(p, action.Destination)
			if !sourceOK || !destinationOK || action.Count == 0 {
				return nil, false
			}
			out = append(out, intent.InventoryAction{
				Kind:        intent.InventoryActionMove,
				Source:      source,
				Destination: destination,
				Count:       int(action.Count),
			})

		case *protocol.SwapStackRequestAction:
			recognized = true
			source, sourceOK := canonicalInventorySlotFor(p, action.Source)
			destination, destinationOK := canonicalInventorySlotFor(p, action.Destination)
			if !sourceOK || !destinationOK || source == destination {
				return nil, false
			}
			out = append(out, intent.InventoryAction{
				Kind:        intent.InventoryActionSwap,
				Source:      source,
				Destination: destination,
			})

		case *protocol.DropStackRequestAction:
			recognized = true
			source, sourceOK := canonicalInventorySlotFor(p, action.Source)
			if !sourceOK || action.Count == 0 {
				return nil, false
			}
			out = append(out, intent.InventoryAction{
				Kind:   intent.InventoryActionDrop,
				Source: source,
				Count:  int(action.Count),
			})

		case *protocol.DestroyStackRequestAction:
			recognized = true
			source, sourceOK := canonicalInventorySlotFor(p, action.Source)
			if !sourceOK || action.Count == 0 {
				return nil, false
			}
			out = append(out, intent.InventoryAction{
				Kind:   intent.InventoryActionDestroy,
				Source: source,
				Count:  int(action.Count),
			})

		case *protocol.CraftRecipeStackRequestAction:
			recognized = true
			if p != nil && p.OpenContainerKind == "minecraft:stonecutter" && len(p.ContainerSlots) > 0 {
				if selection, ok := bedrockStonecutterSelection(action.RecipeNetworkID, p.ContainerSlots[0].ItemID); ok {
					p.WorkstationSelection = selection
					handler.UpdateWorkstationResult(p.OpenContainerKind, p.ContainerSlots, selection)
				}
			}
			craftCount = max(int(action.NumberOfCrafts), 1)
			continue

		case *protocol.AutoCraftRecipeStackRequestAction:
			recognized = true
			craftCount = max(int(action.NumberOfCrafts), 1)
			continue

		case *protocol.ConsumeStackRequestAction:
			recognized = true
			if craftRequest {
				continue
			}
			source, ok := canonicalInventorySlotFor(p, action.Source)
			if !ok || action.Count == 0 {
				return nil, false
			}
			out = append(out, intent.InventoryAction{
				Kind: intent.InventoryActionConsume, Source: source, Count: int(action.Count),
			})
			continue

		case *protocol.CraftRecipeOptionalStackRequestAction,
			*protocol.CraftGrindstoneRecipeStackRequestAction,
			*protocol.CraftLoomRecipeStackRequestAction,
			*protocol.CraftNonImplementedStackRequestAction:
			// These prediction actions are either accompanied by authoritative
			// moves or are successful no-ops. Pumpkin accepts them so the client
			// does not roll its crafting UI back.
			recognized = true
			continue

		default:
			// Skip client-side prediction actions we don't need to handle
			// (e.g. CraftRecipeStackRequestAction, AutoCraftRecipeStackRequestAction).
			// Returning false here would reject the entire request and break
			// operations like Q-drop when the client bundles extra actions.
			slog.Debug(
				"bedrock: skipping unsupported stack request action",
				"type", fmt.Sprintf("%T", raw),
			)
			continue
		}
	}

	return out, recognized
}

func creativeRequestCount(actions []protocol.StackRequestAction) int {
	for _, raw := range actions {
		switch action := raw.(type) {
		case *protocol.TakeStackRequestAction:
			if action.Source.Container.ContainerID == protocol.ContainerCreatedOutput &&
				action.Destination.Container.ContainerID == protocol.ContainerCursor &&
				action.Count > 0 {
				return int(action.Count)
			}
		case *protocol.CraftResultsDeprecatedStackRequestAction:
			if len(action.ResultItems) > 0 && action.ResultItems[0].Count > 0 {
				return int(action.ResultItems[0].Count)
			}
		}
	}
	return 1
}

func canonicalInventorySlot(slot protocol.StackRequestSlotInfo) (int16, bool) {
	return canonicalInventorySlotFor(nil, slot)
}

func canonicalInventorySlotFor(p *player.Player, slot protocol.StackRequestSlotInfo) (int16, bool) {
	switch slot.Container.ContainerID {
	case protocol.ContainerCraftingInput:
		// Bedrock may address crafting slots either by local container index
		// (0-3/0-8) or by the UI inventory slots (28-31/32-40).
		if p != nil && p.OpenContainerKind == "minecraft:crafting_table" {
			if slot.Slot < 9 {
				return intent.InventoryCraftingTableStart + int16(slot.Slot), true
			}
			if slot.Slot >= 32 && slot.Slot <= 40 {
				return intent.InventoryCraftingTableStart + int16(slot.Slot-32), true
			}
			return 0, false
		}
		if slot.Slot < 4 {
			return int16(1 + slot.Slot), true
		}
		if slot.Slot >= 28 && slot.Slot <= 31 {
			return int16(1 + slot.Slot - 28), true
		}
		return 0, false
	case protocol.ContainerCreatedOutput, protocol.ContainerCraftingOutputPreview:
		if slot.Slot != 0 && slot.Slot != 50 {
			return 0, false
		}
		if p != nil && p.OpenContainerKind == "minecraft:crafting_table" {
			return intent.InventoryCraftingTableOutput, true
		}
		if p != nil {
			if index, ok := bedrockWorkstationCanonicalSlot(p.OpenContainerKind, slot.Container.ContainerID, slot.Slot); ok {
				return intent.InventoryContainerStart + int16(index), true
			}
		}
		return 0, true
	case protocol.ContainerFurnaceIngredient, protocol.ContainerBlastFurnaceIngredient, protocol.ContainerSmokerIngredient:
		if p == nil || !handler.IsFurnaceContainer(p.OpenContainerKind) || slot.Slot != 0 {
			return 0, false
		}
		return intent.InventoryFurnaceInput, true
	case protocol.ContainerFurnaceFuel:
		if p == nil || !handler.IsFurnaceContainer(p.OpenContainerKind) || (slot.Slot != 0 && slot.Slot != 1) {
			return 0, false
		}
		return intent.InventoryFurnaceFuel, true
	case protocol.ContainerFurnaceResult:
		if p == nil || !handler.IsFurnaceContainer(p.OpenContainerKind) || (slot.Slot != 0 && slot.Slot != 2) {
			return 0, false
		}
		return intent.InventoryFurnaceOutput, true
	case protocol.ContainerLevelEntity:
		if p == nil {
			return 0, false
		}
		if handler.IsFurnaceContainer(p.OpenContainerKind) {
			if slot.Slot > 2 {
				return 0, false
			}
			return intent.InventoryFurnaceInput + int16(slot.Slot), true
		}
		if int(slot.Slot) >= len(p.ContainerSlots) || len(p.ContainerSlots) == 0 {
			return 0, false
		}
		return intent.InventoryContainerStart + int16(slot.Slot), true
	case protocol.ContainerCrafterLevelEntity:
		if p == nil || p.OpenContainerKind != "minecraft:crafter" || int(slot.Slot) >= len(p.ContainerSlots) {
			return 0, false
		}
		return intent.InventoryContainerStart + int16(slot.Slot), true
	case protocol.ContainerAnvilInput, protocol.ContainerAnvilMaterial, protocol.ContainerAnvilResultPreview,
		protocol.ContainerSmithingTableInput, protocol.ContainerSmithingTableMaterial, protocol.ContainerSmithingTableResultPreview,
		protocol.ContainerSmithingTableTemplate, protocol.ContainerBeaconPayment,
		protocol.ContainerBrewingStandInput, protocol.ContainerBrewingStandResult, protocol.ContainerBrewingStandFuel,
		protocol.ContainerEnchantingInput, protocol.ContainerEnchantingMaterial,
		protocol.ContainerLoomInput, protocol.ContainerLoomDye, protocol.ContainerLoomMaterial, protocol.ContainerLoomResultPreview,
		protocol.ContainerGrindstoneInput, protocol.ContainerGrindstoneAdditional, protocol.ContainerGrindstoneResultPreview,
		protocol.ContainerStonecutterInput, protocol.ContainerStonecutterResultPreview,
		protocol.ContainerCartographyInput, protocol.ContainerCartographyAdditional, protocol.ContainerCartographyResultPreview:
		if p == nil {
			return 0, false
		}
		index, ok := bedrockWorkstationCanonicalSlot(p.OpenContainerKind, slot.Container.ContainerID, slot.Slot)
		if !ok || index >= len(p.ContainerSlots) {
			return 0, false
		}
		return intent.InventoryContainerStart + int16(index), true
	case protocol.ContainerHotBar, protocol.ContainerInventory, protocol.ContainerCombinedHotBarAndInventory:
		if slot.Slot > 35 {
			return 0, false
		}
		if slot.Slot < 9 {
			return int16(player.HotbarStart + int(slot.Slot)), true
		}
		return int16(slot.Slot), true
	case protocol.ContainerArmor:
		if slot.Slot > 3 {
			return 0, false
		}
		return int16(5 + slot.Slot), true
	case protocol.ContainerOffhand:
		if slot.Slot != 0 {
			return 0, false
		}
		return player.OffhandSlot, true
	case protocol.ContainerCursor:
		return intent.InventoryCursorSlot, true
	default:
		return 0, false
	}
}

func bedrockWorkstationCanonicalSlot(kind string, containerID byte, slot byte) (int, bool) {
	single := func(index int) (int, bool) { return index, slot == 0 || slot == 50 }
	switch kind {
	case "minecraft:anvil", "minecraft:chipped_anvil", "minecraft:damaged_anvil":
		switch containerID {
		case protocol.ContainerAnvilInput:
			return single(0)
		case protocol.ContainerAnvilMaterial:
			return single(1)
		case protocol.ContainerAnvilResultPreview, protocol.ContainerCreatedOutput, protocol.ContainerCraftingOutputPreview:
			return single(2)
		}
	case "minecraft:enchanting_table":
		if containerID == protocol.ContainerEnchantingInput {
			return single(0)
		}
		if containerID == protocol.ContainerEnchantingMaterial {
			return single(1)
		}
	case "minecraft:grindstone":
		switch containerID {
		case protocol.ContainerGrindstoneInput:
			return single(0)
		case protocol.ContainerGrindstoneAdditional:
			return single(1)
		case protocol.ContainerGrindstoneResultPreview, protocol.ContainerCreatedOutput, protocol.ContainerCraftingOutputPreview:
			return single(2)
		}
	case "minecraft:loom":
		switch containerID {
		case protocol.ContainerLoomInput:
			return single(0)
		case protocol.ContainerLoomDye:
			return single(1)
		case protocol.ContainerLoomMaterial:
			return single(2)
		case protocol.ContainerLoomResultPreview, protocol.ContainerCreatedOutput, protocol.ContainerCraftingOutputPreview:
			return single(3)
		}
	case "minecraft:smithing_table":
		switch containerID {
		case protocol.ContainerSmithingTableTemplate:
			return single(0)
		case protocol.ContainerSmithingTableInput:
			return single(1)
		case protocol.ContainerSmithingTableMaterial:
			return single(2)
		case protocol.ContainerSmithingTableResultPreview, protocol.ContainerCreatedOutput, protocol.ContainerCraftingOutputPreview:
			return single(3)
		}
	case "minecraft:stonecutter":
		if containerID == protocol.ContainerStonecutterInput {
			return single(0)
		}
		if containerID == protocol.ContainerStonecutterResultPreview || containerID == protocol.ContainerCreatedOutput || containerID == protocol.ContainerCraftingOutputPreview {
			return single(1)
		}
	case "minecraft:cartography_table":
		switch containerID {
		case protocol.ContainerCartographyInput:
			return single(0)
		case protocol.ContainerCartographyAdditional:
			return single(1)
		case protocol.ContainerCartographyResultPreview, protocol.ContainerCreatedOutput, protocol.ContainerCraftingOutputPreview:
			return single(2)
		}
	case "minecraft:brewing_stand":
		switch containerID {
		case protocol.ContainerBrewingStandResult:
			if slot <= 2 {
				return int(slot), true
			}
		case protocol.ContainerBrewingStandInput:
			return single(3)
		case protocol.ContainerBrewingStandFuel:
			return single(4)
		}
	case "minecraft:beacon":
		if containerID == protocol.ContainerBeaconPayment {
			return single(0)
		}
	}
	return 0, false
}

func (s *bedrockSession) allocateStackNetworkID() int32 {
	if s.nextStackNetworkID <= 0 {
		s.nextStackNetworkID = 1
	}
	id := s.nextStackNetworkID
	s.nextStackNetworkID++
	if s.nextStackNetworkID <= 0 {
		s.nextStackNetworkID = 1
	}
	return id
}

func (s *bedrockSession) stackNetworkIDAt(slot int16) int32 {
	if slot == intent.InventoryCursorSlot {
		return s.cursorStackID
	}
	if slot >= intent.InventoryCraftingTableStart && slot <= intent.InventoryCraftingTableOutput {
		return s.craftingNetworkIDs[slot-intent.InventoryCraftingTableStart]
	}
	if slot >= intent.InventoryFurnaceInput && slot <= intent.InventoryFurnaceOutput {
		return s.furnaceNetworkIDs[slot-intent.InventoryFurnaceInput]
	}
	if slot >= intent.InventoryContainerStart && slot < intent.InventoryContainerStart+int16(len(s.containerNetworkIDs)) {
		return s.containerNetworkIDs[slot-intent.InventoryContainerStart]
	}
	if slot < 0 || int(slot) >= len(s.stackNetworkIDs) {
		return 0
	}
	return s.stackNetworkIDs[slot]
}

func (s *bedrockSession) setStackNetworkID(slot int16, id int32) {
	if slot == intent.InventoryCursorSlot {
		s.cursorStackID = id
		return
	}
	if slot >= intent.InventoryCraftingTableStart && slot <= intent.InventoryCraftingTableOutput {
		s.craftingNetworkIDs[slot-intent.InventoryCraftingTableStart] = id
		return
	}
	if slot >= intent.InventoryFurnaceInput && slot <= intent.InventoryFurnaceOutput {
		s.furnaceNetworkIDs[slot-intent.InventoryFurnaceInput] = id
		return
	}
	if slot >= intent.InventoryContainerStart && slot < intent.InventoryContainerStart+int16(len(s.containerNetworkIDs)) {
		s.containerNetworkIDs[slot-intent.InventoryContainerStart] = id
		return
	}
	if slot < 0 || int(slot) >= len(s.stackNetworkIDs) {
		return
	}
	s.stackNetworkIDs[slot] = id
}

func (l *Listener) sessionForPlayer(playerUUID [16]byte) *bedrockSession {
	l.sessionsMu.RLock()
	session := l.sessions[playerUUID]
	l.sessionsMu.RUnlock()
	return session
}

func (l *Listener) applyStackNetworkIDChanges(session *bedrockSession, p *player.Player, actions []protocol.StackRequestAction) {
	if session == nil || p == nil {
		return
	}

	session.stackMu.Lock()
	defer session.stackMu.Unlock()

	for _, raw := range actions {
		switch action := raw.(type) {
		case *protocol.CraftCreativeStackRequestAction:
			session.cursorStackID = session.allocateStackNetworkID()

		case *protocol.CreateStackRequestAction, *protocol.CraftResultsDeprecatedStackRequestAction:
			continue

		case *protocol.TakeStackRequestAction:
			l.transferStackNetworkID(session, p, action.Source, action.Destination)

		case *protocol.PlaceStackRequestAction:
			l.transferStackNetworkID(session, p, action.Source, action.Destination)

		case *protocol.SwapStackRequestAction:
			source, sourceOK := canonicalInventorySlotFor(p, action.Source)
			destination, destinationOK := canonicalInventorySlotFor(p, action.Destination)
			if !sourceOK || !destinationOK {
				continue
			}
			sourceID := session.stackNetworkIDAt(source)
			destinationID := session.stackNetworkIDAt(destination)
			if sourceID == 0 && action.Source.StackNetworkID > 0 {
				sourceID = action.Source.StackNetworkID
			}
			if destinationID == 0 && action.Destination.StackNetworkID > 0 {
				destinationID = action.Destination.StackNetworkID
			}
			session.setStackNetworkID(source, destinationID)
			session.setStackNetworkID(destination, sourceID)

		case *protocol.DropStackRequestAction:
			source, ok := canonicalInventorySlotFor(p, action.Source)
			if ok && canonicalStackAt(p, source).IsEmpty() {
				session.setStackNetworkID(source, 0)
			}

		case *protocol.DestroyStackRequestAction:
			source, ok := canonicalInventorySlotFor(p, action.Source)
			if ok && canonicalStackAt(p, source).IsEmpty() {
				session.setStackNetworkID(source, 0)
			}

		case *protocol.ConsumeStackRequestAction:
			source, ok := canonicalInventorySlotFor(p, action.Source)
			if ok && canonicalStackAt(p, source).IsEmpty() {
				session.setStackNetworkID(source, 0)
			}
		}
	}
}

func (l *Listener) transferStackNetworkID(session *bedrockSession, p *player.Player, sourceInfo, destinationInfo protocol.StackRequestSlotInfo) {
	source, sourceOK := canonicalInventorySlotFor(p, sourceInfo)
	destination, destinationOK := canonicalInventorySlotFor(p, destinationInfo)
	if !sourceOK || !destinationOK {
		return
	}

	sourceID := session.stackNetworkIDAt(source)
	destinationID := session.stackNetworkIDAt(destination)
	if sourceID == 0 && sourceInfo.StackNetworkID > 0 {
		sourceID = sourceInfo.StackNetworkID
	}
	if destinationID == 0 && destinationInfo.StackNetworkID > 0 {
		destinationID = destinationInfo.StackNetworkID
	}

	sourceStack := canonicalStackAt(p, source)
	destinationStack := canonicalStackAt(p, destination)

	if destinationStack.IsEmpty() {
		session.setStackNetworkID(destination, 0)
	} else if destinationID > 0 {
		session.setStackNetworkID(destination, destinationID)
	} else if sourceID > 0 && sourceStack.IsEmpty() {
		session.setStackNetworkID(destination, sourceID)
	} else {
		session.setStackNetworkID(destination, session.allocateStackNetworkID())
	}

	if sourceStack.IsEmpty() {
		session.setStackNetworkID(source, 0)
	} else if sourceID > 0 {
		session.setStackNetworkID(source, sourceID)
	} else {
		session.setStackNetworkID(source, session.allocateStackNetworkID())
	}
}

func (l *Listener) stackResponseContainerInfo(session *bedrockSession, p *player.Player, actions []protocol.StackRequestAction) []protocol.StackResponseContainerInfo {
	if session == nil || p == nil {
		return nil
	}

	type containerKey struct {
		id      byte
		dynamic string
	}
	type changedSlot struct {
		container protocol.FullContainerName
		slot      byte
	}

	changed := make([]changedSlot, 0, len(actions)*2+1)
	seen := make(map[string]struct{}, len(actions)*2+1)
	add := func(slot protocol.StackRequestSlotInfo) {
		if _, ok := canonicalInventorySlotFor(p, slot); !ok {
			return
		}
		key := fmt.Sprintf("%d:%v:%d", slot.Container.ContainerID, slot.Container.DynamicContainerID, slot.Slot)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		changed = append(changed, changedSlot{container: slot.Container, slot: slot.Slot})
	}
	creativeRequest := false
	for _, raw := range actions {
		if _, ok := raw.(*protocol.CraftCreativeStackRequestAction); ok {
			creativeRequest = true
			break
		}
	}

	for _, raw := range actions {
		switch action := raw.(type) {
		case *protocol.CraftCreativeStackRequestAction:
			add(protocol.StackRequestSlotInfo{Container: protocol.FullContainerName{ContainerID: protocol.ContainerCursor}, Slot: 0})
		case *protocol.TakeStackRequestAction:
			if creativeRequest && action.Source.Container.ContainerID == protocol.ContainerCreatedOutput &&
				action.Destination.Container.ContainerID == protocol.ContainerCursor {
				add(action.Destination)
				continue
			}

			add(action.Source)
			add(action.Destination)
			if action.Source.Container.ContainerID == protocol.ContainerCreatedOutput {
				// Taking a result consumes one item from each occupied input slot.
				start, end := byte(28), byte(31)
				if p.OpenContainerKind == "minecraft:crafting_table" {
					start, end = 32, 40
				}
				for slot := start; slot <= end; slot++ {
					add(protocol.StackRequestSlotInfo{
						Container: protocol.FullContainerName{ContainerID: protocol.ContainerCraftingInput},
						Slot:      slot,
					})
				}
			}
		case *protocol.PlaceStackRequestAction:
			add(action.Source)
			add(action.Destination)
		case *protocol.SwapStackRequestAction:
			add(action.Source)
			add(action.Destination)
		case *protocol.DropStackRequestAction:
			add(action.Source)
		case *protocol.DestroyStackRequestAction:
			add(action.Source)
		case *protocol.ConsumeStackRequestAction:
			add(action.Source)
		}
	}

	session.stackMu.Lock()
	defer session.stackMu.Unlock()

	groups := make([]protocol.StackResponseContainerInfo, 0, len(changed))
	indices := make(map[containerKey]int, len(changed))
	for _, entry := range changed {
		canonical, ok := canonicalInventorySlotFor(p, protocol.StackRequestSlotInfo{Container: entry.container, Slot: entry.slot})
		if !ok {
			continue
		}

		stack := canonicalStackAt(p, canonical)
		stackID := session.stackNetworkIDAt(canonical)
		if stack.IsEmpty() {
			stackID = 0
			session.setStackNetworkID(canonical, 0)
		} else if stackID == 0 {
			stackID = session.allocateStackNetworkID()
			session.setStackNetworkID(canonical, stackID)
		}

		key := containerKey{id: entry.container.ContainerID, dynamic: fmt.Sprint(entry.container.DynamicContainerID)}
		index, exists := indices[key]
		if !exists {
			index = len(groups)
			indices[key] = index
			groups = append(groups, protocol.StackResponseContainerInfo{Container: entry.container})
		}
		groups[index].SlotInfo = append(groups[index].SlotInfo, protocol.StackResponseSlotInfo{
			Slot:                 entry.slot,
			HotbarSlot:           entry.slot,
			Count:                byte(min(max(stack.Count, 0), 255)),
			StackNetworkID:       stackID,
			DurabilityCorrection: int32(max(stack.Damage, 0)),
		})
	}
	return groups
}

func canonicalStackAt(p *player.Player, slot int16) player.ItemStack {
	if p == nil {
		return player.ItemStack{}
	}
	if slot == intent.InventoryCursorSlot {
		return p.CarriedItem
	}
	if slot >= intent.InventoryCraftingTableStart && slot < intent.InventoryCraftingTableOutput {
		return p.CraftingGrid[slot-intent.InventoryCraftingTableStart]
	}
	if slot == intent.InventoryCraftingTableOutput {
		return p.CraftingResult
	}
	if slot >= intent.InventoryFurnaceInput && slot <= intent.InventoryFurnaceOutput {
		index := int(slot - intent.InventoryFurnaceInput)
		if index >= 0 && index < len(p.ContainerSlots) {
			return p.ContainerSlots[index]
		}
		return player.ItemStack{}
	}
	if slot >= intent.InventoryContainerStart {
		index := int(slot - intent.InventoryContainerStart)
		if index >= 0 && index < len(p.ContainerSlots) {
			return p.ContainerSlots[index]
		}
		return player.ItemStack{}
	}
	if slot < 0 || int(slot) >= len(p.Inventory) {
		return player.ItemStack{}
	}
	return p.Inventory[slot]
}

func (l *Listener) handlePlayerBlockAction(session *bedrockSession, playerUUID [16]byte, action int32, position protocol.BlockPos, face int32) {
	switch action {
	case protocol.PlayerActionStartBreak:
		session.breakingPos, session.breaking = position, true
		l.broadcastBlockCrack(playerUUID, position, packet.LevelEventStartBlockCracking)
	case protocol.PlayerActionCrackBreak, protocol.PlayerActionContinueDestroyBlock:
		// Holding the button while moving to the next block may emit CONTINUE
		// without a new START. An update event cannot start a fresh crack overlay,
		// so explicitly stop the old target and start the new one.
		if !session.breaking || session.breakingPos != position {
			if session.breaking {
				l.broadcastBlockCrack(playerUUID, session.breakingPos, packet.LevelEventStopBlockCracking)
			}
			session.breakingPos, session.breaking = position, true
			l.broadcastBlockCrack(playerUUID, position, packet.LevelEventStartBlockCracking)
			l.broadcastBlockHitSound(position)
			break
		}
		session.breakingPos, session.breaking = position, true
		l.broadcastBlockCrack(playerUUID, position, packet.LevelEventUpdateBlockCracking)
		l.broadcastBlockHitSound(position)
	case protocol.PlayerActionAbortBreak:
		session.breaking = false
		l.broadcastBlockCrack(playerUUID, position, packet.LevelEventStopBlockCracking)
	case protocol.PlayerActionPredictDestroyBlock, protocol.PlayerActionStopBreak, protocol.PlayerActionCreativePlayerDestroyBlock:
		if session.breaking {
			position = session.breakingPos
		}
		session.breaking = false
		l.broadcastBlockCrack(playerUUID, position, packet.LevelEventStopBlockCracking)
		l.bus.PostBlockInteract(intent.BlockInteractIntent{
			PlayerUUID: playerUUID,
			Action:     intent.BlockActionBreak,
			Position:   spatial.BlockPos{X: position.X(), Y: position.Y(), Z: position.Z()},
			Face:       face,
			// PlayerAction carries no selected-slot field. Zero is a real hotbar
			// slot, so use -1 to preserve the latest MobEquipment selection.
			HotbarSlot: -1,
		})
	case protocol.PlayerActionRespawn:
		l.bus.PostRespawn(intent.RespawnIntent{PlayerUUID: playerUUID})
	case protocol.PlayerActionStopSleeping:
		l.bus.PostWake(intent.WakeIntent{PlayerUUID: playerUUID})
	case protocol.PlayerActionStartSneak:
		l.postPlayerState(playerUUID, intent.PlayerStateSneaking, true)
	case protocol.PlayerActionStopSneak:
		l.postPlayerState(playerUUID, intent.PlayerStateSneaking, false)
	case protocol.PlayerActionStartSprint:
		l.postPlayerState(playerUUID, intent.PlayerStateSprinting, true)
	case protocol.PlayerActionStopSprint:
		l.postPlayerState(playerUUID, intent.PlayerStateSprinting, false)
	case protocol.PlayerActionStartFlying:
		l.postPlayerState(playerUUID, intent.PlayerStateFlying, true)
	case protocol.PlayerActionStopFlying:
		l.postPlayerState(playerUUID, intent.PlayerStateFlying, false)
	}
}

func blockCrackSpeed(duration time.Duration) int32 {
	if duration <= 0 {
		return 0
	}
	speed := int64(65535 / (duration.Seconds() * 20))
	if speed < 1 {
		return 1
	}
	if speed > 65535 {
		return 65535
	}
	return int32(speed)
}

func (l *Listener) blockBreakDuration(playerUUID [16]byte, position protocol.BlockPos) time.Duration {
	block := l.world.GetBlock(int(position.X()), int(position.Y()), int(position.Z()))
	p := l.game.GetPlayer(playerUUID)
	if block.IsAir() || p == nil || p.GameMode == player.GameModeCreative {
		return 0
	}
	// Encoder IDs are stable network hashes, not Dragonfly's sequential runtime
	// IDs. Resolve the hash back before asking Dragonfly for break properties.
	networkID := l.encoder.BlockNetworkID(block)
	runtimeID, ok := dfworld.DefaultBlockRegistry.HashToRuntimeID(networkID)
	if !ok {
		return 750 * time.Millisecond
	}
	bedrockBlock, ok := dfworld.DefaultBlockRegistry.BlockByRuntimeID(runtimeID)
	if !ok {
		return 750 * time.Millisecond
	}
	var held dfitem.Stack
	if heldItem, ok := dfworld.ItemByName(p.HeldItem().ItemID, 0); ok {
		held = dfitem.NewStack(heldItem, 1)
	}
	duration := dfblock.BreakDuration(bedrockBlock, held, dfblock.BreakContext{
		Airborne:   !p.OnGround,
		Underwater: !p.UnderwaterSince.IsZero(),
	})
	if duration > 5*time.Minute {
		return 750 * time.Millisecond
	}
	return duration
}

func (l *Listener) broadcastBlockCrack(playerUUID [16]byte, position protocol.BlockPos, eventType int32) {
	duration := l.blockBreakDuration(playerUUID, position)
	if duration <= 0 && eventType != packet.LevelEventStopBlockCracking {
		return
	}
	event := &packet.LevelEvent{
		EventType: eventType,
		Position:  mgl32.Vec3{float32(position.X()), float32(position.Y()), float32(position.Z())},
		EventData: blockCrackSpeed(duration),
	}
	l.sessionsMu.RLock()
	sessions := make([]*bedrockSession, 0, len(l.sessions))
	for _, session := range l.sessions {
		sessions = append(sessions, session)
	}
	l.sessionsMu.RUnlock()
	for _, session := range sessions {
		_ = session.conn.WritePacket(event)
	}
}

func (l *Listener) broadcastBlockHitSound(position protocol.BlockPos) {
	block := l.world.GetBlock(int(position.X()), int(position.Y()), int(position.Z()))
	if block.IsAir() {
		return
	}
	event := &packet.LevelSoundEvent{
		SoundType: packet.SoundEventHit,
		Position: mgl32.Vec3{
			float32(position.X()) + 0.5,
			float32(position.Y()) + 0.5,
			float32(position.Z()) + 0.5,
		},
		ExtraData: int32(l.encoder.BlockNetworkID(block)),
	}
	l.sessionsMu.RLock()
	sessions := make([]*bedrockSession, 0, len(l.sessions))
	for _, session := range l.sessions {
		sessions = append(sessions, session)
	}
	l.sessionsMu.RUnlock()
	for _, session := range sessions {
		_ = session.conn.WritePacket(event)
	}
}

// inputHasFlag reports whether the given flag is present in a PlayerAuthInput
// InputData optional flag list.
func inputHasFlag(input protocol.InputFlags, flag int32) bool {
	return input.Present() && int(flag) < input.Len() && input.Load(int(flag))
}

func (l *Listener) postInputState(playerUUID [16]byte, input protocol.InputFlags) {
	if inputHasFlag(input, packet.InputFlagStartSneaking) {
		l.postPlayerState(playerUUID, intent.PlayerStateSneaking, true)
	}
	if inputHasFlag(input, packet.InputFlagStopSneaking) {
		l.postPlayerState(playerUUID, intent.PlayerStateSneaking, false)
	}
	if inputHasFlag(input, packet.InputFlagStartSprinting) {
		l.postPlayerState(playerUUID, intent.PlayerStateSprinting, true)
	}
	if inputHasFlag(input, packet.InputFlagStopSprinting) {
		l.postPlayerState(playerUUID, intent.PlayerStateSprinting, false)
	}
	if inputHasFlag(input, packet.InputFlagStartFlying) {
		l.postPlayerState(playerUUID, intent.PlayerStateFlying, true)
	}
	if inputHasFlag(input, packet.InputFlagStopFlying) {
		l.postPlayerState(playerUUID, intent.PlayerStateFlying, false)
	}
}

func (l *Listener) postPlayerState(playerUUID [16]byte, state uint8, enabled bool) {
	l.bus.PostPlayerState(intent.PlayerStateIntent{PlayerUUID: playerUUID, State: state, Enabled: enabled})
}

func (l *Listener) handleUseItemTransaction(session *bedrockSession, playerUUID [16]byte, data *protocol.UseItemTransactionData) {
	if data == nil {
		return
	}
	if !validActionHotbarSlot(data.HotBarSlot, "InventoryTransaction/UseItem") {
		return
	}
	action := uint8(0)
	switch data.ActionType {
	case protocol.UseItemActionBreakBlock:
		action = intent.BlockActionBreak
	case protocol.UseItemActionClickBlock:
		// Holding use produces simulation-tick transactions after the initial
		// input. Stateful blocks must toggle once, not on every held tick.
		if data.TriggerType == protocol.TriggerTypeSimulationTick {
			return
		}
		// Some protocol paths carry the same initial transaction both standalone
		// and embedded in PlayerAuthInput. Collapse that duplicate so doors and
		// fence gates do not immediately toggle back.
		now := time.Now()
		if session != nil && session.lastBlockUsePos == data.BlockPosition &&
			session.lastBlockUseFace == data.BlockFace && session.lastBlockUseSlot == data.HotBarSlot &&
			now.Sub(session.lastBlockUseAt) < 40*time.Millisecond {
			return
		}
		if session != nil {
			session.lastBlockUsePos = data.BlockPosition
			session.lastBlockUseFace = data.BlockFace
			session.lastBlockUseSlot = data.HotBarSlot
			session.lastBlockUseAt = now
		}
		action = intent.BlockActionUse
	case protocol.UseItemActionClickAir:
		l.bus.PostStartUseItem(intent.StartUseItemIntent{PlayerUUID: playerUUID, HotbarSlot: data.HotBarSlot})
		return
	default:
		return
	}
	l.bus.PostBlockInteract(intent.BlockInteractIntent{
		PlayerUUID: playerUUID,
		Action:     action,
		Position: spatial.BlockPos{
			X: data.BlockPosition.X(),
			Y: data.BlockPosition.Y(),
			Z: data.BlockPosition.Z(),
		},
		Face:       data.BlockFace,
		HotbarSlot: data.HotBarSlot,
		ClickX:     data.ClickedPosition[0],
		ClickY:     data.ClickedPosition[1],
		ClickZ:     data.ClickedPosition[2],
	})
}

// validActionHotbarSlot validates the slot snapshot carried by an item action.
// It deliberately does not change the persistent selected slot: Bedrock uses
// MobEquipment for that state, while transaction slots identify only the item
// that participated in a particular use, release, or entity interaction.
func validActionHotbarSlot(slot int32, packetType string) bool {
	if slot >= 0 && slot < 9 {
		return true
	}
	slog.Debug("bedrock action hotbar context rejected",
		"packet_type", packetType, "action_slot", slot)
	return false
}

// acceptClientHotbarSlot records a client-owned selected-slot change and
// queues it for the simulation. A valid selection is never echoed back to the
// originating client: doing so one tick later can overwrite a newer wheel
// selection. Only invalid or dropped changes receive a correction.
func (l *Listener) acceptClientHotbarSlot(session *bedrockSession, playerUUID [16]byte, slot int32, packetType string) bool {
	p := l.game.GetPlayer(playerUUID)
	current := -1
	if p != nil {
		current = p.HeldSlot
	}
	if slot < 0 || slot >= 9 || p == nil {
		slog.Debug("bedrock hotbar selection rejected",
			"packet_type", packetType, "incoming_slot", slot, "current_server_slot", current, "corrected_slot", current)
		if p != nil {
			l.sendHotbarCorrection(session, p, packetType)
		}
		return false
	}
	if session != nil {
		session.stackMu.Lock()
		session.clientHeldSlot = int(slot)
		session.clientHeldSlotSeen = true
		session.stackMu.Unlock()
	}
	slog.Debug("bedrock hotbar selection accepted",
		"packet_type", packetType, "incoming_slot", slot, "current_server_slot", current, "outgoing_slot", "none")
	if !l.bus.PostHotbar(intent.HotbarIntent{PlayerUUID: playerUUID, Slot: slot}) {
		slog.Debug("bedrock hotbar selection queue full; correcting",
			"packet_type", packetType, "incoming_slot", slot, "current_server_slot", current, "corrected_slot", current)
		l.sendHotbarCorrection(session, p, packetType+"/QueueFull")
		return false
	}
	return true
}

func (l *Listener) sendHotbarCorrection(session *bedrockSession, p *player.Player, packetType string) {
	if session == nil || session.conn == nil || p == nil || p.HeldSlot < 0 || p.HeldSlot >= 9 {
		return
	}
	session.stackMu.Lock()
	session.clientHeldSlot = p.HeldSlot
	session.clientHeldSlotSeen = true
	session.stackMu.Unlock()
	slog.Debug("bedrock hotbar correction sent",
		"packet_type", packetType, "current_server_slot", p.HeldSlot, "outgoing_slot", p.HeldSlot, "outgoing_packet", "PlayerHotBar")
	_ = session.conn.WritePacket(&packet.PlayerHotBar{
		SelectedHotBarSlot: uint32(p.HeldSlot),
		WindowID:           protocol.WindowIDInventory,
		SelectHotBarSlot:   true,
	})
}

// handleSubChunkRequest responds to the client's on-demand sub-chunk requests.
// Ground sub-chunk (index = bedrockworld.GroundSubChunkIndex) carries stone;
// all others return SuccessAllAir (no payload required).
func (l *Listener) handleSubChunkRequest(
	conn *minecraft.Conn,
	req *packet.SubChunkRequest,
) {
	dimensionWorld := l.worldForDimension(req.Dimension)
	entries := make([]protocol.SubChunkEntry, 0, len(req.Offsets))
	for _, off := range req.Offsets {
		subY := req.Position.Y() + int32(off[1])
		chunkX := req.Position.X() + int32(off[0])
		chunkZ := req.Position.Z() + int32(off[2])

		entry := protocol.SubChunkEntry{
			Offset: off,
		}
		sectionIndex := int(subY) - coreworld.WorldMinY/coreworld.SectionSize
		if sectionIndex < 0 || sectionIndex >= coreworld.SectionCount {
			entry.Result = protocol.SubChunkResultIndexOutOfBounds
		} else {
			chunk := dimensionWorld.Chunk(chunkX, chunkZ)
			var heightMap []int8
			entry.HeightMapType, heightMap = subChunkHeightMap(chunk, subY)
			entry.HeightMapData = protocol.Option(heightMap)
			entry.RenderHeightMapType = entry.HeightMapType
			entry.RenderHeightMapData = entry.HeightMapData
			section := chunk.Sections[sectionIndex]
			payload, err := l.encoder.EncodeSubChunk(section, subY)
			if err != nil {
				entry.Result = protocol.SubChunkResultChunkNotFound
			} else if len(payload) == 0 {
				entry.Result = protocol.SubChunkResultSuccessAllAir
			} else {
				blockActors, actorErr := encodeBedBlockActors(chunk, subY)
				if actorErr != nil {
					entry.Result = protocol.SubChunkResultChunkNotFound
				} else {
					entry.Result = protocol.SubChunkResultSuccess
					entry.RawPayload = protocol.Option(append(payload, blockActors...))
				}
			}
		}
		entries = append(entries, entry)
	}

	_ = conn.WritePacket(&packet.SubChunk{
		CacheEnabled:    false,
		Dimension:       req.Dimension,
		Position:        req.Position,
		SubChunkEntries: entries,
	})
}

// ── Identity helpers ──────────────────────────────────────────────────────────

// resolveUUID returns the player's canonical [16]byte UUID.
//
//   - Authenticated (online_mode=true): parse the Xbox-issued UUID from
//     identityStr, which is verified by gophertunnel.
//   - Unauthenticated (online_mode=false): generate a deterministic offline
//     UUID (UUID v3, GoCraft namespace + display name). Offline UUIDs use
//     variant bits that keep them in a different range than Xbox UUIDs,
//     preventing accidental collisions.
func resolveUUID(identityStr, displayName string, authenticated bool) ([16]byte, error) {
	if authenticated {
		return parseHexUUID(identityStr)
	}
	return offlineUUID(displayName), nil
}

// parseHexUUID parses a standard UUID string (with dashes) into [16]byte.
func parseHexUUID(s string) ([16]byte, error) {
	cleaned := strings.ReplaceAll(s, "-", "")
	if len(cleaned) != 32 {
		return [16]byte{}, fmt.Errorf("invalid UUID %q", s)
	}
	b, err := hex.DecodeString(cleaned)
	if err != nil {
		return [16]byte{}, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	var u [16]byte
	copy(u[:], b)
	return u, nil
}

// gocraftOfflineNS is the fixed namespace for offline UUID generation.
// Generated once (arbitrary); documented here so it is never changed:
// replacing it would change offline UUIDs for existing players.
//
// Value: SHA-256 of "GoCraft offline namespace" truncated to 16 bytes:
//
//	python3 -c "import hashlib; print(hashlib.sha256(b'GoCraft offline namespace').hexdigest()[:32])"
//	→ 5f3e2a1b4c7d8e9f0a1b2c3d4e5f6a7b
var gocraftOfflineNS = [16]byte{
	0x5f, 0x3e, 0x2a, 0x1b, 0x4c, 0x7d, 0x8e, 0x9f,
	0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f, 0x6a, 0x7b,
}

// offlineUUID generates a deterministic UUID v3 (MD5-based) for an
// unauthenticated player. The UUID is stable across server restarts for the
// same display name, and its version/variant bits distinguish it from Xbox
// UUIDs (which are version 4, random).
//
// This UUID must NOT be treated as a globally trusted identity — it is only
// reliable within the scope of a single server instance where collisions can
// be checked against the connected player list.
func offlineUUID(displayName string) [16]byte {
	h := md5.New()
	h.Write(gocraftOfflineNS[:])
	h.Write([]byte(displayName))
	digest := h.Sum(nil)

	var u [16]byte
	copy(u[:], digest)
	u[6] = (u[6] & 0x0f) | 0x30 // version 3
	u[8] = (u[8] & 0x3f) | 0x80 // variant 1 (RFC 4122)
	return u
}

// xuidLog returns the XUID for structured logging, or "<offline>" when
// unauthenticated to make clear the value is unverified.
func xuidLog(xuid string, authenticated bool) string {
	if authenticated {
		return xuid
	}
	return "<offline>"
}
