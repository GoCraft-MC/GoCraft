package server

import (
	"testing"

	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestSculkSensorDetectsVibrationAndCyclesPhases(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "sculk_sensor", Properties: map[string]string{
		"sculk_sensor_phase": "inactive", "power": "0", "waterlogged": "false",
	}})
	s := &Server{world: w, sessions: session.NewManager(), worldAge: 1}
	// A footstep on top of the sensor has distance zero and must still trigger.
	w.EmitVibration(0, 64, 0)

	s.tickBlockPhysics()
	active := w.GetBlock(0, 64, 0)
	if active.Properties["sculk_sensor_phase"] != "active" || active.Properties["power"] != "15" {
		t.Fatalf("activated sensor = %+v", active.Properties)
	}
	s.worldAge = 31
	s.tickBlockPhysics()
	if phase := w.GetBlock(0, 64, 0).Properties["sculk_sensor_phase"]; phase != "cooldown" {
		t.Fatalf("sensor phase after active period = %q", phase)
	}
	s.worldAge = 41
	s.tickBlockPhysics()
	inactive := w.GetBlock(0, 64, 0)
	if inactive.Properties["sculk_sensor_phase"] != "inactive" || inactive.Properties["power"] != "0" {
		t.Fatalf("inactive sensor = %+v", inactive.Properties)
	}
}
