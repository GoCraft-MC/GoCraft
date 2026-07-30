package world

import (
	"fmt"
	"log/slog"
	"math"
	"sync"

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
	generator Generator
	storage   Storage // nil = no persistence
	dirty     map[[2]int32]struct{}

	// Entities holds all non-player entities (mobs, dropped items, etc.).
	// It is initialised by New and safe for concurrent use.
	Entities *entity.Manager
}

// New creates an empty world that generates chunks with gen on demand.
// storage may be nil for a generation-only world with no persistence.
func New(gen Generator, storage Storage) *World {
	return &World{
		chunks:    make(map[[2]int32]*Chunk),
		generator: gen,
		storage:   storage,
		dirty:     make(map[[2]int32]struct{}),
		Entities:  entity.NewManager(),
	}
}

// Chunk returns the chunk at (x, z).  Lookup order:
//  1. In-memory cache — O(1), no I/O.
//  2. Storage backend (if present) — reads from disk.
//  3. Generator — creates a fresh chunk procedurally.
//
// The returned pointer is stable for the lifetime of the World.
func (w *World) Chunk(x, z int32) *Chunk {
	key := [2]int32{x, z}

	// Fast path: already cached.
	w.mu.RLock()
	if c, ok := w.chunks[key]; ok {
		w.mu.RUnlock()
		return c
	}
	w.mu.RUnlock()

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

	// Insert under write lock; discard if another goroutine raced and won.
	w.mu.Lock()
	defer w.mu.Unlock()
	if existing, ok := w.chunks[key]; ok {
		return existing
	}
	w.chunks[key] = loaded
	return loaded
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
