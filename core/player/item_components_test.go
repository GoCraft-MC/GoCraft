package player

import "testing"

func TestItemComponentsAreCanonicalAndComparable(t *testing.T) {
	first := ItemStack{ItemID: "minecraft:potion", Count: 1}
	if err := first.SetComponent("potion_contents", map[string]any{"custom_effects": []any{}, "potion": "minecraft:healing"}); err != nil {
		t.Fatal(err)
	}
	second := ItemStack{ItemID: "minecraft:potion", Count: 1, Components: `{"minecraft:potion_contents":{"potion":"minecraft:healing","custom_effects":[]}}`}
	if !first.SameItem(second) {
		t.Fatalf("semantically equal components did not match:\n%s\n%s", first.Components, second.Components)
	}
	var contents struct {
		Potion string `json:"potion"`
	}
	if !first.Component("minecraft:potion_contents", &contents) || contents.Potion != "minecraft:healing" {
		t.Fatalf("decoded potion contents = %#v", contents)
	}
}

func TestItemComponentsPreventLossyMerges(t *testing.T) {
	healing := ItemStack{ItemID: "minecraft:potion", Count: 1}
	poison := ItemStack{ItemID: "minecraft:potion", Count: 1}
	if err := healing.SetComponent("potion_contents", map[string]string{"potion": "minecraft:healing"}); err != nil {
		t.Fatal(err)
	}
	if err := poison.SetComponent("potion_contents", map[string]string{"potion": "minecraft:poison"}); err != nil {
		t.Fatal(err)
	}
	if healing.SameItem(poison) {
		t.Fatal("different potion contents merged")
	}
}

func TestSetItemComponentIsAtomic(t *testing.T) {
	stack := ItemStack{ItemID: "minecraft:stone", Count: 1, Components: `{"broken"`}
	before := stack
	if err := stack.SetComponent("lore", []string{"Line"}); err == nil {
		t.Fatal("component update accepted malformed existing data")
	}
	if stack != before {
		t.Fatalf("failed update changed stack from %#v to %#v", before, stack)
	}
}
