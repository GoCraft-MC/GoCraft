package goplugin

import (
	"sort"

	abi "GoCraft/abi/v1"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func playerValue(value *player.Player) abi.Value {
	if value == nil {
		return abi.List()
	}
	edition := "java"
	if value.Edition == player.ClientEditionBedrock {
		edition = "bedrock"
	}
	return abi.List(abi.Bytes(value.UUID[:]), abi.String(value.Username), abi.String(edition))
}

func positionValue(value spatial.BlockPos) abi.Value {
	return abi.List(
		abi.Int64(int64(value.X)),
		abi.Int64(int64(value.Y)),
		abi.Int64(int64(value.Z)),
	)
}

func blockValue(value coreworld.Block) abi.Value {
	keys := make([]string, 0, len(value.Properties))
	for key := range value.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	properties := make([]abi.Value, 0, len(keys))
	for _, key := range keys {
		properties = append(properties,
			abi.List(abi.String(key), abi.String(value.Properties[key])))
	}
	return abi.List(abi.String(value.ResourceLocation()), abi.List(properties...))
}
