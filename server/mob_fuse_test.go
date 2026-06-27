package server

import (
	"testing"
	"time"
)

func TestCreeperFuseIgnite(t *testing.T) {
	f := &CreeperFuse{}
	if f.Tick() {
		t.Fatal("fuse should not tick before ignition")
	}
	f.Ignite()
	if !f.Ignited {
		t.Fatal("fuse should be ignited")
	}
	time.Sleep(1600 * time.Millisecond)
	if !f.Tick() {
		t.Fatal("fuse should have detonated after duration")
	}
}
