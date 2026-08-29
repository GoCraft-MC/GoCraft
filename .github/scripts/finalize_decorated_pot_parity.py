from pathlib import Path
import re


def text(path):
    return Path(path).read_text()


def write(path, value):
    Path(path).write_text(value)


def replace_once(path, old, new):
    value = text(path)
    if old not in value:
        raise SystemExit(f"missing patch target in {path}: {old[:120]!r}")
    write(path, value.replace(old, new, 1))


def replace_all(path, old, new):
    value = text(path)
    if old not in value:
        return 0
    count = value.count(old)
    write(path, value.replace(old, new))
    return count


def append_once(path, marker, addition):
    value = text(path)
    if marker in value:
        return
    write(path, value.rstrip() + "\n\n" + addition.strip() + "\n")


# ---------------------------------------------------------------------------
# Canonical item/entity/container representation.
# ---------------------------------------------------------------------------
replace_once(
    "core/player/item.go",
    '''\t// Enchantments stores sorted resource-location/level pairs as a compact,\n\t// comparable canonical component string.\n\tEnchantments string `json:",omitempty"`\n''',
    '''\t// Enchantments stores sorted resource-location/level pairs as a compact,\n\t// comparable canonical component string.\n\tEnchantments string `json:",omitempty"`\n\t// PotDecorations stores the four side decorations of a decorated-pot item.\n\t// An array keeps ItemStack comparable, which is important for inventory diffing.\n\tPotDecorations [4]string `json:",omitempty"`\n''',
)
replace_once(
    "core/player/item.go",
    '''// SameItem reports whether two stacks may merge without losing components.\nfunc (s ItemStack) SameItem(other ItemStack) bool {\n\treturn s.ItemID == other.ItemID && s.Damage == other.Damage && s.Enchantments == other.Enchantments\n}\n''',
    '''// NormalizePotDecorations returns the complete four-side decoration list.\n// Vanilla treats absent entries as bricks.\nfunc NormalizePotDecorations(decorations [4]string) [4]string {\n\tfor index := range decorations {\n\t\tif decorations[index] == "" {\n\t\t\tdecorations[index] = "minecraft:brick"\n\t\t}\n\t}\n\treturn decorations\n}\n\n// NormalizedPotDecorations returns the meaningful decorated-pot component.\nfunc (s ItemStack) NormalizedPotDecorations() [4]string {\n\tif s.ItemID != "minecraft:decorated_pot" {\n\t\treturn [4]string{}\n\t}\n\treturn NormalizePotDecorations(s.PotDecorations)\n}\n\n// SameItem reports whether two stacks may merge without losing components.\nfunc (s ItemStack) SameItem(other ItemStack) bool {\n\treturn s.ItemID == other.ItemID && s.Damage == other.Damage && s.Enchantments == other.Enchantments &&\n\t\ts.NormalizedPotDecorations() == other.NormalizedPotDecorations()\n}\n''',
)

replace_once(
    "core/entity/entity.go",
    '''\tItemID     string // resource location, e.g. "minecraft:diamond"\n\tItemCount  int\n\tItemDamage int\n''',
    '''\tItemID             string // resource location, e.g. "minecraft:diamond"\n\tItemCount          int\n\tItemDamage         int\n\tItemPotDecorations [4]string\n''',
)

replace_once(
    "core/world/chunk.go",
    '''type BlockEntity struct {\n\tX, Y, Z int\n\tType    string\n\tData    []byte\n\tItems   []ContainerItem\n}\n''',
    '''type BlockEntity struct {\n\tX, Y, Z       int\n\tType          string\n\tData          []byte\n\tItems         []ContainerItem\n\tPotDecorations [4]string\n}\n''',
)
replace_once(
    "core/world/chunk.go",
    '''type ContainerItem struct {\n\tSlot         int\n\tItemID       string\n\tCount        int\n\tDamage       int\n\tEnchantments string\n}\n''',
    '''type ContainerItem struct {\n\tSlot           int\n\tItemID         string\n\tCount          int\n\tDamage         int\n\tEnchantments   string\n\tPotDecorations [4]string\n}\n''',
)

