package world

import "testing"

func TestCarvePumpkin(t *testing.T) {
	carved, ok := CarvePumpkin(Block{Namespace: "minecraft", Name: "pumpkin"}, "west")
	if !ok || carved.ResourceLocation() != "minecraft:carved_pumpkin" || carved.Properties["facing"] != "west" {
		t.Fatalf("carved pumpkin = %+v, ok=%v", carved, ok)
	}
	if _, ok := CarvePumpkin(Block{Namespace: "minecraft", Name: "melon"}, "west"); ok {
		t.Fatal("melon was accepted as a pumpkin")
	}
	if _, ok := CarvePumpkin(Block{Namespace: "minecraft", Name: "pumpkin"}, "up"); ok {
		t.Fatal("vertical pumpkin facing was accepted")
	}
}
