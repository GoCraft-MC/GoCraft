from pathlib import Path


def replace_exact(path, old, new, count=1):
    p = Path(path)
    text = p.read_text()
    actual = text.count(old)
    if actual < count:
        raise SystemExit(f"{path}: expected at least {count} occurrences, found {actual}: {old[:100]!r}")
    text = text.replace(old, new, count)
    p.write_text(text)


def append_once(path, marker, content):
    p = Path(path)
    text = p.read_text()
    if marker not in text:
        p.write_text(text.rstrip() + "\n\n" + content.strip() + "\n")


# ---------------------------------------------------------------------------
# Shared world: neighbor/redstone propagation + vanilla observer scheduling.
# ---------------------------------------------------------------------------
replace_exact(
    "core/world/world.go",
    '''\tw.scheduleBlockNeighborUpdates(x, y, z, oldBlock, block)\n\tw.notifyBlockObserver(x, y, z, block)\n\tw.triggerObservers(x, y, z)\n''',
    '''\tw.scheduleBlockNeighborUpdates(x, y, z, oldBlock, block)\n\tw.notifyBlockObserver(x, y, z, block)\n\tif !oldBlock.Equal(block) {\n\t\tw.triggerObservers(x, y, z)\n\t}\n''',
)
replace_exact(
    "core/world/world.go",
    '''\t\tupdated := redstoneBlockWith(observer, "powered", "true")\n\t\tw.setBlockNoPhysics(pos[0], pos[1], pos[2], updated)\n\t\tw.Redstone.NotifyChange(pos[0], pos[1], pos[2])\n\t\tw.BlockPhysics.ScheduleObserver(pos[0], pos[1], pos[2], w.PhysicsTime(), 2)\n''',
    '''\t\t// Vanilla observers schedule detection two game ticks after the watched\n\t\t// block changes. The scheduled tick starts the two-tick output pulse.\n\t\tw.BlockPhysics.ScheduleObserver(pos[0], pos[1], pos[2], w.PhysicsTime(), 2)\n''',
)
replace_exact(
    "core/world/world.go",
    '''\tif c.Sections[sIdx] == nil {\n\t\tc.Sections[sIdx] = NewSection()\n\t}\n\tc.Sections[sIdx].Set(lx, ly, lz, block)\n\tw.mu.Lock()\n''',
    '''\tif c.Sections[sIdx] == nil {\n\t\tc.Sections[sIdx] = NewSection()\n\t}\n\toldBlock := c.Sections[sIdx].At(lx, ly, lz)\n\tc.Sections[sIdx].Set(lx, ly, lz, block)\n\tw.mu.Lock()\n''',
)
replace_exact(
    "core/world/world.go",
    '''\tw.mu.Unlock()\n\tw.notifyBlockObserver(x, y, z, block)\n}\n\n// scheduleBlockNeighborUpdates''',
    '''\tw.mu.Unlock()\n\tw.notifyBlockObserver(x, y, z, block)\n\tif !oldBlock.Equal(block) {\n\t\tw.triggerObservers(x, y, z)\n\t}\n}\n\n// scheduleBlockNeighborUpdates''',
)
replace_exact(
    "core/world/world.go",
    '''\tif IsRedstoneConductor(placedName) || IsRedstoneSource(placedName) || IsRedstoneLoad(placedName) ||\n\t\tIsRedstoneConductor(oldName) || IsRedstoneSource(oldName) || IsRedstoneLoad(oldName) ||\n\t\tplaced.IsAir() {\n''',
    '''\tif IsRedstoneConductor(placedName) || IsRedstoneSource(placedName) || IsRedstoneLoad(placedName) ||\n\t\tIsRedstoneConductor(oldName) || IsRedstoneSource(oldName) || IsRedstoneLoad(oldName) ||\n\t\tisRedstonePowerConductor(placed) || isRedstonePowerConductor(old) || placed.IsAir() {\n''',
)

