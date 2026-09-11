package handler

import (
	"fmt"
	"math"

	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
)

// FilterPluginTeleport applies the command-teleport event once before the
// edition's movement adapter changes state. Login/respawn corrections do not
// pass through it and cannot be cancelled as player teleports.
func FilterPluginTeleport(bus *coreplugin.Bus, p *player.Player, x, y, z *float64) error {
	if p == nil {
		return fmt.Errorf("target player is unavailable")
	}
	if bus != nil && !bus.EmitPlayerTeleport(p, p.Position.X, p.Position.Y, p.Position.Z, x, y, z, int64(p.Dimension)) {
		return fmt.Errorf("teleport cancelled by a plugin")
	}
	for _, coordinate := range []float64{*x, *y, *z} {
		if math.IsNaN(coordinate) || math.IsInf(coordinate, 0) || math.Abs(coordinate) > 30_000_000 {
			return fmt.Errorf("invalid teleport destination")
		}
	}
	return nil
}

func (d *Dispatcher) eventTeleport(p *player.Player, teleport func(float64, float64, float64) error) func(float64, float64, float64) error {
	if teleport == nil {
		return nil
	}
	return func(x, y, z float64) error {
		if err := FilterPluginTeleport(d.EventBus(), p, &x, &y, &z); err != nil {
			return err
		}
		return teleport(x, y, z)
	}
}
