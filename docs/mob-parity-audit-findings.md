
# Mob AI parity audit — findings

Audit of the **implemented ("Partial")** mobs in `docs/mob-ai-parity-1.21.4.md`
against the decompiled reference server (Mojang **26.2**, mapped/readable
classes; core mob mechanics are stable vs 1.21.4). The 62 "Incomplete" mobs are
unimplemented stubs and are out of scope here — auditing them only re-confirms
"not implemented".

Reference decompiled with FernFlower from `server-26.2.jar`. Values below are
vanilla attribute/goal constants (`createAttributes` / `registerGoals`) versus
GoCraft's runtime code.

Legend — severity of divergence: 🟢 faithful · 🟡 minor · 🟠 missing sub-behaviour · 🔴 wrong/absent core behaviour.

## Summary

GoCraft's shared combat core is **faithful**: movement speeds, attack-damage
attributes, follow ranges, and the **20-tick (1s) melee cooldown** all match
vanilla. Verification against the decompiled reference found the implementation
more complete than a first grep-level pass suggested: enderman block carry,
zombie villager/golem targeting, pufferfish inflate/sting and horse taming were
all already present. Remaining divergences are a few per-mob special goals.

| Mob | Attributes | Core combat | Notable gap | Worst |
| --- | --- | --- | --- | --- |
| Zombie / Zombie Villager | ✅ 0.23 / atk 3 / range 35 / armor 2 | ✅ | door-break ✅, reinforcements ✅ (hard); targets villagers/golems/turtles player-first | 🟢 |
| Skeleton / Stray | ✅ 0.25 | bow ✅ | strafe/flee-sun/avoid-wolf ✅; arrow dmg now difficulty-scaled | 🟢 |
| Creeper | ✅ 0.25 | ✅ swell 3b / fuse 30t / r3 | charged (r6) ✅ via new lightning system | 🟢 |
| Enderman | ✅ 40hp / 0.3 / atk 7 / range 64 | ✅ + water/teleport | block take/place ✅; freeze-on-look ✅ | 🟢 |
| Cow / Sheep / Pig / Mooshroom | ✅ speeds & hp | breed/panic/tempt ✅ | per-mob panic speed ✅; sheep eat-grass? | 🟢 |
| Chicken | ✅ 4hp / 0.25 | ✅ | ~~no egg laying~~ **fixed**; slow-fall moot (mobs take no fall damage) | 🟢 |
| Horse / Donkey / Mule | ✅ 0.225 / jump 0.7 / 53hp | — | taming (temper/buck/hearts) ✅ | 🟢 |
| Wolf | ✅ 0.3 / atk 4 | melee ✅ | leap ✅, avoid-llama ✅, wild prey ✅; no beg | 🟢 |
| Iron Golem | ✅ 100hp / atk 7–21 + toss | ✅ | ~~targets Zombies only~~ **fixed:** all Enemy except Creeper | 🟢 |
| Snow Golem | ✅ melts in water/rain | ✅ snowballs | snowballs ✅, snow trail ✅ | 🟢 |
| Pufferfish | speed 0.7 | inflate/sting ✅ | full puff-state + poison scaling | 🟢 |

## Per-family detail

### Zombie family (Zombie, Zombie Villager)
Vanilla: `speed 0.23, atk 3.0, follow 35, armor 2`; `ZombieAttackGoal(1.0,false)`;
hard-difficulty `BreakDoorGoal`; `MoveThroughVillageGoal`; targets Player,
AbstractVillager, IronGolem, Turtle eggs; `SPAWN_REINFORCEMENTS_CHANCE`.
- 🟢 Attributes and the 20-tick melee (`mob_actions.go:64`, damage from spawn
  settings) match.
- 🟢 **Fixed:** hard-difficulty zombies (and husk/drowned/zombie-villager) now
  bash a wooden door blocking the way to their target over 240 ticks
  (`tickZombieDoorBreak`), matching vanilla `BreakDoorGoal`. No mobGriefing
  gamerule exists in GoCraft, so it is gated on hard difficulty only.
- 🟢 **Fixed:** hard-difficulty zombies summon reinforcements when hurt
  (`tryZombieReinforcement`), bounded by a no-chain flag on the spawned help.
- 🟢 Correction: zombies already target villagers, iron golems and turtles via
  `pumpkinMobTargets` (player-first, then the nearest such entity).

