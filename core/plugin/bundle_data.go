package plugin

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
)

func prepareBundleData(bundle Bundle) (Bundle, error) {
	if bundle.Path == "" {
		return bundle, nil
	}
	directory := filepath.Join(filepath.Dir(bundle.Path), bundle.Manifest.ID)
	if err := createDataDirectory(directory); err != nil {
		return Bundle{}, fmt.Errorf("plugin %s data directory: %w", bundle.Manifest.ID, err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return Bundle{}, fmt.Errorf("open plugin %s data directory: %w", bundle.Manifest.ID, err)
	}
	defer root.Close()
	archive, err := zip.OpenReader(bundle.Path)
	if err != nil {
		return Bundle{}, fmt.Errorf("open plugin %s defaults: %w", bundle.Manifest.ID, err)
	}
	defer archive.Close()
	if err := extractBundleDefaults(root, archive.File); err != nil {
		return Bundle{}, fmt.Errorf("plugin %s defaults: %w", bundle.Manifest.ID, err)
	}
	bundle.DataDirectory = directory
	return bundle, nil
}

func createDataDirectory(directory string) error {
	info, err := os.Lstat(directory)
	if os.IsNotExist(err) {
		return os.Mkdir(directory, 0o755)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("path is not a regular directory")
	}
	return nil
}
