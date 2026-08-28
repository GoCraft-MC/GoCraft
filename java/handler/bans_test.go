package handler

import (
	"path/filepath"
	"testing"
)

func TestPlayerBansPersist(t *testing.T) {
	dir := t.TempDir()
	players := filepath.Join(dir, "banned-players.json")
	ips := filepath.Join(dir, "banned-ips.json")
	if err := ConfigureBans(players, ips); err != nil {
		t.Fatal(err)
	}
	if err := BanPlayer("Steve", "Console", "Testing"); err != nil {
		t.Fatal(err)
	}
	if reason, ok := BanReason("steve", ""); !ok || reason != "Testing" {
		t.Fatalf("BanReason() = %q, %v", reason, ok)
	}
	if err := ConfigureBans(players, ips); err != nil {
		t.Fatal(err)
	}
	if got := BannedPlayers(); len(got) != 1 || got[0] != "Steve" {
		t.Fatalf("BannedPlayers() = %v", got)
	}
	if removed, err := PardonPlayer("STEVE"); err != nil || !removed {
		t.Fatalf("PardonPlayer() = %v, %v", removed, err)
	}
}

func TestIPBansNormalizeRemoteAddress(t *testing.T) {
	dir := t.TempDir()
	if err := ConfigureBans(filepath.Join(dir, "players.json"), filepath.Join(dir, "ips.json")); err != nil {
		t.Fatal(err)
	}
	if err := BanIP("127.0.0.1:25565", "Console", "Testing"); err != nil {
		t.Fatal(err)
	}
	if reason, ok := BanReason("", "127.0.0.1:54321"); !ok || reason != "Testing" {
		t.Fatalf("BanReason() = %q, %v", reason, ok)
	}
	if err := BanIP("not an ip", "Console", "Testing"); err == nil {
		t.Fatal("BanIP accepted an invalid address")
	}
}
