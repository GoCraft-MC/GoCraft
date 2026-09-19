package main

import (
	"GoCraft/internal/gamedata/block"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	flatbuffers "github.com/google/flatbuffers/go"
)

var manifest string

func isMetaKey(k string) bool {
	switch k {
	case "_comment", "_format", "_source", "_coverage", "_gocraft_version":
		return true
	}
	return false
}

type blockStateEntry struct {
	ID         int32             `json:"id"`
	Default    bool              `json:"default"`
	Properties map[string]string `json:"properties"`
}

type blockDefinition struct {
	Type               string            `json:"type"`
	BlockSetType       string            `json:"block_set_type"`
	Properties         map[string]string `json:"properties"`
	TicksToStayPressed int32             `json:"ticks_to_stay_pressed"`
}

type blockEntry struct {
	Properties map[string][]string `json:"properties"`
	States     []blockStateEntry   `json:"states"`
	Definition blockDefinition     `json:"definition"`
}

func main() {
	flag.StringVar(&manifest, "manifest", "", "The manifest to parse")
	flag.Parse()

	data, err := os.ReadFile(manifest)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		panic(fmt.Sprintf("gamedata: parsing blocks.json: %v", err))
	}

	builder := flatbuffers.NewBuilder(6 * 1024 * 1024)
	var blockOffsets []flatbuffers.UOffsetT

	for blockName, rawVal := range raw {
		if isMetaKey(blockName) {
			continue
		}

		var entry blockEntry
		if err := json.Unmarshal(rawVal, &entry); err != nil {
			slog.Warn("gamedata: skipping malformed block entry",
				"block", blockName, "err", err)
			continue
		}

		var stateOffsets []flatbuffers.UOffsetT
		for _, s := range entry.States {
			var statePropOffsets []flatbuffers.UOffsetT
			for k, v := range s.Properties {
				keyOffset := builder.CreateString(k)
				valOffset := builder.CreateString(v)

				block.StatePropertyStart(builder)
				block.StatePropertyAddKey(builder, keyOffset)
				block.StatePropertyAddValue(builder, valOffset)
				statePropOffsets = append(statePropOffsets, block.StatePropertyEnd(builder))
			}

			block.BlockStateStartPropertiesVector(builder, len(statePropOffsets))
			for idx := len(statePropOffsets) - 1; idx >= 0; idx-- { // Backwards array insertion is mandatory
				builder.PrependUOffsetT(statePropOffsets[idx])
			}
			propsVectorOffset := builder.EndVector(len(statePropOffsets))

			block.BlockStateStart(builder)
			block.BlockStateAddId(builder, s.ID)
			block.BlockStateAddIsDefault(builder, s.Default)
			block.BlockStateAddProperties(builder, propsVectorOffset)
			stateOffsets = append(stateOffsets, block.BlockStateEnd(builder))
		}

		block.BlockStartStatesVector(builder, len(stateOffsets))
		for idx := len(stateOffsets) - 1; idx >= 0; idx-- {
			builder.PrependUOffsetT(stateOffsets[idx])
		}
		statesVectorOffset := builder.EndVector(len(stateOffsets))

		var propOptOffsets []flatbuffers.UOffsetT
		for k, vals := range entry.Properties {
			var valOffsets []flatbuffers.UOffsetT
			for _, v := range vals {
				valOffsets = append(valOffsets, builder.CreateString(v))
			}

			block.PropertyOptionStartValuesVector(builder, len(valOffsets))
			for idx := len(valOffsets) - 1; idx >= 0; idx-- {
				builder.PrependUOffsetT(valOffsets[idx])
			}
			valsVectorOffset := builder.EndVector(len(valOffsets))

			keyOffset := builder.CreateString(k)
			block.PropertyOptionStart(builder)
			block.PropertyOptionAddKey(builder, keyOffset)
			block.PropertyOptionAddValues(builder, valsVectorOffset)
			propOptOffsets = append(propOptOffsets, block.PropertyOptionEnd(builder))
		}

		block.BlockStartPropertyOptionsVector(builder, len(propOptOffsets))
		for idx := len(propOptOffsets) - 1; idx >= 0; idx-- {
			builder.PrependUOffsetT(propOptOffsets[idx])
		}
		propOptionsVectorOffset := builder.EndVector(len(propOptOffsets))

		defTypeOffset := builder.CreateString(entry.Definition.Type)
		defSetOffset := builder.CreateString(entry.Definition.BlockSetType)

		block.BlockDefinitionStart(builder)
		block.BlockDefinitionAddType(builder, defTypeOffset)
		block.BlockDefinitionAddBlockSetType(builder, defSetOffset)
		block.BlockDefinitionAddTicksToStayPressed(builder, entry.Definition.TicksToStayPressed)
		definitionOffset := block.BlockDefinitionEnd(builder)

		nameOffset := builder.CreateString(blockName)

		block.BlockStart(builder)
		block.BlockAddName(builder, nameOffset)
		block.BlockAddDefinition(builder, definitionOffset)
		block.BlockAddPropertyOptions(builder, propOptionsVectorOffset)
		block.BlockAddStates(builder, statesVectorOffset)
		blockOffsets = append(blockOffsets, block.BlockEnd(builder))
	}

	block.BlockRegistryStartBlocksVector(builder, len(blockOffsets))
	for idx := len(blockOffsets) - 1; idx >= 0; idx-- {
		builder.PrependUOffsetT(blockOffsets[idx])
	}
	blocksVectorOffset := builder.EndVector(len(blockOffsets))

	block.BlockRegistryStart(builder)
	block.BlockRegistryAddBlocks(builder, blocksVectorOffset)
	rootRegistryOffset := block.BlockRegistryEnd(builder)

	builder.Finish(rootRegistryOffset)
	binaryData := builder.FinishedBytes()

	outPath := "blocks.bin"
	if err := os.WriteFile(outPath, binaryData, 0644); err != nil {
		slog.Error("failed to write binary file", "err", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully compiled FlatBuffers binary: %s (%d bytes)\n", outPath, len(binaryData))
}

func blockStateKey(blockName string, props map[string]string) string {
	if len(props) == 0 {
		return blockName
	}
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sortStrings(keys)

	out := []byte(blockName)
	out = append(out, '[')
	for i, k := range keys {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, k...)
		out = append(out, '=')
		out = append(out, props[k]...)
	}
	out = append(out, ']')
	return string(out)
}

// sortStrings sorts a slice of strings in-place (insertion sort — fine for the
// small property counts found in Minecraft block states).
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		key := ss[i]
		j := i - 1
		for j >= 0 && ss[j] > key {
			ss[j+1] = ss[j]
			j--
		}
		ss[j+1] = key
	}
}
