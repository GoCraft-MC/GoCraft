package entity

// Entity types that were missing from the original GoCraft canonical list but
// are present in the Minecraft Java 1.21.4 entity registry. They live in this
// file so the mob-parity work can be reviewed independently from the generated
// protocol registries.
const (
	TypeEnderDragon EntityType = "minecraft:ender_dragon"
	TypeGiant       EntityType = "minecraft:giant"
)
