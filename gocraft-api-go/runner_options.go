package gocraft

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

type runnerOptions struct {
	socket string
	abi    uint
}

func parseRunnerOptions(arguments []string) (runnerOptions, error) {
	var options runnerOptions
	flags := flag.NewFlagSet("gocraft-go-plugin", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.socket, "sock", "", "GoCraft IPC socket")
	flags.UintVar(&options.abi, "abi", 0, "GoCraft ABI version")
	if err := flags.Parse(arguments); err != nil {
		return options, fmt.Errorf("gocraft: parse runtime arguments: %w", err)
	}
	if strings.TrimSpace(options.socket) == "" {
		return options, fmt.Errorf("gocraft: --sock is required")
	}
	if options.abi != uint(CurrentVersion) {
		return options, fmt.Errorf("gocraft: host requested ABI %d, plugin uses %d", options.abi, CurrentVersion)
	}
	return options, nil
}

func validateMetadata(metadata Metadata, implementation Plugin) error {
	if strings.TrimSpace(metadata.ID) == "" {
		return fmt.Errorf("gocraft: plugin id is required")
	}
	if strings.TrimSpace(metadata.Version) == "" {
		return fmt.Errorf("gocraft: plugin version is required")
	}
	if metadata.APIVersion != CurrentVersion {
		return fmt.Errorf("gocraft: plugin API %d is unsupported, runtime uses %d",
			metadata.APIVersion, CurrentVersion)
	}
	if implementation == nil {
		return fmt.Errorf("gocraft: plugin implementation is required")
	}
	return nil
}
