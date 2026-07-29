package server

import "testing"

func TestPingInfoString(t *testing.T) {
	p := PingInfo{PlayerName: "Notch", LatencyMs: 42}
	got := p.String()
	want := "Notch (42ms)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