# Dust/loads may read power conducted through one ordinary full block. Do not
# allow ordinary full blocks to recursively power each other.
replace_exact(
    "core/world/redstone.go",
    '''\t\t\t} else if IsRedstoneConductor(nbName) {\n\t\t\t\tp := re.powerFromConductorToward(nb[0], nb[1], nb[2], nbBlock, [3]int{x, y, z})\n''',
    '''\t\t\t} else if IsRedstoneConductor(nbName) || isRedstonePowerConductor(nbBlock) {\n\t\t\t\tp := re.powerFromConductorToward(nb[0], nb[1], nb[2], nbBlock, [3]int{x, y, z})\n''',
)
replace_exact(
    "core/world/redstone.go",
    '''\tdefault:\n\t\t// Loads and solid blocks accept both direct source power and power\n\t\t// carried by dust/repeaters. Without the conductor branch, a lamp next\n\t\t// to powered wire would never light even though the wire state changed.\n\t\tbest := 0\n''',
    '''\tdefault:\n\t\t// Loads accept power conducted through one ordinary full block. An\n\t\t// ordinary full block itself only accepts direct source/explicit redstone\n\t\t// conductor power, preventing chains of solid blocks from relaying signal.\n\t\tbest := 0\n\t\tallowOrdinaryConductorInput := !isRedstonePowerConductor(block) || IsRedstoneConductor(name)\n''',
)
replace_exact(
    "core/world/redstone.go",
    '''\t\t\t} else if IsRedstoneConductor(nbName) {\n\t\t\t\tif p := re.powerFromConductorToward(nb[0], nb[1], nb[2], nbBlock, [3]int{x, y, z}); p > best {\n''',
    '''\t\t\t} else if IsRedstoneConductor(nbName) || (allowOrdinaryConductorInput && isRedstonePowerConductor(nbBlock)) {\n\t\t\t\tif p := re.powerFromConductorToward(nb[0], nb[1], nb[2], nbBlock, [3]int{x, y, z}); p > best {\n''',
)

# ---------------------------------------------------------------------------
# Java placement/break path: shared door hinge + attachment support updates.
# ---------------------------------------------------------------------------
replace_exact(
    "java/handler/block.go",
    '"facing": facing, "half": "lower", "hinge": doorHinge(facing, clickX, clickZ),',
    '"facing": facing, "half": "lower", "hinge": coreworld.DoorHinge(w, x, y, z, facing, clickX, clickZ),',
)
replace_exact(
    "java/handler/block.go",
    '''func breakUnsupportedBlocksAbove(x, y, z int, w *coreworld.World, mgr *session.Manager) {\n\tfor _, change := range w.BreakUnsupportedCropsAbove(x, y, z) {\n\t\tif mgr != nil {\n\t\t\tBroadcastBlockChange(change, mgr)\n\t\t}\n\t}\n}\n''',
    '''func breakUnsupportedBlocksAbove(x, y, z int, w *coreworld.World, mgr *session.Manager) {\n\tfor _, change := range w.BreakUnsupportedCropsAbove(x, y, z) {\n\t\tif mgr != nil {\n\t\t\tBroadcastBlockChange(change, mgr)\n\t\t}\n\t}\n\tfor _, change := range w.BreakUnsupportedAttachmentsAround(x, y, z) {\n\t\tif mgr != nil {\n\t\t\tBroadcastBlockChange(change, mgr)\n\t\t}\n\t}\n}\n''',
)

# ---------------------------------------------------------------------------
# Bedrock placement/break path: use the same canonical hinge/support rules.
# ---------------------------------------------------------------------------
replace_exact(
    "server/bedrock_actions.go",
    'hinge := bedrockDoorHinge(facing, i.ClickX, i.ClickZ)',
    'hinge := coreworld.DoorHinge(s.bedrockWorld(), px, py, pz, facing, i.ClickX, i.ClickZ)',
)
replace_exact(
    "server/bedrock_actions.go",
    '''func (s *Server) breakBedrockUnsupportedAbove(x, y, z int) {\n''',
    '''func (s *Server) breakBedrockUnsupportedAbove(x, y, z int) {\n\tfor _, change := range s.bedrockWorld().BreakUnsupportedAttachmentsAround(x, y, z) {\n\t\tif s.sessions != nil {\n\t\t\thandler.BroadcastBlockChange(change, s.sessions)\n\t\t}\n\t}\n''',
)

