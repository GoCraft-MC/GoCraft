# Dragonfly gameplay and interaction parity audit

Audit date: 2026-09-02. GoCraft baseline: `missing-items` at `00685b7`.
Reference implementation: Dragonfly v0.11.0 from the resolved Go module source.
Protocol targets: Java Edition 1.21.4 and Bedrock Edition 1.26.45.

This is a static code audit of behaviour, not a registry-presence check. An item
is not considered implemented merely because it appears in the Java data pack,
Bedrock creative catalogue, a recipe, or a trade. The audit follows the action
from adapter input through canonical state mutation, persistence, and feedback
to both editions.

Dragonfly v0.11.0 has 59 item files and 118 block files with explicit use,
activation, entity-inside, neighbour, random-tick, scheduled-tick, or redstone
behaviour. Dragonfly is a Bedrock-first comparison point rather than a complete
specification. Java 1.21.4 behaviour not represented by Dragonfly is included
where GoCraft's declared Java target still requires it.

## Status legend

| Status | Meaning |
| --- | --- |
| Implemented | The inspected path performs the main server-authoritative behaviour on this adapter. Edge cases may remain. |
| Partial | The action starts or renders, but important state, validation, effects, persistence, or feedback is absent. |
| Missing | No gameplay path was found; registry or UI presence alone does not count. |
| Adapter gap | One edition has a gameplay path and the other does not. |
| N/A | The feature is outside the Java 1.21.4 target because of Bedrock/Dragonfly version skew. |

## Highest-impact findings

1. `player.ItemStack` cannot represent most stateful vanilla components. It
   currently preserves item ID, count, damage, enchantments, pot decorations,
   and fireworks. Potion contents, stew effects, book pages, map data, lodestone
   targets, banner patterns, dyed colour, armour trims, goat-horn instrument,
   crossbow charge, bundle contents, shulker contents, custom names, and lore
   have no canonical storage.
2. GoCraft has no canonical timed status-effect system for players. Several
   foods and the totem send effect packets, but poison, regeneration,
   absorption, resistance, fire resistance, and similar effects do not alter
   authoritative health, damage, movement, or expiry state.
3. Java and Bedrock still dispatch many interactions through separate hardcoded
   switches. This has already produced two large adapter-only sets: several
   block/item actions exist only on Bedrock, while ranged weapons and shields
   exist only on Java.
4. Stateful use is stored as one player-wide `UsingItemID`, timestamp, and
   hotbar index. It cannot model both hands, a charged crossbow stack, potion
   payloads, spyglass state, horn cooldown, or a persistent gliding state.
5. Block breaking trusts the client's completion event. Bedrock calculates a
   Dragonfly break duration for crack feedback, but neither edition validates
   elapsed mining time, tool speed, haste/fatigue, being underwater, or being
   airborne before accepting completion.

These structural gaps should be fixed before adding many isolated switch cases;
otherwise stateful items will appear to work and then lose their data during an
inventory move, save, cross-edition sync, or restart.

## Item and item-use audit

### Food, drink, and effects

| Item or family | Java | Bedrock | Missing or incomplete behaviour |
| --- | --- | --- | --- |
| Ordinary registry-backed foods | Implemented | Implemented | Hunger, saturation, use duration, full-hunger checks, stack consumption, and bowl/bottle remainders are shared. |
| Rotten flesh, raw chicken, spider eye, poisonous potato, pufferfish | Partial | Partial | Java sends client effect packets only. Bedrock's normal timed-use completion does not call the special-effect path; its legacy inventory-consume path does. Neither path stores or ticks player effects authoritatively. |
| Golden apple and enchanted golden apple | Partial | Partial | Food is consumed, but regeneration, absorption, resistance, and fire resistance are client-visible packets rather than canonical effects. Bedrock's normal timed-use path omits them. |
| Honey bottle | Partial | Partial | Nutrition and bottle remainder work. Dragonfly removes poison; GoCraft has no player poison state to remove. |
| Suspicious stew | Partial | Partial | Nutrition and bowl remainder work, but the flower-selected effect is not stored or applied. Every stew stack collapses to the same canonical item. |
| Drinkable potions | Missing | Missing | They are not accepted by the food-use dispatcher, potion type is not stored, the bottle remainder is not produced, and effects are not applied. |
| Splash potions | Missing | Missing | Dispensers and witches may create a generic potion projectile, but players cannot throw one and impacts only play a sound. Potion payload, distance-scaled effects, colour, and instant effects are absent. |
| Lingering potions | Missing | Missing | No player throw path, potion payload, area-effect cloud entity, radius decay, reapplication delay, or tipped-arrow interaction. |
| Milk bucket | Missing | Missing | No drink use, bucket remainder, or effect clearing. |

