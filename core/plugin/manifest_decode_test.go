package plugin

import (
	"strings"
	"testing"
)

const minimalManifest = `id = "fr.oreo.hello"
version = "0.1.0"
api = 1
runtime = "lua"
`

func TestDecodeManifestNamesUnknownFieldsWithTheirLine(t *testing.T) {
	_, err := DecodeManifest(strings.NewReader(minimalManifest + `description = "nope"` + "\n"))
	if err == nil {
		t.Fatal("DecodeManifest() accepted an unknown field")
	}
	if want := `plugin.toml:5: unknown field "description"`; err.Error() != want {
		t.Fatalf("DecodeManifest() error = %q, want %q", err.Error(), want)
	}
}

func TestDecodeManifestReportsSyntaxErrorsWithAPosition(t *testing.T) {
	_, err := DecodeManifest(strings.NewReader("id = \"fr.oreo.hello\"\nversion =\n"))
	if err == nil {
		t.Fatal("DecodeManifest() accepted malformed TOML")
	}
	if !strings.HasPrefix(err.Error(), ManifestFileName+":") {
		t.Fatalf("DecodeManifest() error = %v, want a %s:line:column prefix", err, ManifestFileName)
	}
}

func TestDecodeManifestAcceptsAMinimalManifest(t *testing.T) {
	manifest, err := DecodeManifest(strings.NewReader(minimalManifest))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "fr.oreo.hello" || manifest.Runtime != "lua" || manifest.APIVersion != CurrentAPIVersion {
		t.Fatalf("DecodeManifest() = %+v", manifest)
	}
	if len(manifest.Subscriptions) != 0 || len(manifest.Permissions) != 0 {
		t.Fatalf("DecodeManifest() subscriptions = %+v, permissions = %+v", manifest.Subscriptions, manifest.Permissions)
	}
}