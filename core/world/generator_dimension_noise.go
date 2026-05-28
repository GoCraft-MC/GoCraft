package world

import "math"

// dimensionNoise2D and dimensionNoise3D are deterministic value-noise
// samplers shared by the Nether and End generators. They deliberately use the
// world seed directly, so dimension terrain remains stable across restarts and
// is safe to generate concurrently.
func dimensionNoise2D(seed int64, x, z, scale float64, salt uint64) float64 {
	if scale <= 0 {
		return 0
	}
	x /= scale
	z /= scale
	x0, z0 := int(math.Floor(x)), int(math.Floor(z))
	tx, tz := smoothstep(x-float64(x0)), smoothstep(z-float64(z0))
	value := func(ix, iz int) float64 {
		hash := generatedHash(seed^int64(salt), ix, 0, iz)
		return float64(hash>>11)/float64(uint64(1)<<53)*2 - 1
	}
	return lerp(
		lerp(value(x0, z0), value(x0+1, z0), tx),
		lerp(value(x0, z0+1), value(x0+1, z0+1), tx),
		tz,
	)
}

func dimensionNoise3D(seed int64, x, y, z, scaleX, scaleY, scaleZ float64, salt uint64) float64 {
	if scaleX <= 0 || scaleY <= 0 || scaleZ <= 0 {
		return 0
	}
	x, y, z = x/scaleX, y/scaleY, z/scaleZ
	x0, y0, z0 := int(math.Floor(x)), int(math.Floor(y)), int(math.Floor(z))
	tx := smoothstep(x - float64(x0))
	ty := smoothstep(y - float64(y0))
	tz := smoothstep(z - float64(z0))
	value := func(ix, iy, iz int) float64 {
		hash := generatedHash(seed^int64(salt), ix, iy, iz)
		return float64(hash>>11)/float64(uint64(1)<<53)*2 - 1
	}
	x00 := lerp(value(x0, y0, z0), value(x0+1, y0, z0), tx)
	x10 := lerp(value(x0, y0+1, z0), value(x0+1, y0+1, z0), tx)
	x01 := lerp(value(x0, y0, z0+1), value(x0+1, y0, z0+1), tx)
	x11 := lerp(value(x0, y0+1, z0+1), value(x0+1, y0+1, z0+1), tx)
	return lerp(lerp(x00, x10, ty), lerp(x01, x11, ty), tz)
}

func dimensionFractal2D(seed int64, x, z, scale float64, octaves int, salt uint64) float64 {
	value, amplitude, total := 0.0, 1.0, 0.0
	for octave := 0; octave < octaves; octave++ {
		value += dimensionNoise2D(seed, x, z, scale, salt+uint64(octave)*0x9e3779b97f4a7c15) * amplitude
		total += amplitude
		amplitude *= 0.5
		scale *= 0.5
	}
	if total == 0 {
		return 0
	}
	return value / total
}

func dimensionFractal3D(seed int64, x, y, z, scaleX, scaleY, scaleZ float64, octaves int, salt uint64) float64 {
	value, amplitude, total := 0.0, 1.0, 0.0
	for octave := 0; octave < octaves; octave++ {
		value += dimensionNoise3D(seed, x, y, z, scaleX, scaleY, scaleZ, salt+uint64(octave)*0x9e3779b97f4a7c15) * amplitude
		total += amplitude
		amplitude *= 0.5
		scaleX *= 0.5
		scaleY *= 0.5
		scaleZ *= 0.5
	}
	if total == 0 {
		return 0
	}
	return value / total
}
