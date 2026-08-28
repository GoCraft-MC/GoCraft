package server

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestSpreadLoadedPositionsAreSeparated(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.Chunk(-1, 0)
	w.Chunk(0, 0)
	values := []float64{0.25, 0.5, 0.75, 0.5}
	index := 0
	positions, err := spreadLoadedPositions(w, 0, 0, 4, 8, 2, func() float64 {
		value := values[index]
		index++
		return value
	})
	if err != nil {
		t.Fatal(err)
	}
	if positions[1].X-positions[0].X < 4 {
		t.Fatalf("positions are not separated: %v", positions)
	}
}
