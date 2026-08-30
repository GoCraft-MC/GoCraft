package plugin

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"strings"
)

const maximumCommandTreeSize = 4 << 20

func readBundleEntry(files []*zip.File, name string, limit uint64) ([]byte, error) {
	var match *zip.File
	for _, file := range files {
		cleaned := path.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if cleaned != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("duplicate entry %s", name)
		}
		match = file
	}
	if match == nil {
		return nil, fmt.Errorf("missing entry %s", name)
	}
	if match.UncompressedSize64 > limit {
		return nil, fmt.Errorf("entry %s exceeds %d bytes", name, limit)
	}
	reader, err := match.Open()
	if err != nil {
		return nil, fmt.Errorf("open entry %s: %w", name, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	closeErr := reader.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read entry %s: %w", name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close entry %s: %w", name, closeErr)
	}
	if uint64(len(data)) > limit {
		return nil, fmt.Errorf("entry %s expanded beyond its size limit", name)
	}
	return data, nil
}