### Weapons, use-state, and utility items

| Item or family | Java | Bedrock | Missing or incomplete behaviour |
| --- | --- | --- | --- |
| Snowball, egg, ender pearl, experience bottle, wind charge | Implemented | Implemented | Shared projectiles include motion and impact behaviour. Enchantments and several edition-specific particles/sounds still need coverage. |
| Bow | Partial | Missing | Java draws and fires arrows, but Power, Punch, Flame, Infinity, spectral/tipped payloads, pickup rules, and per-shot critical state are incomplete. Bedrock has no bow use-state path. |
| Crossbow | Partial | Missing | Java has a two-step load/fire flow, but charge is a player-wide boolean and is lost on slot change. Quick Charge, Multishot, Piercing, charged projectile persistence, offhand priority, and firework ammunition are missing. Bedrock has no use path. |
| Trident | Partial | Missing | Java throws a basic projectile. Loyalty, Riptide, Channeling, Impaling, pickup/return ownership, wet checks, and per-stack state are missing. Bedrock has no use path. |
| Shield | Partial | Missing | Java main-hand blocking exists after a fixed delay. Offhand use cannot be started, and axe disable, durability loss, projectile/explosion rules, cooldown, and full feedback are incomplete. Bedrock has no shield use-state path. |
| Firework rocket | Partial | Partial | Ground launch, preserved rocket explosion data, ticking, explosion, and damage work on both adapters. Elytra boost and crossbow-fired rockets are missing. |
| Firework star | Partial | Partial | Some crafting/component decoding exists, but a firework-star component is not preserved as an item stack and all crafting transformations are not round-trippable. |
| Elytra | Partial | Partial | Equipping is possible. Java accepts the start-fall-flying action only by resetting fall distance; no canonical gliding flag or glide physics exists. Bedrock glide input is not modelled, and rocket boost is missing on both. |
| Totem of undying | Partial | Partial | Canonical death prevention and consumption run for both editions. Java receives its animation/effect packets; Bedrock receives neither through the totem path. The granted effects are not authoritative on either edition. |
| Goat horn | Missing | Missing | Dragonfly plays the selected instrument with use duration/cooldown. GoCraft stores neither instrument nor cooldown and has no use action. |
| Spyglass | Missing | Missing | No start/stop use state or remote-player using-item metadata. |
| Fishing rod | Missing | Missing | No hook entity, cast/reel state, bobber physics, hooked entity/item handling, durability, or fishing loot. |
| Brush | Missing | Missing | No brushing progress, cooldown, suspicious sand/gravel dust states, block entity, archaeology loot, or durability. |
| Carrot on a stick / warped fungus on a stick | Missing | Missing | Pig/strider steering boost and durability are absent even though riding exists. |
| Compass / recovery compass | Partial | Partial | Static item identity exists. Lodestone target, dimension, tracking flag, last-death target, and canonical compass state are absent. |
| Empty map / filled map | Partial | Partial | Empty map converts to a filled-map item and cartography accepts it, but map allocation, scale, lock, dimension, colours, decorations, exploration-map target, and live map packets are absent. |
| Writable and written books | Missing | Missing | Pages, title, author, generation, editing/signing packets, validation, copying, and open-book interaction are absent. |
| Music discs | Missing | Missing | Items exist, but insertion/ejection and playback require the missing jukebox behaviour. |
| Dyes, ink sacs, glow ink sacs | Missing | Missing | No sheep dye, sign text colour, glowing sign text, collar colour, or other entity/block dye interaction. The loom accepts dye but cannot preserve a pattern result. |
| Shears | Partial | Partial | Bedrock carves pumpkins and harvests full hives; Java has neither action. Loot-sensitive block breaking exists, but sheep shearing, mooshroom conversion, snow-golem carving, tripwire disarm, entity drops, and complete sounds/durability are absent. |
| Glass bottle | Partial | Partial | Bedrock fills from a full beehive. Java does not. Water-source and water-cauldron filling, dragon-breath collection, incremental cauldron levels, and complete remainders are missing. |
| Water/lava/powder-snow buckets | Partial | Partial | Basic source and full-cauldron transfer works. Fish, axolotl, tadpole, and entity capture/release; waterlogging/unwaterlogging; ultrawarm evaporation; and full dispenser/cauldron cases are incomplete. |
| Spawn eggs | Partial | Partial | A dispenser can spawn a generic entity. Player block/entity use, baby spawning, mob-specific NBT/state, placement validation, and unsupported-entity rejection are absent. |
| Armour stand item | Missing | Missing | No placeable armour-stand entity, pose, equipment interaction, damage rules, or drops. |
| Item frame / glow item frame | Missing | Missing | No placement, support validation, contained stack, rotation, map mode, comparator signal, punch removal, or glow behaviour. |
| Bundle | Missing | Missing | No contents/weight component or insert/extract interaction. |
| Shulker-box items | Partial | Partial | Placed storage works, but breaking spills contents and drops a separate empty box. Vanilla must keep the inventory inside the dropped shulker item. |

