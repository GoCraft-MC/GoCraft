package world

import "testing"

func TestComposterLifecycleOperations(t *testing.T) {
	composter := Block{Namespace: "minecraft", Name: "composter", Properties: map[string]string{"level": "0"}}
	updated, consumed, schedule := AddToComposter(composter, "minecraft:wheat_seeds", 1, 64, 2, 0)
	if !consumed || schedule || updated.Properties["level"] != "1" {
		t.Fatalf("first compost = %+v consumed=%v schedule=%v", updated, consumed, schedule)
	}
	full := Block{Namespace: "minecraft", Name: "composter", Properties: map[string]string{"level": "6"}}
	updated, consumed, schedule = AddToComposter(full, "minecraft:cake", 1, 64, 2, 0)
	if !consumed || !schedule || updated.Properties["level"] != "7" {
		t.Fatalf("final compost = %+v consumed=%v schedule=%v", updated, consumed, schedule)
	}
	if _, consumed, _ = AddToComposter(updated, "minecraft:cake", 1, 64, 2, 0); consumed {
		t.Fatal("level-seven composter consumed another item")
	}
	ready := Block{Namespace: "minecraft", Name: "composter", Properties: map[string]string{"level": "8"}}
	empty, ok := EmptyComposter(ready)
	if !ok || empty.Properties["level"] != "0" {
		t.Fatalf("emptied composter = %+v, ok=%v", empty, ok)
	}
}
