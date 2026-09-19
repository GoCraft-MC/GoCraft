package permission

// Store persists and retrieves the permission Document.
// Implementations must be safe for use from a single goroutine at a time;
// Manager serialises all writes behind its own mutex.
type Store interface {
	// Load returns the stored Document. If no document exists yet the
	// implementation must bootstrap one from DefaultDocument and persist it.
	Load() (Document, error)
	// Save atomically persists doc.
	Save(doc Document) error
	// Close releases any resources held by the store (database connections,
	// open file handles, etc.). The store must not be used after Close returns.
	Close() error
}

// noopStore is used by NewMemory — changes are never persisted.
type noopStore struct{}

func (noopStore) Load() (Document, error) { return DefaultDocument(), nil }
func (noopStore) Save(Document) error     { return nil }
func (noopStore) Close() error            { return nil }
