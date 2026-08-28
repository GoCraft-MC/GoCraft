# Commands

## Built-in commands

| Command | Description |
| --- | --- |
| `/help` | List all available commands |
| `/list` | Show online player names and count |
| `/me <action>` | Broadcast an action message |
| `/msg <player> <message>` | Send a private message |
| `/tell <player> <message>` | Alias for `/msg` |
| `/w <player> <message>` | Alias for `/msg` |
| `/say <message>` | Broadcast a server message |
| `/ver` / `/version` | Report the GoCraft version |
| `/xyz` | Show precise position, block, and chunk coordinates |
| `/tp <x> <y> <z>` | Teleport to coordinates |
| `/tp <player>` | Teleport to a player |
| `/spawn` | Teleport to world spawn |
| `/tphere <player>` | Teleport a player to you |
| `/gamemode <mode>` | Change game mode (`survival`, `creative`, `adventure`, `spectator`) |
| `/gm <mode>` | Alias for `/gamemode` |
| `/fly` | Toggle flight |
| `/walkspeed <value\|reset>` | Set walk speed |
| `/flyspeed <value\|reset>` | Set fly speed |
| `/give <player> <item> [count]` | Give an item to a player |
| `/get <item> [count]` | Give an item to yourself |
| `/heal [player]` | Restore health |
| `/kill [player]` | Kill a player |
| `/god [player]` | Toggle god mode |
| `/ungod [player]` | Remove god mode |
| `/potioneffect <player> <effect> <seconds>` | Apply a potion effect |
| `/summon <mob> [profession]` | Spawn a mob beside you |
| `/locate <village\|biome>` | Find the nearest village or biome |
| `/kick <player> [reason]` | Disconnect a player |
| `/op <player>` | Grant operator status |
| `/deop <player>` | Revoke operator status |
| `/ban <player> [reason]` | Ban a player |
| `/ban-ip <address> [reason]` | Ban an IP address |
| `/pardon <player>` | Unban a player |
| `/pardon-ip <address>` | Unban an IP address |
| `/banlist` | List banned players and IPs |
| `/whitelist <add\|remove\|list\|on\|off>` | Manage the whitelist |
| `/difficulty <level>` | Set difficulty |
| `/time <set\|add> <value>` | Change world time |
| `/weather <clear\|rain\|thunder>` | Set weather |
| `/seed` | Show the world seed |
| `/setspawn` | Set world spawn to your position |
| `/setworldspawn` | Alias for `/setspawn` |
| `/save-all` | Force an immediate world save |
| `/save-on` / `/save-off` | Enable or disable autosaves |
| `/stop` | Shut down the server |
| `/tps` | Show current ticks per second |
| `/mspt` | Show milliseconds per tick |
| `/reload` | Reload configuration |
| `/clear [player]` | Clear a player's inventory |
| `/xp <amount> [player]` | Grant experience |
| `/setblock <x> <y> <z> <block>` | Place a block |
| `/fill <x1> <y1> <z1> <x2> <y2> <z2> <block>` | Fill a region |
| `/clone <from> <to> <dest>` | Clone a region |
| `/tag <player> <add\|remove\|list> [tag]` | Manage player tags |
| `/spawnpoint [player]` | Set a player's spawn point |
| `/effect <player> <effect> [duration] [amplifier]` | Apply or clear effects |
| `/damage <player> <amount>` | Deal damage to a player |
| `/random <min> <max>` | Generate a random number |
| `/rotate <player> <yaw> <pitch>` | Rotate a player |
| `/world <overworld\|nether\|end>` | Switch a player's dimension |

## GoCraft admin commands

| Command | Description |
| --- | --- |
| `/gocraft peditor` | Open the permission editor dashboard (prints a one-time URL) |
| `/gocraft applyedits <key>` | Apply edits uploaded to bytebin from the dashboard |
| `/gocraft user <player> group set <group>` | Assign a player to a group |
| `/gocraft user <player> group remove <group>` | Remove a player from a group |
| `/gocraft user <player> group list` | List a player's groups |
| `/gocraft group <name> create` | Create a new group |
| `/gocraft group <name> delete` | Delete a group |
| `/gocraft group <name> setprefix <prefix>` | Set a group's chat prefix (MiniMessage) |
| `/gocraft group <name> setweight <n>` | Set sort weight (higher = higher rank) |
| `/gocraft group <name> addparent <parent>` | Add a group parent |
| `/gocraft group <name> removeparent <parent>` | Remove a group parent |

