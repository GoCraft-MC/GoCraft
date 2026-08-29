# Vanilla interaction parity audit

Audit date: 2026-08-30. Comparison baseline: GoCraft `vanilla-parity-fix-v2`,
Dragonfly v0.11 block/item behaviours, PumpkinMC's bell and firework paths, and
Java/Bedrock vanilla protocol behaviour. This audit covers interaction and
gameplay code, not whether an item merely exists in a registry.

## Implemented correctly (inspected subset)

- Shared attachment placement chooses standing/wall signs and banners, hanging
  sign variants, and torch support states. Bedrock receives its legacy block
  names and support-facing wall-torch state without changing canonical state.
- Decorated pots place with canonical sherd data, accept items, spill/drop
  contents, and use the proper tool-sensitive cracked loot path.
- Basic water/lava source bucket pickup and placement is present in both entry
  paths, including full cauldron transfer and creative item semantics.
- Basic throwable entities (snowballs, eggs, ender pearls, experience bottles,
  splash potions and wind charges) use canonical entities and server ticking.

## Partially implemented

- Bells now ring from valid player hits, redstone rising edges, and projectile
  impacts. Animation and bell sound are translated for both editions and scoped
  to the dimension. Raid resonance/raider glow is not implemented.
- Firework rockets now launch from block use, preserve supported flight and
  explosion components, tick, accelerate, explode, damage visible nearby living
  targets, consume in non-creative modes, and synchronise to both editions.
  Firework-star crafting components, crossbow-loaded rockets, and Elytra boost
  remain incomplete; GoCraft has no canonical fall-flying state yet.
- Doors, trapdoors, fence gates, buttons, levers, pressure plates, repeaters and
  comparators mutate and propagate their important state. Several Bedrock sound
  events and projectile activation rules (wooden buttons/plates) are incomplete.
- Note blocks can be tuned on Bedrock and retain redstone state, but Java tuning,
  instrument selection, note sound/particle output, and redstone note playback
  are incomplete.
- Cauldrons handle basic full bucket transfers. Bottles, dyed leather armour,
  precipitation/dripstone filling and incremental levels are missing.
- Composters accept a supported compostable set, advance, mature, and dispense
  bone meal, but probability/table coverage and feedback need a full data audit.
- Bee nests/beehives expose honey harvesting, bottles and shears, but bees,
  anger, smoke pacification and occupants are not modelled.
- Respawn anchors accept glowstone charges. Nether spawn assignment, explosion
  outside the Nether and charge consumption on respawn are missing.
- Campfires place/light/extinguish and cook supported recipes. Four-slot visual
  item progress, smoke/hay effects, projectile lighting and complete drop rules
  are incomplete.
- Candles and cakes support basic placement/stacking/eating/extinguishing.
  Candle-cake creation, lighting feedback and all cross-edition sounds are not
  complete.
- Flower pots place and survive on support, but the full pot/unpot interaction
  and contents block-entity behaviour is incomplete.
- Signs and banners now physically place and render on both editions with empty
  block-entity data. Text editing, wax/dye/glow, banner patterns and loom-to-block
  component preservation are not implemented.
- Anvils, grindstones, stonecutters, smithing tables, brewing stands, enchanting
  tables, looms and cartography tables open and have varying amounts of inventory
  logic. They do not all implement full costs, XP, recipe/property data, sounds,
  automation and edition-specific validation.
- Buckets do not yet cover fish/axolotl/tadpole entities, milk use, waterlogging
  every supported block, dispensers or all cauldron cases.
- Shears cover pumpkin carving, beehive harvest and loot-sensitive foliage.
  Sheep shearing, vines/tripwire and complete durability/sound behaviour remain.
- Bone meal covers the supported crops and growth helpers, but not every plant,
  underwater fertilisation, grass feature spreading or all particles.
- Flint and steel/fire charges ignite common targets and prime TNT. Portal
  ignition and every special target/feedback path are incomplete.
- Bows/crossbows/tridents/shields have Java-side draw/load/projectile/blocking
  logic. Bedrock use-state parity, every enchantment, trident loyalty/riptide,
  crossbow fireworks and shield disable rules are incomplete.
- Maps can be created and used in workstation recipes, but terrain tracking,
  decorations, scale/lock data and live map packets are absent.
- Boats and minecarts have canonical entities, placement, riding and basic
  physics. Collision, inventory variants, fluid/rail details and complete
  cross-edition controls still need work.

## Missing

- Jukebox record insertion/ejection, playback, comparator state and sounds.
- Lectern book insertion/removal/page state, UI and comparator output.
- Chiseled bookshelf slot interaction, block-entity inventory and comparator.
- Item frames/glow item frames and armour stands as placeable entities.
- Lodestone compass binding and tracked target data.
- Brush archaeology and suspicious-block progress/loot.
- Fishing rod hook entity, bobber physics, loot and reel interaction.
- Functional filled-map rendering and tracking.

## Implemented but wrong / not vanilla

- Java boat block-use currently permits spawning against any surface; vanilla
  placement must raycast a valid fluid surface and reject obstructed placement.
- Note blocks silently change Bedrock state and have no canonical play event;
  tuning without sound/particle output is not vanilla behaviour.
- Several interaction families still have separate Java and Bedrock mutation
  functions. They often converge on the same world state, but differences in
  sounds, validation and item feedback remain observable.

## Priorities after this pass

1. Add a canonical gliding/fall-flying state, then implement Elytra firework boost.
2. Complete sign editing and banner-pattern component preservation.
3. Add a shared note-block play event and instrument calculation.
4. Fix boat raycast placement and finish vehicle collision/inventory behaviour.
5. Implement jukeboxes, lecterns and chiseled bookshelves using canonical block
   entities instead of protocol-specific container shortcuts.
