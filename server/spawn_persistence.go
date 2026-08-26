package server

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"

	"GoCraft/core/spatial"
)

const worldSpawnFile = "gocraft_spawn.json"

func loadSavedWorldSpawn(worldDir string) (spatial.Vec3, bool) {
	data, err := os.ReadFile(filepath.Join(worldDir, worldSpawnFile))
	if err != nil {
		return spatial.Vec3{}, false
	}
	var position spatial.Vec3
	if json.Unmarshal(data, &position) != nil || !validWorldSpawn(position) {
		return spatial.Vec3{}, false
	}
	return position, true
}

func saveWorldSpawn(worldDir string, position spatial.Vec3) error {
	data, err := json.Marshal(position)
	if err != nil {
		return err
	}
	path := filepath.Join(worldDir, worldSpawnFile)
	temporary := path + ".tmp"
	if err = os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func validWorldSpawn(position spatial.Vec3) bool {
	return !math.IsNaN(position.X) && !math.IsNaN(position.Y) && !math.IsNaN(position.Z) &&
		!math.IsInf(position.X, 0) && !math.IsInf(position.Y, 0) && !math.IsInf(position.Z, 0)
}
