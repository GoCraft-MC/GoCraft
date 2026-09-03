package dispatch

import (
	"fmt"

	"github.com/GoCraft-MC/gocraft-abi/command"
)

func sourceKey(source Source) (string, error) {
	if source.Kind == SourceCore {
		return "core", nil
	}
	if source.Kind != SourcePlugin || source.PluginID == "" {
		return "", fmt.Errorf("command: invalid source")
	}
	return source.PluginID, nil
}

func rootConflict(left, right []command.Node) string {
	names := make(map[string]struct{}, len(left))
	for _, node := range left {
		if literal, ok := node.(command.Literal); ok {
			names[literal.Name] = struct{}{}
		}
	}
	for _, node := range right {
		literal, ok := node.(command.Literal)
		if !ok {
			continue
		}
		if _, exists := names[literal.Name]; exists {
			return literal.Name
		}
	}
	return ""
}
