# Villager Brain parity — Minecraft Java 1.21.4

This document is the merge gate for claiming **100% Mojang Villager Brain parity**. GoCraft implements the behaviour in Go; it does not copy Mojang's Java implementation line-for-line.

## Reference surface

The target is the 1.21.4 `Villager` Brain registration plus `VillagerGoalPackages`, including the memories and sensors they depend on. Paper-specific behavioural fixes that intentionally correct vanilla bugs are retained where applicable.

### Memory modules (30)

- [x] HOME
- [x] JOB_SITE
- [x] POTENTIAL_JOB_SITE
- [x] MEETING_POINT
- [x] NEAREST_LIVING_ENTITIES
- [x] NEAREST_VISIBLE_LIVING_ENTITIES
- [x] VISIBLE_VILLAGER_BABIES
- [x] NEAREST_PLAYERS
- [x] NEAREST_VISIBLE_PLAYER
- [x] NEAREST_VISIBLE_ATTACKABLE_PLAYER
- [x] NEAREST_VISIBLE_WANTED_ITEM
- [x] ITEM_PICKUP_COOLDOWN_TICKS
- [x] WALK_TARGET
- [x] LOOK_TARGET
- [x] INTERACTION_TARGET
- [x] BREED_TARGET
- [x] PATH
- [x] DOORS_TO_CLOSE
- [x] NEAREST_BED
- [x] HURT_BY
- [x] HURT_BY_ENTITY
- [x] NEAREST_HOSTILE
- [x] SECONDARY_JOB_SITE
- [x] HIDING_PLACE
- [x] HEARD_BELL_TIME
- [x] CANT_REACH_WALK_TARGET_SINCE
- [x] LAST_SLEPT
- [x] LAST_WOKEN
- [x] LAST_WORKED_AT_POI
- [x] GOLEM_DETECTED_RECENTLY

The checkbox means a canonical GoCraft state mapping exists. It does **not** by itself mean every behaviour consuming that memory is complete.

### Sensors (9)

- [x] NEAREST_LIVING_ENTITIES
- [x] NEAREST_PLAYERS
- [x] NEAREST_ITEMS
- [x] NEAREST_BED
- [x] HURT_BY
- [x] VILLAGER_HOSTILES
- [x] VILLAGER_BABIES
- [x] SECONDARY_POIS
- [x] GOLEM_DETECTED

## Schedule

- [x] Adult schedule: IDLE 10, WORK 2000, MEET 9000, IDLE 11000, REST 12000
- [x] Baby schedule: IDLE 10, PLAY 3000, IDLE 6000, PLAY 10000, REST 12000
- [x] WORK falls back to IDLE without JOB_SITE
- [x] MEET falls back to IDLE without MEETING_POINT
- [x] PANIC overrides schedule
- [x] Bell reaction can override schedule with HIDE

## Activity packages

### CORE

- [x] swimming remains in shared passive movement
- [x] wooden door interaction / closing
- [x] look target sink mapping
- [x] panic trigger mapping
- [x] wake-up / sleeping-position handling
- [x] ReactToBell / HEARD_BELL_TIME
- [ ] active raid status selection (blocked by missing GoCraft raid manager)
- [x] validate HOME/JOB_SITE/POTENTIAL_JOB_SITE POIs
- [x] MoveToTargetSink via shared navigator
- [ ] POI competitor scan tie-breaking equivalent
- [ ] follow active trading player
- [x] wanted-item sensor + walk target, including Paper MC-157464 sleeping guard
- [x] job/home/meeting POI acquisition hooks
- [ ] POTENTIAL_JOB_SITE ticket lifecycle exactly matching vanilla POI tickets
- [x] profession assignment/reset through canonical POI refresh

### WORK

- [x] route to JOB_SITE end-to-end
- [x] WorkAtPoi timestamp/state
- [ ] WorkAtComposter inventory transformation parity
- [ ] exact weighted stroll-around/stroll-to behaviour selection
- [x] SECONDARY_JOB_SITE discovery for farmers
- [x] mature crop harvest/replant behaviour
- [ ] UseBonemeal inventory/particle parity
- [ ] ShowTradesToPlayer
- [ ] GiveGiftToHero

### PLAY

- [x] exact baby schedule activation
- [x] detect visible villager babies
- [x] PlayTagWithOtherKids-style chase spacing
- [x] village-bound random play target
- [ ] JumpOnBed motion parity
- [ ] exact weighted RunOne selection

### REST

- [x] walk to HOME
- [x] validate HOME
- [x] SleepInBed
- [x] bed-head normalisation and canonical sleeping position
- [x] nearest-bed fallback
- [ ] InsideBrownianWalk / GoToClosestVillage weighting parity

### MEET

- [x] route to MEETING_POINT end-to-end
- [x] stroll/social offset around meeting point
- [x] nearby villager interaction/look memory
- [ ] SocializeAtBell exact timing
- [ ] TradeWithVillager inventory exchange
- [ ] ShowTradesToPlayer
- [ ] GiveGiftToHero

### IDLE

- [x] generic village-bound stroll/look/do-nothing path
- [x] nearby villager interaction target
- [x] BREED_TARGET mapping
- [ ] VillagerMakeLove food/inventory consumption and exact bed reservation lifecycle
- [ ] cat interaction weighting
- [ ] JumpOnBed parity
- [ ] TradeWithVillager inventory exchange
- [ ] GiveGiftToHero

### PANIC

- [x] NEAREST_HOSTILE sensor
- [x] HURT panic integration through existing damage path
- [x] move away from hostile
- [x] schedule override

### PRE_RAID / RAID

- [ ] raid-state sensor and ResetRaidStatus (blocked by missing raid manager)
- [ ] PRE_RAID RingBell behaviour
- [ ] PRE_RAID movement weighting
- [ ] RAID sky-seeing spot movement
- [ ] CelebrateVillagersSurvivedRaid
- [ ] active-raid hiding transition

### HIDE

- [x] bell memory activation
- [x] locate HOME/nearest-bed hiding place
- [x] 300-tick HEARD_BELL_TIME timeout
- [x] move to hiding place
- [ ] exact hidden-state close-distance tick counter

## Supporting villager systems required before the 100% claim

These are used directly by Villager Brain behaviours and therefore are merge gates even though some state lives outside `Brain` itself:

- [ ] 8-slot villager inventory with vanilla stack semantics
- [ ] food-point accounting (bread=4, potato/carrot/beetroot=1; breeding threshold 12)
- [ ] villager item pickup into that inventory
- [ ] food sharing / TradeWithVillager
- [ ] full VillagerMakeLove birth flow
- [ ] gossip/reputation container, sharing and decay
- [ ] Hero of the Village gift selection
- [ ] iron-golem agreement/spawn rules tied to LAST_SLEPT and GOLEM_DETECTED_RECENTLY
- [ ] raid manager/status integration
- [ ] POI ticket reservation/release/reachability parity
- [ ] persistence for villager Brain/inventory/gossip state where vanilla persists it

## Tests

- [x] memory/sensor registry size and names
- [x] adult schedule boundaries
- [x] baby schedule boundaries
- [x] bell -> HIDE
- [x] hostile -> PANIC
- [x] WORK target assignment
- [x] WORK end-to-end movement through passive AI + physics
- [x] MEET target assignment
- [x] MEET end-to-end movement through passive AI + physics
- [x] HOME claim/loss cache behaviour
- [x] bed-head normalisation / sleeping position
- [ ] differential/in-game checks against Java 1.21.4 for every activity package

**Do not mark this document complete or describe GoCraft as having 100% Villager Brain parity until every unchecked merge gate above is implemented and tested.**
