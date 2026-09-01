package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

// EnsureDirectory creates the plugin drop directory when needed.
//
// There is deliberately no default directory constant here. The default lives
// once, in config.PluginsConfig, and callers pass what the admin configured:
// a constant in this package would be a second copy of that value, free to
// drift, and it is what let the server create a hardcoded "plugins" while
// scanning somewhere else entirely.
func EnsureDirectory(directory string) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create plugins directory: %w", err)
	}
	return nil
}

// ScanBundles reads manifests without starting any plugin runtime.
func ScanBundles(directory string) ([]Bundle, error) {
	if err := EnsureDirectory(directory); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("scan plugins directory: %w", err)
	}
	var bundles []Bundle
	ids := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".gcpkg") {
			continue
		}
		opened, err := gcpkg.Open(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		if previous, duplicate := ids[opened.Manifest.ID]; duplicate {
			return nil, fmt.Errorf("plugin %s is present in both %s and %s", opened.Manifest.ID, previous, entry.Name())
		}
		ids[opened.Manifest.ID] = entry.Name()
		bundles = append(bundles, Bundle{Bundle: opened})
	}
	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].Manifest.ID < bundles[j].Manifest.ID
	})
	return bundles, nil
}