# Preserve decorated-pot item components whenever ItemStack <-> ContainerItem
# conversion occurs. The patterns are deliberately keyed by existing component
# fields so ordinary items remain unchanged.
for path in [
    "java/handler/block.go",
    "java/handler/chest.go",
    "java/handler/furnace.go",
    "java/handler/workstation.go",
    "server/bedrock_actions.go",
    "server/bedrock_container.go",
    "server/furnace.go",
    "server/container_automation.go",
]:
    value = text(path)
    value = value.replace(
        "Enchantments: item.Enchantments}",
        "Enchantments: item.Enchantments, PotDecorations: item.PotDecorations}",
    )
    value = value.replace(
        "Enchantments: stack.Enchantments}",
        "Enchantments: stack.Enchantments, PotDecorations: stack.PotDecorations}",
    )
    value = value.replace(
        "Enchantments: held.Enchantments}",
        "Enchantments: held.Enchantments, PotDecorations: held.PotDecorations}",
    )
    write(path, value)

# ---------------------------------------------------------------------------
# Canonical block-entity access for decorated-pot side decorations.
# ---------------------------------------------------------------------------
append_once(
    "core/world/world.go",
    "func (w *World) DecoratedPotDecorations",
    r'''
// DecoratedPotDecorations returns all four vanilla side decorations stored by
// the decorated-pot block entity. Missing entries represent bricks in vanilla.
func (w *World) DecoratedPotDecorations(x, y, z int) [4]string {
	cx := int32(math.Floor(float64(x) / SectionSize))
	cz := int32(math.Floor(float64(z) / SectionSize))
	c := w.Chunk(cx, cz)
	w.containerMu.RLock()
	defer w.containerMu.RUnlock()
	for _, entity := range c.BlockEntities {
		if entity.X == x && entity.Y == y && entity.Z == z {
			return normalizeDecoratedPotDecorations(entity.PotDecorations)
		}
	}
	return normalizeDecoratedPotDecorations([4]string{})
}

// SetDecoratedPotDecorations updates only the decoration component and keeps
// the pot's stored item and opaque block-entity data intact.
func (w *World) SetDecoratedPotDecorations(x, y, z int, decorations [4]string) {
	cx := int32(math.Floor(float64(x) / SectionSize))
	cz := int32(math.Floor(float64(z) / SectionSize))
	c := w.Chunk(cx, cz)
	decorations = normalizeDecoratedPotDecorations(decorations)

	w.containerMu.Lock()
	updated := false
	for index := range c.BlockEntities {
		entity := &c.BlockEntities[index]
		if entity.X != x || entity.Y != y || entity.Z != z {
			continue
		}
		if entity.Type == "" {
			entity.Type = "minecraft:decorated_pot"
		}
		if len(entity.Data) < 2 {
			entity.Data = []byte{10, 0}
		}
		entity.PotDecorations = decorations
		updated = true
		break
	}
	if !updated {
		c.BlockEntities = append(c.BlockEntities, BlockEntity{
			X: x, Y: y, Z: z, Type: "minecraft:decorated_pot", Data: []byte{10, 0},
			PotDecorations: decorations,
		})
	}
	w.containerMu.Unlock()

	w.mu.Lock()
	key := [2]int32{cx, cz}
	w.chunks[key] = c
	w.touchChunkLocked(key)
	w.dirty[key] = struct{}{}
	w.mu.Unlock()
}

func normalizeDecoratedPotDecorations(decorations [4]string) [4]string {
	for index := range decorations {
		if decorations[index] == "" {
			decorations[index] = "minecraft:brick"
		}
	}
	return decorations
}
''',
)

