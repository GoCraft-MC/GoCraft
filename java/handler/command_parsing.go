package handler

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	coreworld "GoCraft/core/world"
	javaworld "GoCraft/java/world"
)

func ParseCommandCoordinate(value string, origin float64) (int, error) {
	if strings.HasPrefix(value, "~") {
		offset := 0.0
		var err error
		if len(value) > 1 {
			offset, err = strconv.ParseFloat(value[1:], 64)
		}
		if err != nil {
			return 0, fmt.Errorf("invalid coordinate %q", value)
		}
		return int(math.Floor(origin + offset)), nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid coordinate %q", value)
	}
	return parsed, nil
}

func parseCommandBlock(value string) (coreworld.Block, error) {
	name, properties := value, map[string]string(nil)
	if open := strings.IndexByte(value, '['); open >= 0 {
		if !strings.HasSuffix(value, "]") {
			return coreworld.Block{}, fmt.Errorf("invalid block state %q", value)
		}
		name, properties = value[:open], make(map[string]string)
		for _, pair := range strings.Split(value[open+1:len(value)-1], ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return coreworld.Block{}, fmt.Errorf("invalid block property %q", pair)
			}
			properties[parts[0]] = parts[1]
		}
	}
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	parts := strings.SplitN(name, ":", 2)
	block := coreworld.Block{Namespace: parts[0], Name: parts[1], Properties: properties}
	if block.ResourceLocation() != "minecraft:air" && javaworld.StateID(block) == 0 {
		return coreworld.Block{}, fmt.Errorf("unknown block state: %s", value)
	}
	return block, nil
}
