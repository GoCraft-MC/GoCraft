package plugin

import (
	"strings"
	"testing"
)

const validTestManifest = `
id = "fr.oreo.shop"
version = "1.0.0"
api = 1
runtime = "go"
`

func TestScanBundlesRejectsDuplicatePluginID(t *testing.T) {
	directory := t.TempDir()
	writeBundle(t, directory, "one.gcpkg", validTestManifest, nil)
	writeBundle(t, directory, "two.gcpkg", validTestManifest, nil)
	_, err := ScanBundles(directory)
	if err == nil || !strings.Contains(err.Error(), "present in both") {
		t.Fatalf("ScanBundles() error = %v", err)
	}
}
