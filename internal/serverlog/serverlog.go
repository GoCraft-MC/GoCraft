// Package serverlog manages GoCraft's Paper-style log files.
package serverlog

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const latestLogName = "latest.log"

// Open rotates an existing latest.log, prunes old compressed logs, and opens a
// fresh latest.log for the current server run. The caller must close the file.
func Open(directory string, maxArchives int) (*os.File, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, errors.New("log directory cannot be empty")
	}
	if maxArchives < 1 {
		return nil, fmt.Errorf("max log archives must be at least 1, got %d", maxArchives)
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	if err := rotateLatest(directory); err != nil {
		return nil, err
	}
	if err := pruneArchives(directory, maxArchives); err != nil {
		return nil, err
	}

	path := filepath.Join(directory, latestLogName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open latest log: %w", err)
	}
	return file, nil
}

func rotateLatest(directory string) error {
	latestPath := filepath.Join(directory, latestLogName)
	info, err := os.Stat(latestPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect latest log: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("latest log path is a directory: %s", latestPath)
	}
	if info.Size() == 0 {
		if err := os.Remove(latestPath); err != nil {
			return fmt.Errorf("remove empty latest log: %w", err)
		}
		return nil
	}

	source, err := os.Open(latestPath)
	if err != nil {
		return fmt.Errorf("open previous latest log: %w", err)
	}

	date := info.ModTime().Format("2006-01-02")
	archivePath, archive, err := createArchive(directory, date)
	if err != nil {
		_ = source.Close()
		return err
	}

	compressed := gzip.NewWriter(archive)
	compressed.Name = latestLogName
	compressed.ModTime = info.ModTime()

	_, copyErr := io.Copy(compressed, source)
	gzipCloseErr := compressed.Close()
	archiveCloseErr := archive.Close()
	sourceCloseErr := source.Close()
	if err := errors.Join(copyErr, gzipCloseErr, archiveCloseErr, sourceCloseErr); err != nil {
		_ = os.Remove(archivePath)
		return fmt.Errorf("compress previous latest log: %w", err)
	}

	if err := os.Remove(latestPath); err != nil {
		_ = os.Remove(archivePath)
		return fmt.Errorf("remove rotated latest log: %w", err)
	}
	return nil
}

func createArchive(directory, date string) (string, *os.File, error) {
	for index := 1; ; index++ {
		name := fmt.Sprintf("%s-%d.log.gz", date, index)
		path := filepath.Join(directory, name)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", nil, fmt.Errorf("create rotated log %s: %w", name, err)
		}
		return path, file, nil
	}
}

type archiveFile struct {
	name    string
	modTime int64
}

func pruneArchives(directory string, maxArchives int) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("list log archives: %w", err)
	}

	archives := make([]archiveFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log.gz") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect log archive %s: %w", entry.Name(), err)
		}
		archives = append(archives, archiveFile{name: entry.Name(), modTime: info.ModTime().UnixNano()})
	}

	sort.Slice(archives, func(i, j int) bool {
		if archives[i].modTime == archives[j].modTime {
			return archives[i].name < archives[j].name
		}
		return archives[i].modTime < archives[j].modTime
	})

	removeCount := len(archives) - maxArchives
	if removeCount < 0 {
		removeCount = 0
	}
	for _, archive := range archives[:removeCount] {
		path := filepath.Join(directory, archive.name)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove old log archive %s: %w", archive.name, err)
		}
	}
	return nil
}
