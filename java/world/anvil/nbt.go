// Package anvil implements Minecraft Anvil region file I/O for GoCraft.
//
// Anvil is the on-disk format for Minecraft Java Edition worlds.  Each
// region file (r.X.Z.mca) covers 32×32 chunk columns; chunk data is stored
// as zlib-compressed NBT.
//
// This package is Java-edition-specific and lives under java/world/anvil.
// A future bedrock/world/leveldb package will provide equivalent I/O for
// Bedrock Edition without touching this package.
package anvil

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
)

// ── Tag types ─────────────────────────────────────────────────────────────────

type tagType byte

const (
	tagEnd      tagType = 0
	tagByte     tagType = 1
	tagShort    tagType = 2
	tagInt      tagType = 3
	tagLong     tagType = 4
	tagFloat    tagType = 5
	tagDouble   tagType = 6
	tagByteArr  tagType = 7
	tagString   tagType = 8
	tagList     tagType = 9
	tagCompound tagType = 10
	tagIntArr   tagType = 11
	tagLongArr  tagType = 12
)

// ── Tag value ─────────────────────────────────────────────────────────────────

// Tag is a parsed NBT value.  Only the field corresponding to typ is valid.
type Tag struct {
	typ      tagType
	byteV    int8
	shortV   int16
	intV     int32
	longV    int64
	floatV   float32
	doubleV  float64
	bytesV   []byte
	strV     string
	listElem tagType
	listV    []Tag
	compound map[string]Tag
	intsV    []int32
	longsV   []int64
}

// Type returns the NBT tag type.
func (t Tag) Type() tagType { return t.typ }

// Byte returns the TAG_Byte payload as int8, or 0.
func (t Tag) Byte() int8 { return t.byteV }

// Short returns the TAG_Short payload, or 0.
func (t Tag) Short() int16 { return t.shortV }

// Int returns the TAG_Int payload, or 0.
func (t Tag) Int() int32 { return t.intV }

// Long returns the TAG_Long payload, or 0.
func (t Tag) Long() int64 { return t.longV }

// Str returns the TAG_String payload, or "".
func (t Tag) Str() string { return t.strV }

// List returns the TAG_List elements, or nil.
func (t Tag) List() []Tag { return t.listV }

// LongArray returns the TAG_Long_Array elements, or nil.
func (t Tag) LongArray() []int64 { return t.longsV }

// Get returns the named entry from a TAG_Compound, or a zero Tag.
func (t Tag) Get(name string) Tag {
	if t.compound == nil {
		return Tag{}
	}
	return t.compound[name]
}

// ── Reader ────────────────────────────────────────────────────────────────────

// ReadRootCompound reads a standard (file-format) named root TAG_Compound.
// The root name (usually "") is discarded; the returned map contains the
// compound's named children.
func ReadRootCompound(r io.Reader) (map[string]Tag, error) {
	_, tag, err := readNamed(r)
	if err != nil {
		return nil, err
	}
	if tag.typ != tagCompound {
		return nil, fmt.Errorf("nbt: expected root TAG_Compound, got type %d", tag.typ)
	}
	return tag.compound, nil
}

// readNamed reads a type byte, a UTF-8 name, and a payload.
// Returns a zero Tag (typ == tagEnd) and empty name for TAG_End entries.
func readNamed(r io.Reader) (name string, tag Tag, err error) {
	var typBuf [1]byte
	if _, err := io.ReadFull(r, typBuf[:]); err != nil {
		return "", Tag{}, fmt.Errorf("nbt: reading tag type: %w", err)
	}
	typ := tagType(typBuf[0])
	if typ == tagEnd {
		return "", Tag{typ: tagEnd}, nil
	}
	name, err = readMUTF8(r)
	if err != nil {
		return "", Tag{}, fmt.Errorf("nbt: reading tag name: %w", err)
	}
	tag, err = readPayload(r, typ)
	return name, tag, err
}

