# GoCraft permissions

GoCraft has one canonical permission service for Java and Bedrock players. Its data is stored in
`permissions.json`, reloaded atomically through the editor, and checked for every command execution.

## Browser editor

1. As an operator, run `/gocraft peditor`. From the console, run `gocraft peditor`.
2. Open the generated link. Java receives a clickable link; Bedrock receives the URL as text.
3. Create groups, choose parent groups, set each command to Allow, Deny, or Unset, and assign players.
4. Select **Save edits**.
5. Copy the displayed `gocraft applyedits <link>` command into the server console, or run
   `/gocraft applyedits <link>` in game.

Editor links contain a random 192-bit token, expire after the configured lifetime, cannot be applied
twice, and reject an apply if another editor changed permissions after the link was created.

For a remotely hosted server, change `public_url` to the HTTPS address players can reach and put a
TLS reverse proxy in front of the editor. The safe default listens only on localhost:

```yaml
permission_editor:
  enabled: true
  address: 127.0.0.1:8080
  public_url: http://127.0.0.1:8080
  session_minutes: 15
```

## Permission nodes

Every registered command automatically receives `gocraft.command.<command>`. Examples:

- `gocraft.command.spawn`
- `gocraft.command.give`
- `gocraft.command.gocraft`
- `gocraft.command.*` for all GoCraft commands
- `*` for every permission

An explicit user rule takes priority over group rules. More specific nodes take priority over
wildcards, a child group's equally specific rule takes priority over its parent, and Deny wins an
equal-priority conflict. Players always inherit `default`. Operators retain a full bypass so an
invalid group edit cannot lock the server owner out.

Commands that were public before this system remain allowed when no rule is set. Commands that were
operator-only remain denied by default, but a group can now grant them individually. Java command
suggestions hide commands the player cannot use and refresh immediately after edits are applied.

## File format

The editor owns this file, but it remains human-readable:

```json
{
  "version": 1,
  "groups": {
    "default": {},
    "builder": {
      "parents": ["default"],
      "permissions": {
        "gocraft.command.give": true,
        "gocraft.command.stop": false
      }
    }
  },
  "users": {
    "alex": {"groups": ["builder"]}
  }
}
```

Names are matched case-insensitively. Group inheritance cycles, unknown parents, unknown user groups,
malformed permission nodes, oversized editor requests, and stale editor sessions are rejected.

This is a native GoCraft permission system inspired by the LuckPerms workflow. It does not load the
LuckPerms plugin or its database and does not yet implement contexts, weights, tracks, temporary
permissions, prefixes/suffixes, UUID-based assignments, or LuckPerms import/export compatibility.
