package jvm

import (
	"fmt"
	"testing"
	"time"

	"GoCraft/core/player"
	"GoCraft/core/plugin"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func BenchmarkNativeEventJVM(b *testing.B) {
	_, loaded := nativeJVMFixture(b)
	// Measurement-only: count slow samples instead of losing their mutations.
	bus := plugin.NewBus(b.Context(), time.Second)
	if err := bus.Attach(loaded); err != nil {
		b.Fatal(err)
	}
	p := player.New([16]byte{1}, "Alex", player.ClientEditionJava)
	emit := func() time.Duration {
		started := time.Now()
		text := "original"
		if !bus.EmitPlayerChat(p, &text) || text != "rewritten" {
			b.Fatal("round trip failed")
		}
		return time.Since(started)
	}
	started := time.Now()
	emit()
	b.Logf("cold event without shaped warm-up: %s", time.Since(started))
	if err := loaded.Warm(b.Context(), &abi.Event{Type: plugin.EventPlayerChat, Fields: plugin.BlankEvent(plugin.EventPlayerChat)}); err != nil {
		b.Fatal(err)
	}
	started = time.Now()
	emit()
	b.Logf("first event after shaped warm-up: %s", time.Since(started))
	for _, count := range []int{1, 100, 500, 600, 1000} {
		b.Run(fmt.Sprintf("warmed_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			overruns := 0
			for range b.N {
				for range count {
					if emit() > 2*time.Millisecond {
						overruns++
					}
				}
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*count), "ns/event")
			b.ReportMetric(100*float64(overruns)/float64(b.N*count), "over2ms_percent")
		})
	}
}