// readPayload reads the payload for a tag of the given type.
func readPayload(r io.Reader, typ tagType) (Tag, error) {
	t := Tag{typ: typ}
	var buf [8]byte

	switch typ {
	case tagByte:
		if _, err := io.ReadFull(r, buf[:1]); err != nil {
			return t, fmt.Errorf("nbt: TAG_Byte: %w", err)
		}
		t.byteV = int8(buf[0])

	case tagShort:
		if _, err := io.ReadFull(r, buf[:2]); err != nil {
			return t, fmt.Errorf("nbt: TAG_Short: %w", err)
		}
		t.shortV = int16(binary.BigEndian.Uint16(buf[:2]))

	case tagInt:
		if _, err := io.ReadFull(r, buf[:4]); err != nil {
			return t, fmt.Errorf("nbt: TAG_Int: %w", err)
		}
		t.intV = int32(binary.BigEndian.Uint32(buf[:4]))

	case tagLong:
		if _, err := io.ReadFull(r, buf[:8]); err != nil {
			return t, fmt.Errorf("nbt: TAG_Long: %w", err)
		}
		t.longV = int64(binary.BigEndian.Uint64(buf[:8]))

	case tagFloat:
		if _, err := io.ReadFull(r, buf[:4]); err != nil {
			return t, fmt.Errorf("nbt: TAG_Float: %w", err)
		}
		t.floatV = math.Float32frombits(binary.BigEndian.Uint32(buf[:4]))

	case tagDouble:
		if _, err := io.ReadFull(r, buf[:8]); err != nil {
			return t, fmt.Errorf("nbt: TAG_Double: %w", err)
		}
		t.doubleV = math.Float64frombits(binary.BigEndian.Uint64(buf[:8]))

	case tagByteArr:
		n, err := readInt32(r)
		if err != nil {
			return t, fmt.Errorf("nbt: TAG_Byte_Array length: %w", err)
		}
		if n < 0 {
			return t, fmt.Errorf("nbt: TAG_Byte_Array negative length %d", n)
		}
		t.bytesV = make([]byte, n)
		if _, err := io.ReadFull(r, t.bytesV); err != nil {
			return t, fmt.Errorf("nbt: TAG_Byte_Array data: %w", err)
		}

	case tagString:
		var err error
		t.strV, err = readMUTF8(r)
		if err != nil {
			return t, fmt.Errorf("nbt: TAG_String: %w", err)
		}

	case tagList:
		var elemBuf [1]byte
		if _, err := io.ReadFull(r, elemBuf[:]); err != nil {
			return t, fmt.Errorf("nbt: TAG_List element type: %w", err)
		}
		t.listElem = tagType(elemBuf[0])
		n, err := readInt32(r)
		if err != nil {
			return t, fmt.Errorf("nbt: TAG_List length: %w", err)
		}
		if n < 0 {
			return t, fmt.Errorf("nbt: TAG_List negative length %d", n)
		}
		t.listV = make([]Tag, n)
		for i := int32(0); i < n; i++ {
			elem, err := readPayload(r, t.listElem)
			if err != nil {
				return t, fmt.Errorf("nbt: TAG_List[%d]: %w", i, err)
			}
			t.listV[i] = elem
		}

	case tagCompound:
		t.compound = make(map[string]Tag)
		for {
			name, entry, err := readNamed(r)
			if err != nil {
				return t, fmt.Errorf("nbt: TAG_Compound entry: %w", err)
			}
			if entry.typ == tagEnd {
				break
			}
			t.compound[name] = entry
		}

	case tagIntArr:
		n, err := readInt32(r)
		if err != nil {
			return t, fmt.Errorf("nbt: TAG_Int_Array length: %w", err)
		}
		if n < 0 {
			return t, fmt.Errorf("nbt: TAG_Int_Array negative length %d", n)
		}
		t.intsV = make([]int32, n)
		for i := int32(0); i < n; i++ {
			if _, err := io.ReadFull(r, buf[:4]); err != nil {
				return t, fmt.Errorf("nbt: TAG_Int_Array[%d]: %w", i, err)
			}
			t.intsV[i] = int32(binary.BigEndian.Uint32(buf[:4]))
		}

	case tagLongArr:
		n, err := readInt32(r)
		if err != nil {
			return t, fmt.Errorf("nbt: TAG_Long_Array length: %w", err)
		}
		if n < 0 {
			return t, fmt.Errorf("nbt: TAG_Long_Array negative length %d", n)
		}
		t.longsV = make([]int64, n)
		for i := int32(0); i < n; i++ {
			if _, err := io.ReadFull(r, buf[:8]); err != nil {
				return t, fmt.Errorf("nbt: TAG_Long_Array[%d]: %w", i, err)
			}
			t.longsV[i] = int64(binary.BigEndian.Uint64(buf[:8]))
		}

	default:
		return t, fmt.Errorf("nbt: unknown tag type %d", typ)
	}
	return t, nil
}

// readInt32 reads a big-endian signed 32-bit integer.
func readInt32(r io.Reader) (int32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(buf[:])), nil
}

// readMUTF8 reads a 2-byte-length-prefixed MUTF-8 string (NBT string encoding).
func readMUTF8(r io.Reader) (string, error) {
	var lenBuf [2]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return "", fmt.Errorf("nbt: reading string length: %w", err)
	}
	n := int(binary.BigEndian.Uint16(lenBuf[:]))
	if n == 0 {
		return "", nil
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("nbt: reading string body (%d bytes): %w", n, err)
	}
	return string(b), nil
}

// ── Writer helpers ────────────────────────────────────────────────────────────
//
// These write raw bytes to an io.Writer with no error accumulation — each
// call returns an error immediately.  The chunk encoder (encode.go) builds
// into a bytes.Buffer where writes cannot fail, so errors are silently
// swallowed there; the helpers are kept generic for correctness.

// WriteRootCompound writes a standard named-root NBT compound. Keys are sorted
// recursively so round-trip fixtures are deterministic.
func WriteRootCompound(w io.Writer, root map[string]Tag) {
	wByte(w, byte(tagCompound))
	writeMUTF8(w, "")
	writeCompoundPayload(w, root)
}

