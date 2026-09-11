package world

import "testing"

func TestLecternComparatorTracksCurrentPage(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	lectern := Block{Namespace: "minecraft", Name: "lectern", Properties: map[string]string{
		"has_book": "true", "powered": "false", "facing": "north",
	}}
	w.SetBlock(0, 64, -1, lectern)
	w.SetContainerItems(0, 64, -1, "minecraft:lectern", []ContainerItem{{
		Slot: 0, ItemID: "minecraft:written_book", Count: 1,
		Components: `{"minecraft:written_book_content":{"pages":[0,1,2,3,4,5,6,7,8,9]}}`,
	}})
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "comparator", Properties: map[string]string{
		"facing": "north", "mode": "compare", "powered": "false",
	}})

	w.Redstone.FlushUpdates()
	if got := w.Redstone.PowerAt(0, 64, 0); got != 1 {
		t.Fatalf("first-page comparator output = %d, want 1", got)
	}
	if !w.SetLecternPage(0, 64, -1, 4, 10) {
		t.Fatal("valid lectern page update was rejected")
	}
	w.Redstone.FlushUpdates()
	if got := w.Redstone.PowerAt(0, 64, 0); got != 7 {
		t.Fatalf("middle-page comparator output = %d, want 7", got)
	}
	if !w.SetLecternPage(0, 64, -1, 99, 10) {
		t.Fatal("clamped lectern page update was rejected")
	}
	w.Redstone.FlushUpdates()
	if got := w.Redstone.PowerAt(0, 64, 0); got != 15 {
		t.Fatalf("last-page comparator output = %d, want 15", got)
	}
	if got := w.GetBlockEntity(0, 64, -1); got.LecternPage != 9 || got.LecternPageCount != 10 {
		t.Fatalf("stored lectern page = %d/%d, want 9/10", got.LecternPage, got.LecternPageCount)
	}
}

func TestLecternPageUpdateRejectsInvalidState(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	if w.SetLecternPage(0, 64, 0, 0, 1) {
		t.Fatal("page update without a lectern was accepted")
	}
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "lectern", Properties: map[string]string{"has_book": "true"}})
	if w.SetLecternPage(0, 64, 0, 0, 0) {
		t.Fatal("page update with an invalid page count was accepted")
	}
}