### Tools and block-use items

| Item or family | Java | Bedrock | Missing or incomplete behaviour |
| --- | --- | --- | --- |
| Axes | Partial | Partial | Stripping, scraping oxidation, and removing wax are present. Complete sound/particle feedback and enchantment durability rules remain. |
| Hoes | Partial | Partial | Tilling works. Bedrock drops hanging roots from rooted dirt; Java does not. Java also applies durability outside the interaction helper while Bedrock applies it inside, leaving duplicate logic to keep aligned. |
| Shovels | Partial | Partial | Dirt paths and campfire extinguishing work. Full flattenable set, sound parity, and durability/enchantment rules need verification. |
| Flint and steel / fire charge | Partial | Partial | Both adapters create fire, light candles/campfires, and ignite portals. Bedrock directly primes TNT; Java has no TNT target branch. Projectile/dispenser and feedback rules remain incomplete. |
| Honeycomb | Missing | Partial | Bedrock waxes copper. Java has axe unwaxing/scraping but no honeycomb waxing action. |
| Bone meal | Partial | Partial | Supported crops and saplings work. Grass-area features, flowers, moss, azalea, mangrove, underwater plants/coral, fungi/nylium, sea pickles, and particles are incomplete. |
| Ender eye | Implemented | Implemented | Stronghold launch and portal-frame insertion are present. Structure search and feedback still deserve runtime tests. |

## Block interaction audit

### Bedrock behaviour with no Java equivalent

These are confirmed adapter gaps in the current switches, not speculative
feature requests.

| Interaction | Bedrock | Java |
| --- | --- | --- |
| Carve pumpkin with shears and drop seeds | Implemented | Missing |
| Harvest a full bee nest/hive with shears or bottle | Implemented | Missing |
| Wax copper with honeycomb | Implemented | Missing |
| Add a candle to a cake | Implemented | Missing |
| Put a plant into, or remove it from, a flower pot | Implemented | Missing |
| Add compostables, mature the composter, collect bone meal | Partial | Missing |
| Charge a respawn anchor with glowstone | Partial | Missing |
| Ignite TNT directly with flint and steel/fire charge | Implemented | Missing |
| Tune a note block | Partial | Missing |

The Java dispatcher contains none of the corresponding item/block names except
for generic placement or support checks. These behaviours should move into
shared canonical operations before Java calls are added.

### Missing or incomplete on both editions

