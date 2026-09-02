package handler

import "bytes"

const goCraftComponentsNBTKey = "GoCraftComponents"

// nbtGoCraftComponents encodes the canonical extension component object inside
// Java's minecraft:custom_data component. The value remains valid vanilla NBT
// while GoCraft adapters gain typed codecs for individual components.
func nbtGoCraftComponents(encoded string) []byte {
	var buf bytes.Buffer
	buf.WriteByte(0x0a)
	writeNBTStringEntry(&buf, goCraftComponentsNBTKey, encoded)
	buf.WriteByte(0)
	return buf.Bytes()
}

func readGoCraftComponents(r *bytes.Reader) (string, error) {
	rootType, err := r.ReadByte()
	if err != nil || rootType == 0 {
		return "", err
	}
	if rootType != 10 {
		return "", skipNBTPayload(r, rootType)
	}
	var result string
	for {
		childType, readErr := r.ReadByte()
		if readErr != nil {
			return "", readErr
		}
		if childType == 0 {
			return result, nil
		}
		name, readErr := readNBTStringValue(r)
		if readErr != nil {
			return "", readErr
		}
		if childType == 8 && name == goCraftComponentsNBTKey {
			result, readErr = readNBTStringValue(r)
		} else {
			readErr = skipNBTPayload(r, childType)
		}
		if readErr != nil {
			return "", readErr
		}
	}
}