# ---------------------------------------------------------------------------
# Vanilla decorated-pot loot: dynamic sherds + copy_components.
# ---------------------------------------------------------------------------
replace_once(
    "core/blockloot/loot.go",
    '''type Context struct {\n\tBlock        coreworld.Block\n\tTool         player.ItemStack\n\tEnchantments map[string]int\n\tExplosion    float64\n\tRandom       *rand.Rand\n\tBlockAt      func(dx, dy, dz int) coreworld.Block\n}\n''',
    '''type Context struct {\n\tBlock          coreworld.Block\n\tTool           player.ItemStack\n\tEnchantments   map[string]int\n\tExplosion      float64\n\tRandom         *rand.Rand\n\tBlockAt        func(dx, dy, dz int) coreworld.Block\n\tPotDecorations [4]string\n}\n''',
)
replace_once(
    "core/blockloot/loot.go",
    '''\tcase "minecraft:dynamic":\n\t\tif ctx.Block.ResourceLocation() == "minecraft:decorated_pot" {\n\t\t\treturn []player.ItemStack{{ItemID: "minecraft:brick", Count: 4}}, true\n\t\t}\n''',
    '''\tcase "minecraft:dynamic":\n\t\tif ctx.Block.ResourceLocation() == "minecraft:decorated_pot" {\n\t\t\tdecorations := player.NormalizePotDecorations(ctx.PotDecorations)\n\t\t\tstacks := make([]player.ItemStack, 0, len(decorations))\n\t\t\tfor _, decoration := range decorations {\n\t\t\t\tstacks = append(stacks, player.ItemStack{ItemID: decoration, Count: 1})\n\t\t\t}\n\t\t\treturn stacks, true\n\t\t}\n''',
)
replace_once(
    "core/blockloot/loot.go",
    '''\t\tcase "minecraft:copy_components", "minecraft:copy_state":\n\t\t\t// ItemStack currently stores identity/count/damage only. These\n\t\t\t// functions do not alter which item or how many items are dropped.\n''',
    '''\t\tcase "minecraft:copy_components":\n\t\t\tif ctx.Block.ResourceLocation() == "minecraft:decorated_pot" {\n\t\t\t\tfor index := range stacks {\n\t\t\t\t\tif stacks[index].ItemID == "minecraft:decorated_pot" {\n\t\t\t\t\t\tstacks[index].PotDecorations = player.NormalizePotDecorations(ctx.PotDecorations)\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\tcase "minecraft:copy_state":\n\t\t\t// Block-state copying does not currently affect a canonical ItemStack.\n''',
)

# ---------------------------------------------------------------------------
# Java Anvil: keep pot block-entity sherds and pot item components on disk.
# ---------------------------------------------------------------------------
replace_once(
    "java/world/anvil/decode.go",
    '''\t\titems := decodeContainerItems(data["Items"])\n\t\tdelete(data, "Items")\n''',
    '''\t\titems := decodeContainerItems(data["Items"])\n\t\tpotDecorations := decodePotDecorations(data["sherds"])\n\t\tdelete(data, "Items")\n''',
)
replace_once(
    "java/world/anvil/decode.go",
    '''\t\tentities = append(entities, coreworld.BlockEntity{X: x, Y: y, Z: z, Type: entityType, Data: payload.Bytes(), Items: items})\n''',
    '''\t\tentities = append(entities, coreworld.BlockEntity{\n\t\t\tX: x, Y: y, Z: z, Type: entityType, Data: payload.Bytes(), Items: items, PotDecorations: potDecorations,\n\t\t})\n''',
)
replace_once(
    "java/world/anvil/decode.go",
    '''\t\tdamage, enchantments := 0, ""\n\t\tif components := entry.compound["components"]; components.typ == tagCompound {\n\t\t\tdamage = numericTagValue(components.compound["minecraft:damage"])\n\t\t\tenchantments = decodeItemEnchantments(components.compound["minecraft:enchantments"])\n\t\t}\n\t\titems = append(items, coreworld.ContainerItem{Slot: slot, ItemID: itemID, Count: count, Damage: damage, Enchantments: enchantments})\n''',
    '''\t\tdamage, enchantments := 0, ""\n\t\tvar potDecorations [4]string\n\t\tif components := entry.compound["components"]; components.typ == tagCompound {\n\t\t\tdamage = numericTagValue(components.compound["minecraft:damage"])\n\t\t\tenchantments = decodeItemEnchantments(components.compound["minecraft:enchantments"])\n\t\t\tpotDecorations = decodePotDecorations(components.compound["minecraft:pot_decorations"])\n\t\t}\n\t\titems = append(items, coreworld.ContainerItem{\n\t\t\tSlot: slot, ItemID: itemID, Count: count, Damage: damage, Enchantments: enchantments, PotDecorations: potDecorations,\n\t\t})\n''',
)
append_once(
    "java/world/anvil/decode.go",
    "func decodePotDecorations",
    r'''
func decodePotDecorations(tag Tag) [4]string {
	var decorations [4]string
	if tag.typ != tagList {
		return decorations
	}
	for index, entry := range tag.listV {
		if index >= len(decorations) {
			break
		}
		if entry.typ == tagString {
			decorations[index] = entry.Str()
		}
	}
	return decorations
}
''',
)

