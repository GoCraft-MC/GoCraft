package server

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// loadBedrockPackFromBytes writes raw pack bytes to a temp file and loads it
// via resource.ReadPath. Used to load in-memory generated .mcaddon files.
func loadBedrockPackFromBytes(data []byte) (*resource.Pack, error) {
	f, err := os.CreateTemp("", "gocraft-pack-*.mcaddon")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	f.Close()
	return resource.ReadPath(f.Name())
}

// loadBedrockPacks loads all Bedrock-format packs from the given file paths.
// Each path may be:
//   - a .mcpack or .zip file (single resource pack with manifest.json at root)
//   - a .mcaddon file  (zip archive containing resource_packs/ and/or
//     behavior_packs/ sub-directories, each holding one pack)
//
// All loaded packs are returned in a flat slice, in the order they appear
// across paths and within each .mcaddon.
func loadBedrockPacks(paths []string) ([]*resource.Pack, error) {
	var packs []*resource.Pack
	for _, path := range paths {
		loaded, err := loadOneBedrockPath(path)
		if err != nil {
			return nil, fmt.Errorf("resource pack %q: %w", path, err)
		}
		packs = append(packs, loaded...)
	}
	return packs, nil
}

// loadOneBedrockPath loads one or more packs from a single file path.
func loadOneBedrockPath(path string) ([]*resource.Pack, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".mcaddon" {
		return loadMcaddon(path)
	}
	pack, err := resource.ReadPath(path)
	if err != nil {
		return nil, err
	}
	return []*resource.Pack{pack}, nil
}

// loadMcaddon opens a .mcaddon zip archive and loads each sub-pack it contains.
//
// The .mcaddon format is a zip whose top-level entries are pack directories;
// each pack directory holds a manifest.json that identifies it as a resource
// pack or behavior pack.  Sub-packs are extracted to a temporary directory
// so gophertunnel can load them with resource.ReadPath.
//
// If the file has no nested pack layout (manifest.json at root), it is treated
// as a plain resource pack and loaded directly.
func loadMcaddon(path string) ([]*resource.Pack, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open mcaddon: %w", err)
	}
	defer r.Close()

	// Collect top-level directories that contain a manifest.json directly
	// inside them (i.e. exactly one path separator deep).
	type subDir struct{ name string }
	var subDirs []subDir
	seen := map[string]bool{}
	hasRootManifest := false

	for _, f := range r.File {
		if f.Name == "manifest.json" {
			hasRootManifest = true
			continue
		}
		// "PackDir/manifest.json"
		parts := strings.SplitN(f.Name, "/", 3)
		if len(parts) == 2 && parts[1] == "manifest.json" && !seen[parts[0]] {
			seen[parts[0]] = true
			subDirs = append(subDirs, subDir{parts[0]})
		}
	}

	if hasRootManifest || len(subDirs) == 0 {
		// Plain pack — load directly (gophertunnel handles the zip).
		pack, err := resource.ReadPath(path)
		if err != nil {
			return nil, err
		}
		return []*resource.Pack{pack}, nil
	}

	// Extract each sub-pack into its own temp directory so gophertunnel can
	// load it.  We use a single temp base directory per mcaddon file.
	tmpBase, err := os.MkdirTemp("", "gocraft-mcaddon-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	var packs []*resource.Pack
	for _, sd := range subDirs {
		dir := filepath.Join(tmpBase, sd.name)
		if err := extractZipDir(r, sd.name+"/", dir); err != nil {
			return nil, fmt.Errorf("extract sub-pack %q: %w", sd.name, err)
		}
		pack, err := resource.ReadPath(dir)
		if err != nil {
			return nil, fmt.Errorf("load sub-pack %q: %w", sd.name, err)
		}
		packs = append(packs, pack)
	}
	return packs, nil
}

// extractZipDir copies all zip entries whose name starts with prefix into dst,
// stripping the prefix from each target path.
func extractZipDir(r *zip.ReadCloser, prefix, dst string) error {
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(f.Name, prefix)
		if rel == "" || strings.HasSuffix(f.Name, "/") {
			continue // directory entry — created implicitly below
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			return readErr
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
