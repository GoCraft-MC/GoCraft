package permission

import (
	"path/filepath"
	"testing"
)

func TestAllowedResolvesGroupsWildcardsAndUserOverrides(t *testing.T) {
	manager := NewMemory()
	document := manager.Snapshot()
	document.Groups["moderator"] = Group{
		Parents: []string{"default"},
		Permissions: map[string]bool{
			"gocraft.command.*":    true,
			"gocraft.command.stop": false,
		},
	}
	document.Users["alex"] = User{
		Groups:      []string{"moderator"},
		Permissions: map[string]bool{"gocraft.command.tp": false},
	}
	if err := manager.Replace(document); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		node    string
		allowed bool
	}{
		{"gocraft.command.give", true},
		{"gocraft.command.stop", false},
		{"gocraft.command.tp", false},
	} {
		if got := manager.Allowed("Alex", test.node, false, false); got != test.allowed {
			t.Errorf("Allowed(%q) = %v, want %v", test.node, got, test.allowed)
		}
	}
	if !manager.Allowed("someone", "gocraft.command.help", false, true) {
		t.Fatal("an unset permission did not retain its public command default")
	}
	if !manager.Allowed("operator", "anything", true, false) {
		t.Fatal("operator wildcard bypass was not honored")
	}
}

func TestPermissionDocumentPersistsAndRejectsCycles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "permissions.json")
	manager, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	document := manager.Snapshot()
	document.Groups["builder"] = Group{Permissions: map[string]bool{"gocraft.command.give": true}}
	document.Users["steve"] = User{Groups: []string{"builder"}}
	if err := manager.Replace(document); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Allowed("Steve", "gocraft.command.give", false, false) {
		t.Fatal("persisted group permission was not restored")
	}

	invalid := reloaded.Snapshot()
	invalid.Groups["one"] = Group{Parents: []string{"two"}}
	invalid.Groups["two"] = Group{Parents: []string{"one"}}
	if err := reloaded.Replace(invalid); err == nil {
		t.Fatal("cyclic group inheritance was accepted")
	}
	invalid = reloaded.Snapshot()
	invalid.Groups["bad group"] = Group{}
	if err := reloaded.Replace(invalid); err == nil {
		t.Fatal("group name containing whitespace was accepted")
	}
}
