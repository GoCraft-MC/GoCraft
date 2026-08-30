package world

import "testing"

func TestBellTracksRedstoneRisingAndFallingEdges(t *testing.T) {
	w := attachmentTestWorld(t)
	bell := Block{Namespace: "minecraft", Name: "bell", Properties: map[string]string{
		"attachment": "floor", "facing": "north", "powered": "false",
	}}
	w.SetBlock(0, 64, 0, bell)
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "redstone_block"})
	result := w.Redstone.FlushUpdates()
	if w.GetBlock(0, 64, 0).Properties["powered"] != "true" || len(result.PoweredLoads) == 0 {
		t.Fatalf("powered bell=%+v result=%+v", w.GetBlock(0, 64, 0), result)
	}
	w.SetBlock(1, 64, 0, Air)
	result = w.Redstone.FlushUpdates()
	if w.GetBlock(0, 64, 0).Properties["powered"] != "false" || len(result.UnpoweredLoads) == 0 {
		t.Fatalf("unpowered bell=%+v result=%+v", w.GetBlock(0, 64, 0), result)
	}
}
