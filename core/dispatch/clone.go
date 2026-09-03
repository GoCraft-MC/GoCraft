package dispatch

import "github.com/GoCraft-MC/gocraft-abi/command"

func remapRoot(root command.Root, allocate func() command.ExecID) (command.Root, map[command.ExecID]command.ExecID) {
	remapped := make(map[command.ExecID]command.ExecID)
	return command.Root{Children: remapNodes(root.Children, allocate, remapped)}, remapped
}

func remapNodes(nodes []command.Node, allocate func() command.ExecID, remapped map[command.ExecID]command.ExecID) []command.Node {
	cloned := make([]command.Node, 0, len(nodes))
	for _, node := range nodes {
		switch typed := node.(type) {
		case command.Literal:
			typed.Children = remapNodes(typed.Children, allocate, remapped)
			typed.Exec = remapExecutor(typed.Exec, allocate, remapped)
			cloned = append(cloned, typed)
		case command.Argument:
			typed = cloneArgument(typed)
			typed.Children = remapNodes(typed.Children, allocate, remapped)
			typed.Exec = remapExecutor(typed.Exec, allocate, remapped)
			cloned = append(cloned, typed)
		}
	}
	return cloned
}

func cloneArgument(argument command.Argument) command.Argument {
	argument.Enum = append([]string(nil), argument.Enum...)
	if argument.IntegerMin != nil {
		value := *argument.IntegerMin
		argument.IntegerMin = &value
	}
	if argument.IntegerMax != nil {
		value := *argument.IntegerMax
		argument.IntegerMax = &value
	}
	if argument.DecimalMin != nil {
		value := *argument.DecimalMin
		argument.DecimalMin = &value
	}
	if argument.DecimalMax != nil {
		value := *argument.DecimalMax
		argument.DecimalMax = &value
	}
	return argument
}

func remapExecutor(executor command.ExecID, allocate func() command.ExecID, remapped map[command.ExecID]command.ExecID) command.ExecID {
	if executor == 0 {
		return 0
	}
	if global, ok := remapped[executor]; ok {
		return global
	}
	global := allocate()
	remapped[executor] = global
	return global
}