| Block or family | Status | Missing or incomplete behaviour |
| --- | --- | --- |
| Jukebox | Missing | Record insert/eject, stored record, playback event, stop event, comparator output, note particle, persistence, and cross-edition sound translation. |
| Lectern | Missing | Book insert/remove, pages, current page, UI, page-turn events, comparator output, redstone pulse, and persistence. It currently exists only as a generated village workstation/redstone name. |
| Chiseled bookshelf | Missing | Six-slot inventory, targeted slot selection, book insertion/removal, block state, comparator signal, vibration, and persistence. |
| Item/glow item frame | Missing | See item table; Dragonfly implements this as a stateful block. |
| Dragon egg | Partial | Gravity is registered, but activate/punch teleport, particles, and creative exception are absent. |
| Note block | Partial | Bedrock can increment the note and redstone stores `powered`. Java cannot tune it. Neither adapter calculates instrument from the block below or emits canonical note sound/particle on click or rising edge. |
| Signs and hanging signs | Partial | Placement and empty block entities work. Text editing, front/back text, filtering, wax, dye/glow, click events, and persistence/round-trip of text components are absent. |
| Banners | Partial | Placement works. Pattern layers, loom output, shields carrying patterns, map markers, block-entity persistence, and wash-off in cauldrons are absent. |
| Beacon | Partial | Both adapters can open a one-slot screen. Pyramid level, beam obstruction/colour, payment validation, selected effects, periodic area application, and persistence are absent. |
| Enchanting table | Partial | The screen accepts item/lapis slots, but offers, bookshelf power, seed, XP/lapis cost, selection packet, and enchant application are absent. |
| Brewing stand | Partial | Persistent slots and generic hopper movement exist, but blaze fuel, brew timer, ingredient transformations, potion payloads, bottle properties, and slot-aware automation are absent. |
| Anvil | Partial | Basic material/same-item repair exists. Rename, enchant merging, prior-work penalty, XP cost, too-expensive rules, anvil damage, and output feedback are absent. |
| Grindstone | Partial | Basic combining/output exists. Enchantment removal rules, curses, XP refund, costs, and sounds are absent. |
| Smithing table | Partial | Upgrade item-ID transforms exist. Armour trims and trim material/pattern components cannot be produced or preserved. |
| Loom | Partial | It consumes a banner and dye but returns an unchanged banner because pattern data has no canonical representation. Selection, six-layer limit, and full pattern rules are absent. |
| Stonecutter | Partial | Recipe selection/output exists. Adapter selection parity and complete feedback/validation require tests. |
| Cartography table | Partial | It returns a generic filled map for paper, map, or glass pane. Scale, clone count, lock state, map identity, and data preservation are absent. |
| Respawn anchor | Partial | Bedrock can charge it only. Nether spawn assignment, charge use on respawn, comparator output, explosion outside the Nether, and Java interaction are absent. |
| Bee nest / beehive | Partial | Bedrock can harvest honey. Bees, occupants, entry/exit, honey production, anger, smoke pacification, Silk Touch data, dripping, and Java harvest are absent. |
| Campfire | Partial | Placement, lighting/extinguishing, four stored cooking slots, cooking completion, and damage are present. Item rendering, per-slot progress persistence, smoke height, hay signal, bee pacification, projectile lighting, soul variants, and complete drop/waterlogging rules remain. |
| Candles / candle cakes / cake | Partial | Core stacking, eating, lighting, and extinguishing exist, but Java candle-cake creation, projectile lighting, cake collision details, particles, sounds, and complete waterlogging/support behaviour remain. |
| Cauldrons | Partial | Full bucket transfers work. Bottles, incremental water/powder levels, dyed leather washing/dyeing, banner cleaning, shulker dyeing, precipitation, dripstone filling, entity extinguishing, and comparator details are absent. |
| Sponge | Partial | Wet-sponge smelting and bucket remainder exist. Dry-sponge water absorption, 65-block search, wet conversion, neighbour reaction, Nether drying, and particles are absent. |
| Coral and coral blocks | Partial | Generation/placement exists. Loss-of-water scheduled death and dead variants are absent. |
| Cactus | Partial | Contact damage and support removal are present. Random growth, height limit, side-neighbour survival, item destruction, and entity collision details are absent. |
| Sugar cane | Partial | Placement/support removal exists. Random growth, height limit, water-adjacency validation, and complete survival rules are absent. |
| Bamboo / bamboo sapling | Partial | Generation, placement, support, and bamboo block axis exist. Sapling conversion, random growth, leaves/stage/age transitions, height selection, and bone-meal growth are absent. |
| Kelp | Partial | Placement/data exists. Random growth, age, water-column conversion, top/body state, and support updates are absent. |
| Vines / cave vines / twisting and weeping vines | Partial | Some placement/generation exists. Climbing state, random spread/growth, berry harvest, bone meal, support updates, and shears rules are incomplete or absent. |
| Cocoa | Partial | A bounded legacy growth tick exists. Correct jungle-log attachment/facing survival, placement interaction, and bone meal are absent. |
| Fire | Partial | Placement, scheduled spread/burnout, and contact damage exist. Fire immunity/effects, rain extinguishing, portal interaction details, gamerules, block-specific flammability, and complete soul-fire rules remain. |
| Fluids and waterlogging | Partial | Source placement, simple spread, collision checks, and lava/water hardening exist. Flow vectors, finite levels/source formation, waterlogged fluid ticking, ultrawarm evaporation, entity pushing, dripstone, and many replaceability rules are incomplete. |
| Boats and minecarts | Partial | Placement, mounting, basic movement, rails, powered/detector/activator effects, and TNT minecart fuse exist. Placement is too permissive, and collision, fluid physics, fall damage, passenger rules, chest/hopper/furnace inventories, furnace fuel, and complete cross-edition control are missing. |

