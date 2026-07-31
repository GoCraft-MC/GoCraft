package world

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"

	"GoCraft/core/entity"
)

// World holds all chunks that have been loaded or generated, and delegates
// chunk creation to a Generator for positions not yet in cache.
//
// If a Storage is provided, World tries to load chunks from it before
// generating, and marks modified chunks dirty for flushing on Close.
//
// All public methods are safe for concurrent use.  Note that chunk content
// (Section.Set) is not protected by the world mutex; concurrent modification
// of the same chunk column from multiple goroutines is a known M8 limitation
// that will be addressed by per-chunk locking in a later milestone.
type World struct {
	mu        sync.RWMutex
	chunks    map[[2]int32]*Chunk
	inflight  map[[2]int32]chan struct{}
	generator Generator
	storage   Storage // nil = no persistence
	dirty     map[[2]int32]struct{}

	pregenQueue  chan [2]int32
	pregenQueued map[[2]int32]struct{}
	pregenStop   chan struct{}
	pregenWG     sync.WaitGroup
	closeOnce    sync.Once

	// Entities holds all non-player entities (mobs, dropped items, etc.).
	// It is initialised by New and safe for concurrent use.
	Entities *entity.Manager

	// Village entity spawning
	villagersEnabled bool
	spawnedVillages  map[[2]int]struct{}
	spawnedMu        sync.Mutex
	nextVillagerID   atomic.Int32
}

// New creates an empty world that generates chunks with gen on demand.
// storage may be nil for a generation-only world with no persistence.
func New(gen Generator, storage Storage, villagersEnabled bool) *World {
	w := &World{
		chunks:       make(map[[2]int32]*Chunk),
		inflight:     make(map[[2]int32]chan struct{}),
		generator:    gen,
		storage:      storage,
		dirty:        make(map[[2]int32]struct{}),
		pregenQueue:  make(chan [2]int32, 4096),
		pregenQueued: make(map[[2]int32]struct{}),
		pregenStop:       make(chan struct{}),
		Entities:         entity.NewManager(),
		villagersEnabled: villagersEnabled,
		spawnedVillages:  make(map[[2]int]struct{}),
	}
	const workers = 2
	w.pregenWG.Add(workers)
	for i := 0; i < workers; i++ {
		go w.pregenerationWorker()
	}
	return w
}

// Chunk returns the chunk at (x, z).  Lookup order:
//  1. In-memory cache — O(1), no I/O.
//  2. Storage backend (if present) — reads from disk.
//  3. Generator — creates a fresh chunk procedurally.
//
// The returned pointer is stable for the lifetime of the World.
func (w *World) Chunk(x, z int32) *Chunk {
	key := [2]int32{x, z}

	// Fast path: already cached. If a background worker is already generating
	// the same chunk, wait for it instead of doing the expensive work twice.
	w.mu.Lock()
	if c, ok := w.chunks[key]; ok {
		w.mu.Unlock()
		return c
	}
	if ready, ok := w.inflight[key]; ok {
		w.mu.Unlock()
		<-ready
		w.mu.RLock()
		c := w.chunks[key]
		w.mu.RUnlock()
		return c
	}
	ready := make(chan struct{})
	w.inflight[key] = ready
	w.mu.Unlock()

	// Try loading from storage before generating.
	var loaded *Chunk
	if w.storage != nil {
		c, err := w.storage.LoadChunk(x, z)
		if err != nil {
			slog.Warn("world: failed to load chunk from storage",
				"x", x, "z", z, "err", err)
		} else if c != nil {
			loaded = c
		}
	}

	// Fall back to generator if storage had nothing.
	if loaded == nil {
		loaded = w.generator.Generate(x, z)
	}

	w.mu.Lock()
	w.chunks[key] = loaded
	delete(w.inflight, key)
	close(ready)
	w.mu.Unlock()

	if w.villagersEnabled {
		if og, ok := w.generator.(*OverworldGenerator); ok {
			for _, vc := range og.VillageCentersNear(x, z) {
				vkey := [2]int{vc.WorldX, vc.WorldZ}
				w.spawnedMu.Lock()
				_, already := w.spawnedVillages[vkey]
				if !already {
					w.spawnedVillages[vkey] = struct{}{}
				}
				w.spawnedMu.Unlock()
				if already {
					continue
				}
				wellY := og.SurfaceHeight(vc.WorldX, vc.WorldZ)
				if wellY <= SeaLevel || wellY > 210 {
					continue
				}
				villagerCount := 3 + int(vc.Hash%3)
				for i := 0; i < villagerCount; i++ {
					id := w.nextVillagerID.Add(1) + 10_000_000
					var uuid [16]byte
					binary.BigEndian.PutUint64(uuid[:8], uint64(vc.WorldX)*0x9e3779b185ebca87^uint64(vc.WorldZ)*0xc2b2ae3d27d4eb4f)
					binary.BigEndian.PutUint64(uuid[8:], uint64(i+1)*0x6c62272e07bb0142^villageSalt)
					vx := float64(vc.WorldX) + float64(i-villagerCount/2)
					vz := float64(vc.WorldZ) + 1.5
					v := entity.New(int32(id), uuid, entity.TypeVillager, vx, float64(wellY+1), vz)
					w.Entities.Add(v)
				}
			}
		}
	}
	return loaded
}