replace_once(
    "java/world/anvil/merge.go",
    '''\t\tif entity.Items != nil {\n\t\t\tcompound["Items"] = containerItemsTag(entity.Items)\n\t\t}\n\t\tentries = append(entries, Tag{typ: tagCompound, compound: compound})\n''',
    '''\t\tif entity.Items != nil {\n\t\t\tcompound["Items"] = containerItemsTag(entity.Items)\n\t\t}\n\t\tif entity.Type == "minecraft:decorated_pot" || entity.Type == "DecoratedPot" {\n\t\t\tcompound["sherds"] = potDecorationsTag(entity.PotDecorations)\n\t\t}\n\t\tentries = append(entries, Tag{typ: tagCompound, compound: compound})\n''',
)
replace_once(
    "java/world/anvil/merge.go",
    '''\t\tif enchantments := encodeItemEnchantments(item.Enchantments); enchantments.typ != tagEnd {\n\t\t\tcomponents["minecraft:enchantments"] = enchantments\n\t\t}\n\t\tif len(components) != 0 {\n''',
    '''\t\tif enchantments := encodeItemEnchantments(item.Enchantments); enchantments.typ != tagEnd {\n\t\t\tcomponents["minecraft:enchantments"] = enchantments\n\t\t}\n\t\tif item.ItemID == "minecraft:decorated_pot" {\n\t\t\tcomponents["minecraft:pot_decorations"] = potDecorationsTag(item.PotDecorations)\n\t\t}\n\t\tif len(components) != 0 {\n''',
)
append_once(
    "java/world/anvil/merge.go",
    "func potDecorationsTag",
    r'''
func potDecorationsTag(decorations [4]string) Tag {
	decorations = player.NormalizePotDecorations(decorations)
	values := make([]Tag, 0, len(decorations))
	for _, decoration := range decorations {
		values = append(values, Tag{typ: tagString, strV: decoration})
	}
	return Tag{typ: tagList, listElem: tagString, listV: values}
}
''',
)

# ---------------------------------------------------------------------------
# Java protocol item component 61 (pot_decorations).
# ---------------------------------------------------------------------------
replace_once(
    "java/handler/inventory.go",
    '''\tif maxDamage <= 0 {\n\t\tcomponentCount := int32(0)\n\t\tif len(enchantments) > 0 {\n\t\t\tcomponentCount = 1\n\t\t}\n\t\tb.VarInt(int32(item.Count)).\n\t\t\tVarInt(id).\n\t\t\tVarInt(componentCount).\n\t\t\tVarInt(0) // components_to_remove\n\t\tencodeSlotEnchantments(b, enchantments)\n\t\treturn\n\t}\n''',
    '''\tif maxDamage <= 0 {\n\t\tcomponentCount := int32(0)\n\t\tif len(enchantments) > 0 {\n\t\t\tcomponentCount++\n\t\t}\n\t\tif item.ItemID == "minecraft:decorated_pot" {\n\t\t\tcomponentCount++\n\t\t}\n\t\tb.VarInt(int32(item.Count)).\n\t\t\tVarInt(id).\n\t\t\tVarInt(componentCount).\n\t\t\tVarInt(0) // components_to_remove\n\t\tencodeSlotEnchantments(b, enchantments)\n\t\tencodeSlotPotDecorations(b, item)\n\t\treturn\n\t}\n''',
)
append_once(
    "java/handler/inventory.go",
    "func encodeSlotPotDecorations",
    r'''
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
''',
)

