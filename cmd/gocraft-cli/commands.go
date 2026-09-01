package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"GoCraft/core/command"
)

// The command trees a compiler handed over, turned into what a bundle ships.
//
// §15 splits the work this way on purpose: an annotation processor is the only
// thing that can see a plugin's source, and this is the only thing that writes
// the bundle format. The processor extracts, this encodes, and neither
// reimplements the other — which is what keeps build-time and load-time
// validation the same code rather than two opinions.
//
// The intermediate carries no executor ids. They are minted here, once, walking
// the tree in declaration order: an id is an artefact of the tree, and a
// compiler that invented its own would be a second set to keep in step.

// intermediateVersion is the shape this reads. A build tool newer than the
// server is a real situation, and refusing it by number beats failing on a
// field that moved.
const intermediateVersion = 1

// maximumIntermediateSize bounds what a build tool may hand over. The file is a
// few fields per command; anything past this is a mistake or a different file.
const maximumIntermediateSize = 4 << 20

type intermediateFile struct {
	Version  int                `json:"version"`
	Commands []intermediateNode `json:"commands"`
}

type intermediateNode struct {
	Name       string             `json:"name"`
	Argument   bool               `json:"argument"`
	Kind       string             `json:"kind"`
	Permission string             `json:"permission"`
	Runs       bool               `json:"runs"`
	Minimum    *float64           `json:"min"`
	Maximum    *float64           `json:"max"`
	Options    []string           `json:"options"`
	Custom     string             `json:"custom"`
	Children   []intermediateNode `json:"children"`
}

// readCommandTree loads an intermediate and encodes the tree it describes.
func readCommandTree(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read command trees: %w", err)
	}
	if len(data) > maximumIntermediateSize {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, maximumIntermediateSize)
	}
	var file intermediateFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if file.Version != intermediateVersion {
		return nil, fmt.Errorf("%s: version %d is unsupported, this tool reads %d",
			path, file.Version, intermediateVersion)
	}
	if len(file.Commands) == 0 {
		return nil, fmt.Errorf("%s: declares no commands", path)
	}

	var executor command.ExecID
	next := func() command.ExecID {
		executor++
		return executor
	}
	root := command.Root{}
	for _, declared := range file.Commands {
		node, err := convertIntermediate(declared, next)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		root.Children = append(root.Children, node)
	}
	// Encoded through the host's own encoder, which validates first. A tree the
	// server would refuse fails on the machine that has the source.
	return command.EncodeTree(root)
}

func convertIntermediate(node intermediateNode, next func() command.ExecID) (command.Node, error) {
	children := make([]command.Node, 0, len(node.Children))
	for _, child := range node.Children {
		converted, err := convertIntermediate(child, next)
		if err != nil {
			return nil, err
		}
		children = append(children, converted)
	}
	if len(children) == 0 {
		children = nil
	}
	var exec command.ExecID
	if node.Runs {
		exec = next()
	}
	if !node.Argument {
		if node.Kind != "" || node.Options != nil || node.Minimum != nil || node.Maximum != nil {
			return nil, fmt.Errorf("literal %q carries argument fields", node.Name)
		}
		return command.Literal{
			Name: node.Name, Permission: node.Permission, Exec: exec, Children: children,
		}, nil
	}
	if node.Permission != "" {
		return nil, fmt.Errorf("argument %q carries a permission", node.Name)
	}
	kind, err := argumentKind(node.Kind)
	if err != nil {
		return nil, fmt.Errorf("argument %q: %w", node.Name, err)
	}
	argument := command.Argument{
		Name: node.Name, Type: kind, Enum: node.Options,
		CustomType: node.Custom, Exec: exec, Children: children,
	}
	switch kind {
	case command.ArgInteger:
		argument.IntegerMin, argument.IntegerMax = wholeBound(node.Minimum), wholeBound(node.Maximum)
	case command.ArgDecimal:
		argument.DecimalMin, argument.DecimalMax = node.Minimum, node.Maximum
	default:
		if node.Minimum != nil || node.Maximum != nil {
			return nil, fmt.Errorf("argument %q is %s and carries a range", node.Name, node.Kind)
		}
	}
	return argument, nil
}

// argumentKind reads the neutral spelling the intermediate uses.
//
// Named rather than numbered on the wire between two build tools: a number
// would mean the compiler and this program agreeing on an enum order neither
// of them declares.
func argumentKind(kind string) (command.ArgType, error) {
	switch kind {
	case "integer":
		return command.ArgInteger, nil
	case "decimal":
		return command.ArgDecimal, nil
	case "string":
		return command.ArgString, nil
	case "greedy":
		return command.ArgGreedy, nil
	case "player":
		return command.ArgPlayer, nil
	case "block_pos":
		return command.ArgBlockPos, nil
	case "block_state":
		return command.ArgBlockState, nil
	case "item":
		return command.ArgItem, nil
	case "duration":
		return command.ArgDuration, nil
	case "enum":
		return command.ArgEnum, nil
	case "custom":
		return command.ArgCustom, nil
	case "":
		return 0, fmt.Errorf("no kind")
	default:
		return 0, fmt.Errorf("unknown kind %q", kind)
	}
}

func wholeBound(value *float64) *int64 {
	if value == nil {
		return nil
	}
	whole := int64(*value)
	return &whole
}
