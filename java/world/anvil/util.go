package anvil

import (
	"bytes"
	"compress/zlib"
	"os"
)

// zlibCompress compresses data with zlib (deflate) and returns the result.
func zlibCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// openRegionFile opens a region file for reading.
// Returns (nil, err) if the file does not exist or cannot be opened.
func openRegionFile(path string) (*os.File, error) {
	return os.Open(path)
}

// renameFile renames src to dst, replacing dst if it exists.
func renameFile(src, dst string) error {
	return os.Rename(src, dst)
}
