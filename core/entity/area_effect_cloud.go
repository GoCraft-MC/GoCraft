package entity

import "GoCraft/core/player"

// NewAreaEffectCloud creates the stationary cloud left by a lingering potion.
func NewAreaEffectCloud(id int32, uuid [16]byte, x, y, z float64, potion player.ItemStack) *Entity {
	cloud := New(id, uuid, TypeAreaEffectCloud, x, y, z)
	cloud.ProjectileItem = potion
	cloud.ProjectileItem.Count = 1
	cloud.CloudRadius = 3
	cloud.CloudRadiusGrowth = -0.005
	cloud.CloudRadiusOnUse = -0.5
	cloud.CloudDurationTicks = 600
	cloud.CloudReapplicationDelay = 40
	cloud.CloudTargets = make(map[int32]int64)
	return cloud
}
