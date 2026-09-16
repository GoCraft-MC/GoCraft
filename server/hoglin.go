package server

import (
	corentity "GoCraft/core/entity"
)

// hoglinConversionTime mirrors Hoglin.CONVERSION_TIME: a hoglin left outside the
// Nether zombifies into a zoglin after this many ticks.
const hoglinConversionTime = 300

// tickHoglinConversion ports Hoglin.customServerAiStep: while a hoglin is outside
// the Nether it accumulates timeInOverworld, and once that exceeds
// CONVERSION_TIME (300) it converts into a zoglin. Returning to the Nether resets
// the timer. Called every tick (not on the staggered hostile-AI cadence) so the
// wall-clock conversion time matches vanilla.
func (s *Server) tickHoglinConversion(e *corentity.Entity) {
	if e == nil || e.Dead {
		return
	}
	state := parityState(e)
	// PIGLINS_ZOMBIFY is false only in the Nether; every other dimension converts.
	if s.simulationDimension == dimensionNether {
		state.timeInOverworld = 0
		return
	}
	state.timeInOverworld++
	if state.timeInOverworld > hoglinConversionTime {
		s.lightningConvert(e, corentity.TypeZoglin, nil)
	}
}