replace_once(
    "java/handler/crafting.go",
    '''\tdamage := int32(0)\n\tenchantments := ""\n''',
    '''\tdamage := int32(0)\n\tenchantments := ""\n\tvar potDecorations [4]string\n''',
)
replace_once(
    "java/handler/crafting.go",
    '''\t\tcase 13: // attribute modifiers, including the final showTooltip flag\n''',
    '''\t\tcase 61: // pot_decorations: array of item registry IDs\n\t\t\tlength, readErr := protocol.ReadVarInt(r)\n\t\t\tif readErr != nil || length < 0 || length > 64 {\n\t\t\t\treturn player.ItemStack{}, fmt.Errorf("invalid pot decoration count %d: %w", length, readErr)\n\t\t\t}\n\t\t\tfor entry := int32(0); entry < length; entry++ {\n\t\t\t\tdecorationID, idErr := protocol.ReadVarInt(r)\n\t\t\t\tif idErr != nil {\n\t\t\t\t\treturn player.ItemStack{}, idErr\n\t\t\t\t}\n\t\t\t\tdecoration := javaworld.ItemName(decorationID)\n\t\t\t\tif decoration == "" {\n\t\t\t\t\treturn player.ItemStack{}, fmt.Errorf("unknown pot decoration item ID %d", decorationID)\n\t\t\t\t}\n\t\t\t\tif entry < int32(len(potDecorations)) {\n\t\t\t\t\tpotDecorations[entry] = decoration\n\t\t\t\t}\n\t\t\t}\n\t\tcase 13: // attribute modifiers, including the final showTooltip flag\n''',
)
replace_once(
    "java/handler/crafting.go",
    '''\treturn player.ItemStack{ItemID: name, Count: int(count), Damage: int(damage), Enchantments: enchantments}, nil\n''',
    '''\treturn player.ItemStack{\n\t\tItemID: name, Count: int(count), Damage: int(damage), Enchantments: enchantments, PotDecorations: potDecorations,\n\t}, nil\n''',
)

# ---------------------------------------------------------------------------
# Java break/place/drop path.
# ---------------------------------------------------------------------------
replace_once(
    "java/handler/block.go",
    '''\t\tlootBlock := broken\n\t\tif broken.ResourceLocation() == "minecraft:decorated_pot" && held.EnchantmentLevel("minecraft:silk_touch") == 0 && blockloot.BreaksDecoratedPot(held.ItemID) {\n''',
    '''\t\tlootBlock := broken\n\t\tpotDecorations := [4]string{}\n\t\tif broken.ResourceLocation() == "minecraft:decorated_pot" {\n\t\t\tpotDecorations = w.DecoratedPotDecorations(int(bx), int(by), int(bz))\n\t\t}\n\t\tcontainerItems := w.ContainerItems(int(bx), int(by), int(bz))\n\t\tif broken.ResourceLocation() == "minecraft:decorated_pot" && held.EnchantmentLevel("minecraft:silk_touch") == 0 && blockloot.BreaksDecoratedPot(held.ItemID) {\n''',
)
replace_once(
    "java/handler/block.go",
    '''\t\t\tEnchantments: enchantments,\n\t\t\tBlockAt: func(dx, dy, dz int) coreworld.Block {\n''',
    '''\t\t\tEnchantments:   enchantments,\n\t\t\tPotDecorations: potDecorations,\n\t\t\tBlockAt: func(dx, dy, dz int) coreworld.Block {\n''',
)
replace_once(
    "java/handler/block.go",
    '''\t\t\tif isJavaStorageContainer(broken.ResourceLocation()) || broken.ResourceLocation() == "minecraft:decorated_pot" || IsFurnaceContainer(broken.ResourceLocation()) {\n\t\t\t\tfor _, item := range w.ContainerItems(int(bx), int(by), int(bz)) {\n''',
    '''\t\t\tif isJavaStorageContainer(broken.ResourceLocation()) || broken.ResourceLocation() == "minecraft:decorated_pot" || IsFurnaceContainer(broken.ResourceLocation()) {\n\t\t\t\tfor _, item := range containerItems {\n''',
)
replace_once(
    "java/handler/block.go",
    '''\tdropped.ItemID, dropped.ItemCount, dropped.ItemDamage = stack.ItemID, stack.Count, stack.Damage\n''',
    '''\tdropped.ItemID, dropped.ItemCount, dropped.ItemDamage = stack.ItemID, stack.Count, stack.Damage\n\tdropped.ItemPotDecorations = stack.PotDecorations\n''',
)
replace_once(
    "java/handler/block.go",
    '''\tcase block.ResourceLocation() == "minecraft:decorated_pot":\n\t\tapplyBlockChange(px, py, pz, block, w, mgr)\n\t\tw.SetContainerItems(px, py, pz, block.ResourceLocation(), nil)\n''',
    '''\tcase block.ResourceLocation() == "minecraft:decorated_pot":\n\t\tblock.Properties = map[string]string{\n\t\t\t"facing": chestFacingFromYaw(p.Rotation.Yaw), "cracked": "false",\n\t\t\t"waterlogged": strconv.FormatBool(placingInWater),\n\t\t}\n\t\tapplyBlockChange(px, py, pz, block, w, mgr)\n\t\tw.SetContainerItems(px, py, pz, block.ResourceLocation(), nil)\n\t\tw.SetDecoratedPotDecorations(px, py, pz, held.NormalizedPotDecorations())\n''',
)

