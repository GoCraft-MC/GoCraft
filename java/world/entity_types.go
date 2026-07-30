package world

// entityTypeIDs maps canonical entity resource locations to their numeric
// protocol IDs for Minecraft Java Edition 1.21.4 (protocol 769).
//
// Populated at init time by registry.go from the embedded registries.json.
// Replacing the JSON file with the full Minecraft data-generator output updates
// all IDs without changing Go source.
var entityTypeIDs map[string]int32

// EntityTypeID returns the numeric protocol ID for the given canonical entity
// resource location.  Returns -1 and logs a one-time warning when the entity
// type is not in the registry.
func EntityTypeID(resourceLocation string) int32 {
	id, ok := entityTypeIDs[resourceLocation]
	if !ok {
		warnUnknown("minecraft:entity_type", resourceLocation)
		return -1
	}
	return id
}