# ---------------------------------------------------------------------------
# Observer scheduled pulse: first tick ON, two ticks later OFF.
# ---------------------------------------------------------------------------
replace_exact(
    "server/server.go",
    '''\t\tcase coreworld.UpdateObserver:\n\t\t\tobserver := s.world.GetBlock(u.X, u.Y, u.Z)\n\t\t\tif observer.ResourceLocation() == "minecraft:observer" && observer.Properties["powered"] == "true" {\n\t\t\t\tobserver = bedrockCopyBlock(observer)\n\t\t\t\tobserver.Properties["powered"] = "false"\n\t\t\t\ts.world.SetBlock(u.X, u.Y, u.Z, observer)\n''',
    '''\t\tcase coreworld.UpdateObserver:\n\t\t\tobserver := s.world.GetBlock(u.X, u.Y, u.Z)\n\t\t\tif observer.ResourceLocation() == "minecraft:observer" {\n\t\t\t\tobserver = bedrockCopyBlock(observer)\n\t\t\t\tif observer.Properties["powered"] == "true" {\n\t\t\t\t\tobserver.Properties["powered"] = "false"\n\t\t\t\t} else {\n\t\t\t\t\tobserver.Properties["powered"] = "true"\n\t\t\t\t\ts.world.BlockPhysics.ScheduleObserver(u.X, u.Y, u.Z, s.worldAge, 2)\n\t\t\t\t}\n\t\t\t\ts.world.SetBlock(u.X, u.Y, u.Z, observer)\n''',
)

# Snowballs are always zero-damage except against blazes, even if a caller
# accidentally supplied a generic projectile damage value.
replace_exact(
    "server/server.go",
    '''\tif projectile.Type == corentity.TypeSnowball && target.Type == corentity.TypeBlaze {\n\t\treturn 3\n\t}\n\treturn projectile.ProjectileDamage\n''',
    '''\tif projectile.Type == corentity.TypeSnowball {\n\t\tif target.Type == corentity.TypeBlaze {\n\t\t\treturn 3\n\t\t}\n\t\treturn 0\n\t}\n\treturn projectile.ProjectileDamage\n''',
)

# Add an impact-only queue so zero-damage snowballs still produce hurt/panic/
# knockback semantics through the same simulation-thread path.
replace_exact(
    "core/world/world.go",
    '''func (w *World) QueueEntityDamageFrom(entityID int32, amount float32, sourceX, sourceZ float64) bool {\n\treturn w.queueEntityDamage(entityID, amount, sourceX, sourceZ, true, [16]byte{}, false)\n}\n''',
    '''func (w *World) QueueEntityDamageFrom(entityID int32, amount float32, sourceX, sourceZ float64) bool {\n\treturn w.queueEntityDamage(entityID, amount, sourceX, sourceZ, true, [16]byte{}, false)\n}\n\n// QueueEntityImpactFrom records a zero-health-damage hit with a source. It is\n// used by projectiles such as snowballs that still trigger the vanilla hurt\n// reaction and knockback without reducing ordinary entity health.\nfunc (w *World) QueueEntityImpactFrom(entityID int32, sourceX, sourceZ float64) bool {\n\tif _, ok := w.Entities.Get(entityID); !ok {\n\t\treturn false\n\t}\n\tw.damageMu.Lock()\n\tw.pendingDamage[entityID] = EntityDamage{Amount: 0, SourceX: sourceX, SourceZ: sourceZ, HasSource: true}\n\tw.damageMu.Unlock()\n\treturn true\n}\n''',
)
replace_exact(
    "server/server.go",
    '''\t\t\t} else if damage := projectileDamageAgainst(projectile, target); damage > 0 {\n\t\t\t\tif owner := s.playerByEntityID(projectile.OwnerEntityID); owner != nil {\n\t\t\t\t\ts.world.QueueEntityDamageFromPlayer(target.EntityID, damage, start.X, start.Z, owner.UUID)\n\t\t\t\t} else {\n\t\t\t\t\ts.world.QueueEntityDamageFrom(target.EntityID, damage, start.X, start.Z)\n\t\t\t\t}\n\t\t\t}\n''',
    '''\t\t\t} else if damage := projectileDamageAgainst(projectile, target); damage > 0 {\n\t\t\t\tif owner := s.playerByEntityID(projectile.OwnerEntityID); owner != nil {\n\t\t\t\t\ts.world.QueueEntityDamageFromPlayer(target.EntityID, damage, start.X, start.Z, owner.UUID)\n\t\t\t\t} else {\n\t\t\t\t\ts.world.QueueEntityDamageFrom(target.EntityID, damage, start.X, start.Z)\n\t\t\t\t}\n\t\t\t} else if projectile.Type == corentity.TypeSnowball {\n\t\t\t\ts.world.QueueEntityImpactFrom(target.EntityID, start.X, start.Z)\n\t\t\t}\n''',
)

