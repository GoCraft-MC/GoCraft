package nbt

import (
	"GoCraft/core/itemregistry"
	"GoCraft/core/player"
	"GoCraft/java/protocol"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync/atomic"

	javaworld "GoCraft/java/world"
)

var configuredItemTooltips atomic.Value

type itemTooltipOptions struct {
	showDurability        bool
	showAttributes        bool
	hideVanillaAttributes bool
	legacyCombat          bool
}

func init() {
	configuredItemTooltips.Store(itemTooltipOptions{
		showDurability:        true,
		showAttributes:        true,
		hideVanillaAttributes: true,
	})
}

// ConfigureItemTooltips installs the server-wide presentation used for all
// outgoing item stacks. The gameplay attributes remain authoritative even
// when their vanilla tooltip section is hidden.
func ConfigureItemTooltips(showDurability, showAttributes, hideVanillaAttributes, legacyCombat bool) {
	configuredItemTooltips.Store(itemTooltipOptions{
		showDurability:        showDurability,
		showAttributes:        showAttributes,
		hideVanillaAttributes: hideVanillaAttributes,
		legacyCombat:          legacyCombat,
	})
}

func SkipNetworkNBT(r *bytes.Reader) error {
	tagType, err := r.ReadByte()
	if err != nil {
		return err
	}

	return skipNBTPayload(r, tagType)
}

// encodeSlot appends a 1.20.5+ slot encoding to b.
//
//	Empty:     VarInt(0)
//	Non-empty: VarInt(count) VarInt(item_type) VarInt(0/*add*/) VarInt(0/*remove*/)
func EncodeSlot(b *protocol.Builder, item player.ItemStack) {
	if item.IsEmpty() {
		b.VarInt(0)
		return
	}

	id := javaworld.ItemID(item.ItemID)
	if id < 0 {
		// Item not in our table — send as empty rather than sending a wrong ID.
		b.VarInt(0)
		return
	}

	maxDamage := player.MaxDurability(item.ItemID)
	enchantments := item.EnchantmentLevels()
	if maxDamage <= 0 {
		componentCount := int32(0)
		if potionID(item) >= 0 {
			componentCount++
		}

		if len(enchantments) > 0 {
			componentCount++
		}

		if item.ItemID == "minecraft:decorated_pot" {
			componentCount++
		}

		if item.ItemID == "minecraft:firework_rocket" {
			componentCount++
		}

		if item.Components != "" {
			componentCount++
		}

		b.VarInt(int32(item.Count)).
			VarInt(id).
			VarInt(componentCount).
			VarInt(0) // components_to_remove
		encodeSlotPotionContents(b, item)
		encodeSlotEnchantments(b, enchantments)
		encodeSlotPotDecorations(b, item)
		encodeSlotFireworks(b, item)
		encodeSlotExtensionComponents(b, item)
		return
	}

	damage := item.Damage
	if damage < 0 {
		damage = 0
	} else if damage >= maxDamage {
		damage = maxDamage - 1
	}

	options := configuredItemTooltips.Load().(itemTooltipOptions)
	type loreLine struct {
		text  string
		color string
	}

	lore := make([]loreLine, 0, 7)
	if attackDamage, attackSpeed, ok := player.AttackAttributes(item.ItemID); ok && options.showAttributes {
		speedText := fmt.Sprintf("%g", attackSpeed)
		if options.legacyCombat {
			speedText = "Instant"
		}

		lore = append(lore,
			loreLine{"", "gray"},
			loreLine{"When in Main Hand:", "gray"},
			loreLine{fmt.Sprintf(" %g Attack Damage", attackDamage), "dark_green"},
			loreLine{" " + speedText + " Attack Speed", "dark_green"})
	}

	if armour := player.ArmorPoints(item.ItemID); armour > 0 && options.showAttributes {
		lore = append(lore,
			loreLine{"", "gray"},
			loreLine{armorTooltipHeading(item.ItemID), "gray"},
			loreLine{fmt.Sprintf(" %d Armor", armour), "blue"})

		if toughness := player.ArmorToughness(item.ItemID); toughness > 0 {
			lore = append(lore, loreLine{fmt.Sprintf(" %g Armor Toughness", toughness), "blue"})
		}

		if resistance := player.ArmorKnockbackResistance(item.ItemID); resistance > 0 {
			lore = append(lore, loreLine{fmt.Sprintf(" %g%% Knockback Resistance", resistance*100), "blue"})
		}
	}

	if options.showDurability {
		remaining := maxDamage - damage
		color := "green"
		if remaining*5 <= maxDamage {
			color = "red"
		} else if remaining*2 <= maxDamage {
			color = "yellow"
		}

		lore = append(lore, loreLine{fmt.Sprintf("Durability: %d / %d", remaining, maxDamage), color})
	}

	componentCount := int32(2) // max_damage + damage
	if len(lore) > 0 {
		componentCount++
	}

	if options.hideVanillaAttributes {
		componentCount++
	}

	if len(enchantments) > 0 {
		componentCount++
	}

	if item.Components != "" {
		componentCount++
	}

	b.VarInt(int32(item.Count)).
		VarInt(id).
		VarInt(componentCount).
		VarInt(0). // components_to_remove
		VarInt(2).VarInt(int32(maxDamage)).
		VarInt(3).VarInt(int32(damage))
	encodeSlotEnchantments(b, enchantments)
	encodeSlotExtensionComponents(b, item)

	if len(lore) > 0 {
		b.VarInt(8).VarInt(int32(len(lore)))
		for _, line := range lore {
			b.Write(LoreTextComponent{
				Text:  line.text,
				Color: line.color,
			})
		}
	}

	if options.hideVanillaAttributes {
		// Component 13 overrides the item's default attribute list with an empty
		// non-displayed list. This hides the client-visible 1024 attack speed;
		// actual damage, armour, and cooldown remain server-authoritative.
		b.VarInt(13).VarInt(0).Bool(false)
	}
}

