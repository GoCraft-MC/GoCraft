package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"GoCraft/core/command"
	corepermission "GoCraft/core/permission"
	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	"GoCraft/java/handler"
	"GoCraft/java/session"
)

// pluginCommandServer is the smallest server that can answer a plugin command:
// a registry to hold the tree and a permission manager to judge the sender.
func pluginCommandServer(t *testing.T) *Server {
	t.Helper()
	registry := coreplugin.NewRegistry(context.Background(), 2*time.Millisecond, nil, nil)
	return &Server{pluginRegistry: registry, permissions: corepermission.NewMemory()}
}

func registerShopCommand(t *testing.T, server *Server, handler command.Handler) {
	t.Helper()
	root := command.Root{Children: []command.Node{command.Literal{
		Name: "shop", Children: []command.Node{
			command.Argument{Name: "price", Type: command.ArgDecimal, Exec: 1},
		},
	}}}
	source := command.Source{Kind: command.SourcePlugin, PluginID: "shop"}
	handlers := map[command.ExecID]command.Handler{1: handler}
	if err := server.pluginRegistry.Commands().Register(source, root, handlers); err != nil {
		t.Fatal(err)
	}
}

func TestRunPluginCommandReachesTheHandler(t *testing.T) {
	server := pluginCommandServer(t)
	var price float64
	var senderName string
	registerShopCommand(t, server, func(_ context.Context, call *command.Context) error {
		price, _ = call.Args.Decimal("price")
		senderName = call.Sender.Name()
		return nil
	})

	handled, err := server.runPluginCommand(&player.Player{Username: "oreo"}, "shop 12.5")
	if !handled || err != nil {
		t.Fatalf("run = (%t, %v), want (true, nil)", handled, err)
	}
	if price != 12.5 || senderName != "oreo" {
		t.Fatalf("handler saw price %v from %q", price, senderName)
	}
}

func TestRunPluginCommandIgnoresALineItDoesNotOwn(t *testing.T) {
	server := pluginCommandServer(t)
	registerShopCommand(t, server, func(context.Context, *command.Context) error { return nil })

	handled, err := server.runPluginCommand(nil, "gamemode creative")
	if handled || err != nil {
		t.Fatalf("run = (%t, %v), want (false, nil)", handled, err)
	}
}

// A server with no plugin registry answers nothing rather than panicking: the
// bridge is installed before plugins load, and may never have any.
func TestRunPluginCommandToleratesNoRegistry(t *testing.T) {
	handled, err := (&Server{}).runPluginCommand(nil, "shop 1")
	if handled || err != nil {
		t.Fatalf("run = (%t, %v), want (false, nil)", handled, err)
	}
}

// A permission refusal is rewritten into a sentence: core/command's sentinel
// says what happened, not what to show someone.
func TestRunPluginCommandRewritesAPermissionRefusal(t *testing.T) {
	server := pluginCommandServer(t)
	root := command.Root{Children: []command.Node{command.Literal{
		Name: "admin", Permission: "shop.admin",
		Children: []command.Node{command.Literal{Name: "reload", Exec: 1}},
	}}}
	source := command.Source{Kind: command.SourcePlugin, PluginID: "shop"}
	handlers := map[command.ExecID]command.Handler{
		1: func(context.Context, *command.Context) error { return nil },
	}
	if err := server.pluginRegistry.Commands().Register(source, root, handlers); err != nil {
		t.Fatal(err)
	}

	// Hidden from a player who lacks the node, and reachable from the console,
	// which holds everything.
	handled, err := server.runPluginCommand(&player.Player{Username: "oreo"}, "admin reload")
	if handled {
		t.Fatalf("a hidden branch was claimed, err = %v", err)
	}
	handled, err = server.pluginRegistry.Commands().Execute(context.Background(),
		server.consoleCommandSender(), "admin reload", server.commandResolvers())
	if !handled || err != nil {
		t.Fatalf("console run = (%t, %v), want (true, nil)", handled, err)
	}
}

func TestRunPluginCommandReportsAnArgumentFailure(t *testing.T) {
	server := pluginCommandServer(t)
	registerShopCommand(t, server, func(context.Context, *command.Context) error { return nil })

	handled, err := server.runPluginCommand(&player.Player{Username: "oreo"}, "shop cheap")
	if !handled {
		t.Fatal("a line naming a plugin command was reported as unhandled")
	}
	if err == nil || !strings.Contains(err.Error(), "must be a number") {
		t.Fatalf("argument failure = %v", err)
	}
}

func TestSplitIdentifierDefaultsAndRefuses(t *testing.T) {
	cases := []struct {
		raw       string
		namespace string
		name      string
		ok        bool
	}{
		{"stone", "minecraft", "stone", true},
		{"Minecraft:Stone", "minecraft", "stone", true},
		{"oreo:shop/token", "oreo", "shop/token", true},
		{"", "", "", false},
		{":stone", "", "", false},
		{"minecraft:", "", "", false},
		{"minecraft:st one", "", "", false},
		{"minecraft:stone!", "", "", false},
	}
	for _, testCase := range cases {
		namespace, name, ok := splitIdentifier(testCase.raw)
		if ok != testCase.ok || namespace != testCase.namespace || name != testCase.name {
			t.Fatalf("splitIdentifier(%q) = (%q, %q, %t), want (%q, %q, %t)",
				testCase.raw, namespace, name, ok, testCase.namespace, testCase.name, testCase.ok)
		}
	}
}

// The version is read once per tick and nothing is sent while it stands still.
// A resend on every tick would mean shipping the whole command list twenty
// times a second to everyone.
func TestResendChangedCommandsOnlyFollowsTheVersion(t *testing.T) {
	dispatcher := handler.NewDispatcher()
	registry := command.NewRegistry()
	dispatcher.SetCommandRegistry(registry)
	server := &Server{cmds: dispatcher, sessions: session.NewManager()}

	dispatcher.Register("first", func(handler.CommandContext) error { return nil })
	server.resendChangedCommands()
	settled := server.commandTreeVersion
	if settled == 0 {
		t.Fatal("the first tick recorded no version")
	}

	server.resendChangedCommands()
	if server.commandTreeVersion != settled {
		t.Fatal("an unchanged tree moved the recorded version")
	}

	dispatcher.Register("second", func(handler.CommandContext) error { return nil })
	server.resendChangedCommands()
	if server.commandTreeVersion <= settled {
		t.Fatalf("a new command left the version at %d", server.commandTreeVersion)
	}
}

// A server whose dispatcher has no registry never claims a version, so it
// never resends.
func TestResendChangedCommandsToleratesNoRegistry(t *testing.T) {
	server := &Server{cmds: handler.NewDispatcher(), sessions: session.NewManager()}
	server.resendChangedCommands()
	if server.commandTreeVersion != 0 {
		t.Fatalf("version = %d with no registry", server.commandTreeVersion)
	}
}