# ---------------------------------------------------------------------------
# Decorated-pot shatter/intact rules. The shared loot table already contains
# the vanilla alternatives; expose the tool tag and make live break paths pass
# Silk Touch state and set cracked only for a valid shattering tool.
# ---------------------------------------------------------------------------
append_once(
    "core/blockloot/loot.go",
    "func BreaksDecoratedPot",
    '''// BreaksDecoratedPot reports whether the item is in vanilla's\n// #minecraft:breaks_decorated_pots item tag.\nfunc BreaksDecoratedPot(itemID string) bool {\n\treturn data().itemTagSets["minecraft:breaks_decorated_pots"][itemID]\n}\n''',
)

# Java live loot context currently omits enchantments. Populate it and mark a
# decorated pot cracked only for a non-Silk shattering tool before evaluating.
replace_exact(
    "java/handler/block.go",
    '''\t\tlootContext := blockloot.Context{\n\t\t\tBlock: broken,\n\t\t\tTool:  held,\n''',
    '''\t\tlootBlock := broken\n\t\tif broken.ResourceLocation() == "minecraft:decorated_pot" && held.EnchantmentLevel("minecraft:silk_touch") == 0 && blockloot.BreaksDecoratedPot(held.ItemID) {\n\t\t\tlootBlock = copyBlockProperties(broken)\n\t\t\tlootBlock.Properties["cracked"] = "true"\n\t\t}\n\t\tenchantments := make(map[string]int)\n\t\tfor _, enchantment := range held.EnchantmentLevels() {\n\t\t\tenchantments[enchantment.ID] = enchantment.Level\n\t\t}\n\t\tlootContext := blockloot.Context{\n\t\t\tBlock:        lootBlock,\n\t\t\tTool:         held,\n\t\t\tEnchantments: enchantments,\n''',
)

# Bedrock break path uses the same shared loot evaluator.
replace_exact(
    "server/server.go",
    '''\t\tlootContext := blockloot.Context{\n\t\t\tBlock: block,\n\t\t\tTool:  held,\n''',
    '''\t\tlootBlock := block\n\t\tif block.ResourceLocation() == "minecraft:decorated_pot" && held.EnchantmentLevel("minecraft:silk_touch") == 0 && blockloot.BreaksDecoratedPot(held.ItemID) {\n\t\t\tlootBlock = bedrockCopyBlock(block)\n\t\t\tlootBlock.Properties["cracked"] = "true"\n\t\t}\n\t\tenchantments := make(map[string]int)\n\t\tfor _, enchantment := range held.EnchantmentLevels() {\n\t\t\tenchantments[enchantment.ID] = enchantment.Level\n\t\t}\n\t\tlootContext := blockloot.Context{\n\t\t\tBlock:        lootBlock,\n\t\t\tTool:         held,\n\t\t\tEnchantments: enchantments,\n''',
)