func potionID(item player.ItemStack) int32 {
	name, ok := player.PotionName(item)
	if !ok || name == "" {
		return -1
	}

	return javaworld.PotionID("minecraft:" + name)
}

func encodeSlotPotionContents(b *protocol.Builder, item player.ItemStack) {
	id := potionID(item)
	if id < 0 {
		return
	}

	// Component 41: optional base potion, optional colour, custom effects.
	b.VarInt(41).
		Bool(true).
		VarInt(id).
		Bool(false).
		VarInt(0)
}

func encodeSlotExtensionComponents(b *protocol.Builder, item player.ItemStack) {
	if item.Components == "" {
		return
	}

	b.VarInt(0).
		Write(GocraftComponents{
			Value: item.NormalizedComponents(),
		})
}

func encodeSlotEnchantments(b *protocol.Builder, enchantments []player.EnchantmentLevel) {
	if len(enchantments) == 0 {
		return
	}

	b.VarInt(10).
		VarInt(int32(len(enchantments)))
	for _, enchantment := range enchantments {
		b.VarInt(javaworld.EnchantmentID(enchantment.ID)).
			VarInt(int32(enchantment.Level))
	}
}

func armorInventorySlot(itemID string) int {
	definition, ok := itemregistry.Lookup(itemID)
	if !ok || definition.Equipment == nil {
		return -1
	}

	switch definition.Equipment.Slot {
	case "head":
		return 5
	case "chest":
		return 6
	case "legs":
		return 7
	case "feet":
		return 8
	default:
		return -1
	}
}

func armorTooltipHeading(itemID string) string {
	switch armorInventorySlot(itemID) {
	case 5:
		return "When on Head:"
	case 6:
		return "When on Body:"
	case 7:
		return "When on Legs:"
	case 8:
		return "When on Feet:"
	default:
		return "When Worn:"
	}
}

func encodeSlotPotDecorations(b *protocol.Builder, item player.ItemStack) {
	if item.ItemID != "minecraft:decorated_pot" {
		return
	}

	decorations := item.NormalizedPotDecorations()
	b.VarInt(61).VarInt(int32(len(decorations)))
	for _, decoration := range decorations {
		b.VarInt(javaworld.ItemID(decoration))
	}
}

