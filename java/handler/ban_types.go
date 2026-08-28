package handler

import (
	"strings"
	"sync"
)

type banRecord struct {
	Name    string `json:"name,omitempty"`
	IP      string `json:"ip,omitempty"`
	Created string `json:"created"`
	Source  string `json:"source"`
	Expires string `json:"expires"`
	Reason  string `json:"reason"`
}

type banRegistry struct {
	mu      sync.RWMutex
	path    string
	address bool
	entries map[string]banRecord
}

func normalizeBanValue(value string) string   { return strings.ToLower(normalizeBanDisplay(value)) }
func normalizeBanDisplay(value string) string { return strings.TrimSpace(value) }
