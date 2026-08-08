package bedrock

import (
	"strconv"
	"testing"

	bedrockworld "GoCraft/bedrock/world"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

func TestCreativeCatalogueIsPopulated(t *testing.T) {
	l := &Listener{}
	l.initCreativeContent()
	if len(l.creativeGroups) == 0 || len(l.creativeItems) < 1000 {
		t.Fatalf("creative catalogue = %d groups/%d items", len(l.creativeGroups), len(l.creativeItems))
	}
	for _, item := range l.creativeNames {
		if item.name == "minecraft:oak_log" {
			return
		}
	}
	t.Fatal("creative catalogue does not contain minecraft:oak_log")
}

func TestCanonicalLightItemUsesSelectedBedrockBlockState(t *testing.T) {
	encoder := bedrockworld.NewEncoder()
	l := &Listener{encoder: encoder}
	instance := l.itemInstance(player.ItemStack{ItemID: "minecraft:light", Count: 1, Damage: 11}, 7)
	wantBlock := encoder.BlockNetworkID(coreworld.Block{
		Namespace: "minecraft", Name: "light", Properties: map[string]string{"level": "11"},
	})
	if instance.Stack.NetworkID == 0 || instance.Stack.BlockRuntimeID != int32(wantBlock) {
		t.Fatalf("light item = network %d block %d, want nonzero/%d", instance.Stack.NetworkID, instance.Stack.BlockRuntimeID, wantBlock)
	}
	if instance.Stack.NBTData != nil {
		t.Fatalf("light level was encoded as durability NBT: %v", instance.Stack.NBTData)
	}
}

func TestBedrockLightCreativeItemsUseCanonicalJavaIdentity(t *testing.T) {
	for level := int16(0); level <= 15; level++ {
		name, meta := canonicalCreativeIdentity("minecraft:light_block_"+strconv.Itoa(int(level)), 0)
		if name != "minecraft:light" || meta != level {
			t.Fatalf("level %d canonical identity = %q/%d", level, name, meta)
		}
	}
}