See [Permissions](permissions.md) for the full permission system documentation.

## Permission nodes

Every command literal has its own permission node in the format `gocraft.command.<name>`. Commands default to either **Public** (any player) or **Operator** (requires op or an explicit grant).

| Command | Permission node | Default |
| --- | --- | --- |
| `/ban` | `gocraft.command.ban` | Operator |
| `/ban-ip` | `gocraft.command.ban-ip` | Operator |
| `/banlist` | `gocraft.command.banlist` | Operator |
| `/clear` | `gocraft.command.clear` | Operator |
| `/clone` | `gocraft.command.clone` | Operator |
| `/damage` | `gocraft.command.damage` | Operator |
| `/defaultgamemode` | `gocraft.command.defaultgamemode` | Operator |
| `/deop` | `gocraft.command.deop` | Operator |
| `/difficulty` | `gocraft.command.difficulty` | Operator |
| `/effect` | `gocraft.command.effect` | Operator |
| `/experience` | `gocraft.command.experience` | Operator |
| `/fill` | `gocraft.command.fill` | Operator |
| `/fly` | `gocraft.command.fly` | Operator |
| `/flyspeed` | `gocraft.command.flyspeed` | Operator |
| `/gamemode` | `gocraft.command.gamemode` | Operator |
| `/get` | `gocraft.command.get` | Operator |
| `/give` | `gocraft.command.give` | Operator |
| `/gm` | `gocraft.command.gm` | Operator |
| `/gocraft` | `gocraft.command.gocraft` | Operator |
| `/god` | `gocraft.command.god` | Operator |
| `/heal` | `gocraft.command.heal` | Operator |
| `/help` | `gocraft.command.help` | Public |
| `/kick` | `gocraft.command.kick` | Operator |
| `/kill` | `gocraft.command.kill` | Operator |
| `/list` | `gocraft.command.list` | Public |
| `/locate` | `gocraft.command.locate` | Operator |
| `/me` | `gocraft.command.me` | Public |
| `/msg` | `gocraft.command.msg` | Public |
| `/mspt` | `gocraft.command.mspt` | Operator |
| `/op` | `gocraft.command.op` | Public (bootstrap-protected) |
| `/pardon` | `gocraft.command.pardon` | Operator |
| `/pardon-ip` | `gocraft.command.pardon-ip` | Operator |
| `/potioneffect` | `gocraft.command.potioneffect` | Operator |
| `/random` | `gocraft.command.random` | Operator |
| `/reload` | `gocraft.command.reload` | Operator |
| `/rotate` | `gocraft.command.rotate` | Operator |
| `/save-all` | `gocraft.command.save-all` | Operator |
| `/save-off` | `gocraft.command.save-off` | Operator |
| `/save-on` | `gocraft.command.save-on` | Operator |
| `/say` | `gocraft.command.say` | Operator |
| `/seed` | `gocraft.command.seed` | Operator |
| `/setblock` | `gocraft.command.setblock` | Operator |
| `/setspawn` | `gocraft.command.setspawn` | Operator |
| `/setworldspawn` | `gocraft.command.setworldspawn` | Operator |
| `/spawn` | `gocraft.command.spawn` | Public |
| `/spawnpoint` | `gocraft.command.spawnpoint` | Operator |
| `/stop` | `gocraft.command.stop` | Operator |
| `/summon` | `gocraft.command.summon` | Operator |
| `/tag` | `gocraft.command.tag` | Operator |
| `/tell` | `gocraft.command.tell` | Public |
| `/time` | `gocraft.command.time` | Operator |
| `/tp` | `gocraft.command.tp` | Operator |
| `/tphere` | `gocraft.command.tphere` | Operator |
| `/tps` | `gocraft.command.tps` | Operator |
| `/ungod` | `gocraft.command.ungod` | Operator |
| `/ver` | `gocraft.command.ver` | Public |
| `/version` | `gocraft.command.version` | Public |
| `/w` | `gocraft.command.w` | Public |
| `/walkspeed` | `gocraft.command.walkspeed` | Operator |
| `/weather` | `gocraft.command.weather` | Operator |
| `/whitelist` | `gocraft.command.whitelist` | Operator |
| `/world` | `gocraft.command.world` | Operator |
| `/xp` | `gocraft.command.xp` | Operator |
| `/xyz` | `gocraft.command.xyz` | Public |

For gaps relative to vanilla Paper/Spigot commands, see [command-parity.md](command-parity.md).
