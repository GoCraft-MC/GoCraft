package world

// legacyRandom implements the 48-bit java.util.Random stream used by
// Minecraft's legacy structure placements.
type legacyRandom struct {
	seed uint64
}

func newLegacyRandom(seed int64) *legacyRandom {
	return &legacyRandom{seed: (uint64(seed) ^ 0x5deece66d) & ((1 << 48) - 1)}
}

func (r *legacyRandom) next(bits uint) uint32 {
	r.seed = (r.seed*0x5deece66d + 0xb) & ((1 << 48) - 1)
	return uint32(r.seed >> (48 - bits))
}

func (r *legacyRandom) nextDouble() float64 {
	high := uint64(r.next(26))
	low := uint64(r.next(27))
	return float64(high<<27|low) / float64(uint64(1)<<53)
}

func (r *legacyRandom) nextInt(bound int) int {
	if bound <= 0 {
		panic("legacyRandom.nextInt requires a positive bound")
	}
	if bound&(bound-1) == 0 {
		return int((int64(bound) * int64(r.next(31))) >> 31)
	}
	for {
		bits := int64(r.next(31))
		value := bits % int64(bound)
		if bits-value+int64(bound-1) < 1<<31 {
			return int(value)
		}
	}
}

func (r *legacyRandom) skipLong() {
	r.next(32)
	r.next(32)
}
