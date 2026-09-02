package jvm

import (
	"runtime"
	"strings"
	"testing"
)

const goodTable = `{
  "version": "25.0.3",
  "platforms": {
    "linux-amd64":      {"url": "https://example.invalid/jdk.tar.gz", "sha256": "a3f1", "size": 45812736, "strip": 1, "bin": "bin/java"},
    "linux-amd64-musl": {"url": "https://example.invalid/musl.tar.gz", "sha256": "b4e2", "size": 45812737, "strip": 1, "bin": "bin/java"}
  }
}`

func TestLookupPinResolvesAPlatform(t *testing.T) {
	artifact, version, err := lookupPin([]byte(goodTable), "linux-amd64")
	if err != nil {
		t.Fatalf("lookupPin() = %v", err)
	}
	if version != "25.0.3" {
		t.Fatalf("lookupPin() version = %q", version)
	}
	if artifact.URL != "https://example.invalid/jdk.tar.gz" || artifact.SHA256 != "a3f1" ||
		artifact.Size != 45812736 || artifact.Strip != 1 || artifact.Bin != "bin/java" {
		t.Fatalf("lookupPin() artifact = %+v", artifact)
	}
}

// A glibc tarball does not run on Alpine, so musl is a row of its own rather
// than a variant resolved after the download.
func TestLookupPinSeparatesMuslFromGlibc(t *testing.T) {
	glibc, _, err := lookupPin([]byte(goodTable), "linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	musl, _, err := lookupPin([]byte(goodTable), "linux-amd64-musl")
	if err != nil {
		t.Fatal(err)
	}
	if glibc.URL == musl.URL {
		t.Fatal("lookupPin() returned the same artifact for glibc and musl")
	}
}

func TestLookupPinNamesTheMissingPlatformAndWhatIsPinned(t *testing.T) {
	_, _, err := lookupPin([]byte(goodTable), "plan9-riscv64")
	if err == nil {
		t.Fatal("lookupPin() invented a JDK for an unpinned platform")
	}
	if !strings.Contains(err.Error(), "plan9-riscv64") {
		t.Fatalf("lookupPin() error = %v, want the platform named", err)
	}
	if !strings.Contains(err.Error(), "linux-amd64") {
		t.Fatalf("lookupPin() error = %v, want the pinned platforms listed", err)
	}
}

// The table this build ships has no rows. The test states that as the current
// answer rather than leaving it to be discovered at boot, and it will fail the
// day someone fills the table in — which is exactly when the message about
// pointing JAVA_HOME somewhere stops being the right advice.
func TestShippedPinTableHasNoPlatformsYet(t *testing.T) {
	_, _, err := lookupPin(jdkPins, platformKey())
	if err == nil {
		t.Skip("the pin table now has an entry for this platform; drop this test")
	}
	if !strings.Contains(err.Error(), "no pinned platforms at all") {
		t.Fatalf("lookupPin(jdkPins) error = %v, want the empty table explained", err)
	}
	if !strings.Contains(err.Error(), "JAVA_HOME") {
		t.Fatalf("lookupPin(jdkPins) error = %v, want the admin told what to do instead", err)
	}
}

func TestLookupPinRejectsIncompleteRows(t *testing.T) {
	for _, testCase := range []struct {
		name string
		row  string
		want string
	}{
		{"no url", `{"sha256": "a3", "size": 1, "bin": "bin/java"}`, "no url"},
		{"no sha256", `{"url": "https://x.invalid", "size": 1, "bin": "bin/java"}`, "no sha256"},
		{"no size", `{"url": "https://x.invalid", "sha256": "a3", "bin": "bin/java"}`, "no size"},
		{"no bin", `{"url": "https://x.invalid", "sha256": "a3", "size": 1}`, "where java is"},
		{
			// Bin is joined onto whatever directory the fetch produced, so a
			// path that climbs out of it would run something from elsewhere.
			name: "escaping bin",
			row:  `{"url": "https://x.invalid", "sha256": "a3", "size": 1, "bin": "../../usr/bin/java"}`,
			want: "invalid bin path",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			table := `{"version": "25.0.3", "platforms": {"linux-amd64": ` + testCase.row + `}}`
			_, _, err := lookupPin([]byte(table), "linux-amd64")
			if err == nil {
				t.Fatal("lookupPin() accepted an incomplete row")
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("lookupPin() error = %v, want it to mention %q", err, testCase.want)
			}
		})
	}
}

func TestLookupPinRejectsAnUnusableTable(t *testing.T) {
	for _, testCase := range []struct{ name, table string }{
		{"malformed", `{"version":`},
		{"no version", `{"platforms": {}}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, _, err := lookupPin([]byte(testCase.table), "linux-amd64"); err == nil {
				t.Fatal("lookupPin() accepted an unusable table")
			}
		})
	}
}

func TestPlatformKeyNamesThisMachine(t *testing.T) {
	key := platformKey()
	if !strings.HasPrefix(key, runtime.GOOS+"-"+runtime.GOARCH) {
		t.Fatalf("platformKey() = %q, want it to start with the Go platform", key)
	}
	if runtime.GOOS != "linux" && strings.HasSuffix(key, "-musl") {
		t.Fatalf("platformKey() = %q, musl is a Linux concern only", key)
	}
}

func TestMuslLoaderPathIsNamedAfterTheMachine(t *testing.T) {
	// GOARCH and the loader name disagree on every architecture that matters,
	// which is why the mapping is written out rather than interpolated.
	if got := muslLoaderPath("amd64"); got != "/lib/ld-musl-x86_64.so.1" {
		t.Fatalf("muslLoaderPath(amd64) = %q", got)
	}
	if got := muslLoaderPath("arm64"); got != "/lib/ld-musl-aarch64.so.1" {
		t.Fatalf("muslLoaderPath(arm64) = %q", got)
	}
	if got := muslLoaderPath("mips64le"); got != "" {
		t.Fatalf("muslLoaderPath(mips64le) = %q, want empty for an unmapped arch", got)
	}
}
