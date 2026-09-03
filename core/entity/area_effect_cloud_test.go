package entity

import (
	"testing"

	"GoCraft/core/player"
)

func TestNewAreaEffectCloudUsesVanillaDefaults(t *testing.T) {
	stack := player.ItemStack{ItemID: "minecraft:lingering_potion", Count: 3}
	cloud := NewAreaEffectCloud(12, [16]byte{1}, 1, 2, 3, stack)
	if cloud.Type != TypeAreaEffectCloud || cloud.ProjectileItem.Count != 1 {
		t.Fatalf("cloud identity = %+v", cloud)
	}
	if cloud.CloudRadius != 3 || cloud.CloudRadiusGrowth != -0.005 || cloud.CloudRadiusOnUse != -0.5 {
		t.Fatalf("cloud radius settings = %v, %v, %v", cloud.CloudRadius, cloud.CloudRadiusGrowth, cloud.CloudRadiusOnUse)
	}
	if cloud.CloudDurationTicks != 600 || cloud.CloudReapplicationDelay != 40 || cloud.CloudTargets == nil {
		t.Fatalf("cloud lifecycle settings = %+v", cloud)
	}
}
