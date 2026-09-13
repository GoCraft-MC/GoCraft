package world

// ChunkCoordsForX converts an absolute X block coordinate to its chunk X.
// It is the single-axis counterpart to ChunkCoordsFor and preserves correct
// floor division for negative coordinates.
func ChunkCoordsForX(x int) int32 {
	cx, _ := ChunkCoordsFor(x, 0)
	return cx
}

// ChunkCoordsForZ converts an absolute Z block coordinate to its chunk Z.
// It is the single-axis counterpart to ChunkCoordsFor and preserves correct
// floor division for negative coordinates.
func ChunkCoordsForZ(z int) int32 {
	_, cz := ChunkCoordsFor(0, z)
	return cz
}
