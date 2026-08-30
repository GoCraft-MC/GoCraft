package plugin

import (
	"path/filepath"
	"strings"
	"testing"

	"GoCraft/core/command"
	"google.golang.org/protobuf/encoding/protowire"
)

func testCommandTree() string {
	node := protowire.AppendTag(nil, 1, protowire.VarintType)
	node = protowire.AppendVarint(node, 1)
	node = protowire.AppendTag(node, 2, protowire.BytesType)
	node = protowire.AppendString(node, "hello")
	node = protowire.AppendTag(node, 6, protowire.VarintType)
	node = protowire.AppendVarint(node, 1)
	tree := protowire.AppendTag(nil, 1, protowire.VarintType)
	tree = protowire.AppendVarint(tree, command.CommandWireVersion)
	tree = protowire.AppendTag(tree, 2, protowire.BytesType)
	return string(protowire.AppendBytes(tree, node))
}

func commandManifest(tree string) string {
	return validTestManifest + "[commands]\ntree = \"" + tree + "\"\n"
}

func TestOpenBundleDecodesGeneratedCommandTree(t *testing.T) {
	directory := t.TempDir()
	writeBundle(t, directory, "commands.gcpkg", commandManifest("commands.pb"), map[string]string{
		"commands.pb": testCommandTree(),
	})
	bundle, err := OpenBundle(filepath.Join(directory, "commands.gcpkg"))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Commands == nil || len(bundle.Commands.Children) != 1 {
		t.Fatalf("commands = %#v", bundle.Commands)
	}
	literal := bundle.Commands.Children[0].(command.Literal)
	if literal.Name != "hello" || literal.Exec != 1 {
		t.Fatalf("literal = %+v", literal)
	}
}

func TestOpenBundleRejectsInvalidCommandAssets(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		extra    map[string]string
		want     string
	}{
		{name: "unsafe reference", manifest: commandManifest("../commands.pb"), want: "invalid command tree path"},
		{name: "missing asset", manifest: commandManifest("commands.pb"), want: "missing entry commands.pb"},
		{name: "invalid protobuf", manifest: commandManifest("commands.pb"), extra: map[string]string{"commands.pb": "bad"}, want: "command tree"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			writeBundle(t, directory, "invalid.gcpkg", tc.manifest, tc.extra)
			_, err := OpenBundle(filepath.Join(directory, "invalid.gcpkg"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("OpenBundle() error = %v, want %q", err, tc.want)
			}
		})
	}
}
