package main

import (
	"fmt"
	"log/slog"
	"os"

	gocraft "GoCraft/gocraft-api-go"
)

var metadata = gocraft.Metadata{
	ID: "example-go", Version: "1.0.0", APIVersion: gocraft.CurrentVersion,
}

type examplePlugin struct {
	context gocraft.Context
}

func (p *examplePlugin) OnLoad(context gocraft.Context) error {
	p.context = context
	context.Logger().Info("loaded", "data", context.DataDirectory())
	if err := context.Events().OnPlayerJoin(func(event *gocraft.PlayerJoinEvent) {
		context.Logger().Info("player joined", "player", event.Player.Username,
			"edition", event.Player.Edition)
	}); err != nil {
		return err
	}
	if err := context.Events().OnBlockBreak(func(event *gocraft.BlockBreakEvent) {
		context.Logger().Info("block broken", "player", event.Player.Username,
			"block", event.Block.ID, "position", event.Position)
	}); err != nil {
		return err
	}
	// The path, as commands.pb spells it. The executor id that tree assigns is
	// the tree's business, not this file's.
	return context.Commands().Register("greet", func(call *gocraft.CommandContext) error {
		call.Reply(fmt.Sprintf("Hello, %s!", call.SenderName))
		return nil
	})
}

func (p *examplePlugin) OnEnable() error {
	p.context.Logger().Info("enabled")
	return nil
}

func (p *examplePlugin) OnDisable() error {
	p.context.Logger().Info("disabled")
	return nil
}

func main() {
	if err := gocraft.Run(metadata, &examplePlugin{}); err != nil {
		slog.Error("example plugin stopped", "err", err)
		os.Exit(1)
	}
}