# ---------------------------------------------------------------------------
# Bedrock live break/place path.
# ---------------------------------------------------------------------------
server_path = Path("server/server.go")
server_text = server_path.read_text()
server_text = server_text.replace(
    '''\t\tlootBlock := block\n\t\tif block.ResourceLocation() == "minecraft:decorated_pot" && held.EnchantmentLevel("minecraft:silk_touch") == 0 && blockloot.BreaksDecoratedPot(held.ItemID) {\n''',
    '''\t\tlootBlock := block\n\t\tpotDecorations := [4]string{}\n\t\tif block.ResourceLocation() == "minecraft:decorated_pot" {\n\t\t\tpotDecorations = actionWorld.DecoratedPotDecorations(x, y, z)\n\t\t}\n\t\tif block.ResourceLocation() == "minecraft:decorated_pot" && held.EnchantmentLevel("minecraft:silk_touch") == 0 && blockloot.BreaksDecoratedPot(held.ItemID) {\n''',
    1,
)
server_text = server_text.replace(
    '''\t\t\tEnchantments: enchantments,\n\t\t\tBlockAt: func(dx, dy, dz int) coreworld.Block {\n''',
    '''\t\t\tEnchantments:   enchantments,\n\t\t\tPotDecorations: potDecorations,\n\t\t\tBlockAt: func(dx, dy, dz int) coreworld.Block {\n''',
    1,
)
server_text = server_text.replace(
    '''\t\t\t\tstack := player.ItemStack{ItemID: item.ItemID, Count: item.Count, Damage: item.Damage}\n''',
    '''\t\t\t\tstack := player.ItemStack{\n\t\t\t\t\tItemID: item.ItemID, Count: item.Count, Damage: item.Damage, Enchantments: item.Enchantments, PotDecorations: item.PotDecorations,\n\t\t\t\t}\n''',
    1,
)
server_text = server_text.replace(
    '''\t\tstack := player.ItemStack{ItemID: e.ItemID, Count: e.ItemCount, Damage: e.ItemDamage}\n''',
    '''\t\tstack := player.ItemStack{\n\t\t\tItemID: e.ItemID, Count: e.ItemCount, Damage: e.ItemDamage, PotDecorations: e.ItemPotDecorations,\n\t\t}\n''',
    1,
)
# newDroppedItemInWorld assigns the canonical stack to the entity in one place.
server_text = server_text.replace(
    '''\tdropped.ItemID = stack.ItemID\n\tdropped.ItemCount = stack.Count\n\tdropped.ItemDamage = stack.Damage\n''',
    '''\tdropped.ItemID = stack.ItemID\n\tdropped.ItemCount = stack.Count\n\tdropped.ItemDamage = stack.Damage\n\tdropped.ItemPotDecorations = stack.PotDecorations\n''',
    1,
)
server_path.write_text(server_text)

replace_once(
    "server/bedrock_actions.go",
    '''\tplaced, valid := s.bedrockPlacementState(p, block, px, py, pz, i)\n''',
    '''\tplaced, valid := s.bedrockPlacementState(p, block, px, py, pz, i)\n''',
)
replace_once(
    "server/bedrock_actions.go",
    '''\tif isBedrockGenericContainer(name) || name == "minecraft:decorated_pot" {\n\t\ts.bedrockWorld().SetContainerItems(px, py, pz, name, nil)\n\t}\n''',
    '''\tif isBedrockGenericContainer(name) || name == "minecraft:decorated_pot" {\n\t\ts.bedrockWorld().SetContainerItems(px, py, pz, name, nil)\n\t}\n\tif name == "minecraft:decorated_pot" {\n\t\ts.bedrockWorld().SetDecoratedPotDecorations(px, py, pz, held.NormalizedPotDecorations())\n\t}\n''',
)
replace_once(
    "server/bedrock_actions.go",
    '''\tcase name == "minecraft:redstone_lamp":\n''',
    '''\tcase name == "minecraft:decorated_pot":\n\t\tprops = map[string]string{"facing": frontFacing, "cracked": "false", "waterlogged": "false"}\n\tcase name == "minecraft:redstone_lamp":\n''',
)

