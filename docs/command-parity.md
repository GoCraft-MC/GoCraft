# Command parity inventory

This inventory compares GoCraft with the dedicated Java 1.21.4 command classes in Mojang's official
[server mappings](https://piston-data.mojang.com/v1/objects/0b1e60cc509cfb0172573ae56b436c29febbc187/server.txt)
and with Paper's current [command reference](https://docs.papermc.io/paper/reference/commands/).

## Implemented in GoCraft

`effect`, `fly`, `flyspeed`, `gamemode` (`gm`), `give`, `gocraft`, `god`, `heal`, `help`, `kick`,
`kill`, `list`, `locate`, `mspt`, `op`, `potioneffect`, `seed`, `setspawn`, `spawn`, `spawnboat`,
`summon`, `time`, `timings`, `tp`, `tphere`, `tps`, `ungod`, `version` (`ver`), `walkspeed`,
`whitelist`, `world`, and `xyz`.

Several names overlap vanilla but currently implement a smaller argument/behavior subset: `effect`,
`gamemode`, `give`, `kill`, `locate`, `summon`, `time`, and `tp`. GoCraft's `setspawn` changes the
world spawn; it is not yet the vanilla `setworldspawn` command grammar. `spawn`, `fly`, `god`,
`heal`, `tphere`, `world`, and the speed commands are GoCraft conveniences rather than vanilla.

## Missing vanilla 1.21.4 commands

- Administration: `ban`, `ban-ip`, `banlist`, `deop`, `defaultgamemode`, `difficulty`, `pardon`,
  `pardon-ip`, `save-all`, `save-off`, `save-on`, `setidletimeout`, and `stop`.
- World and building: `clone`, `fill`, `fillbiome`, `forceload`, `place`, `setblock`, `spawnpoint`,
  `setworldspawn`, `spreadplayers`, `weather`, and `worldborder`.
- Player and item: `advancement`, `attribute`, `clear`, `damage`, `enchant`, `experience` (`xp`),
  `item`, `loot`, `recipe`, `ride`, `rotate`, and `spectate`.
- Messaging and presentation: `me`, `msg` (`tell`, `w`), `particle`, `playsound`, `say`, `stopsound`,
  `tellraw`, `title`, and `tm` (`teammsg`).
- Automation and data: `bossbar`, `data`, `datapack`, `execute`, `function`, `gamerule`, `random`,
  `reload`, `return`, `schedule`, `scoreboard`, `tag`, `team`, `tick`, `trigger`, and `transfer`.
- Dedicated-server/developer commands: `jfr`, `perf`, `publish`, and Mojang's debug-only commands.

## Missing Paper/Bukkit commands

GoCraft already provides the commonly requested `help`, `version`, `tps`, `mspt`, and `timings`
commands. It does not implement Bukkit/Paper's `plugins`, `reload`, `restart`, or `spark`, and it does
not implement the `/paper` tree (`chunkinfo`, `debug`, `dumpitem`, `dumplisteners`, `dumpplugins`,
`entity`, `fixlight`, `heap`, `holderinfo`, `mobcaps`, `playermobcaps`, `reload`, `syncloadinfo`, and
`version`). Several of those inspect Bukkit plugins or Paper internals that GoCraft does not have.

## Recommended implementation order

1. Server safety: `stop`, `save-all`, `deop`, bans/pardons, `difficulty`, `gamerule`, and `weather`.
2. Daily administration: `clear`, `setblock`, `fill`, `enchant`, `xp`, `spawnpoint`,
   `setworldspawn`, `say`, and private messaging.
3. World control: `clone`, `forceload`, `worldborder`, particles, sounds, titles, and teams.
4. Data-driven command engine: `execute`, `data`, `scoreboard`, functions, schedules, loot, and
   recipes. These depend on selector, NBT-path, and data-pack infrastructure and should share it.
5. Native diagnostics for useful Paper concepts such as heap, mob caps, chunks, and profiling;
   plugin-introspection commands should wait for GoCraft's plugin API.
