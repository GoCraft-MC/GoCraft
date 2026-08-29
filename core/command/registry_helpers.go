package command

import "fmt"

func sourceKey(source Source) (string, error) {
	if source.Kind == SourceCore {
		return "core", nil
	}
	if source.Kind != SourcePlugin || source.PluginID == "" {
		return "", fmt.Errorf("command: invalid source")
	}
	return source.PluginID, nil
}

func collectExecutors(nodes []Node, out map[ExecID]struct{}) {
	for _, node := range nodes {
		switch typed := node.(type) {
		case Literal:
			if typed.Exec != 0 {
				out[typed.Exec] = struct{}{}
			}
			collectExecutors(typed.Children, out)
		case Argument:
			if typed.Exec != 0 {
				out[typed.Exec] = struct{}{}
			}
			collectExecutors(typed.Children, out)
		}
	}
}

func rootConflict(left, right []Node) string {
	names := make(map[string]struct{}, len(left))
	for _, node := range left {
		if literal, ok := node.(Literal); ok {
			names[literal.Name] = struct{}{}
		}
	}
	for _, node := range right {
		literal, ok := node.(Literal)
		if !ok {
			continue
		}
		if _, exists := names[literal.Name]; exists {
			return literal.Name
		}
	}
	return ""
}
