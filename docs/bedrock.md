# Bedrock Edition & Cross-Play

GoCraft includes a native Bedrock adapter backed by the same canonical simulation as Java Edition. Both editions share the same world, entities, players, chat, combat, block changes, and time — with no external proxy required.

Current network compatibility target: **Minecraft Bedrock Edition 1.26.45 (protocol 2169)**.

## Enabling Bedrock

```yaml
bedrock:
  enabled: true
  address: 0.0.0.0:19106   # UDP port for RakNet connections
  online_mode: true          # require Xbox Live authentication
```

Open the UDP port on your firewall/Pterodactyl egg. Bedrock clients connect using the standard "Add Server" flow.

## What's shared between editions

| Feature | Shared |
| --- | --- |
| World (blocks, chunks, terrain) | Yes |
| Players — position, movement | Yes |
| Players — visibility and despawn | Yes |
| Chat messages | Yes |
| Commands | Yes |
| Combat, health, death, respawn | Yes |
| Block break and placement | Yes |
| Basic inventory (held item, drops) | Yes |
| Time of day | Yes |
| Mob spawn, tick, despawn | Yes |
| Equipment and armor | Yes |
| Sleeping | Yes |
| Permissions and ranks | Yes |
| Custom items (resource/behavior packs) | Yes — see [Custom Items](custom-items.md) |

## Not a translator

GoCraft is **not** a protocol translator like Geyser. The Bedrock adapter does not consume Java packets and re-encode them. Both adapters read from the same canonical game core independently:

```
                    ┌─────────────────────────────┐
                    │         core/               │
                    │  World · Entity · Player    │  ← no Java, no Bedrock
                    └────────────┬────────────────┘
                                 │ canonical state
               ┌─────────────────┴──────────────────┐
               ▼                                    ▼
   ┌───────────────────────┐          ┌───────────────────────┐
   │     java/ adapter     │          │   bedrock/ adapter    │
   │  native Java packets  │          │  native Bedrock packets│
   │  Java state IDs       │          │  Bedrock runtime IDs  │
   └───────────────────────┘          └───────────────────────┘
         Java client                       Bedrock client
```

This design means GoCraft can evolve both protocol implementations independently without the coupling issues that proxy-based translators face.

## Item registry

Bedrock clients receive the Pumpkin/BDS 1.26.40 complete item registry (1,976 entries including data-driven components) and a 1,875-entry grouped Creative catalogue. Custom items defined in `packs/` are appended to this registry at startup — see [Custom Items](custom-items.md).

## Resource and behavior packs

Bedrock packs can be pushed to clients at login. Supported formats:

| File | Description |
| --- | --- |
| `.mcpack` | Single Bedrock resource or behavior pack |
| `.zip` | Same as `.mcpack` but with a `.zip` extension |
| `.mcaddon` | Archive containing multiple sub-packs (resource + behavior) |

Configure in `server.yml`:

```yaml
resource_pack:
  bedrock:
    enabled: true
    paths:
      - packs/MyTextures.mcpack
      - packs/Ore_Boost_V1.2.mcaddon
    forced: false   # kick players who decline
```

`.mcaddon` files are automatically extracted — every sub-directory containing a `manifest.json` is loaded as a separate pack and sent to connecting clients.

## Known limitations (beta)

- Advanced container transactions (e.g. Bedrock-specific crafting screens) are incomplete
- Some villager and mob AI behaviours differ between editions
- Not every vanilla UI element, particle, and sound has full cross-edition parity
- Biome palette exact-match is still in progress
- Behavior pack server-side logic (custom loot tables, AI overrides) does not run — GoCraft is a Go server, not a vanilla Bedrock engine

See [Architecture](architecture.md) for how the two adapters interact at the code level.
