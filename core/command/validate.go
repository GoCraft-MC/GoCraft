package command

import (
	"fmt"
	"strings"
)

func Validate(root *Root) error {
	if root == nil {
		return fmt.Errorf("command tree: nil root")
	}
	if len(root.Children) == 0 {
		return fmt.Errorf("command tree: root has no commands")
	}
	for _, child := range root.Children {
		if _, ok := child.(Literal); !ok {
			return fmt.Errorf("command tree: root commands must be literals")
		}
	}
	return validateChildren("/", root.Children)
}

func validateChildren(path string, children []Node) error {
	seen := make(map[string]struct{}, len(children))
	for _, child := range children {
		name, kind, err := nodeIdentity(child)
		if err != nil {
			return fmt.Errorf("command tree %s: %w", path, err)
		}
		key := kind + ":" + name
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("command tree %s: duplicate %s %q", path, kind, name)
		}
		seen[key] = struct{}{}
		if err := validateNode(path, child); err != nil {
			return err
		}
	}
	return nil
}

func validateNode(parent string, node Node) error {
	name, _, _ := nodeIdentity(node)
	path := strings.TrimSuffix(parent, "/") + "/" + name
	var children []Node
	var executor ExecID
	switch typed := node.(type) {
	case Literal:
		children, executor = typed.Children, typed.Exec
	case Argument:
		children, executor = typed.Children, typed.Exec
		if typed.Type < ArgInteger || typed.Type > ArgCustom {
			return fmt.Errorf("command tree %s: invalid argument type %d", path, typed.Type)
		}
		if err := validateArgument(typed); err != nil {
			return fmt.Errorf("command tree %s: %w", path, err)
		}
		if typed.Type == ArgGreedy && len(children) != 0 {
			return fmt.Errorf("command tree %s: greedy argument must be last", path)
		}
	default:
		return fmt.Errorf("command tree %s: root may only appear at the top", path)
	}
	// An executable node may still have children, which is how an optional
	// argument is spelled: /kill runs on its own and /kill <player> runs on
	// someone else. Refusing that would make the IR unable to express half of
	// what a vanilla command tree already does.
	if executor == 0 && len(children) == 0 {
		return fmt.Errorf("command tree %s: leaf has no executor", path)
	}
	return validateChildren(path, children)
}

func nodeIdentity(node Node) (name, kind string, err error) {
	switch typed := node.(type) {
	case Literal:
		name, kind = typed.Name, "literal"
	case Argument:
		name, kind = typed.Name, "argument"
	case Root:
		return "", "root", fmt.Errorf("nested root")
	default:
		return "", "unknown", fmt.Errorf("unknown node type %T", node)
	}
	if name == "" || strings.TrimSpace(name) != name || strings.ContainsAny(name, " \t\r\n/") {
		return "", kind, fmt.Errorf("invalid %s name %q", kind, name)
	}
	return name, kind, nil
}
