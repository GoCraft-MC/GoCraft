package handler

import (
	"testing"

	coreworld "GoCraft/core/world"
	javaworld "GoCraft/java/world"
)

func TestJavaFencesConnectAndDisconnectAfterNeighbourChanges(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	fence := coreworld.Block{Namespace: "minecraft", Name: "oak_fence"}
	w.SetBlock(0, 64, 0, fence)
	w.SetBlock(1, 64, 0, fence)
	refreshJavaConnectedBlocks(1, 64, 0, w, nil)

	left, right := w.GetBlock(0, 64, 0), w.GetBlock(1, 64, 0)
	if left.Properties["east"] != "true" || right.Properties["west"] != "true" {
		t.Fatalf("fence connection states left=%v right=%v", left.Properties, right.Properties)
	}
	if javaworld.StateID(left) == 0 || javaworld.StateID(right) == 0 {
		t.Fatal("connected fences did not resolve to Java state IDs")
	}

	w.SetBlock(1, 64, 0, coreworld.Air)
	refreshJavaConnectedBlocks(1, 64, 0, w, nil)
	if got := w.GetBlock(0, 64, 0).Properties["east"]; got != "false" {
		t.Fatalf("fence east after break = %q, want false", got)
	}
}

func TestJavaIronBarsConnectToEachOther(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	bars := coreworld.Block{Namespace: "minecraft", Name: "iron_bars"}
	w.SetBlock(0, 64, 0, bars)
	w.SetBlock(0, 64, 1, bars)
	refreshJavaConnectedBlocks(0, 64, 1, w, nil)
	if w.GetBlock(0, 64, 0).Properties["south"] != "true" || w.GetBlock(0, 64, 1).Properties["north"] != "true" {
		t.Fatalf("iron bars did not connect: first=%v second=%v", w.GetBlock(0, 64, 0).Properties, w.GetBlock(0, 64, 1).Properties)
	}
}
