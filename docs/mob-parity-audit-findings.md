
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
vanilla. Divergences are concentrated in **per-mob special goals** (ranged
attacks, targeting sets, sun/avoid behaviour, block manipulation, egg/trail
lifecycle), which are simplified or absent.

| Mob | Attributes | Core combat | Notable gap | Worst |
| --- | --- | --- | --- | --- |
| Zombie / Zombie Villager | ✅ 0.23 / atk 3 / range 35 / armor 2 | ✅ | ~~no door-break~~ **fixed (hard)**; no reinforcements, targets players only | 🟡 |
| Skeleton / Stray | ✅ 0.25 | bow ✅ | ~~no strafe-kite / flee-sun / avoid-wolf~~ **fixed**; arrow dmg flat 3 | 🟡 |
| Creeper | ✅ 0.25 | ✅ swell 3b / fuse 30t / r3 | dead `CreeperFuse` struct; no charged (r6) | 🟡 |
| Enderman | ✅ 40hp / 0.3 / atk 7 / range 64 | ✅ + water/teleport | no block take/place, freeze-on-look partial | 🟠 |
| Cow / Sheep / Pig / Mooshroom | ✅ speeds & hp | breed/panic/tempt ✅ | panic speed not per-mob; sheep eat-grass? | 🟡 |
| Chicken | ✅ 4hp / 0.25 | ✅ | ~~no egg laying~~ **fixed**; slow-fall moot (mobs take no fall damage) | 🟢 |
| Horse / Donkey / Mule | ✅ 0.225 / jump 0.7 / 53hp | — | taming/rearing partial | 🟠 |
| Wolf | ✅ 0.3 / atk 4 | melee ✅ (range 1.8, cd 20) | no leap-at-target, beg, avoid-llama; wild prey targeting? | 🟡 |
| Iron Golem | ✅ 100hp / atk 7–21 + toss | ✅ | ~~targets Zombies only~~ **fixed:** all Enemy except Creeper | 🟢 |
| Snow Golem | ✅ melts in water/rain | ✅ snowballs | ~~no ranged attack~~ **fixed**; no snow trail | 🟡 |
| Pufferfish | speed 0.7 | — | inflate/sting to verify | 🟠 |

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
- 🟠 No zombie **reinforcement** summon on damage.
- 🟠 Targeting: no evidence zombies actively hunt villagers / iron golems / baby
  turtles — appears to target players only.

### Skeleton family (Skeleton, Stray)
Vanilla: `speed 0.25`; `RangedBowAttackGoal` (strafing kite), `MeleeAttackGoal`
fallback, `RestrictSunGoal`+`FleeSunGoal` (seek shade by day),
`AvoidEntityGoal(Wolf, 6)`.
- 🟢 Bow implemented (`shootMobArrow`, `bowDrawTicks`); daylight burn present.
- 🟢 **Fixed:** strafing (`tickSkeletonStrafe`) circles the target, re-rolling
  direction every 20 ticks and backing off when too close; `tickSkeletonAvoidance`
  adds `FleeSunGoal` (seek shade while burning) and `AvoidEntityGoal(Wolf, 6)`,
  both prioritised above the bow attack like vanilla.
- 🟡 Arrow damage flat **3** (`mob_environment.go:266`); vanilla ≈2 base scaled by
  difficulty/power.

### Creeper — 🟢 faithful
Swell ≤3 blocks + LOS, fuse 30 ticks, explosion radius 3 — all match
`SwellGoal`/`Creeper`.
- 🟡 De-swell on retreat is `-2/tick` vs vanilla `-1`.
- 🟡 `server/mob_fuse.go` `CreeperFuse` is **dead code** and, unlike the live
  tick-based path, is wall-clock (`time.Since`, 1500ms) — would drift under lag.
- 🟠 Charged creeper (lightning → radius 6) not represented.

