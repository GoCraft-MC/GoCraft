package world

import "testing"

func TestRailsConnectIntoCurve(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	rail := Block{Namespace: "minecraft", Name: "rail", Properties: map[string]string{"shape": "north_south"}}
	w.SetBlock(0, 64, 0, rail)
	w.SetBlock(1, 64, 0, rail)
	w.SetBlock(1, 64, 1, rail)
	if got := w.GetBlock(1, 64, 0).Properties["shape"]; got != "south_west" {
		t.Fatalf("corner shape = %q, want south_west", got)
	}
}

func TestRailConnectsUpSlope(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	rail := Block{Namespace: "minecraft", Name: "rail", Properties: map[string]string{"shape": "north_south"}}
	w.SetBlock(0, 64, 0, rail)
	w.SetBlock(1, 65, 0, rail)
	if got := w.GetBlock(0, 64, 0).Properties["shape"]; got != "ascending_east" {
		t.Fatalf("lower rail shape = %q, want ascending_east", got)
	}
}
