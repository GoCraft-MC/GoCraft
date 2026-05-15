// Package spatial provides edition-agnostic 3-D position and rotation types
// shared by the Java and Bedrock adapters and the GoCraft game core.
package spatial

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Vec3 is a floating-point 3-D position used for entity positions and movement.
type Vec3 struct {
	X, Y, Z float64
}

// Add returns v + other.
func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// Sub returns v − other.
func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{v.X - other.X, v.Y - other.Y, v.Z - other.Z}
}

// Distance returns the Euclidean distance to other.
func (v Vec3) Distance(other Vec3) float64 {
	dx, dy, dz := v.X-other.X, v.Y-other.Y, v.Z-other.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// ToBlock returns the block position containing v (floor of each axis).
func (v Vec3) ToBlock() BlockPos {
	return BlockPos{
		X: int32(math.Floor(v.X)),
		Y: int32(math.Floor(v.Y)),
		Z: int32(math.Floor(v.Z)),
	}
}

// String returns a human-readable representation.
func (v Vec3) String() string {
	return fmt.Sprintf("(%.3f, %.3f, %.3f)", v.X, v.Y, v.Z)
}

// BlockPos is a signed integer block position.
type BlockPos struct {
	X, Y, Z int32
}

// Encode packs the block position into the 64-bit Minecraft protocol format.
//
// Layout (Java Edition 1.14+):
//
//	bits 63-38: X (26 bits, signed)
//	bits 37-12: Z (26 bits, signed)
//	bits 11-0:  Y (12 bits, signed)
func (b BlockPos) Encode() int64 {
	return ((int64(b.X) & 0x3FFFFFF) << 38) |
		((int64(b.Z) & 0x3FFFFFF) << 12) |
		(int64(b.Y) & 0xFFF)
}

// EncodeBytes writes the position as 8 big-endian bytes (for protocol packets).
func (b BlockPos) EncodeBytes() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(b.Encode()))
	return buf
}

// DecodeBlockPos decodes the 64-bit Minecraft packed block position.
func DecodeBlockPos(encoded int64) BlockPos {
	x := int32(encoded >> 38)
	z := int32(encoded << 26 >> 38) // sign-extend the 26-bit Z field
	y := int32(encoded << 52 >> 52) // sign-extend the 12-bit Y field
	return BlockPos{X: x, Y: y, Z: z}
}

// String returns a human-readable representation.
func (b BlockPos) String() string {
	return fmt.Sprintf("[%d, %d, %d]", b.X, b.Y, b.Z)
}

// Rotation holds the yaw and pitch angles in degrees (as used by Minecraft).
// Yaw is the horizontal rotation (0 = south, clockwise).
// Pitch is the vertical angle (positive = looking down).
type Rotation struct {
	Yaw   float32
	Pitch float32
}

// DefaultSpawnPos is the world-origin spawn used when no spawn has been set.
var DefaultSpawnPos = Vec3{X: 0, Y: 64, Z: 0}
