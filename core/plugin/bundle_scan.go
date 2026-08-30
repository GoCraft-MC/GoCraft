package plugin

import (
	"archive/zip"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"GoCraft/core/command"
)

const DefaultDirectory = "plugins"

// EnsureDirectory creates the server's plugin drop directory when needed.
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
		bundle, err := OpenBundle(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		if previous, duplicate := ids[bundle.Manifest.ID]; duplicate {
			return nil, fmt.Errorf("plugin %s is present in both %s and %s", bundle.Manifest.ID, previous, entry.Name())
		}
		ids[bundle.Manifest.ID] = entry.Name()
		bundles = append(bundles, bundle)
	}
	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].Manifest.ID < bundles[j].Manifest.ID
	})
	return bundles, nil
}

// OpenBundle validates one archive and decodes its root plugin.toml.
func OpenBundle(bundlePath string) (Bundle, error) {
	archive, err := zip.OpenReader(bundlePath)
	if err != nil {
		return Bundle{}, fmt.Errorf("open plugin bundle %s: %w", bundlePath, err)
	}
	defer archive.Close()
	var manifestFile *zip.File
	for _, file := range archive.File {
		name := strings.ReplaceAll(file.Name, "\\", "/")
		clean := path.Clean(name)
		if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
			return Bundle{}, fmt.Errorf("plugin bundle %s contains unsafe path %q", bundlePath, file.Name)
		}
		if clean != ManifestFileName {
			continue
		}
		if manifestFile != nil {
			return Bundle{}, fmt.Errorf("plugin bundle %s contains duplicate plugin.toml", bundlePath)
		}
		manifestFile = file
	}
	if manifestFile == nil {
		return Bundle{}, fmt.Errorf("plugin bundle %s has no root plugin.toml", bundlePath)
	}
	if manifestFile.UncompressedSize64 > maximumManifestSize {
		return Bundle{}, fmt.Errorf("plugin bundle %s: plugin.toml exceeds %d bytes", bundlePath, maximumManifestSize)
	}
	reader, err := manifestFile.Open()
	if err != nil {
		return Bundle{}, fmt.Errorf("open %s plugin.toml: %w", bundlePath, err)
	}
	manifest, decodeErr := DecodeManifest(reader)
	closeErr := reader.Close()
	if decodeErr != nil {
		return Bundle{}, fmt.Errorf("plugin bundle %s: %w", bundlePath, decodeErr)
	}
	if closeErr != nil {
		return Bundle{}, fmt.Errorf("close %s plugin.toml: %w", bundlePath, closeErr)
	}
	var commands *command.Root
	if manifest.CommandTree != "" {
		encoded, err := readBundleEntry(archive.File, manifest.CommandTree, maximumCommandTreeSize)
		if err != nil {
			return Bundle{}, fmt.Errorf("plugin bundle %s: %w", bundlePath, err)
		}
		tree, err := command.DecodeTree(encoded)
		if err != nil {
			return Bundle{}, fmt.Errorf("plugin bundle %s command tree: %w", bundlePath, err)
		}
		commands = &tree
	}
	absolutePath, err := filepath.Abs(bundlePath)
	if err != nil {
		return Bundle{}, fmt.Errorf("resolve plugin bundle %s: %w", bundlePath, err)
	}
	return Bundle{Path: absolutePath, Manifest: manifest, Commands: commands}, nil
}
