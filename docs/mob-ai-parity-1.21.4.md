# Minecraft 1.21.4 mob AI parity

This document is the acceptance gate for GoCraft mob behaviour parity.

Reference behaviour is Minecraft Java 1.21.4. Paper 1.21.4 patches are reviewed
where they intentionally fix or expose vanilla mob behaviour. Behaviour is
ported semantically into Go; Mojang/Paper Java source is not copied into this
repository.

A mob is **Complete** only when all applicable areas are implemented and tested:

- goal/Brain priority and state transitions;
- correct navigation model and pathfinding controls;
- target selection, anger, avoidance and ally/owner rules;
- melee/ranged/special attacks, cooldowns and difficulty effects;
- player/entity/block/item interactions;
- environmental behaviour, conversions and immunities;
- breeding/taming/age/lifecycle where applicable;
- Java and Bedrock-visible metadata/state;
- end-to-end movement/attack tests, not only helper-state assertions.

The source-of-truth executable registry is `server/mob_parity_registry.go` and is
pinned to the complete 83-mob Java 1.21.4 set by
`server/mob_parity_registry_test.go`. Villager has its own dedicated Brain port
and remains in this list so the global coverage gate cannot silently omit it.

## Review regression coverage

Navigation tests cover stationary shulkers (including the hostile pursuit
fallback), Creative-player temptation with wall occlusion, and piglin/spider
retaliation through the hostile controller. Sunlight burning deliberately excludes
wither skeletons and zombie horses; the undead tag is not a sunlight-burn tag.
These checks do not complete the remaining per-mob acceptance criteria below.

## Current audit

