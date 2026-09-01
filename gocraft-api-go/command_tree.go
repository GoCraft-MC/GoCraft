package gocraft

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"strings"

	"google.golang.org/protobuf/proto"

	wire "GoCraft/abi/v1/wire"
)

// maximumCommandTreeSize matches the cap the host reads a bundle entry under.
// A plugin opening its own bundle is not a reason to trust its size.
const maximumCommandTreeSize = 4 << 20

// commandTree is what a bundle declared, indexed by the path a handler
// registers against.
//
// The executor ids the tree assigns are deliberately not part of the plugin
// API. They are assigned by whatever built the tree, so naming one in plugin
// source would be writing it down a second time, free to disagree with the
// first the day a command is inserted above it. A path cannot drift from the
// tree that defines it.
type commandTree struct {
	executors map[string]uint32
	// paths, in declaration order, so a registration that misses can say what
	// the bundle actually declares instead of only that it missed.
	paths []string
}

// loadCommandTree reads the tree out of the bundle this plugin was loaded from.
//
// The host sends the path rather than the tree itself: it already told the
// runtime where the bundle is, and a copy on the wire would be a second
// definition of something both sides can read from one file.
//
// A plugin that declares no commands has no tree and no error — most do not.
func loadCommandTree(bundlePath, entry string) (*commandTree, error) {
	if entry == "" {
		return &commandTree{executors: map[string]uint32{}}, nil
	}
	if bundlePath == "" {
		return nil, fmt.Errorf("gocraft: the host sent a command tree but no bundle")
	}
	encoded, err := readBundleEntry(bundlePath, entry)
	if err != nil {
		return nil, err
	}
	var decoded wire.CommandTree
	if err := proto.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("gocraft: decode %s: %w", entry, err)
	}
	tree := &commandTree{executors: make(map[string]uint32)}
	tree.index("", decoded.GetChildren())
	return tree, nil
}

// index walks the tree, naming every executable node by the path that reaches
// it: literals as written, arguments in angle brackets, exactly as the Java
// facade spells them. One vocabulary across runtimes, or an author moving
// between them learns it twice.
func (t *commandTree) index(prefix string, nodes []*wire.CommandNode) {
	for _, node := range nodes {
		label := node.GetName()
		if node.GetKind() == wire.CommandNodeKind_COMMAND_NODE_KIND_ARGUMENT {
			label = "<" + label + ">"
		}
		reached := label
		if prefix != "" {
			reached = prefix + " " + label
		}
		if executor := node.GetExecutor(); executor != 0 {
			if _, taken := t.executors[reached]; !taken {
				t.executors[reached] = executor
				t.paths = append(t.paths, reached)
			}
		}
		t.index(reached, node.GetChildren())
	}
}

// lookup resolves a registration path, and explains itself when it cannot. The
// list is the point: a typo in a path is otherwise a command that silently
// never runs, which is the afternoon nobody wants to spend.
func (t *commandTree) lookup(commandPath string) (uint32, error) {
	normalised := strings.Join(strings.Fields(commandPath), " ")
	if normalised == "" {
		return 0, fmt.Errorf("gocraft: a command path is required")
	}
	executor, ok := t.executors[normalised]
	if !ok {
		if len(t.paths) == 0 {
			return 0, fmt.Errorf("gocraft: this bundle declares no commands, so %q cannot be registered", normalised)
		}
		return 0, fmt.Errorf("gocraft: no command %q in this bundle; it declares %s",
			normalised, strings.Join(t.paths, ", "))
	}
	return executor, nil
}

func readBundleEntry(bundlePath, entry string) ([]byte, error) {
	archive, err := zip.OpenReader(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("gocraft: open bundle: %w", err)
	}
	defer archive.Close()
	for _, file := range archive.File {
		if path.Clean(strings.ReplaceAll(file.Name, `\`, "/")) != entry {
			continue
		}
		if file.UncompressedSize64 > maximumCommandTreeSize {
			return nil, fmt.Errorf("gocraft: %s exceeds %d bytes", entry, maximumCommandTreeSize)
		}
		reader, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("gocraft: open %s: %w", entry, err)
		}
		defer reader.Close()
		return io.ReadAll(io.LimitReader(reader, maximumCommandTreeSize))
	}
	return nil, fmt.Errorf("gocraft: bundle has no %s", entry)
}
