package server

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// dimensionWorldDirectories follows the server-panel-friendly Bukkit naming
// convention. Keeping each dimension beside the overworld makes all three
// persistent worlds visible at the Pterodactyl root.
func dimensionWorldDirectories(overworld string) (nether, end string) {
	clean := filepath.Clean(overworld)
	parent, name := filepath.Dir(clean), filepath.Base(clean)
	return filepath.Join(parent, name+"_nether"), filepath.Join(parent, name+"_end")
}

// prepareDimensionWorldDirectory copies an older Vanilla-style DIM folder to
// the new sibling directory the first time it is used. The source is retained,
// making the migration non-destructive and safe to roll back.
func prepareDimensionWorldDirectory(target, legacy string) error {
	if _, err := os.Stat(target); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking target dimension directory: %w", err)
	}
	if _, err := os.Stat(legacy); os.IsNotExist(err) {
		return os.MkdirAll(target, 0o755)
	} else if err != nil {
		return fmt.Errorf("checking legacy dimension directory: %w", err)
	}

	return filepath.WalkDir(legacy, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(legacy, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		return copyDimensionFile(path, destination, info.Mode().Perm())
	})
}

func copyDimensionFile(source, destination string, permissions fs.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, permissions)
	if err != nil {
		_ = input.Close()
		return err
	}
	_, copyErr := io.Copy(output, input)
	inputCloseErr := input.Close()
	outputCloseErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if inputCloseErr != nil {
		return inputCloseErr
	}
	return outputCloseErr
}
