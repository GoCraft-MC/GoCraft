package plugin

import (
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	// ManifestFileName is the manifest at the root of a bundle, and of the
	// source directory a bundle is built from.
	ManifestFileName = "plugin.toml"

	CurrentAPIVersion   = 1
	maximumManifestSize = 1 << 20
)

type manifestFile struct {
	ID         string `toml:"id"`
	Version    string `toml:"version"`
	APIVersion uint32 `toml:"api"`
	Runtime    string `toml:"runtime"`
	Entry      string `toml:"entry"`
	Subscribe  struct {
		Events      []string `toml:"events"`
		Permissions []string `toml:"perms"`
	} `toml:"subscribe"`
	Commands struct {
		Tree string `toml:"tree"`
	} `toml:"commands"`
}

// DecodeManifest reads and validates one plugin.toml. It is the only manifest
// parser in the project: the host calls it when opening a bundle and the CLI
// calls it when building one, so build-time and load-time validation cannot
// drift apart.
func DecodeManifest(reader io.Reader) (Manifest, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maximumManifestSize+1))
	if err != nil {
		return Manifest{}, fmt.Errorf("read %s: %w", ManifestFileName, err)
	}
	if len(data) > maximumManifestSize {
		return Manifest{}, fmt.Errorf("%s exceeds %d bytes", ManifestFileName, maximumManifestSize)
	}
	var file manifestFile
	decoder := toml.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return Manifest{}, fmt.Errorf("decode %s: %w", ManifestFileName, err)
	}
	manifest := Manifest{
		ID: file.ID, Version: file.Version, APIVersion: file.APIVersion,
		Runtime: file.Runtime, Entry: file.Entry, CommandTree: file.Commands.Tree,
		Permissions: append([]string(nil), file.Subscribe.Permissions...),
	}
	for _, event := range file.Subscribe.Events {
		manifest.Subscriptions = append(manifest.Subscriptions, Subscription{Event: event, Priority: PriorityNormal})
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validateManifest(manifest Manifest) error {
	if !validPluginID(manifest.ID) {
		return fmt.Errorf("plugin manifest: invalid id %q", manifest.ID)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("plugin %s: version is required", manifest.ID)
	}
	if manifest.APIVersion != CurrentAPIVersion {
		return fmt.Errorf("plugin %s: API %d is unsupported, host uses %d", manifest.ID, manifest.APIVersion, CurrentAPIVersion)
	}
	if strings.TrimSpace(manifest.Runtime) == "" {
		return fmt.Errorf("plugin %s: runtime is required", manifest.ID)
	}
	if manifest.CommandTree != "" && !validBundleReference(manifest.CommandTree) {
		return fmt.Errorf("plugin %s: invalid command tree path %q", manifest.ID, manifest.CommandTree)
	}
	permissions := make(map[string]struct{}, len(manifest.Permissions))
	for _, permission := range manifest.Permissions {
		if strings.TrimSpace(permission) == "" {
			return fmt.Errorf("plugin %s: empty subscribed permission", manifest.ID)
		}
		if _, duplicate := permissions[permission]; duplicate {
			return fmt.Errorf("plugin %s: duplicate subscribed permission %s", manifest.ID, permission)
		}
		permissions[permission] = struct{}{}
	}
	seen := make(map[string]struct{}, len(manifest.Subscriptions))
	for _, subscription := range manifest.Subscriptions {
		if strings.TrimSpace(subscription.Event) == "" {
			return fmt.Errorf("plugin %s: empty event subscription", manifest.ID)
		}
		if _, duplicate := seen[subscription.Event]; duplicate {
			return fmt.Errorf("plugin %s: duplicate subscription to %s", manifest.ID, subscription.Event)
		}
		seen[subscription.Event] = struct{}{}
	}
	return nil
}

func validBundleReference(reference string) bool {
	if strings.Contains(reference, `\`) || path.IsAbs(reference) {
		return false
	}
	cleaned := path.Clean(reference)
	return cleaned == reference && cleaned != "." && cleaned != ".." && !strings.HasPrefix(cleaned, "../")
}

func validPluginID(id string) bool {
	if id == "" || id[0] == '.' || id[len(id)-1] == '.' {
		return false
	}
	for _, character := range id {
		if character != '.' && character != '-' && character != '_' &&
			(character < 'a' || character > 'z') && (character < '0' || character > '9') {
			return false
		}
	}
	return !strings.Contains(id, "..")
}
