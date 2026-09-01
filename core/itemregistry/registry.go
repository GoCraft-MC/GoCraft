package itemregistry

import (
	"fmt"
	"sort"
	"strings"
)

const vanillaVersion = "1.21.4"

// Registry is an immutable item-definition and tag index. NewRegistry is
// exported so custom-item systems can eventually build compatible registries
// without coupling the canonical model to a protocol adapter.
type Registry struct {
	definitions map[string]*Definition
	tags        map[string]map[string]struct{}
}

var vanilla = mustLoadVanilla()

// Version returns the pinned vanilla base version.
func Version() string { return vanillaVersion }

// Lookup returns the canonical definition for itemID.
func Lookup(itemID string) (*Definition, bool) { return vanilla.Lookup(itemID) }

// HasTag reports whether itemID belongs to a canonical Minecraft item tag.
func HasTag(itemID, tag string) bool { return vanilla.HasTag(itemID, tag) }

// RepairsWith reports whether ingredientID is accepted by the item's static
// repair component.
func RepairsWith(itemID, ingredientID string) bool {
	return vanilla.RepairsWith(itemID, ingredientID)
}

// Count returns the number of embedded vanilla and compatibility definitions.
func Count() int { return len(vanilla.definitions) }

// NewRegistry validates and indexes detached definitions. Definitions and tag
// slices are copied so later caller mutation cannot change the registry.
func NewRegistry(definitions []Definition) (*Registry, error) {
	registry := &Registry{
		definitions: make(map[string]*Definition, len(definitions)),
		tags:        make(map[string]map[string]struct{}),
	}
	for _, source := range definitions {
		definition := cloneDefinition(source)
		sort.Strings(definition.Tags)
		if err := validateDefinition(&definition); err != nil {
			return nil, err
		}
		if _, duplicate := registry.definitions[definition.ID]; duplicate {
			return nil, fmt.Errorf("itemregistry: duplicate definition %s", definition.ID)
		}
		registry.definitions[definition.ID] = &definition
		for _, tag := range definition.Tags {
			members := registry.tags[tag]
			if members == nil {
				members = make(map[string]struct{})
				registry.tags[tag] = members
			}
			members[definition.ID] = struct{}{}
		}
	}
	return registry, nil
}

func (r *Registry) Lookup(itemID string) (*Definition, bool) {
	if r == nil {
		return nil, false
	}
	definition, ok := r.definitions[itemID]
	return definition, ok
}

func (r *Registry) HasTag(itemID, tag string) bool {
	if r == nil {
		return false
	}
	_, ok := r.tags[tag][itemID]
	return ok
}

func (r *Registry) RepairsWith(itemID, ingredientID string) bool {
	definition, ok := r.Lookup(itemID)
	if !ok || definition.Repair == nil || ingredientID == "" {
		return false
	}
	ingredient := definition.Repair.Ingredient
	if strings.HasPrefix(ingredient, "#") {
		return r.HasTag(ingredientID, strings.TrimPrefix(ingredient, "#"))
	}
	return ingredient == ingredientID
}
