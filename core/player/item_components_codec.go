package player

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const maxItemComponentsBytes = 1 << 20

func decodeItemComponents(encoded string) (map[string]json.RawMessage, error) {
	components := map[string]json.RawMessage{}
	if encoded == "" {
		return components, nil
	}
	if len(encoded) > maxItemComponentsBytes {
		return nil, errors.New("item components exceed size limit")
	}
	decoder := json.NewDecoder(strings.NewReader(encoded))
	if err := decoder.Decode(&components); err != nil {
		return nil, fmt.Errorf("decode item components: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, err
	}
	normalized := make(map[string]json.RawMessage, len(components))
	for id, raw := range components {
		id = normalizeComponentID(id)
		if id == "" {
			return nil, errors.New("empty item component ID")
		}
		value, err := canonicalJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("decode item component %s: %w", id, err)
		}
		normalized[id] = value
	}
	return normalized, nil
}

func normalizeComponentID(id string) string {
	id = strings.TrimSpace(id)
	if id != "" && !strings.ContainsRune(id, ':') {
		return "minecraft:" + id
	}
	return id
}

func canonicalJSON(raw []byte) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values")
	}
	return err
}
