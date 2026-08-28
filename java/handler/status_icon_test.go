package handler

import (
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"GoCraft/config"
)

func TestLoadServerIconUsesEmbeddedDefault(t *testing.T) {
	dataURI, err := LoadServerIcon("")
	if err != nil {
		t.Fatalf("LoadServerIcon: %v", err)
	}
	icon := decodeStatusIcon(t, dataURI)
	if icon.Bounds().Dx() != 64 || icon.Bounds().Dy() != 64 {
		t.Fatalf("icon dimensions = %v, want 64x64", icon.Bounds())
	}
}

func TestLoadServerIconResizesConfiguredPNG(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 96, 48))
	for y := range 48 {
		for x := range 96 {
			source.Set(x, y, color.NRGBA{R: 30, G: 180, B: 220, A: 255})
		}
	}
	path := filepath.Join(t.TempDir(), "wide.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, source); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	dataURI, err := LoadServerIcon(path)
	if err != nil {
		t.Fatalf("LoadServerIcon: %v", err)
	}
	icon := decodeStatusIcon(t, dataURI)
	if _, _, _, alpha := icon.At(32, 0).RGBA(); alpha != 0 {
		t.Fatalf("top padding alpha = %d, want transparent", alpha)
	}
	if _, _, _, alpha := icon.At(32, 32).RGBA(); alpha == 0 {
		t.Fatal("scaled image center is transparent")
	}
}

func TestBuildStatusJSONIncludesFavicon(t *testing.T) {
	want := "data:image/png;base64,test"
	payload, err := buildStatusJSON(&config.Config{MOTD: "GoCraft", MaxPlayers: 20}, want)
	if err != nil {
		t.Fatalf("buildStatusJSON: %v", err)
	}
	var response statusResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if response.Favicon != want {
		t.Fatalf("favicon = %q, want %q", response.Favicon, want)
	}
}

func decodeStatusIcon(t *testing.T, dataURI string) image.Image {
	t.Helper()
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(dataURI, prefix) {
		t.Fatalf("icon URI has invalid prefix")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURI, prefix))
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	icon, err := png.Decode(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("decode PNG: %v", err)
	}
	return icon
}