# ---------------------------------------------------------------------------
# Dropped-item wire representation for Java + Bedrock.
# ---------------------------------------------------------------------------
replace_once(
    "java/handler/mob.go",
    '''\t\tencodeSlot(b, player.ItemStack{ItemID: e.ItemID, Count: e.ItemCount, Damage: e.ItemDamage})\n''',
    '''\t\tencodeSlot(b, player.ItemStack{\n\t\t\tItemID: e.ItemID, Count: e.ItemCount, Damage: e.ItemDamage, PotDecorations: e.ItemPotDecorations,\n\t\t})\n''',
)
replace_once(
    "bedrock/sync.go",
    '''\t\titem := l.itemInstance(player.ItemStack{ItemID: entity.ItemID, Count: entity.ItemCount, Damage: entity.ItemDamage}, 1)\n''',
    '''\t\titem := l.itemInstance(player.ItemStack{\n\t\t\tItemID: entity.ItemID, Count: entity.ItemCount, Damage: entity.ItemDamage, PotDecorations: entity.ItemPotDecorations,\n\t\t}, 1)\n''',
)
replace_once(
    "bedrock/sync.go",
    '''\tif enchantments := bedrockEnchantments(stack); len(enchantments) > 0 {\n\t\tnbtData["ench"] = enchantments\n\t}\n\tif len(nbtData) == 0 {\n''',
    '''\tif enchantments := bedrockEnchantments(stack); len(enchantments) > 0 {\n\t\tnbtData["ench"] = enchantments\n\t}\n\tif stack.ItemID == "minecraft:decorated_pot" {\n\t\tdecorations := stack.NormalizedPotDecorations()\n\t\tsherds := make([]any, 0, len(decorations))\n\t\tfor _, decoration := range decorations {\n\t\t\tsherds = append(sherds, decoration)\n\t\t}\n\t\tnbtData["id"] = "DecoratedPot"\n\t\tnbtData["sherds"] = sherds\n\t}\n\tif len(nbtData) == 0 {\n''',
)

# ---------------------------------------------------------------------------
# Regression tests: exact pot data, all door cursor sides, Bedrock placement,
# and zero-damage snowball impact metadata.
# ---------------------------------------------------------------------------
Path("core/blockloot/decorated_pot_test.go").write_text(r'''package blockloot

import (
	"testing"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

func TestDecoratedPotIntactDropPreservesDecorations(t *testing.T) {
	decorations := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:skull_pottery_sherd", "minecraft:heart_pottery_sherd"}
	drops := Drops(Context{
		Block: coreworld.Block{Namespace: "minecraft", Name: "decorated_pot", Properties: map[string]string{"cracked": "false"}},
		PotDecorations: decorations,
	})
	if len(drops) != 1 || drops[0].ItemID != "minecraft:decorated_pot" || drops[0].Count != 1 {
		t.Fatalf("intact pot drops = %#v", drops)
	}
	if got := drops[0].NormalizedPotDecorations(); got != decorations {
		t.Fatalf("intact pot decorations = %#v, want %#v", got, decorations)
	}
}

func TestDecoratedPotShatterReturnsOriginalSherds(t *testing.T) {
	decorations := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:skull_pottery_sherd", "minecraft:angler_pottery_sherd"}
	drops := Drops(Context{
		Block: coreworld.Block{Namespace: "minecraft", Name: "decorated_pot", Properties: map[string]string{"cracked": "true"}},
		Tool: player.ItemStack{ItemID: "minecraft:iron_pickaxe", Count: 1},
		PotDecorations: decorations,
	})
	counts := map[string]int{}
	for _, drop := range drops {
		counts[drop.ItemID] += drop.Count
	}
	want := map[string]int{"minecraft:angler_pottery_sherd": 2, "minecraft:brick": 1, "minecraft:skull_pottery_sherd": 1}
	if len(counts) != len(want) {
		t.Fatalf("shattered drops = %#v, want %#v", counts, want)
	}
	for item, count := range want {
		if counts[item] != count {
			t.Fatalf("%s count = %d, want %d (all=%#v)", item, counts[item], count, counts)
		}
	}
}
''')