### Broadly implemented but still needs conformance tests

The following families have a meaningful shared implementation and are not the
first missing-feature targets: beds and respawn points, ordinary chests/barrels
and per-player ender chests, furnaces/smokers/blast furnaces, crafting grids,
doors/trapdoors/fence gates, levers/buttons/pressure plates, repeaters,
comparators, daylight detectors, redstone wire/torches/lamps, pistons,
observers, sculk sensors, rails, hoppers, droppers, dispensers, crafters,
decorated pots, crop/stem growth, sapling trees, falling blocks, portals,
explosions, and ordinary attachment placement. Their remaining edge cases
should be covered by differential tests rather than being labelled complete.

## Entity interaction audit

GoCraft has a useful shared base for feeding, baby growth, breeding, taming,
sitting, saddling, mounting, villagers, boats, minecarts, and basic attacks.
The following interaction families are still missing or incomplete on both
adapters unless noted:

| Interaction | Status | Gap |
| --- | --- | --- |
| Sheep shearing and dyeing | Missing | No wool colour/state, regrowth, sheared flag, drops, sound, or tool durability action. |
| Cow/mooshroom/goat milking | Missing | Buckets cannot create milk; mooshroom bowls/stew and conversion interactions are absent. |
| Fish/axolotl/tadpole bucket capture | Missing | No capture/release data, bucket replacement, variant preservation, or water placement. |
| Leads and leash knots | Missing | No leash ownership, fence knot entity, distance physics, drop, or detach interaction. |
| Name tags | Missing | No custom-name component, anvil naming prerequisite, entity naming, persistence, or visibility rules. |
| Horse/donkey/llama equipment | Partial | Saddling/mounting exists. Inventory UI, armour, carpet, chest attachment, storage, jump charge, temper/buck details, and equipment persistence are absent. |
| Wolf/cat/parrot ownership | Partial | Taming and sit toggle exist. Collar dye, follow/teleport rules, owner defence breadth, shoulder parrots, gifts, and complete metadata are absent. |
| Turtle/frog/sniffer/armadillo special actions | Partial | Generic food/breeding tables exist, but egg laying, scute/drop timing, frogspawn, sniff/dig, brushing, rolling, and species-specific goals are absent. |
| Villager trading | Partial | Trading UI/catalogue exists. Gossip, restocking rules, demand, curing discounts, reputation, wandering trader details, and full profession workflows are incomplete. |
| Chest boats and storage minecarts | Missing | Entity types can spawn, but there is no portable container state or open/transfer interaction. |
| Armour stands | Missing | No entity implementation or interactions. |

## Player actions, combat, and survival audit

| Action or system | Java | Bedrock | Gap |
| --- | --- | --- | --- |
| Movement, sprint, sneak, flight permission | Implemented | Implemented | Swimming, crawling, gliding, pose transitions, collision validation, and speed-effect integration are incomplete. |
| Drop one stack / drop one item | Missing | Implemented | Java Player Action statuses 3 and 4 are not handled. Bedrock inventory transactions support drops. |
| Swap main hand and offhand | Missing | Partial | Java Player Action status 6 is not handled. Bedrock inventory swaps exist, but use actions still assume a hotbar item. |
| Offhand use | Missing | Missing | Java acknowledges and ignores non-main-hand `Use Item`; Bedrock use intents carry only a hotbar slot. Shields, food, rockets, and utility items therefore cannot use normal offhand semantics. |
| Melee attack | Partial | Partial | Basic damage, range, cooldown gate, armour/toughness, knockback, mace fall bonus, and durability exist. Critical hits, sweeping attacks, attack-speed-per-item timing, fire aspect, damage/knockback enchantments, shield disable, statistics, and exhaustion are missing. |
| Armour and defensive enchantments | Partial | Partial | Base armour/toughness/knockback resistance work. Protection families, Feather Falling, Thorns, Respiration, Aqua Affinity, Soul Speed, Swift Sneak, Frost Walker, and equipment-triggered effects are not applied. |
| Projectile combat | Partial | Partial | Shared collision/damage exists. Tipped/spectral effects, arrow embedding/pickup, criticals, bow/crossbow/trident enchantments, owner rules, and shield interaction are incomplete. |
| Status effects | Missing | Missing | Client effect packets exist, but there is no canonical effect collection, tick/expiry engine, attribute modification, periodic damage/heal, immunity, cure, persistence, or cross-edition sync source. |
| Hunger, regeneration, starvation | Partial | Partial | Core hunger/exhaustion and natural regeneration/starvation exist. Activity costs, difficulty rules, status-effect interaction, peaceful behaviour, and all exhaustion sources are incomplete. |
| Fall, fire, lava, cactus, berry, void, drowning | Partial | Partial | Main damage paths exist. Effect/enchantment mitigation, fire ticks, freezing/powder snow, suffocation, cramming, border damage, lightning, and many block hazards are absent or simplified. |
| Sleep and respawn | Partial | Partial | Beds set spawn and sleep at night. Occupancy, monsters nearby, dimension explosions, obstruction, sleep percentage/gamerules, anchor respawn, charge use, and exact wake placement remain. |

