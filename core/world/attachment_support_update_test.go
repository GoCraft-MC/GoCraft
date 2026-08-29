package world

import "testing"

func TestAttachmentSupportUpdateRetainsRemovedBlock(t *testing.T) {
	w := attachmentTestWorld(t)
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	sign := Block{Namespace: "minecraft", Name: "oak_wall_sign", Properties: map[string]string{"facing": "east"}}
	w.SetBlock(1, 64, 0, sign)
	w.SetBlock(0, 64, 0, Air)

	updates := w.ApplyAttachmentSupportUpdatesAround(0, 64, 0)
	if len(updates) != 1 {
		t.Fatalf("updates=%+v, want one removal", updates)
	}
	update := updates[0]
	if !update.Removed || update.Previous.ResourceLocation() != "minecraft:oak_wall_sign" || !update.Change.Block.IsAir() {
		t.Fatalf("update=%+v", update)
	}
}
