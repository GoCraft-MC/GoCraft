package handler

import (
	"fmt"
	"math"
	"strconv"
)

func cmdRotate(ctx CommandContext) error {
	if len(ctx.Args) != 3 {
		return fmt.Errorf("usage: /rotate <player> <yaw> <pitch>")
	}
	target := findCanonicalPlayer(ctx, ctx.Args[0])
	if target == nil {
		return fmt.Errorf("player not found: %s", ctx.Args[0])
	}
	yaw, err := strconv.ParseFloat(ctx.Args[1], 32)
	if err != nil || math.IsNaN(yaw) || math.IsInf(yaw, 0) {
		return fmt.Errorf("invalid yaw: %s", ctx.Args[1])
	}
	pitch, err := strconv.ParseFloat(ctx.Args[2], 32)
	if err != nil || pitch < -90 || pitch > 90 || math.IsNaN(pitch) {
		return fmt.Errorf("pitch must be between -90 and 90")
	}
	target.Rotation.Yaw = float32(math.Mod(yaw, 360))
	target.Rotation.Pitch = float32(pitch)
	if ctx.TeleportPlayer == nil {
		return fmt.Errorf("teleport service is unavailable")
	}
	position := target.Position
	if err := ctx.TeleportPlayer(target, position.X, position.Y, position.Z); err != nil {
		return err
	}
	return sendCommandMessage(ctx, fmt.Sprintf("Rotated %s to %.1f %.1f", target.Username, yaw, pitch))
}
