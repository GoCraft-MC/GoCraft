package handler

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
)

const serverIconSize = 64

//go:embed assets/server-icon.png
var defaultServerIcon []byte

// LoadServerIcon returns the Java status favicon data URI. A configured PNG
// is scaled to fit the vanilla 64x64 canvas; a missing file uses GoCraft's
// embedded icon so fresh binary-only installs still have artwork.
func LoadServerIcon(path string) (string, error) {
	data := defaultServerIcon
	if strings.TrimSpace(path) != "" {
		configured, err := os.ReadFile(path)
		switch {
		case err == nil:
			data = configured
		case !os.IsNotExist(err):
			return "", fmt.Errorf("reading server icon: %w", err)
		}
	}

	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("decoding server icon PNG: %w", err)
	}
	icon, err := resizeServerIcon(source)
	if err != nil {
		return "", err
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, icon); err != nil {
		return "", fmt.Errorf("encoding server icon PNG: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes()), nil
}

func resizeServerIcon(source image.Image) (*image.NRGBA, error) {
	bounds := source.Bounds()
	sourceWidth, sourceHeight := bounds.Dx(), bounds.Dy()
	if sourceWidth < 1 || sourceHeight < 1 {
		return nil, fmt.Errorf("server icon has empty dimensions")
	}

	targetWidth, targetHeight := serverIconSize, serverIconSize
	if sourceWidth > sourceHeight {
		targetHeight = max(1, sourceHeight*serverIconSize/sourceWidth)
	} else if sourceHeight > sourceWidth {
		targetWidth = max(1, sourceWidth*serverIconSize/sourceHeight)
	}
	offsetX := (serverIconSize - targetWidth) / 2
	offsetY := (serverIconSize - targetHeight) / 2
	destination := image.NewNRGBA(image.Rect(0, 0, serverIconSize, serverIconSize))

	for y := range targetHeight {
		sourceY := bounds.Min.Y + y*sourceHeight/targetHeight
		for x := range targetWidth {
			sourceX := bounds.Min.X + x*sourceWidth/targetWidth
			destination.Set(offsetX+x, offsetY+y, source.At(sourceX, sourceY))
		}
	}
	return destination, nil
}