// QueuePregeneration schedules every chunk in the square around (cx, cz),
// centre-out. Calls are cheap and duplicate coordinates are coalesced. The
// workers populate the normal in-memory cache, so a foreground player reaching
// a queued chunk does not wait for terrain generation.
func (w *World) QueuePregeneration(cx, cz, radius int32) int {
	if radius < 0 {
		return 0
	}
	queued := 0
	for ring := int32(0); ring <= radius; ring++ {
		for dx := -ring; dx <= ring; dx++ {
			for dz := -ring; dz <= ring; dz++ {
				if ring > 0 && abs32World(dx) != ring && abs32World(dz) != ring {
					continue
				}
				key := [2]int32{cx + dx, cz + dz}
				w.mu.Lock()
				_, loaded := w.chunks[key]
				_, loading := w.inflight[key]
				_, pending := w.pregenQueued[key]
				if loaded || loading || pending {
					w.mu.Unlock()
					continue
				}
				w.pregenQueued[key] = struct{}{}
				select {
				case w.pregenQueue <- key:
					queued++
				default:
					delete(w.pregenQueued, key)
				}
				w.mu.Unlock()
			}
		}
	}
	return queued
}

func (w *World) pregenerationWorker() {
	defer w.pregenWG.Done()
	for {
		select {
		case <-w.pregenStop:
			return
		case key := <-w.pregenQueue:
			w.Chunk(key[0], key[1])
			w.mu.Lock()
			delete(w.pregenQueued, key)
			w.mu.Unlock()
		}
	}
}

func abs32World(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// SurfaceY returns the highest non-air block at absolute x/z, loading or
// generating the containing chunk when necessary.
func (w *World) SurfaceY(x, z int) int {
	cx := int32(math.Floor(float64(x) / SectionSize))
	cz := int32(math.Floor(float64(z) / SectionSize))
	localX := x - int(cx)*SectionSize
	localZ := z - int(cz)*SectionSize
	return w.Chunk(cx, cz).HighestBlockY(localX, localZ)
}

// SetBlock places block at absolute world coordinates (x, y, z).
// The chunk is loaded or generated on demand.
// Out-of-bounds Y is silently ignored.
func (w *World) SetBlock(x, y, z int, block Block) {
	if y < WorldMinY || y > WorldMaxY {
		return
	}
	cx := int32(math.Floor(float64(x) / SectionSize))
	cz := int32(math.Floor(float64(z) / SectionSize))
	c := w.Chunk(cx, cz)

	relY := y - WorldMinY
	sIdx := relY / SectionSize
	lx := x - int(cx)*SectionSize
	ly := relY % SectionSize
	lz := z - int(cz)*SectionSize

	if c.Sections[sIdx] == nil {
		c.Sections[sIdx] = NewSection()
	}
	c.Sections[sIdx].Set(lx, ly, lz, block)
	if block.IsAir() && len(c.BlockEntities) > 0 {
		filtered := c.BlockEntities[:0]
		for _, entity := range c.BlockEntities {
			if entity.X == x && entity.Y == y && entity.Z == z {
				continue
			}
			filtered = append(filtered, entity)
		}
		c.BlockEntities = filtered
	}

	// Mark chunk dirty for persistence, but only when there is a storage
	// backend — avoids the extra lock acquisition otherwise.
	if w.storage != nil {
		w.mu.Lock()
		w.dirty[[2]int32{cx, cz}] = struct{}{}
		w.mu.Unlock()
	}
}

// GetBlock returns the canonical Block at absolute world coordinates.
// Returns Air for out-of-bounds Y or sections that have never been written to.
func (w *World) GetBlock(x, y, z int) Block {
	if y < WorldMinY || y > WorldMaxY {
		return Air
	}
	cx := int32(math.Floor(float64(x) / SectionSize))
	cz := int32(math.Floor(float64(z) / SectionSize))
	c := w.Chunk(cx, cz)

	relY := y - WorldMinY
	sIdx := relY / SectionSize
	lx := x - int(cx)*SectionSize
	ly := relY % SectionSize
	lz := z - int(cz)*SectionSize

	if c.Sections[sIdx] == nil {
		return Air
	}
	return c.Sections[sIdx].At(lx, ly, lz)
}

// Flush writes all dirty (modified) chunks to storage and clears the dirty set.
// Does nothing if no storage backend was provided.
func (w *World) Flush() error {
	if w.storage == nil {
		return nil
	}

	// Swap dirty set under the lock so writers can continue marking new dirty
	// chunks while we flush.
	w.mu.Lock()
	dirty := w.dirty
	w.dirty = make(map[[2]int32]struct{})
	w.mu.Unlock()

	for key := range dirty {
		w.mu.RLock()
		c, ok := w.chunks[key]
		w.mu.RUnlock()
		if !ok {
			continue // chunk was never inserted (shouldn't happen, but be safe)
		}
		if err := w.storage.SaveChunk(c); err != nil {
			return fmt.Errorf("world: saving chunk (%d,%d): %w", key[0], key[1], err)
		}
	}
	return nil
}

// Close flushes all dirty chunks and closes the storage backend.
// Should be called once on server shutdown.
func (w *World) Close() error {
	w.closeOnce.Do(func() {
		close(w.pregenStop)
		w.pregenWG.Wait()
	})
	if err := w.Flush(); err != nil {
		return err
	}
	if w.storage != nil {
		return w.storage.Close()
	}
	return nil
}

// LoadedCount returns the number of chunks currently held in memory.
func (w *World) LoadedCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.chunks)
}
