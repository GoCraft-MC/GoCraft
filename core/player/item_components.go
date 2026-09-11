package player

import (
	"encoding/json"
	"errors"
)

// SetComponents validates and canonicalises a complete component object.
func (s *ItemStack) SetComponents(encoded string) error {
	if s == nil {
		return errors.New("nil item stack")
	}
	normalized, err := NormalizeItemComponents(encoded)
	if err == nil {
		s.Components = normalized
	}
	return err
}

// NormalizeItemComponents returns deterministic component JSON.
func NormalizeItemComponents(encoded string) (string, error) {
	components, err := decodeItemComponents(encoded)
	if err != nil || len(components) == 0 {
		return "", err
	}
	raw, err := json.Marshal(components)
	if err != nil || len(raw) > maxItemComponentsBytes {
		if err == nil {
			err = errors.New("item components exceed size limit")
		}
		return "", err
	}
	return string(raw), nil
}

// SetComponent atomically stores a JSON component. Nil removes it.
func (s *ItemStack) SetComponent(id string, value any) error {
	if s == nil {
		return errors.New("nil item stack")
	}
	components, err := decodeItemComponents(s.Components)
	if err != nil {
		return err
	}
	id = normalizeComponentID(id)
	if id == "" {
		return errors.New("empty item component ID")
	}
	if value == nil {
		delete(components, id)
	} else if raw, marshalErr := json.Marshal(value); marshalErr != nil {
		return marshalErr
	} else if components[id], err = canonicalJSON(raw); err != nil {
		return err
	}
	encoded, err := json.Marshal(components)
	if err != nil || len(encoded) > maxItemComponentsBytes {
		return errors.New("item components exceed size limit")
	}
	s.Components = string(encoded)
	if len(components) == 0 {
		s.Components = ""
	}
	return nil
}

// Component decodes a stored component into target.
func (s ItemStack) Component(id string, target any) bool {
	components, err := decodeItemComponents(s.Components)
	raw, ok := components[normalizeComponentID(id)]
	return err == nil && ok && target != nil && json.Unmarshal(raw, target) == nil
}

func (s ItemStack) NormalizedComponents() string {
	encoded, err := NormalizeItemComponents(s.Components)
	if err != nil {
		return s.Components
	}
	return encoded
}
