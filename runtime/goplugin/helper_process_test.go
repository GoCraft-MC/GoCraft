package goplugin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"GoCraft/runtime/link"
	gocraft "github.com/GoCraft-MC/gocraft-api-go"
)

type helperPlugin struct{}

// Race-instrumented helpers also initialize the imported gameplay registries.
// Match the normal runtime startup allowance, not its per-event budget.
const helperStartTimeout = 30 * time.Second

func (*helperPlugin) OnLoad(context gocraft.Context) error {
	if os.Getenv("GOCRAFT_NATIVE_EVENTS") == "1" {
		if err := context.Events().OnPlayerChat(func(event *gocraft.PlayerChatEvent, control gocraft.EventControl) {
			if event.Message == "cancel" {
				control.Cancel()
			}
			event.Message = "rewritten"
		}); err != nil {
			return err
		}
	}
	if os.Getenv("GOCRAFT_NATIVE_PLUGIN_FAILURE") == "load" {
		return errors.New("load failure")
	}
	if err := context.Events().OnBlockBreak(func(event *gocraft.BlockBreakEvent, control gocraft.EventControl) {
		control.Cancel()
	}); err != nil {
		return err
	}
	return context.Commands().Register("give <amount>", func(call *gocraft.CommandContext) error {
		value, ok := call.Args.Integer("amount")
		if !ok {
			return fmt.Errorf("amount is missing")
		}
		call.Reply(fmt.Sprintf("amount=%d", value))
		return nil
	})
}

func (*helperPlugin) OnEnable() error {
	if os.Getenv("GOCRAFT_NATIVE_PLUGIN_FAILURE") == "enable" {
		return errors.New("enable failure")
	}
	return nil
}
func (*helperPlugin) OnDisable() error { return nil }

func TestNativePluginHelperProcess(t *testing.T) {
	if os.Getenv("GOCRAFT_NATIVE_PLUGIN_HELPER") != "1" {
		return
	}
	separator := 0
	for index, argument := range os.Args {
		if argument == "--" {
			separator = index
			break
		}
	}
	os.Args = append([]string{os.Args[0]}, os.Args[separator+1:]...)
	err := gocraft.Run(gocraft.Metadata{
		ID: "example", Version: "1.0.0", APIVersion: gocraft.CurrentVersion,
	}, &helperPlugin{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func helperSpawn(string) link.Spawn {
	return helperSpawnFailure("")
}

func helperSpawnFailure(failure string) link.Spawn {
	return func(socket string) *exec.Cmd {
		command := exec.Command(os.Args[0],
			"-test.run=TestNativePluginHelperProcess", "--",
			"--sock", socket, "--abi", "1")
		command.Env = append(os.Environ(), "GOCRAFT_NATIVE_PLUGIN_HELPER=1",
			"GOCRAFT_NATIVE_PLUGIN_FAILURE="+failure)
		command.Stderr = os.Stderr
		return command
	}
}
