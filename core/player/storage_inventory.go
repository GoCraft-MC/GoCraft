package player

import "sync"

// StorageInventory owns vehicle contents independently of any viewer's screen.
// Transactions read current slots under the lock, preventing stale screens
// from duplicating items when multiple players use the same inventory.
type StorageInventory struct {
	mu      sync.Mutex
	slots   []ItemStack
	removed bool
}

func NewStorageInventory(size int) *StorageInventory {
	return &StorageInventory{slots: make([]ItemStack, size)}
}

func (s *StorageInventory) Snapshot() []ItemStack {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ItemStack(nil), s.slots...)
}

func (s *StorageInventory) Update(change func([]ItemStack) []ItemStack) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.removed {
		return false
	}
	updated := change(append([]ItemStack(nil), s.slots...))
	copy(s.slots, updated)
	return true
}

// Drain removes the inventory exactly once, even while a screen is open.
func (s *StorageInventory) Drain() []ItemStack {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	slots := s.slots
	s.slots, s.removed = nil, true
	return slots
}
