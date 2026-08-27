// Package customitems implements GoCraft's cross-edition custom item system.
//
// Items are defined in YAML files inside a packs/ directory. Each sub-directory
// is one pack. GoCraft reads them at startup, assigns stable CustomModelData
// (CMD) values for Java clients and component-based runtime IDs for Bedrock
// clients, then:
//
//   - Generates a Java resource pack ZIP (models + textures) and serves it
//     over a local HTTP server so Java clients see the correct textures.
//
//   - Generates a Bedrock .mcaddon (resource pack + behavior pack) in memory
//     and injects it into the Bedrock listener's pack list so Bedrock clients
//     receive the item definitions and textures on login.
//
// Pack layout (one directory per pack inside packs/):
//
//	packs/
//	  mypacks/
//	    items.yml         ← item definitions (see ItemDef / PackDef)
//	    textures/
//	      ruby.png        ← 16×16 PNG referenced by items.yml
//	      ruby_sword.png
//	  .registry.yml       ← auto-generated; do not edit by hand
package customitems

// ItemDef describes one custom item in a pack's items.yml.
type ItemDef struct {
	// DisplayName is the MiniMessage-formatted name shown to players.
	// Example: "<red>Ruby" or "<gradient:#ff0000:#aa0000>Ruby Sword"
	DisplayName string `yaml:"display_name"`

	// Material is the vanilla Java Edition base material (lowercase).
	// All custom items of the same material share one model-override file.
	// Examples: "paper", "diamond_sword", "iron_ingot"
	Material string `yaml:"material"`

	// Texture is the PNG filename inside the pack's textures/ directory.
	// Example: "ruby.png"
	Texture string `yaml:"texture"`

	// Parent is the item model parent used in the generated Java model JSON.
	// Defaults to "item/generated" (flat items).
	// Use "item/handheld" for swords / tools.
	Parent string `yaml:"parent,omitempty"`

	// MaxStackSize is the maximum number of items in one stack (Bedrock only for now).
	// Defaults to 64.
	MaxStackSize int `yaml:"max_stack_size,omitempty"`

	// HandEquipped marks this as a weapon/tool for Bedrock's first-person hand
	// rendering (larger item in hand). Defaults to false.
	HandEquipped bool `yaml:"hand_equipped,omitempty"`
}

// PackDef is the top-level structure of a pack's items.yml file.
type PackDef struct {
	// Namespace is the unique identifier prefix for all items in this pack.
	// Must be lowercase and contain no spaces.  Example: "myserver"
	Namespace string `yaml:"namespace"`

	// Items maps item IDs to their definitions.
	// The full item identifier will be "<namespace>:<id>".
	Items map[string]ItemDef `yaml:"items"`
}

// LoadedPack is a parsed pack with texture bytes read into memory.
type LoadedPack struct {
	Def      PackDef
	Dir      string
	Textures map[string][]byte // filename → PNG bytes
}

// ResolvedItem is a single custom item with its assigned network IDs.
type ResolvedItem struct {
	Namespace string
	ID        string
	Def       ItemDef

	// CMD is the Java CustomModelData integer assigned to this item.
	CMD int

	// BedrockRuntimeID is the item runtime ID sent to Bedrock clients in
	// the StartGame packet's Items list.
	BedrockRuntimeID int16

	TextureData []byte // PNG bytes, may be nil if texture file was not found
}

// Key returns the fully-qualified identifier "namespace:id".
func (r *ResolvedItem) Key() string { return r.Namespace + ":" + r.ID }
