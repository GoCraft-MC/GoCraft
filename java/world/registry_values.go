package world

// Registry-backed Java protocol IDs used outside terrain encoding.
var (
	soundEventIDs    map[string]int32
	mobEffectIDs     map[string]int32
	enchantmentIDs   map[string]int32
	enchantmentNames map[int32]string
)

// EnchantmentID returns the protocol-769 dynamic-registry ID, or -1 when the
// embedded Java 1.21.4 registry does not contain name.
func EnchantmentID(name string) int32 {
	id, ok := enchantmentIDs[name]
	if !ok {
		return -1
	}
	return id
}

// EnchantmentName returns the resource location for a protocol-769 ID.
func EnchantmentName(id int32) string { return enchantmentNames[id] }

// SoundEventID returns the protocol ID of a 1.21.4 sound event, or -1 when the
// embedded registry does not contain name.
func SoundEventID(name string) int32 {
	id, ok := soundEventIDs[name]
	if !ok {
		return -1
	}
	return id
}

// MobEffectID returns the protocol ID of a 1.21.4 mob effect, or -1 when the
// embedded registry does not contain name.
func MobEffectID(name string) int32 {
	id, ok := mobEffectIDs[name]
	if !ok {
		return -1
	}
	return id
}

// MobEffectNames returns sorted, namespace-free names suitable for command
// literals and tab completion.
func MobEffectNames() []string {
	names := make([]string, 0, len(mobEffectIDs))
	for name := range mobEffectIDs {
		if len(name) > len("minecraft:") && name[:len("minecraft:")] == "minecraft:" {
			name = name[len("minecraft:"):]
		}
		names = append(names, name)
	}
	sortStrings(names)
	return names
}
