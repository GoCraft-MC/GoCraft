package anvil

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLevelMetadata(t *testing.T) {
	worldDir := t.TempDir()
	file, err := os.Create(filepath.Join(worldDir, "level.dat"))
	if err != nil {
		t.Fatal(err)
	}
	compressed := gzip.NewWriter(file)
	WriteRootCompound(compressed, map[string]Tag{
		"Data": {typ: tagCompound, compound: map[string]Tag{
			"DataVersion":   {typ: tagInt, intV: 4189},
			"LevelName":     {typ: tagString, strV: "fixture"},
			"SpawnX":        {typ: tagInt, intV: 12},
			"SpawnY":        {typ: tagInt, intV: 81},
			"SpawnZ":        {typ: tagInt, intV: -7},
			"GameType":      {typ: tagInt, intV: 1},
			"hardcore":      {typ: tagByte, byteV: 1},
			"allowCommands": {typ: tagByte, byteV: 1},
			"WorldGenSettings": {typ: tagCompound, compound: map[string]Tag{
				"seed": {typ: tagLong, longV: -8675309},
			}},
			"Version": {typ: tagCompound, compound: map[string]Tag{
				"Name": {typ: tagString, strV: "1.21.4"},
			}},
		}},
	})
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	metadata, err := LoadLevelMetadata(worldDir)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.DataVersion != 4189 || metadata.LevelName != "fixture" || metadata.Seed != -8675309 {
		t.Fatalf("metadata=%+v", metadata)
	}
	if metadata.SpawnX != 12 || metadata.SpawnY != 81 || metadata.SpawnZ != -7 || metadata.VersionName != "1.21.4" {
		t.Fatalf("spawn/version metadata=%+v", metadata)
	}
	if !metadata.Hardcore || !metadata.AllowCommands || metadata.GameType != 1 {
		t.Fatalf("game metadata=%+v", metadata)
	}
}
