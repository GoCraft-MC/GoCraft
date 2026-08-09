package serverlog

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestOpenRotatesLatestAndCreatesFreshLog(t *testing.T) {
	directory := t.TempDir()
	previous := "previous server output\nsecond line\n"
	latestPath := filepath.Join(directory, latestLogName)
	if err := os.WriteFile(latestPath, []byte(previous), 0o644); err != nil {
		t.Fatal(err)
	}
	modTime := time.Date(2026, time.August, 9, 14, 30, 0, 0, time.Local)
	if err := os.Chtimes(latestPath, modTime, modTime); err != nil {
		t.Fatal(err)
	}

	latest, err := Open(directory, 10)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if _, err := latest.WriteString("current server output\n"); err != nil {
		t.Fatal(err)
	}
	if err := latest.Close(); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(directory, "2026-08-09-1.log.gz")
	if got := readGzipFile(t, archivePath); got != previous {
		t.Fatalf("archive content = %q, want %q", got, previous)
	}
	current, err := os.ReadFile(latestPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(current), "current server output\n"; got != want {
		t.Fatalf("latest.log content = %q, want %q", got, want)
	}
}

func TestOpenNumbersMultipleArchivesOnSameDay(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "2026-08-09-1.log.gz"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	latestPath := filepath.Join(directory, latestLogName)
	if err := os.WriteFile(latestPath, []byte("next run"), 0o644); err != nil {
		t.Fatal(err)
	}
	modTime := time.Date(2026, time.August, 9, 16, 0, 0, 0, time.Local)
	if err := os.Chtimes(latestPath, modTime, modTime); err != nil {
		t.Fatal(err)
	}

	latest, err := Open(directory, 10)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := latest.Close(); err != nil {
		t.Fatal(err)
	}

	if got := readGzipFile(t, filepath.Join(directory, "2026-08-09-2.log.gz")); got != "next run" {
		t.Fatalf("second archive content = %q, want %q", got, "next run")
	}
}

func TestOpenPrunesOldestArchives(t *testing.T) {
	directory := t.TempDir()
	baseTime := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.Local)
	for index := 1; index <= 3; index++ {
		name := filepath.Join(directory, fmt.Sprintf("2026-08-%02d-1.log.gz", index))
		if err := os.WriteFile(name, []byte("archive"), 0o644); err != nil {
			t.Fatal(err)
		}
		modTime := baseTime.Add(time.Duration(index) * time.Hour)
		if err := os.Chtimes(name, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}
	latestPath := filepath.Join(directory, latestLogName)
	if err := os.WriteFile(latestPath, []byte("previous latest"), 0o644); err != nil {
		t.Fatal(err)
	}
	latestModTime := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.Local)
	if err := os.Chtimes(latestPath, latestModTime, latestModTime); err != nil {
		t.Fatal(err)
	}

	latest, err := Open(directory, 2)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := latest.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var archives []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".log.gz") {
			archives = append(archives, entry.Name())
		}
	}
	slices.Sort(archives)
	want := []string{"2026-08-03-1.log.gz", "2026-08-09-1.log.gz"}
	if !slices.Equal(archives, want) {
		t.Fatalf("archives = %v, want %v", archives, want)
	}
}

func TestOpenDoesNotArchiveEmptyLatest(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, latestLogName), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	latest, err := Open(directory, 10)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := latest.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != latestLogName {
		t.Fatalf("log directory entries = %v, want only %s", entries, latestLogName)
	}
}

func TestOpenRejectsInvalidConfiguration(t *testing.T) {
	if _, err := Open("", 10); err == nil {
		t.Fatal("Open() with empty directory returned nil error")
	}
	if _, err := Open(t.TempDir(), 0); err == nil {
		t.Fatal("Open() with zero archive cap returned nil error")
	}
}

func readGzipFile(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
