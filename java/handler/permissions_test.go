package handler

import (
	"strings"
	"testing"

	"GoCraft/core/player"
	"GoCraft/java/session"
)

func TestAdministrativeCommandsRequireOperator(t *testing.T) {
	dispatcher := NewDispatcher()
	RegisterBuiltins(dispatcher)
	for _, name := range []string{`gamemode`, `tp`, `tphere`, `give`, `kill`, `fly`, `god`, `ungod`, `heal`, `effect`} {
		command, ok := dispatcher.cmds[name]
		if !ok {
			t.Errorf(`administrative command %q is not registered`, name)
			continue
		}
		if !command.operatorOnly {
			t.Errorf(`administrative command %q is not operator-only`, name)
		}
	}
	for _, name := range []string{`help`, `list`, `xyz`, `version`, `op`} {
		command, ok := dispatcher.cmds[name]
		if !ok {
			t.Errorf(`public command %q is not registered`, name)
			continue
		}
		if command.operatorOnly {
			t.Errorf(`bootstrap/public command %q unexpectedly requires operator`, name)
		}
	}
}

func TestDispatcherChecksPermissionNodePerCommand(t *testing.T) {
	dispatcher := NewDispatcher()
	dispatcher.RegisterOperator("restricted", func(ctx CommandContext) error {
		return sendCommandMessage(ctx, "executed")
	})
	var checkedNode string
	dispatcher.SetPermissionChecker(func(_ *player.Player, node string, defaultAllowed bool) bool {
		checkedNode = node
		if defaultAllowed {
			t.Error("operator command unexpectedly defaults to allowed")
		}
		return true
	})
	var reply string
	dispatcher.Dispatch("/restricted", CommandContext{
		Player: player.New([16]byte{9}, "builder", player.ClientEditionBedrock),
		Reply: func(message string) error {
			reply = message
			return nil
		},
	})
	if checkedNode != "gocraft.command.restricted" || reply != "executed" {
		t.Fatalf("node/reply = %q/%q", checkedNode, reply)
	}

	dispatcher.SetPermissionChecker(func(_ *player.Player, _ string, _ bool) bool { return false })
	dispatcher.Dispatch("/restricted", CommandContext{
		Player: player.New([16]byte{8}, "denied", player.ClientEditionJava),
		Reply:  func(message string) error { reply = message; return nil },
	})
	if !strings.Contains(reply, "permission") {
		t.Fatalf("denial reply = %q", reply)
	}
}

func TestHelpListsOnlyPermittedCommands(t *testing.T) {
	dispatcher := NewDispatcher()
	RegisterBuiltins(dispatcher)
	var reply string
	dispatcher.Dispatch("/help", CommandContext{
		Player: player.New([16]byte{7}, "viewer", player.ClientEditionBedrock),
		Reply:  func(message string) error { reply = message; return nil },
	})
	if !strings.Contains(reply, "/help") || strings.Contains(reply, "/gamemode") {
		t.Fatalf("permission-filtered help = %q", reply)
	}
}

func TestGodModeBlocksNormalDamageButKillOverridesIt(t *testing.T) {
	p := player.New([16]byte{1}, `invincible`, player.ClientEditionJava)
	p.GodMode = true
	target := &session.Session{Player: p}
	if DamagePlayer(target, 5, `was tested`, nil) {
		t.Fatal(`normal damage was applied while god mode was enabled`)
	}
	if health, _, _, _ := p.HealthSnapshot(); health != p.MaxHealth {
		t.Fatalf(`god-mode health = %v, want %v`, health, p.MaxHealth)
	}
	if !KillPlayer(target, `was killed`, nil) {
		t.Fatal(`/kill did not override god mode`)
	}
}

func TestBedrockGameModeCommandRefreshesEditionState(t *testing.T) {
	p := player.New([16]byte{2}, `bedrock-builder`, player.ClientEditionBedrock)
	p.GameMode = player.GameModeCreative
	var (
		refreshed *player.Player
		reply     string
	)
	err := cmdGameMode(CommandContext{
		Player: p,
		Args:   []string{"survival"},
		SyncAbilities: func(changed *player.Player) {
			refreshed = changed
		},
		Reply: func(message string) error {
			reply = message
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.GameMode != player.GameModeSurvival || refreshed != p {
		t.Fatalf("game mode = %v, refreshed = %p, want survival/%p", p.GameMode, refreshed, p)
	}
	if reply != "Game mode changed to survival" {
		t.Fatalf("reply = %q", reply)
	}
}
