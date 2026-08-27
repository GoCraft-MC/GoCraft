package handler

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

func (registry *banRegistry) configure(path string, address bool) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.path, registry.address = path, address
	registry.entries = make(map[string]banRecord)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var records []banRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}
	for _, record := range records {
		value := record.Name
		if address {
			value = record.IP
		}
		if key := normalizeBanValue(value); key != "" {
			registry.entries[key] = record
		}
	}
	return nil
}

func (registry *banRegistry) add(value, source, reason string) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	record := banRecord{Created: time.Now().Format("2006-01-02 15:04:05 -0700"), Source: source, Expires: "forever", Reason: reason}
	if registry.address {
		record.IP = normalizeBanDisplay(value)
	} else {
		record.Name = normalizeBanDisplay(value)
	}
	registry.entries[normalizeBanValue(value)] = record
	return registry.saveLocked()
}

func (registry *banRegistry) remove(value string) (bool, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	key := normalizeBanValue(value)
	if _, ok := registry.entries[key]; !ok {
		return false, nil
	}
	delete(registry.entries, key)
	return true, registry.saveLocked()
}

func (registry *banRegistry) reason(value string) (string, bool) {
	registry.mu.RLock()
	record, ok := registry.entries[normalizeBanValue(value)]
	registry.mu.RUnlock()
	return record.Reason, ok
}