# The dynamic cracked-pot entry is the four original ingredients. Until a pot
# has explicit decoration data, vanilla's default is four bricks.
replace_exact(
    "core/blockloot/loot.go",
    '''\tcase "minecraft:dynamic":\n\t\t// Dynamic container contents are handled by the world container store.\n\t\treturn nil, true\n''',
    '''\tcase "minecraft:dynamic":\n\t\tif ctx.Block.ResourceLocation() == "minecraft:decorated_pot" {\n\t\t\treturn []player.ItemStack{{ItemID: "minecraft:brick", Count: 4}}, true\n\t\t}\n\t\t// Other dynamic container contents are handled by the world container store.\n\t\treturn nil, true\n''',
)

# ---------------------------------------------------------------------------
# Regression tests for adapter placement, observer timing, pot rules, snowball.
# ---------------------------------------------------------------------------
Path("java/handler/vanilla_parity_test.go").write_text(r'''package handler

import (
    "testing"

    "GoCraft/core/player"
    coreworld "GoCraft/core/world"
)

func TestJavaLeverPlacementAllAttachmentFaces(t *testing.T) {
    tests := []struct {
        face int32
        wantFace, wantFacing string
    }{
        {0, "ceiling", "south"}, {1, "floor", "south"},
        {2, "wall", "north"}, {3, "wall", "south"},
        {4, "wall", "west"}, {5, "wall", "east"},
    }
    for _, tc := range tests {
        w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
        x, y, z := 0, 64, 0
        off := faceOffset[tc.face]
        w.SetBlock(x-int(off[0]), y-int(off[1]), z-int(off[2]), coreworld.Block{Namespace:"minecraft", Name:"stone"})
        got, ok := javaButtonPlacementState(coreworld.Block{Namespace:"minecraft", Name:"lever"}, tc.face, 0, w, x, y, z)
        if !ok { t.Fatalf("face %d rejected", tc.face) }
        if got.Properties["face"] != tc.wantFace || got.Properties["facing"] != tc.wantFacing {
            t.Fatalf("face %d => face=%q facing=%q, want %q/%q", tc.face, got.Properties["face"], got.Properties["facing"], tc.wantFace, tc.wantFacing)
        }
        w.Close()
    }
}

func TestJavaDoorPlacementUsesSharedVanillaHinge(t *testing.T) {
    p := player.New([16]byte{}, "door", player.ClientEditionJava)
    w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
    defer w.Close()
    w.SetBlock(0, 63, 0, coreworld.Block{Namespace:"minecraft", Name:"stone"})
    p.Rotation.Yaw = 180 // north in GoCraft's placement convention
    if !placeDoorBlock(p, 0, 64, 0, "minecraft:oak_door", 0.2, 0.5, w, nil) { t.Fatal("door placement rejected") }
    lower := w.GetBlock(0,64,0)
    upper := w.GetBlock(0,65,0)
    if lower.Properties["hinge"] != coreworld.DoorHinge(w,0,64,0,lower.Properties["facing"],0.2,0.5) { t.Fatalf("lower hinge=%q", lower.Properties["hinge"]) }
    if upper.Properties["hinge"] != lower.Properties["hinge"] || upper.Properties["facing"] != lower.Properties["facing"] { t.Fatalf("upper/lower mismatch: %+v %+v", lower.Properties, upper.Properties) }
}
''')

