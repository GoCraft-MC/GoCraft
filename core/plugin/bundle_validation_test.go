package plugin

import (
	"path/filepath"
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

func TestOpenBundleRejectsUnsafeArchivePath(t *testing.T) {
	directory := t.TempDir()
	writeBundle(t, directory, "unsafe.gcpkg", validTestManifest, map[string]string{"../escape": "bad"})
	_, err := OpenBundle(filepath.Join(directory, "unsafe.gcpkg"))
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("OpenBundle() error = %v", err)
	}
}

func TestOpenBundleRejectsUnknownManifestField(t *testing.T) {
	directory := t.TempDir()
	writeBundle(t, directory, "unknown.gcpkg", validTestManifest+"unknown = true\n", nil)
	_, err := OpenBundle(filepath.Join(directory, "unknown.gcpkg"))
	if err == nil || !strings.Contains(err.Error(), `unknown field "unknown"`) {
		t.Fatalf("OpenBundle() error = %v", err)
	}
	if !strings.Contains(err.Error(), ManifestFileName+":") {
		t.Fatalf("OpenBundle() error = %v, want a %s line reference", err, ManifestFileName)
	}
}

func TestOpenBundleRejectsUnsupportedAPI(t *testing.T) {
	directory := t.TempDir()
	manifest := strings.Replace(validTestManifest, "api = 1", "api = 2", 1)
	writeBundle(t, directory, "future.gcpkg", manifest, nil)
	_, err := OpenBundle(filepath.Join(directory, "future.gcpkg"))
	if err == nil || !strings.Contains(err.Error(), "API 2 is unsupported") {
		t.Fatalf("OpenBundle() error = %v", err)
	}
}

func TestOpenBundleRejectsInvalidPermissionLists(t *testing.T) {
	tests := []struct {
		name        string
		permissions string
		want        string
	}{
		{name: "empty", permissions: `[""]`, want: "empty subscribed permission"},
		{name: "duplicate", permissions: `["shop.use", "shop.use"]`, want: "duplicate subscribed permission"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			manifest := validTestManifest + "[subscribe]\nperms = " + tc.permissions + "\n"
			writeBundle(t, directory, "invalid.gcpkg", manifest, nil)
			_, err := OpenBundle(filepath.Join(directory, "invalid.gcpkg"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("OpenBundle() error = %v, want %q", err, tc.want)
			}
		})
	}
}
