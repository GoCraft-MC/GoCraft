package main

import (
	"fmt"
	"log/slog"
	"os"

	gocraft "github.com/GoCraft-MC/gocraft-api-go"
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
			"block", event.Block.ID, "position", event.Pos)
	}); err != nil {
		return err
	}
	return nil
}

// Commands is asked twice and never by this file: once by the build, to put the
// shape in the bundle, and once by the loader, to bind what runs. Declaring it
// here is what makes the two agree.
func (p *examplePlugin) Commands() *gocraft.CommandSet {
	set := gocraft.NewCommandSet()
	set.Command("greet").Permission("example.greet").Runs(func(call *gocraft.CommandContext) error {
		call.Reply(fmt.Sprintf("Hello, %s!", call.SenderName))
		return nil
	})
	return set
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
