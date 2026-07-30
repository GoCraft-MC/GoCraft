package world

import "sync"

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

// LoadedCount returns the number of chunks currently held in memory.
func (w *World) LoadedCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.chunks)
}