func writeCompoundPayload(w io.Writer, compound map[string]Tag) {
	keys := make([]string, 0, len(compound))
	for key := range compound {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := compound[key]
		if entry.typ == tagEnd {
			continue
		}
		writeNamedHeader(w, entry.typ, key)
		writePayload(w, entry)
	}
	writeCompoundEnd(w)
}

func writePayload(w io.Writer, tag Tag) {
	switch tag.typ {
	case tagByte:
		wByte(w, byte(tag.byteV))
	case tagShort:
		wUint16BE(w, uint16(tag.shortV))
	case tagInt:
		wUint32BE(w, uint32(tag.intV))
	case tagLong:
		wUint64BE(w, uint64(tag.longV))
	case tagFloat:
		wUint32BE(w, math.Float32bits(tag.floatV))
	case tagDouble:
		wUint64BE(w, math.Float64bits(tag.doubleV))
	case tagByteArr:
		wUint32BE(w, uint32(len(tag.bytesV)))
		wBytes(w, tag.bytesV)
	case tagString:
		writeMUTF8(w, tag.strV)
	case tagList:
		wByte(w, byte(tag.listElem))
		wUint32BE(w, uint32(len(tag.listV)))
		for _, entry := range tag.listV {
			writePayload(w, entry)
		}
	case tagCompound:
		writeCompoundPayload(w, tag.compound)
	case tagIntArr:
		wUint32BE(w, uint32(len(tag.intsV)))
		for _, value := range tag.intsV {
			wUint32BE(w, uint32(value))
		}
	case tagLongArr:
		wUint32BE(w, uint32(len(tag.longsV)))
		for _, value := range tag.longsV {
			wUint64BE(w, uint64(value))
		}
	}
}

func cloneCompound(source map[string]Tag) map[string]Tag {
	cloned := make(map[string]Tag, len(source))
	for key, value := range source {
		cloned[key] = cloneTag(value)
	}
	return cloned
}

func cloneTag(source Tag) Tag {
	cloned := source
	cloned.bytesV = append([]byte(nil), source.bytesV...)
	cloned.intsV = append([]int32(nil), source.intsV...)
	cloned.longsV = append([]int64(nil), source.longsV...)
	if source.listV != nil {
		cloned.listV = make([]Tag, len(source.listV))
		for i, entry := range source.listV {
			cloned.listV[i] = cloneTag(entry)
		}
	}
	if source.compound != nil {
		cloned.compound = cloneCompound(source.compound)
	}
	return cloned
}
func wByte(w io.Writer, b byte) {
	//nolint:errcheck
	w.Write([]byte{b})
}

func wBytes(w io.Writer, b []byte) {
	//nolint:errcheck
	w.Write(b)
}

func wUint16BE(w io.Writer, v uint16) {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], v)
	wBytes(w, buf[:])
}

func wUint32BE(w io.Writer, v uint32) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	wBytes(w, buf[:])
}

func wUint64BE(w io.Writer, v uint64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	wBytes(w, buf[:])
}

// writeMUTF8 writes a 2-byte-length-prefixed MUTF-8 string.
func writeMUTF8(w io.Writer, s string) {
	b := []byte(s)
	wUint16BE(w, uint16(len(b)))
	wBytes(w, b)
}

// writeNamedHeader writes a tag type byte followed by a tag name.
// Call this before writing the tag's payload.
func writeNamedHeader(w io.Writer, typ tagType, name string) {
	wByte(w, byte(typ))
	writeMUTF8(w, name)
}

// writeTagByte writes a named TAG_Byte entry.
func writeTagByte(w io.Writer, name string, v int8) {
	writeNamedHeader(w, tagByte, name)
	wByte(w, byte(v))
}

// writeTagInt writes a named TAG_Int entry.
func writeTagInt(w io.Writer, name string, v int32) {
	writeNamedHeader(w, tagInt, name)
	wUint32BE(w, uint32(v))
}

// writeTagString writes a named TAG_String entry.
func writeTagString(w io.Writer, name string, v string) {
	writeNamedHeader(w, tagString, name)
	writeMUTF8(w, v)
}

// writeCompoundOpen writes the header for a named TAG_Compound.
// The caller must close it with writeCompoundEnd.
func writeCompoundOpen(w io.Writer, name string) {
	writeNamedHeader(w, tagCompound, name)
}

// writeCompoundEnd writes a TAG_End byte to close a compound.
func writeCompoundEnd(w io.Writer) {
	wByte(w, byte(tagEnd))
}

// writeListHeader writes a named TAG_List header (type, name, element type,
// element count).  The caller must write exactly count payloads of elemType.
func writeListHeader(w io.Writer, name string, elemType tagType, count int) {
	writeNamedHeader(w, tagList, name)
	wByte(w, byte(elemType))
	wUint32BE(w, uint32(count))
}

// writeTagLongArray writes a named TAG_Long_Array entry.
func writeTagLongArray(w io.Writer, name string, data []int64) {
	writeNamedHeader(w, tagLongArr, name)
	wUint32BE(w, uint32(len(data)))
	for _, v := range data {
		wUint64BE(w, uint64(v))
	}
}