func encodeSlotFireworks(b *protocol.Builder, item player.ItemStack) {
	if item.ItemID != "minecraft:firework_rocket" {
		return
	}

	data := item.EffectiveFireworks()
	b.VarInt(56).VarInt(int32(data.Flight)).VarInt(int32(data.ExplosionCount))
	for index := range int(data.ExplosionCount) {
		explosion := data.Explosions[index]
		b.VarInt(int32(explosion.Shape)).VarInt(int32(explosion.ColorCount))
		for color := range int(explosion.ColorCount) {
			b.Int(explosion.Colors[color])
		}

		b.VarInt(int32(explosion.FadeColorCount))
		for color := range int(explosion.FadeColorCount) {
			b.Int(explosion.FadeColors[color])
		}

		b.Bool(explosion.Trail).Bool(explosion.Twinkle)
	}
}

func ReadPlainSlot(r *bytes.Reader) (player.ItemStack, error) {
	count, err := protocol.ReadVarInt(r)
	if err != nil {
		return player.ItemStack{}, err
	}

	if count <= 0 {
		return player.ItemStack{}, nil
	}

	itemID, err := protocol.ReadVarInt(r)
	if err != nil {
		return player.ItemStack{}, err
	}

	added, err := protocol.ReadVarInt(r)
	if err != nil {
		return player.ItemStack{}, err
	}

	removed, err := protocol.ReadVarInt(r)
	if err != nil {
		return player.ItemStack{}, err
	}

	damage := int32(0)
	enchantments := ""
	var potDecorations [4]string
	var fireworks player.FireworkData
	hasFireworks := false
	components := ""
	potionName := ""
	for range added {
		componentType, err := protocol.ReadVarInt(r)
		if err != nil {
			return player.ItemStack{}, err
		}

		switch componentType {
		case 0: // custom_data: preserve GoCraft's canonical extension object
			components, err = readGoCraftComponents(r)
			if err != nil {
				return player.ItemStack{}, fmt.Errorf("reading custom item data: %w", err)
			}

		case 2: // max_damage
			if _, err := protocol.ReadVarInt(r); err != nil {
				return player.ItemStack{}, err
			}

		case 3: // damage
			damage, err = protocol.ReadVarInt(r)
			if err != nil {
				return player.ItemStack{}, err
			}

		case 8: // lore: VarInt count followed by optional anonymous NBT
			lines, err := protocol.ReadVarInt(r)
			if err != nil || lines < 0 || lines > 256 {
				return player.ItemStack{}, fmt.Errorf("invalid lore line count %d: %w", lines, err)
			}

			for range lines {
				if err := SkipNetworkNBT(r); err != nil {
					return player.ItemStack{}, fmt.Errorf("reading lore: %w", err)
				}
			}

		case 10: // enchantments: registry ID/level pairs
			length, readErr := protocol.ReadVarInt(r)
			if readErr != nil || length < 0 || length > 256 {
				return player.ItemStack{}, fmt.Errorf("invalid enchantment count %d: %w", length, readErr)
			}

			stack := player.ItemStack{ItemID: "minecraft:stone", Count: 1}
			for range length {
				enchantmentID, idErr := protocol.ReadVarInt(r)
				level, levelErr := protocol.ReadVarInt(r)
				name := javaworld.EnchantmentName(enchantmentID)
				if idErr != nil || levelErr != nil || name == "" || level < 1 || level > 255 {
					return player.ItemStack{}, fmt.Errorf("invalid enchantment id=%d level=%d", enchantmentID, level)
				}

				stack.Enchant(name, int(level))
			}

			enchantments = stack.Enchantments

		case 61: // pot_decorations: array of item registry IDs
			length, readErr := protocol.ReadVarInt(r)
			if readErr != nil || length < 0 || length > 64 {
				return player.ItemStack{}, fmt.Errorf("invalid pot decoration count %d: %w", length, readErr)
			}

			for entry := range length {
				decorationID, idErr := protocol.ReadVarInt(r)
				if idErr != nil {
					return player.ItemStack{}, idErr
				}

				decoration := javaworld.ItemName(decorationID)
				if decoration == "" {
					return player.ItemStack{}, fmt.Errorf("unknown pot decoration item ID %d", decorationID)
				}

				if entry < int32(len(potDecorations)) {
					potDecorations[entry] = decoration
				}
			}

		case 13: // attribute modifiers, including the final showTooltip flag
			attributes, readErr := protocol.ReadVarInt(r)
			if readErr != nil || attributes < 0 || attributes > 256 {
				return player.ItemStack{}, fmt.Errorf("invalid attribute modifier count %d: %w", attributes, readErr)
			}

			for range attributes {
				if _, readErr = protocol.ReadVarInt(r); readErr != nil {
					return player.ItemStack{}, readErr
				}

				if _, readErr = protocol.ReadString(r); readErr != nil {
					return player.ItemStack{}, readErr
				}

				if _, readErr = protocol.ReadDouble(r); readErr != nil {
					return player.ItemStack{}, readErr
				}

				if _, readErr = protocol.ReadVarInt(r); readErr != nil {
					return player.ItemStack{}, readErr
				}

				if _, readErr = protocol.ReadVarInt(r); readErr != nil {
					return player.ItemStack{}, readErr
				}
			}

			if _, readErr = protocol.ReadBool(r); readErr != nil {
				return player.ItemStack{}, readErr
			}

		case 41: // potion_contents: optional potion, colour, custom effects
			hasPotion, readErr := protocol.ReadBool(r)
			if readErr != nil {
				return player.ItemStack{}, readErr
			}

			if hasPotion {
				potionRegistryID, idErr := protocol.ReadVarInt(r)
				if idErr != nil || javaworld.PotionName(potionRegistryID) == "" {
					return player.ItemStack{}, fmt.Errorf("invalid potion registry ID %d", potionRegistryID)
				}

				potionName = javaworld.PotionName(potionRegistryID)
			}

			hasColour, colourErr := protocol.ReadBool(r)
			if colourErr != nil {
				return player.ItemStack{}, colourErr
			}

			if hasColour {
				if _, colourErr = protocol.ReadInt(r); colourErr != nil {
					return player.ItemStack{}, colourErr
				}
			}

			customEffects, effectErr := protocol.ReadVarInt(r)
			if effectErr != nil || customEffects != 0 {
				return player.ItemStack{}, fmt.Errorf("unsupported custom potion effect count %d", customEffects)
			}

		case 56: // fireworks: flight duration and bounded explosion list
			flight, readErr := protocol.ReadVarInt(r)
			if readErr != nil || flight < 0 || flight > 255 {
				return player.ItemStack{}, fmt.Errorf("invalid firework flight %d: %w", flight, readErr)
			}

			length, readErr := protocol.ReadVarInt(r)
			if readErr != nil || length < 0 || length > player.MaxFireworkExplosions {
				return player.ItemStack{}, fmt.Errorf("invalid firework explosion count %d: %w", length, readErr)
			}

			fireworks.Flight = uint8(flight)
			fireworks.ExplosionCount = uint8(length)
			for explosionIndex := range length {
				explosion, readErr := readJavaFireworkExplosion(r)
				if readErr != nil {
					return player.ItemStack{}, readErr
				}

				fireworks.Explosions[explosionIndex] = explosion
			}

			hasFireworks = true
		default:
			return player.ItemStack{}, fmt.Errorf("unsupported item component %d", componentType)
		}
	}

	for range removed {
		if _, err := protocol.ReadVarInt(r); err != nil {
			return player.ItemStack{}, err
		}
	}

	name := javaworld.ItemName(itemID)
	if name == "" {
		return player.ItemStack{}, fmt.Errorf("unknown item ID %d", itemID)
	}

	if damage < 0 {
		damage = 0
	}

	stack := player.ItemStack{
		ItemID: name, Count: int(count), Damage: int(damage), Enchantments: enchantments, PotDecorations: potDecorations,
		HasFireworks: hasFireworks, Fireworks: fireworks,
	}

	if components != "" {
		if err := stack.SetComponents(components); err != nil {
			return player.ItemStack{}, fmt.Errorf("invalid canonical item components: %w", err)
		}
	}

	if potionName != "" {
		if existing, _ := player.PotionName(stack); existing == "" {
			if err := stack.SetComponent("potion_contents", map[string]string{"potion": potionName}); err != nil {
				return player.ItemStack{}, err
			}
		}
	}

	return stack, nil
}

