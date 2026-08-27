package handler

import (
	"encoding/json"
	"os"
	"sort"
)

func (registry *banRegistry) values() []string {
	registry.mu.RLock()
	values := make([]string, 0, len(registry.entries))
	for _, record := range registry.entries {
		value := record.Name
		if registry.address {
			value = record.IP
		}
		values = append(values, value)
	}
	registry.mu.RUnlock()
	sort.Strings(values)
	return values
}

func (registry *banRegistry) saveLocked() error {
	records := make([]banRecord, 0, len(registry.entries))
	for _, record := range registry.entries {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return normalizeBanValue(records[i].Name+records[i].IP) < normalizeBanValue(records[j].Name+records[j].IP)
	})
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil || registry.path == "" {
		return err
	}
	return os.WriteFile(registry.path, append(data, '\n'), 0o644)
}