## Container and automation audit

- Ender chests are correctly backed by each player's `EnderChestInventory` on
  both adapters; the previous generic-world-container concern no longer applies.
- Ordinary chests, double chests, barrels, furnaces, hoppers, dispensers,
  droppers, crafters, and placed shulker boxes have canonical storage and basic
  adapter UIs.
- Shulker boxes use the wrong break contract: contents spill into the world and
  the dropped box cannot retain them.
- Hopper insertion filtering is specialised only for furnaces. Brewing stands
  and other sided inventories accept items too generically, and hopper
  minecarts/chest minecarts do not expose storage.
- Dispensers support a useful subset: basic projectiles, TNT, water/lava
  buckets, bone meal, flint and steel, and spawn eggs. Missing families include
  armour/equipment, boats/minecarts, shears, glass bottles, honeycomb, skulls,
  pumpkins, shulker placement, and correct potion/arrow payloads.
- Enchanting, brewing, beacon, loom, cartography, and smithing screens may open
  successfully even when their defining operation is absent. UI-open success
  must not be used as a parity signal.

## Bedrock/Java version-skew exclusions

Dragonfly v0.11.0 contains newer Bedrock content such as copper golem statues,
copper lanterns/torches/chains/bars, additional copper oxidation families, and
other items not present in Java 1.21.4. Those features are **N/A for Java
parity**, not Java bugs. They still require a separate Bedrock 1.26.45 audit for
placement, activation, oxidation, waxing, loot, recipes, entities, and network
states. GoCraft should not expose them to Java 1.21.4 unless it intentionally
defines them as custom content with a resource pack.

## Implementation order

1. Add extensible canonical item components and update inventory equality,
   persistence, Java component codecs, Bedrock NBT codecs, containers, dropped
   items, and recipes together.
2. Add a canonical player status-effect engine. Route foods, potions, beacons,
   totems, mobs, commands, milk, and adapter packets through it.
3. Replace the edition-specific block-use switches with shared item-use and
   block-activation operations; first port the nine confirmed Bedrock-only
   interactions to Java.
4. Add canonical hand/use state, charged-crossbow state, and gliding. Then wire
   Bedrock bow/crossbow/trident/shield use and Elytra rockets.
5. Implement the stateful block entities: jukebox, lectern, chiseled bookshelf,
   signs, banners, item frames, and portable shulker contents.
6. Implement missing entity interactions: buckets/milking, sheep, leads, name
   tags, animal equipment, and vehicle inventories.
7. Add random-tick/neighbor behaviours for sponge, coral, cactus, sugar cane,
   bamboo, kelp, and vines.
8. Add adapter conformance tests that run the same canonical interaction
   scenario through Java and Bedrock inputs and compare world, player,
   inventory, entity, sound, particle, and persistence results.

## Inspected source areas

- GoCraft canonical state: `core/player`, `core/entity`, `core/world`,
  `core/itemregistry`, `core/blockloot`, and `core/intent`.
- GoCraft Java actions: `java/handler/block.go`, `inventory.go`,
  `projectile.go`, `health.go`, `boat.go`, `trade.go`, `chest.go`,
  `workstation.go`, `crafting.go`, and `play.go`.
- GoCraft Bedrock actions: `bedrock/listener.go`, `bedrock/sync.go`,
  `server/bedrock_actions.go`, `bedrock_container.go`, `server.go`,
  `container_automation.go`, `animal_interaction.go`, and `firework.go`.
- Dragonfly reference: `server/item`, `server/block`, item NBT/component types,
  and their activation/use interfaces in the v0.11.0 module source.

This document is a code-derived backlog. Each `Missing` or `Partial` row should
become a focused implementation issue and a cross-edition test before it is
changed to `Implemented`.