func readJavaFireworkExplosion(r *bytes.Reader) (player.FireworkExplosion, error) {
	var explosion player.FireworkExplosion
	shape, err := protocol.ReadVarInt(r)
	if err != nil || shape < 0 || shape > 4 {
		return explosion, fmt.Errorf("invalid firework shape %d: %w", shape, err)
	}

	explosion.Shape = uint8(shape)
	colors, err := protocol.ReadVarInt(r)
	if err != nil || colors < 0 || colors > player.MaxFireworkColors {
		return explosion, fmt.Errorf("invalid firework color count %d: %w", colors, err)
	}

	explosion.ColorCount = uint8(colors)
	for index := range colors {
		color, readErr := protocol.ReadInt(r)
		if readErr != nil {
			return explosion, readErr
		}

		explosion.Colors[index] = color
	}

	fades, err := protocol.ReadVarInt(r)
	if err != nil || fades < 0 || fades > player.MaxFireworkColors {
		return explosion, fmt.Errorf("invalid firework fade count %d: %w", fades, err)
	}

	explosion.FadeColorCount = uint8(fades)
	for index := range fades {
		color, readErr := protocol.ReadInt(r)
		if readErr != nil {
			return explosion, readErr
		}

		explosion.FadeColors[index] = color
	}

	explosion.Trail, err = protocol.ReadBool(r)
	if err != nil {
		return explosion, err
	}

	explosion.Twinkle, err = protocol.ReadBool(r)
	return explosion, err
}

