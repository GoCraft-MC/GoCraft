package anvil

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Anvil region file constants.
const (
	sectorSize    = 4096              // bytes per sector
	headerSectors = 2                 // header occupies sectors 0 and 1
	headerSize    = headerSectors * sectorSize
)

// regionCoords returns the region file coordinates for chunk (cx, cz).
// Each region covers a 32×32 grid of chunks.
func regionCoords(cx, cz int32) (rx, rz int32) {
	return cx >> 5, cz >> 5
}

// chunkIndex returns the 0-based index of chunk (cx, cz) within its region file.
func chunkIndex(cx, cz int32) int {
	return int((cx&31) + (cz&31)*32)
}

// regionPath builds the file path for a region file.
func regionPath(worldDir string, rx, rz int32) string {
	return filepath.Join(worldDir, "region", fmt.Sprintf("r.%d.%d.mca", rx, rz))
}

// readRawChunk reads the compressed chunk bytes (compression byte + payload)
// for chunk index idx from f.  Returns nil if the chunk is not present.
func readRawChunk(f *os.File, idx int) ([]byte, error) {
	// Each location entry is 4 bytes at offset idx*4 within the first 4096 bytes.
	var locBuf [4]byte
	if _, err := f.ReadAt(locBuf[:], int64(idx*4)); err != nil {
		return nil, fmt.Errorf("anvil: reading location entry %d: %w", idx, err)
	}
	// Unpack: 3-byte sector offset | 1-byte sector count.
	sectorOffset := uint32(locBuf[0])<<16 | uint32(locBuf[1])<<8 | uint32(locBuf[2])
	sectorCount := locBuf[3]
	if sectorOffset < uint32(headerSectors) || sectorCount == 0 {
		return nil, nil // chunk not present
	}

	byteOffset := int64(sectorOffset) * sectorSize

	// First 4 bytes at the chunk data position are the data length (including
	// the compression byte), followed by 1 compression byte.
	var hdr [5]byte
	if _, err := f.ReadAt(hdr[:], byteOffset); err != nil {
		return nil, fmt.Errorf("anvil: reading chunk data header at %d: %w", byteOffset, err)
	}
	dataLen := int(binary.BigEndian.Uint32(hdr[:4]))
	if dataLen < 1 {
		return nil, fmt.Errorf("anvil: invalid chunk data length %d at idx %d", dataLen, idx)
	}

	// raw = [compression_byte, compressed_data…]
	raw := make([]byte, dataLen)
	if _, err := f.ReadAt(raw, byteOffset+4); err != nil {
		return nil, fmt.Errorf("anvil: reading chunk data at %d: %w", byteOffset, err)
	}
	return raw, nil
}

// decompressChunk decompresses raw chunk bytes (format: [compression][data…])
// and returns an io.Reader for the uncompressed NBT stream.
func decompressChunk(raw []byte) (io.ReadCloser, error) {
	if len(raw) < 1 {
		return nil, fmt.Errorf("anvil: empty chunk data")
	}
	compression := raw[0]
	payload := raw[1:]

	switch compression {
	case 1: // GZip
		gr, err := gzip.NewReader(bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("anvil: gzip reader: %w", err)
		}
		return gr, nil
	case 2: // Zlib (deflate)
		zr, err := zlib.NewReader(bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("anvil: zlib reader: %w", err)
		}
		return zr, nil
	case 3: // Uncompressed
		return io.NopCloser(bytes.NewReader(payload)), nil
	default:
		return nil, fmt.Errorf("anvil: unknown compression type %d", compression)
	}
}

// loadChunkFromRegion loads and decodes the chunk at (cx, cz) from the region
// file in worldDir.  Returns (nil, nil) if the region file or chunk is absent.
func loadChunkFromRegion(worldDir string, cx, cz int32) (map[string]Tag, error) {
	rx, rz := regionCoords(cx, cz)
	path := regionPath(worldDir, rx, rz)

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("anvil: opening region %s: %w", path, err)
	}
	defer f.Close()

	idx := chunkIndex(cx, cz)
	raw, err := readRawChunk(f, idx)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	rc, err := decompressChunk(raw)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return ReadRootCompound(rc)
}

