package jvm

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"GoCraft/core/plugin"
)

// jdkPins is the pin table, and it lives in this package on purpose.
//
// The obvious shape — one table in the server that knows about every
// interpreter — would leak exactly what the rest of the design refuses: core/
// would learn what a JDK is. Adding a runtime must touch no core file, so each
// runtime carries its own pins, its own platform keys and its own quirks.
//
// Where the artifacts come from is pinned rather than resolved through an API.
// A resolver is what you need when you serve any requested version; this needs
// exactly one per runtime, and a pinned table is reproducible, auditable as a
// diff, and adds no third-party service to the boot path.
//
// The table currently declares its version and no platforms, which is not an
// oversight: an entry is a URL, a size and a sha256 that must be taken from a
// real Adoptium release and verified, and a fabricated checksum is worse than a
// missing one — it fails at the end of a 45 MB download instead of at once, and
// it looks authoritative in review. Until the rows are filled, lookupPin says
// so by name and automatic provisioning is simply unavailable.
//
//go:embed jdk.json
var jdkPins []byte

// minimumJavaVersion is the language baseline from §13: scoped values are final
// in 25 (JEP 506), and the runtime uses them instead of the
// setContextClassLoader dance. A 24 that almost works is not worth supporting.
const minimumJavaVersion = 25

type pinTable struct {
	Version   string               `json:"version"`
	Platforms map[string]pinnedJDK `json:"platforms"`
}

type pinnedJDK struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Strip  int    `json:"strip"`
	Bin    string `json:"bin"`
}

// lookupPin resolves one platform to the artifact the provisioner should fetch,
// and returns the pinned JDK version alongside it for the cache key.
//
// A malformed row is refused here rather than at the end of a download: every
// field is required, and Bin has to be a relative path inside the archive
// because it is joined onto whatever directory the fetch produced.
func lookupPin(table []byte, key string) (plugin.Artifact, string, error) {
	var pins pinTable
	if err := json.Unmarshal(table, &pins); err != nil {
		return plugin.Artifact{}, "", fmt.Errorf("jvm: unreadable pin table: %w", err)
	}
	if strings.TrimSpace(pins.Version) == "" {
		return plugin.Artifact{}, "", fmt.Errorf("jvm: pin table declares no version")
	}
	entry, ok := pins.Platforms[key]
	if !ok {
		return plugin.Artifact{}, "", fmt.Errorf(
			"jvm: no JDK pinned for %s; this platform has %s", key, describePlatforms(pins.Platforms))
	}
	if err := entry.validate(key); err != nil {
		return plugin.Artifact{}, "", err
	}
	return plugin.Artifact{
		URL:    entry.URL,
		SHA256: entry.SHA256,
		Size:   entry.Size,
		Strip:  entry.Strip,
		Bin:    entry.Bin,
	}, pins.Version, nil
}

func (entry pinnedJDK) validate(key string) error {
	switch {
	case strings.TrimSpace(entry.URL) == "":
		return fmt.Errorf("jvm: pinned JDK for %s has no url", key)
	case strings.TrimSpace(entry.SHA256) == "":
		return fmt.Errorf("jvm: pinned JDK for %s has no sha256", key)
	case entry.Size <= 0:
		return fmt.Errorf("jvm: pinned JDK for %s has no size", key)
	case strings.TrimSpace(entry.Bin) == "":
		return fmt.Errorf("jvm: pinned JDK for %s does not say where java is", key)
	}
	// Bin is joined onto the directory the fetch produced, so a path that
	// escapes it would run something from outside the artifact entirely.
	cleaned := path.Clean(entry.Bin)
	if path.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") ||
		strings.Contains(entry.Bin, `\`) {
		return fmt.Errorf("jvm: pinned JDK for %s has an invalid bin path %q", key, entry.Bin)
	}
	return nil
}

// describePlatforms turns the table into something an admin can act on: which
// platforms this build does pin, or that it pins none at all.
func describePlatforms(platforms map[string]pinnedJDK) string {
	if len(platforms) == 0 {
		return "no pinned platforms at all, so install a JDK " +
			"and point JAVA_HOME at it"
	}
	keys := make([]string, 0, len(platforms))
	for key := range platforms {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return "pinned platforms are " + strings.Join(keys, ", ")
}