### Enderman
Vanilla: `40hp, 0.3, atk 7, follow 64`; freeze-when-looked-at, take/leave block,
anger-on-stare, water/rain damage + teleport.
- 🟢 Attributes match; water damage + teleport implemented (`tickEndermanWater`);
  stare-based aggro present (`isPlayerStaringAtEnderman`).
- 🟠 No **block pick up / place** (`EndermanTakeBlockGoal`/`LeaveBlockGoal`).
- 🟡 Freeze-while-stared and teleport-on-damage only partially modelled.

### Passive animals (Cow, Sheep, Pig, Mooshroom, Chicken)
Vanilla: shared `Animal` goals — `PanicGoal`, `BreedGoal(1.0)`, `TemptGoal`,
`FollowParentGoal(1.1)`; per-mob panic speed (Sheep 1.25, Pig 1.25, Chicken 1.4).
- 🟢 Speeds/health match; breeding (`animal_lifecycle.go`), panic (`panicTick=60`)
  and tempt are implemented.
- 🟡 Panic speed appears flat, not per-mob; verify FollowParent and Sheep
  `EatBlockGoal` (grass → regrow wool).
- 🟢 **Fixed:** chickens lay an egg every 6000-12000 ticks (`EggLayTicks` in
  `tickAnimalLifecycle`). Slow-fall is moot — GoCraft applies no fall damage to
  mobs.

### Horse family (Horse, Donkey, Mule)
Vanilla: `speed 0.225, jump 0.7, hp 53 base`; `RunAroundLikeCrazyGoal` (taming
buck/rear), `TemptGoal(1.25)`, `RandomStandGoal`.
- 🟢 Attributes match.
- 🟠 Taming/rearing, jump strength application, and donkey/mule chest inventory
  are partial — needs a dedicated pass.

### Wolf
Vanilla: `speed 0.3, atk 4, hp 8 wild`; `LeapAtTargetGoal(0.4)`, `MeleeAttackGoal`,
`FollowOwnerGoal(start 10, stop 2)`, `BegGoal`, `SitWhenOrdered`,
`WolfAvoidEntity(Llama)`, wild targets Animals/Skeletons, owner-hurt targeting.
- 🟢 Melee (range 1.8, cd 20) and owner-combat (`tickTamedWolfCombat`) present.
- 🟡 No `LeapAtTargetGoal` pounce, no `BegGoal`, no llama avoidance; confirm wild
  wolves hunt prey (sheep/rabbit) and skeletons.

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
- 🟠 No snow-trail placement.

### Pufferfish
Vanilla: `PufferfishPuffGoal` (inflate when a player/mob is near, contact damage
+ poison, deflate after).
- 🟠 Inflation state, contact damage and poison need verification — likely
  partial.

## Recommended fix priority

1. 🔴 **Iron Golem targeting** — one-line-ish widen from Zombies-only to all
   `Enemy` except Creeper. High player impact, cheap.
2. 🔴 **Snow Golem snowball attack** — add a ranged goal mirroring
   `RangedAttackGoal(1.25, 20, 10)`.
3. 🟠 **Chicken eggs** + **Zombie door-breaking/reinforcements** — visible,
   commonly-noticed gaps.
4. 🟠 **Skeleton strafe + flee-sun**, **Enderman block take/place** — behavioural
   polish.
5. 🟡 Cleanups: delete dead `CreeperFuse`; per-mob panic speeds; skeleton arrow
   damage scaling.

## Method (reproducible)

```
# decompile a vanilla class from the reference jar
unzip -o server-26.2.jar 'net/minecraft/world/entity/<path>/<Class>.class' -d classes
java -cp "<IntelliJ>/plugins/java-decompiler/lib/java-decompiler.jar" \
  org.jetbrains.java.decompiler.main.decompiler.ConsoleDecompiler \
  -e=server-26.2.jar classes out
# then compare createAttributes()/registerGoals() vs GoCraft server/mob_*.go
```
