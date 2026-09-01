package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func mustDecode(path string, target any) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		panic(fmt.Errorf("decode %s: %w", path, err))
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func directory(path string) string {
	if index := strings.LastIndexAny(path, `/\\`); index >= 0 {
		return path[:index]
	}
	return "."
}
