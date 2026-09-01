package gocraft

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	wire "GoCraft/abi/v1/wire"
)

func literalNode(name string, executor uint32, children ...*wire.CommandNode) *wire.CommandNode {
	return &wire.CommandNode{
		Kind: wire.CommandNodeKind_COMMAND_NODE_KIND_LITERAL,
		Name: name, Executor: executor, Children: children,
	}
}

func argumentNode(name string, executor uint32, children ...*wire.CommandNode) *wire.CommandNode {
	return &wire.CommandNode{
		Kind:         wire.CommandNodeKind_COMMAND_NODE_KIND_ARGUMENT,
		Name:         name,
		ArgumentType: wire.CommandArgumentType_COMMAND_ARGUMENT_TYPE_DECIMAL,
		Executor:     executor, Children: children,
	}
}

// shopNodes is the tree the tests in this package register against: one path
// ending in an argument, one ending in a literal.
func shopNodes() []*wire.CommandNode {
	return []*wire.CommandNode{literalNode("shop", 0,
		literalNode("sell", 0, argumentNode("price", 7)),
		literalNode("reload", 9),
	)}
}

func testCommandTree() *commandTree {
	tree := &commandTree{executors: map[string]uint32{}}
	tree.index("", shopNodes())
	return tree
}

// bundleWithCommands writes a real .gcpkg holding a real command tree, so the
// path a plugin registers is read the way it will be in production rather than
// from a fixture built in memory.
func bundleWithCommands(t *testing.T, entry string, nodes []*wire.CommandNode) string {
	t.Helper()
	encoded, err := proto.Marshal(&wire.CommandTree{Version: 1, Children: nodes})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plugin.gcpkg")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	writer, err := archive.Create(entry)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(encoded); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadCommandTreeIndexesEveryExecutablePath(t *testing.T) {
	bundle := bundleWithCommands(t, "commands.pb", shopNodes())
	tree, err := loadCommandTree(bundle, "commands.pb")
	if err != nil {
		t.Fatal(err)
	}

	executor, err := tree.lookup("shop sell <price>")
	if err != nil || executor != 7 {
		t.Fatalf("shop sell = %d, %v", executor, err)
	}
	if executor, err := tree.lookup("shop reload"); err != nil || executor != 9 {
		t.Fatalf("shop reload = %d, %v", executor, err)
	}
	// A node carrying no executor is not a path a handler can bind to.
	if _, err := tree.lookup("shop"); err == nil {
		t.Fatal("a node with no executor was registrable")
	}
}

// Spacing is not part of the contract; the path is.
func TestLookupNormalisesSpacing(t *testing.T) {
	tree := testCommandTree()
	if _, err := tree.lookup("  shop   sell   <price> "); err != nil {
		t.Fatal(err)
	}
	if _, err := tree.lookup(""); err == nil {
		t.Fatal("an empty path was accepted")
	}
}

// A plugin declaring no commands has no tree, and that is not a failure.
func TestLoadCommandTreeWithoutAnEntry(t *testing.T) {
	tree, err := loadCommandTree("", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tree.lookup("anything"); err == nil ||
		!strings.Contains(err.Error(), "declares no commands") {
		t.Fatalf("empty tree reported %v", err)
	}
}

// A bundle that promises a tree and does not carry one fails the load: its
// commands would never run, and an admin deserves the reason now rather than a
// plugin that half works.
func TestLoadCommandTreeRefusesAMissingEntry(t *testing.T) {
	bundle := bundleWithCommands(t, "elsewhere.pb", shopNodes())
	if _, err := loadCommandTree(bundle, "commands.pb"); err == nil {
		t.Fatal("a missing command tree loaded")
	}
	if _, err := loadCommandTree("", "commands.pb"); err == nil {
		t.Fatal("a command tree with no bundle loaded")
	}
}

func TestPluginRegistersACommandByPathAtLoad(t *testing.T) {
	bundle := bundleWithCommands(t, "commands.pb", shopNodes())
	state := newRuntimeState(Metadata{ID: "commands"}, &wireCommandPlugin{})
	if _, err := state.load(loadRequest{
		pluginID: "commands", dataDirectory: "data",
		bundlePath: bundle, commandTree: "commands.pb",
	}); err != nil {
		t.Fatal(err)
	}
	if _, registered := state.context.commands.handlers[7]; !registered {
		t.Fatal("the path did not resolve to the executor the tree assigns")
	}
}
