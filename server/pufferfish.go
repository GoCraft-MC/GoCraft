package server

import corentity "GoCraft/core/entity"

// updatePufferState mirrors vanilla's InflateGoal counters. Inflation changes
// immediately to half-puffed, reaches full after 40 ticks, then deflates in
// two delayed stages after a threat leaves.
func updatePufferState(fish *corentity.Entity, threat bool) bool {
	if fish == nil || fish.Type != corentity.TypePufferfish {
		return false
	}
	before := fish.PufferState
	if threat {
		fish.PufferDeflateTicks = 0
		if fish.PufferInflateTicks == 0 {
			fish.PufferInflateTicks = 1
		}
		if fish.PufferState == 0 {
			fish.PufferState = 1
		} else if fish.PufferInflateTicks > 40 && fish.PufferState == 1 {
			fish.PufferState = 2
		}
		fish.PufferInflateTicks++
	} else {
		fish.PufferInflateTicks = 0
		if fish.PufferDeflateTicks == 0 {
			fish.PufferDeflateTicks = 1
		}
		if fish.PufferState == 2 && fish.PufferDeflateTicks > 60 {
			fish.PufferState = 1
		} else if fish.PufferState == 1 && fish.PufferDeflateTicks > 100 {
			fish.PufferState = 0
		}
		fish.PufferDeflateTicks++
	}
	return before != fish.PufferState
}
