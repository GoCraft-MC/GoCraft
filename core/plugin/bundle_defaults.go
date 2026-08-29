package plugin

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	defaultConfigPrefix    = "config/"
	maximumDefaultFiles    = 128
	maximumDefaultFileSize = 1 << 20
	maximumDefaultBytes    = 16 << 20
)

func extractBundleDefaults(root *os.Root, files []*zip.File) error {
	count, total := 0, uint64(0)
	for _, file := range files {
		name := path.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if !strings.HasPrefix(name, defaultConfigPrefix) || file.FileInfo().IsDir() {
			continue
		}
		if file.Mode()&os.ModeType != 0 {
			return fmt.Errorf("%s is not a regular file", file.Name)
		}
		count++
		total += file.UncompressedSize64
		if count > maximumDefaultFiles || file.UncompressedSize64 > maximumDefaultFileSize || total > maximumDefaultBytes {
			return fmt.Errorf("configuration defaults exceed extraction limits")
		}
		name = filepath.FromSlash(strings.TrimPrefix(name, defaultConfigPrefix))
		if err := root.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			return fmt.Errorf("create %s parent: %w", name, err)
		}
		if err := extractDefault(root, name, file); err != nil {
			return err
		}
	}
	return nil
}

func extractDefault(root *os.Root, name string, source *zip.File) error {
	reader, err := source.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", source.Name, err)
	}
	defer reader.Close()
	target, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	written, copyErr := io.Copy(target, io.LimitReader(reader, maximumDefaultFileSize+1))
	if written > maximumDefaultFileSize {
		copyErr = fmt.Errorf("file exceeds extraction limit")
	}
	closeErr := target.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		_ = root.Remove(name)
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}
