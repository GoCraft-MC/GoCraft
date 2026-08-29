package command

func remapRoot(root Root, allocate func() ExecID) (Root, map[ExecID]ExecID) {
	remapped := make(map[ExecID]ExecID)
	return Root{Children: remapNodes(root.Children, allocate, remapped)}, remapped
}

func remapNodes(nodes []Node, allocate func() ExecID, remapped map[ExecID]ExecID) []Node {
	cloned := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		switch typed := node.(type) {
		case Literal:
			typed.Children = remapNodes(typed.Children, allocate, remapped)
			typed.Exec = remapExecutor(typed.Exec, allocate, remapped)
			cloned = append(cloned, typed)
		case Argument:
			typed.Enum = append([]string(nil), typed.Enum...)
			typed.Children = remapNodes(typed.Children, allocate, remapped)
			typed.Exec = remapExecutor(typed.Exec, allocate, remapped)
			cloned = append(cloned, typed)
		}
	}
	return cloned
}

func remapExecutor(executor ExecID, allocate func() ExecID, remapped map[ExecID]ExecID) ExecID {
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