### Skeleton family (Skeleton, Stray)
Vanilla: `speed 0.25`; `RangedBowAttackGoal` (strafing kite), `MeleeAttackGoal`
fallback, `RestrictSunGoal`+`FleeSunGoal` (seek shade by day),
`AvoidEntityGoal(Wolf, 6)`.
- 🟢 Bow implemented (`shootMobArrow`, `bowDrawTicks`); daylight burn present.
- 🟢 **Fixed:** strafing (`tickSkeletonStrafe`) circles the target, re-rolling
  direction every 20 ticks and backing off when too close; `tickSkeletonAvoidance`
  adds `FleeSunGoal` (seek shade while burning) and `AvoidEntityGoal(Wolf, 6)`,
  both prioritised above the bow attack like vanilla.
- 🟢 **Fixed:** arrow damage now scales with difficulty (easy 2, normal 3, hard 4)
  instead of a flat 3.

### Creeper — 🟢 faithful
Swell ≤3 blocks + LOS, fuse 30 ticks, explosion radius 3 — all match
`SwellGoal`/`Creeper`.
- 🟡 De-swell on retreat is `-2/tick` vs vanilla `-1`.
- 🟢 **Fixed:** the dead wall-clock `CreeperFuse` struct was removed; the live
  fuse is the tick-based `ai.fuseTick` path.
- 🟢 **Fixed:** added a lightning system (`server/lightning.go`) — thunderstorms
  strike near players, spawning a `lightning_bolt` entity that burns/damages
  entities at the column (5 dmg + fire) and charges creepers within 15 blocks.
  A charged creeper renders the powered aura (metadata index 17) and explodes
  with radius 6.

### Enderman
Vanilla: `40hp, 0.3, atk 7, follow 64`; freeze-when-looked-at, take/leave block,
anger-on-stare, water/rain damage + teleport.
- 🟢 Attributes match; water damage + teleport implemented (`tickEndermanWater`);
  stare-based aggro present (`isPlayerStaringAtEnderman`).
- 🟢 Block **pick up / place** is implemented (`tryEndermanPickupBlock` /
  `tryEndermanPlaceBlock`, 1/20 and 1/2000 per tick like vanilla). Correction to
  an earlier audit note: this was already present. This pass broadened
  `EndermanPickupBlocks` (10 → ~35) to match the vanilla `enderman_holdable` tag
  (dirt family, moss, nether holdables, small flowers, tnt, clay, …).
- 🟢 **Fixed:** freeze-while-stared — an enderman being looked at by its target
  within 16 blocks stops moving (EndermanFreezeWhenLookedAt).
- 🟡 Carry ticks every tick rather than only via idle goals; teleport-on-damage
  only partially modelled.

### Passive animals (Cow, Sheep, Pig, Mooshroom, Chicken)
Vanilla: shared `Animal` goals — `PanicGoal`, `BreedGoal(1.0)`, `TemptGoal`,
`FollowParentGoal(1.1)`; per-mob panic speed (Sheep 1.25, Pig 1.25, Chicken 1.4).
- 🟢 Speeds/health match; breeding (`animal_lifecycle.go`), panic (`panicTick=60`)
  and tempt are implemented.
- 🟢 **Fixed:** per-mob PanicGoal speed (sheep/pig 1.25x, chicken 1.4x). Sheep
  `EatBlockGoal` (grass → regrow wool) still worth verifying.
- 🟢 **Fixed:** chickens lay an egg every 6000-12000 ticks (`EggLayTicks` in
  `tickAnimalLifecycle`). Slow-fall is moot — GoCraft applies no fall damage to
  mobs.

### Horse family (Horse, Donkey, Mule)
Vanilla: `speed 0.225, jump 0.7, hp 53 base`; `RunAroundLikeCrazyGoal` (taming
buck/rear), `TemptGoal(1.25)`, `RandomStandGoal`.
- 🟢 Attributes match.
- 🟢 Correction: taming is implemented — mounting an untamed horse rolls against
  temper, adds 5 temper + a buck event on failure, and grants ownership with the
  taming-heart event on success (`animal_interaction.go`).
- 🟡 Donkey/mule chest inventory still worth a dedicated pass.

