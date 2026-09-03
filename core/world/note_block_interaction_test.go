package world

import "testing"

func TestTuneNoteBlockAdvancesAndWraps(t *testing.T) {
	block := Block{Namespace: "minecraft", Name: "note_block", Properties: map[string]string{"note": "23", "instrument": "harp"}}
	block, ok := TuneNoteBlock(block)
	if !ok || block.Properties["note"] != "24" || block.Properties["instrument"] != "harp" {
		t.Fatalf("first tuning = %+v, ok=%v", block, ok)
	}
	block, ok = TuneNoteBlock(block)
	if !ok || block.Properties["note"] != "0" {
		t.Fatalf("wrapped tuning = %+v, ok=%v", block, ok)
	}
	if _, ok := TuneNoteBlock(Block{Namespace: "minecraft", Name: "stone"}); ok {
		t.Fatal("stone was accepted as a note block")
	}
}