// saveChunkToRegion writes c to the appropriate region file in worldDir.
//
// Algorithm:
//  1. Read all raw chunk data from the existing region file (if present).
//  2. Compress and insert the new chunk data at the slot for (cx, cz).
//  3. Write a completely new region file to a temp path, then rename it
//     atomically over the original.  This prevents corruption if the process
//     is killed mid-write.
func saveChunkToRegion(worldDir string, cx, cz int32, nbt []byte) error {
	rx, rz := regionCoords(cx, cz)
	path := regionPath(worldDir, rx, rz)

	// Ensure the region directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("anvil: creating region dir: %w", err)
	}

	// Snapshot existing raw chunks from disk (to preserve chunks we don't own).
	rawChunks := make([][]byte, 1024) // indexed by chunkIndex

	if f, err := os.Open(path); err == nil {
		for i := 0; i < 1024; i++ {
			// Skip the slot we're about to overwrite.
			thisCX := rx*32 + int32(i%32)
			thisCZ := rz*32 + int32(i/32)
			if thisCX == cx && thisCZ == cz {
				continue
			}
			raw, _ := readRawChunk(f, i)
			rawChunks[i] = raw // nil is fine — means absent
		}
		f.Close()
	}

	// Compress the new chunk NBT using zlib (compression type 2).
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(nbt); err != nil {
		return fmt.Errorf("anvil: compressing chunk NBT: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("anvil: finalising zlib: %w", err)
	}
	// raw format: [compression_byte=2] + [compressed data]
	newRaw := make([]byte, 1+compressed.Len())
	newRaw[0] = 2
	copy(newRaw[1:], compressed.Bytes())
	rawChunks[chunkIndex(cx, cz)] = newRaw

	// Write the region file to a temp path first.
	tmpPath := path + ".tmp"
	if err := writeRegionFile(tmpPath, rawChunks); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Atomic rename over the original.
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("anvil: renaming temp region file: %w", err)
	}
	return nil
}

// writeRegionFile builds a complete region file from rawChunks and writes it
// to path.  rawChunks[i] holds the raw bytes (compression byte + payload) for
// chunk index i, or nil if the chunk is absent.
func writeRegionFile(path string, rawChunks [][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("anvil: creating region file %s: %w", path, err)
	}
	defer f.Close()

	// Reserve space for the 8192-byte header.
	header := make([]byte, headerSize)
	if _, err := f.Write(header); err != nil {
		return fmt.Errorf("anvil: writing header placeholder: %w", err)
	}

	// Write chunk data sectors; track each chunk's sector offset.
	type location struct {
		offset uint32 // in sectors
		count  byte
	}
	locs := make([]location, 1024)
	currentSector := uint32(headerSectors) // start after header

	for i, raw := range rawChunks {
		if raw == nil {
			continue
		}

		// Chunk data on disk: 4-byte length | raw bytes (compression + payload).
		// Length field counts the raw bytes (including the compression byte).
		chunkLen := uint32(len(raw))
		var chunk bytes.Buffer
		var lenBuf [4]byte
		binary.BigEndian.PutUint32(lenBuf[:], chunkLen)
		chunk.Write(lenBuf[:])
		chunk.Write(raw)

		// Pad to a sector boundary.
		sectorCount := (chunk.Len() + sectorSize - 1) / sectorSize
		padded := make([]byte, sectorCount*sectorSize)
		copy(padded, chunk.Bytes())

		if _, err := f.Write(padded); err != nil {
			return fmt.Errorf("anvil: writing chunk %d data: %w", i, err)
		}

		locs[i] = location{offset: currentSector, count: byte(sectorCount)}
		currentSector += uint32(sectorCount)
	}

	// Write the completed header.
	for i, loc := range locs {
		off := i * 4
		header[off] = byte(loc.offset >> 16)
		header[off+1] = byte(loc.offset >> 8)
		header[off+2] = byte(loc.offset)
		header[off+3] = loc.count
		// Timestamps (second 4096-byte block) remain zero.
	}
	if _, err := f.WriteAt(header, 0); err != nil {
		return fmt.Errorf("anvil: writing region header: %w", err)
	}

	return nil
}
