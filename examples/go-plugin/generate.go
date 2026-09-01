//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	wire "github.com/GoCraft-MC/gocraft-abi/abi/v1/wire"
	"google.golang.org/protobuf/proto"
)

func main() {
	tree := &wire.CommandTree{
		Version: 1,
		Children: []*wire.CommandNode{{
			Kind:       wire.CommandNodeKind_COMMAND_NODE_KIND_LITERAL,
			Name:       "greet",
			Permission: "example.greet",
			Executor:   1,
		}},
	}
	encoded, err := proto.Marshal(tree)
	if err != nil {
		panic(err)
	}
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot find generator directory")
	}
	target := filepath.Join(filepath.Dir(source), "commands.pb")
	if err := os.WriteFile(target, encoded, 0o644); err != nil {
		panic(err)
	}
	fmt.Println(target)
}
