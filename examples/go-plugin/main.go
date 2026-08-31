package main

import (
	"fmt"
	"log/slog"
	"os"

	"GoCraft/pluginapi"
)

var metadata = pluginapi.Metadata{
	ID: "example-go", Version: "1.0.0", APIVersion: pluginapi.CurrentVersion,
}

type examplePlugin struct {
	context pluginapi.Context
}

func (p *examplePlugin) OnLoad(context pluginapi.Context) error {
	p.context = context
	context.Logger().Info("loaded", "data", context.DataDirectory())
	if err := context.Events().OnPlayerJoin(func(event *pluginapi.PlayerJoinEvent) {
		context.Logger().Info("player joined", "player", event.Player.Username,
			"edition", event.Player.Edition)
	}); err != nil {
		return err
	}
	if err := context.Events().OnBlockBreak(func(event *pluginapi.BlockBreakEvent) {
		context.Logger().Info("block broken", "player", event.Player.Username,
			"block", event.Block.ID, "position", event.Position)
	}); err != nil {
		return err
	}
	return context.Commands().Register(1, func(call *pluginapi.CommandContext) error {
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
	if err := pluginapi.Run(metadata, &examplePlugin{}); err != nil {
		slog.Error("example plugin stopped", "err", err)
		os.Exit(1)
	}
}