func skipReaderBytes(r *bytes.Reader, n int) error {
	if n < 0 || n > r.Len() {
		return io.ErrUnexpectedEOF
	}

	_, err := r.Seek(int64(n), io.SeekCurrent)
	return err
}

func skipNBTString(r *bytes.Reader) error {
	_, err := readNBTStringValue(r)
	return err
}

func readNBTLength(r *bytes.Reader) (int, error) {
	var raw [4]byte
	if _, err := io.ReadFull(r, raw[:]); err != nil {
		return 0, err
	}

	n := int(int32(binary.BigEndian.Uint32(raw[:])))
	if n < 0 || n > r.Len() {
		return 0, fmt.Errorf("invalid NBT length %d", n)
	}

	return n, nil
}

func readNBTStringValue(r *bytes.Reader) (string, error) {
	var raw [2]byte
	if _, err := io.ReadFull(r, raw[:]); err != nil {
		return "", err
	}

	length := int(binary.BigEndian.Uint16(raw[:]))
	if length > r.Len() {
		return "", io.ErrUnexpectedEOF
	}

	value := make([]byte, length)
	if _, err := io.ReadFull(r, value); err != nil {
		return "", err
	}

	return string(value), nil
}

func skipNBTPayload(r *bytes.Reader, tagType byte) error {
	switch tagType {
	case 0:
		return nil

	case 1:
		return skipReaderBytes(r, 1)

	case 2:
		return skipReaderBytes(r, 2)

	case 3, 5:
		return skipReaderBytes(r, 4)

	case 4, 6:
		return skipReaderBytes(r, 8)

	case 7:
		n, err := readNBTLength(r)
		if err != nil {
			return err
		}

		return skipReaderBytes(r, n)

	case 8:
		return skipNBTString(r)

	case 9:
		elementType, err := r.ReadByte()
		if err != nil {
			return err
		}

		n, err := readNBTLength(r)
		if err != nil {
			return err
		}

		for range n {
			if err := skipNBTPayload(r, elementType); err != nil {
				return err
			}
		}

		return nil

	case 10:
		for {
			childType, err := r.ReadByte()
			if err != nil {
				return err
			}

			if childType == 0 {
				return nil
			}

			if err := skipNBTString(r); err != nil {
				return err
			}

			if err := skipNBTPayload(r, childType); err != nil {
				return err
			}
		}

	case 11:
		n, err := readNBTLength(r)
		if err != nil {
			return err
		}

		return skipReaderBytes(r, n*4)

	case 12:
		n, err := readNBTLength(r)
		if err != nil {
			return err
		}

		return skipReaderBytes(r, n*8)

	default:
		return fmt.Errorf("invalid NBT tag type %d", tagType)
	}
}