append_once(
    "core/world/vanilla_interaction_test.go",
    "TestObserverSchedulesTwoTickPulse",
    r'''func TestObserverSchedulesTwoTickPulse(t *testing.T) {
    w := New(&FlatGenerator{}, nil, false)
    defer w.Close()
    w.SetPhysicsTime(100)
    w.SetBlock(0,64,0, Block{Namespace:"minecraft", Name:"observer", Properties:map[string]string{"facing":"east","powered":"false"}})
    // Ignore placement's own pending bookkeeping, then change the exact watched block.
    w.BlockPhysics.DrainDue(100)
    w.SetBlock(1,64,0, Block{Namespace:"minecraft", Name:"stone"})
    if got := w.GetBlock(0,64,0).Properties["powered"]; got == "true" { t.Fatal("observer powered before scheduled detection tick") }
    if due := w.BlockPhysics.DrainDue(101); len(due) != 0 { t.Fatalf("observer fired early: %+v", due) }
    due := w.BlockPhysics.DrainDue(102)
    if len(due) != 1 || due[0].Kind != UpdateObserver { t.Fatalf("due=%+v, want one observer update", due) }
}

func TestObserverOnlyWatchesItsFront(t *testing.T) {
    w := New(&FlatGenerator{}, nil, false)
    defer w.Close()
    w.SetPhysicsTime(10)
    w.SetBlock(0,64,0, Block{Namespace:"minecraft", Name:"observer", Properties:map[string]string{"facing":"east","powered":"false"}})
    w.BlockPhysics.DrainDue(10)
    w.SetBlock(0,64,-1, Block{Namespace:"minecraft", Name:"stone"})
    if due := w.BlockPhysics.DrainDue(12); len(due) != 0 { t.Fatalf("side change triggered observer: %+v", due) }
    w.SetBlock(1,64,0, Block{Namespace:"minecraft", Name:"fire"})
    if due := w.BlockPhysics.DrainDue(12); len(due) != 1 || due[0].Kind != UpdateObserver { t.Fatalf("front fire change did not trigger: %+v", due) }
}
''',
)

append_once(
    "core/blockloot/loot_test.go",
    "TestDecoratedPotVanillaBreakRules",
    r'''func TestDecoratedPotVanillaBreakRules(t *testing.T) {
    intact := Drops(Context{Block:block("decorated_pot", map[string]string{"cracked":"false"}), Tool:player.ItemStack{}, Random:rand.New(rand.NewSource(1))})
    wantDrop(t, intact, "minecraft:decorated_pot", 1)

    silk := Drops(Context{Block:block("decorated_pot", map[string]string{"cracked":"false"}), Tool:player.ItemStack{ItemID:"minecraft:diamond_pickaxe", Count:1}, Enchantments:map[string]int{"minecraft:silk_touch":1}, Random:rand.New(rand.NewSource(1))})
    wantDrop(t, silk, "minecraft:decorated_pot", 1)

    cracked := Drops(Context{Block:block("decorated_pot", map[string]string{"cracked":"true"}), Tool:player.ItemStack{ItemID:"minecraft:diamond_pickaxe", Count:1}, Random:rand.New(rand.NewSource(1))})
    if count(cracked, "minecraft:brick") != 4 { t.Fatalf("cracked pot drops=%+v, want four bricks", cracked) }

    if !BreaksDecoratedPot("minecraft:diamond_pickaxe") { t.Fatal("diamond pickaxe missing breaks_decorated_pots tag") }
    if BreaksDecoratedPot("") { t.Fatal("empty hand incorrectly shatters pot") }
}
''',
)

Path("server/projectile_parity_test.go").write_text(r'''package server

import (
    "testing"
    corentity "GoCraft/core/entity"
)

func TestSnowballDamageIsZeroExceptBlaze(t *testing.T) {
    snowball := corentity.New(1, [16]byte{}, corentity.TypeSnowball, 0,0,0)
    snowball.ProjectileDamage = 99 // generic/default damage must never leak through.
    pig := corentity.New(2, [16]byte{}, corentity.TypePig, 0,0,0)
    blaze := corentity.New(3, [16]byte{}, corentity.TypeBlaze, 0,0,0)
    if got := projectileDamageAgainst(snowball, pig); got != 0 { t.Fatalf("snowball->pig=%v, want 0", got) }
    if got := projectileDamageAgainst(snowball, blaze); got != 3 { t.Fatalf("snowball->blaze=%v, want 3", got) }
}
''')