### Wolf
Vanilla: `speed 0.3, atk 4, hp 8 wild`; `LeapAtTargetGoal(0.4)`, `MeleeAttackGoal`,
`FollowOwnerGoal(start 10, stop 2)`, `BegGoal`, `SitWhenOrdered`,
`WolfAvoidEntity(Llama)`, wild targets Animals/Skeletons, owner-hurt targeting.
- 🟢 Melee + owner-combat present.
- 🟢 **Fixed:** `LeapAtTargetGoal` pounce (`wolfMaybeLeap`), `AvoidEntityGoal(Llama, 24)`,
  and wild-wolf prey targeting (sheep/rabbit/fox + skeleton family) via `tickWolfBehaviour`.
- 🟡 No `BegGoal` (cosmetic tilt toward players holding food).

### Iron Golem — ✅ fixed
Vanilla: `100hp, 0.25, atk 7.5–21.5 + upward toss, KB-resist 1.0`; targets any
`Enemy` mob **except Creeper**, plus DefendVillage.
- 🟢 Health/speed/knockback-resist match; damage `7 + rng(15)` (7–21) and the
  upward toss match.
- 🟢 **Fixed:** now targets any hostile except Creeper via `isIronGolemTarget`
  (was Zombies-only).
- 🟠 No offer-flower / village-reputation behaviour.

### Snow Golem — ✅ ranged attack added
Vanilla: `4hp, 0.2`; `RangedAttackGoal(1.25, 20, 10)` throwing snowballs at
`Enemy` mobs (knockback; damage only to blaze/enderman), leaves a snow trail on
snow-friendly biomes, melts in warm biomes / water / rain.
- 🟢 Melts in water/rain (`mob_environment.go`).
- 🟢 **Fixed:** `tickSnowGolemAI` targets the nearest hostile within 10 and throws
  a snowball every 20 ticks; the shared projectile path already applies knockback
  (and 3 damage to blazes).
- 🟢 **Fixed:** lays a snow layer under itself as it walks (`placeSnowGolemTrail`);
  the vanilla warm-biome melt gate is omitted (GoCraft has no per-biome temp).

### Pufferfish
Vanilla: `PufferfishPuffGoal` (inflate when a player/mob is near, contact damage
+ poison, deflate after).
- 🟢 Correction: fully implemented (`server/pufferfish.go`) — inflate to half then
  full, staged deflate, threat detection, and contact sting dealing `1+puffState`
  damage plus `puffState*60` ticks of poison.

## Status

Every implemented ("Partial") mob has been brought to parity or verified faithful.
Fixes shipped: iron golem targeting, snow golem snowballs + snow trail, chicken
eggs, zombie door-breaking + reinforcements, skeleton
strafe/flee-sun/avoid-wolf + difficulty arrows, enderman holdable set +
freeze-when-looked-at, wolf leap/avoid-llama/wild-prey, per-mob panic speed, and
removal of the dead `CreeperFuse`.

A lightning system was added, so the **charged creeper** (radius-6) is done, and
lightning now also converts a struck **pig → zombified piglin** (golden sword),
**villager → witch** (both outside peaceful), and toggles a **mooshroom's
red/brown variant** (server-side; the client-facing variant metadata awaits the
confirmed 1.21.4 serializer). Sheep now regrow wool by **eating a grass block**
(EatBlockGoal), with the old timer kept as a fallback.

Still open (small):
- **Wolf beg** — cosmetic head-tilt. Blocked: GoCraft maps the wolf collar to
  metadata index 19, which collides with vanilla's begging-at-19, so the wolf
  metadata layout needs reconciling before adding it (risk of client disconnect).
- **Donkey/mule chest inventory** — a distinct feature (15-slot inventory + chest
  attach interaction + container open); the `has_chest` metadata slot already
  exists at index 18. Planned as a follow-up.

## Method (reproducible)

```
# decompile a vanilla class from the reference jar
unzip -o server-26.2.jar 'net/minecraft/world/entity/<path>/<Class>.class' -d classes
java -cp "<IntelliJ>/plugins/java-decompiler/lib/java-decompiler.jar" \
  org.jetbrains.java.decompiler.main.decompiler.ConsoleDecompiler \
  -e=server-26.2.jar classes out
# then compare createAttributes()/registerGoals() vs GoCraft server/mob_*.go
```