| Mob | Controller target | Navigation | Status |
| --- | --- | --- | --- |
| Allay | allay Brain | flying | Incomplete |
| Armadillo | armadillo Brain | ground | Incomplete |
| Axolotl | axolotl Brain | amphibious | Incomplete |
| Bat | bat goals/hanging | flying | Incomplete |
| Bee | bee goals/hive/flower/anger | flying | Incomplete |
| Blaze | blaze attack goals | flying | Incomplete |
| Bogged | skeleton ranged goals + bogged effects | ground | Incomplete |
| Breeze | breeze Brain | jumping | Incomplete |
| Camel | camel Brain/sit/dash | ground | Incomplete |
| Cat | cat goals/taming/sit/follow | ground | Incomplete |
| Cave Spider | spider goals + poison | climbing | Incomplete |
| Chicken | animal goals/egg lifecycle | ground | Partial |
| Cod | schooling fish goals | aquatic | Incomplete |
| Cow | animal goals | ground | Partial |
| Creaking | creaking Brain/heart/gaze | ground | Incomplete |
| Creeper | swell/avoid/ignite goals | ground | Partial |
| Dolphin | dolphin goals/treasure/play | aquatic | Incomplete |
| Donkey | horse goals | ground | Partial |
| Drowned | drowned amphibious goals | amphibious | Incomplete |
| Elder Guardian | guardian goals + elder pulse | aquatic | Incomplete |
| Ender Dragon | dragon phase manager | boss flight | Missing controller |
| Enderman | enderman goals/anger/teleport/carry | teleporting | Partial |
| Endermite | endermite goals/lifetime | ground | Incomplete |
| Evoker | evoker spell goals | ground | Incomplete |
| Fox | fox goals/sleep/pounce/trust | ground | Incomplete |
| Frog | frog Brain/tongue/frogspawn | amphibious | Incomplete |
| Ghast | random flight/fireball goals | flying | Incomplete |
| Giant | vanilla giant goals | ground | Missing controller |
| Glow Squid | squid goals | aquatic | Incomplete |
| Goat | goat Brain/ram/horn | ground | Incomplete |
| Guardian | guardian beam/thorns goals | aquatic | Incomplete |
| Hoglin | hoglin Brain | ground | Incomplete |
| Horse | horse goals/taming | ground | Partial |
| Husk | zombie goals + husk effects | ground | Incomplete |
| Illusioner | illusioner spells/bow/copies | ground | Incomplete |
| Iron Golem | golem goals/village/reputation | ground | Partial |
| Llama | llama goals/caravan/spit | ground | Incomplete |
| Magma Cube | magma-cube hop/size goals | jumping | Incomplete |
| Mooshroom | animal goals + mooshroom interactions | ground | Partial |
| Mule | horse goals | ground | Partial |
| Ocelot | ocelot avoid/trust goals | ground | Incomplete |
| Panda | panda goals/genes/actions | ground | Incomplete |
| Parrot | parrot fly/follow/imitate/dance | flying | Incomplete |
| Phantom | circle/swoop phases | flying | Incomplete |
| Pig | animal goals/saddle/lightning | ground | Partial |
| Piglin | piglin Brain/admire/barter/anger | ground | Incomplete |
| Piglin Brute | brute Brain | ground | Incomplete |
| Pillager | crossbow/raid goals | ground | Incomplete |
| Polar Bear | bear goals/cub defence | ground | Incomplete |
| Pufferfish | fish goals/inflate/sting | aquatic | Partial |
| Rabbit | rabbit jump/raid-crop goals | ground | Incomplete |
| Ravager | ravager attack/stun/roar | ground | Incomplete |
| Salmon | schooling fish goals | aquatic | Incomplete |
| Sheep | sheep goals/eat grass | ground | Partial |
| Shulker | attach/peek/teleport/bullet goals | anchored | Incomplete |
| Silverfish | infest/wake-friends goals | ground | Incomplete |
| Skeleton | bow/melee goals | ground | Partial |
| Skeleton Horse | horse/trap goals | ground | Incomplete |
| Slime | slime hop/size goals | jumping | Incomplete |
| Sniffer | sniffer Brain/sniff/dig | ground | Incomplete |
| Snow Golem | ranged/snow/melt goals | ground | Partial |
| Spider | spider climb/daylight goals | climbing | Incomplete |
| Squid | squid goals | aquatic | Incomplete |
| Stray | skeleton goals + stray arrows | ground | Partial |
| Strider | lava/tempt/ride goals | lava | Incomplete |
| Tadpole | tadpole swim/grow | aquatic | Incomplete |
| Trader Llama | llama/trader defence goals | ground | Incomplete |
| Tropical Fish | schooling fish goals | aquatic | Incomplete |
| Turtle | turtle home/egg/travel goals | amphibious | Incomplete |
| Vex | vex owner/charge/lifetime goals | flying | Incomplete |
| Villager | dedicated 1.21.4 villager Brain | ground | Separate parity pass (#76) |
| Vindicator | axe/Johnny/raid goals | ground | Incomplete |
| Wandering Trader | wander/trade/invisibility/despawn | ground | Incomplete |
| Warden | warden Brain/anger/sonic boom | ground | Incomplete |
| Witch | potion selection/drink/raid heal | ground | Incomplete |
| Wither | wither boss controller | boss flight | Incomplete |
| Wither Skeleton | skeleton melee/wither goals | ground | Incomplete |
| Wolf | tame/follow/owner-combat/anger | ground | Partial |
| Zoglin | zoglin Brain | ground | Incomplete |
| Zombie | zombie goals/doors/reinforcement | ground | Partial |
| Zombie Horse | horse goals | ground | Incomplete |
| Zombie Villager | zombie goals/conversion/infection | ground | Partial |
| Zombified Piglin | universal anger goals | ground | Incomplete |

## Merge rule

Do not change this document to claim complete parity merely because a mob can
spawn, wander or damage a player. The final parity PR is merge-ready only when
every row has its applicable behaviour covered and the complete test suite is
green. Large behaviour families should be committed in reviewable batches, but
all batches live behind this single exhaustive coverage gate.