Path("core/world/decorated_pot_test.go").write_text(r'''package world

import "testing"

func TestDecoratedPotDecorationsSurviveContainerUpdates(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "decorated_pot", Properties: map[string]string{"facing": "north", "cracked": "false", "waterlogged": "false"}})
	want := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:skull_pottery_sherd", "minecraft:heart_pottery_sherd"}
	w.SetDecoratedPotDecorations(0, 64, 0, want)
	w.SetContainerItems(0, 64, 0, "minecraft:decorated_pot", []ContainerItem{{Slot: 0, ItemID: "minecraft:diamond", Count: 3}})
	if got := w.DecoratedPotDecorations(0, 64, 0); got != want {
		t.Fatalf("decorations = %#v, want %#v", got, want)
	}
}
''')

append_once(
    "core/world/vanilla_interaction_test.go",
    "TestDoorHingeCursorSidesAllFacings",
    r'''
func TestDoorHingeCursorSidesAllFacings(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	tests := []struct {
		facing       string
		firstX, firstZ float32
		secondX, secondZ float32
	}{
		{facing: "north", firstX: 0.25, firstZ: 0.5, secondX: 0.75, secondZ: 0.5},
		{facing: "south", firstX: 0.25, firstZ: 0.5, secondX: 0.75, secondZ: 0.5},
		{facing: "east", firstX: 0.5, firstZ: 0.25, secondX: 0.5, secondZ: 0.75},
		{facing: "west", firstX: 0.5, firstZ: 0.25, secondX: 0.5, secondZ: 0.75},
	}
	for _, test := range tests {
		t.Run(test.facing, func(t *testing.T) {
			first := DoorHinge(w, 0, 64, 0, test.facing, test.firstX, test.firstZ)
			second := DoorHinge(w, 0, 64, 0, test.facing, test.secondX, test.secondZ)
			if first == second {
				t.Fatalf("hinge did not change across cursor halves for %s: %q", test.facing, first)
			}
		})
	}
}
''',
)

Path("server/decorated_pot_parity_test.go").write_text(r'''package server

import (
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestBedrockDecoratedPotPlacementPreservesDecorations(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{81}, "bedrock-potter", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	p.Rotation.Yaw = 0
	decorations := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:skull_pottery_sherd", "minecraft:heart_pottery_sherd"}
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:decorated_pot", Count: 1, PotDecorations: decorations}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	w.SetBlock(0, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	s := &Server{game: g, world: w, sessions: session.NewManager()}
	if !s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
		PlayerUUID: p.UUID, Position: spatial.BlockPos{X: 0, Y: 63, Z: 0}, Face: 1, ClickX: 0.5, ClickY: 1, ClickZ: 0.5,
	}, w.GetBlock(0, 63, 0)) {
		t.Fatal("decorated pot placement was not handled")
	}
	pot := w.GetBlock(0, 64, 0)
	if pot.ResourceLocation() != "minecraft:decorated_pot" || pot.Properties["cracked"] != "false" {
		t.Fatalf("placed pot state = %#v", pot)
	}
	if got := w.DecoratedPotDecorations(0, 64, 0); got != decorations {
		t.Fatalf("placed decorations = %#v, want %#v", got, decorations)
	}
}

func TestSnowballImpactQueuesZeroDamageReaction(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	pig := testPassiveEntity(991, "minecraft:pig")
	w.Entities.Add(pig)
	if !w.QueueEntityImpactFrom(pig.EntityID, -1, 0) {
		t.Fatal("zero-damage impact was not queued")
	}
	events := w.DrainEntityDamage()
	event, ok := events[pig.EntityID]
	if !ok || event.Amount != 0 || !event.HasSource {
		t.Fatalf("impact event = %#v, present=%v", event, ok)
	}
}
''')

# Anvil helper regression: block entity + item component use the same four names.
append_once(
    "java/world/anvil/storage_roundtrip_test.go",
    "TestDecoratedPotDecorationTagRoundTrip",
    r'''
func TestDecoratedPotDecorationTagRoundTrip(t *testing.T) {
	want := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:skull_pottery_sherd", "minecraft:heart_pottery_sherd"}
	if got := decodePotDecorations(potDecorationsTag(want)); got != want {
		t.Fatalf("decorations = %#v, want %#v", got, want)
	}
}
''',
)
