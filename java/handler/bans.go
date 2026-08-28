package handler

import (
	"fmt"
	"net"
	"strings"
)

var (
	playerBans banRegistry
	ipBans     banRegistry
)

func ConfigureBans(playerPath, ipPath string) error {
	if err := playerBans.configure(playerPath, false); err != nil {
		return fmt.Errorf("player bans: %w", err)
	}
	if err := ipBans.configure(ipPath, true); err != nil {
		return fmt.Errorf("IP bans: %w", err)
	}
	return nil
}

func BanPlayer(name, source, reason string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("player name is required")
	}
	return playerBans.add(name, source, defaultBanReason(reason))
}

func PardonPlayer(name string) (bool, error) { return playerBans.remove(name) }
func BannedPlayers() []string                { return playerBans.values() }

func BanIP(address, source, reason string) error {
	ip, err := normalizeIPAddress(address)
	if err != nil {
		return err
	}
	return ipBans.add(ip, source, defaultBanReason(reason))
}

func PardonIP(address string) (bool, error) {
	ip, err := normalizeIPAddress(address)
	if err != nil {
		return false, err
	}
	return ipBans.remove(ip)
}

func BannedIPs() []string { return ipBans.values() }

func BanReason(name, address string) (string, bool) {
	if reason, banned := playerBans.reason(name); banned {
		return reason, true
	}
	ip, err := normalizeIPAddress(address)
	if err != nil {
		return "", false
	}
	return ipBans.reason(ip)
}

func normalizeIPAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	if host, _, err := net.SplitHostPort(address); err == nil {
		address = host
	}
	address = strings.Trim(address, "[]")
	if parsed := net.ParseIP(address); parsed != nil {
		return parsed.String(), nil
	}
	return "", fmt.Errorf("invalid IP address %q", address)
}

func defaultBanReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "Banned by an operator."
	}
	return reason
}
