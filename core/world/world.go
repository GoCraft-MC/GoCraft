package world

import (
	"math"
	"sync"
)

// World holds all chunks that have been loaded or generated, and delegates
// chunk creation to a Generator for positions not yet in cache.
//
// All methods are safe for concurrent use.
type World struct {
	mu        sync.RWMutex
	chunks    map[[2]int32]*Chunk
	generator Generator
}

// New creates an empty world that generates chunks with gen on demand.
func New(gen Generator) *World {
	return &World{
		chunks:    make(map[[2]int32]*Chunk),
		generator: gen,
	}
}

// Chunk returns the chunk at (x, z), generating it if it has not been loaded yet.
func (w *World) Chunk(x, z int32) *Chunk {
	key := [2]int32{x, z}

	w.mu.RLock()
	if c, ok := w.chunks[key]; ok {
		w.mu.RUnlock()
		return c
	}
	w.mu.RUnlock()

	c := w.generator.Generate(x, z)

	w.mu.Lock()
	defer w.mu.Unlock()
	// Check again — another goroutine may have generated the same chunk.
	if existing, ok := w.chunks[key]; ok {
		return existing
	}
	w.chunks[key] = c
	return c
}

// SetBlock places block at absolute world coordinates (x, y, z).
// The chunk is loaded or generated on demand; out-of-bounds Y is silently ignored.
// If the target section does not exist yet it is created as all-air before the
// block is written, so the rest of the section remains correct.
//
// SetBlock is NOT safe for concurrent calls on the same chunk column.
// Single-goroutine per-player access is the norm in M8; a per-chunk mutex will
// be added when multiple goroutines need to modify the same column.
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
}

// GetBlock returns the canonical Block at absolute world coordinates.
// Returns Air for out-of-bounds Y or ungenerated sections.
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

// LoadedCount returns the number of chunks currently held in memory.
func (w *World) LoadedCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.chunks)
}
