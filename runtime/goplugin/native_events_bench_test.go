package goplugin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"GoCraft/core/player"
	"GoCraft/core/plugin"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

func BenchmarkNativeEventIPC(b *testing.B) {
	runtime, bundle, _ := nativeEventRuntime(b)
	// Measurement-only deadline: retain slow samples and report overruns.
	// Production and the functional round-trip tests still use 2 ms.
	bus := plugin.NewBus(b.Context(), time.Second)
	started := time.Now()
	loaded, err := runtime.Load(b.Context(), bundle)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("cold process load: %s", time.Since(started))
	if err := bus.Attach(loaded); err != nil {
		b.Fatal(err)
	}
	p := player.New([16]byte{1}, "Alex", player.ClientEditionJava)
	emit := func() time.Duration {
		started := time.Now()
		message := "original"
		if !bus.EmitPlayerChat(p, &message) || message != "rewritten" {
			b.Fatal("round trip failed")
		}
		return time.Since(started)
	}
	started = time.Now()
	emit()
	b.Logf("first actual event: %s", time.Since(started))
	for range 200 {
		emit()
	}
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

// Same host path, but no socket, serialization or SDK execution. Compare this
// with NativeEventIPC; do not subtract noisy wall-clock samples as exact costs.
type benchmarkInstance struct{}

func (benchmarkInstance) Manifest() gcpkg.Manifest {
	return gcpkg.Manifest{ID: "bench", Subscriptions: []gcpkg.Subscription{{Event: plugin.EventPlayerChat}}}
}
func (benchmarkInstance) Unload(context.Context) error { return nil }

func (benchmarkInstance) Dispatch(context.Context, *abi.Event) (abi.Verdict, error) {
	return abi.Verdict{Mutations: []abi.Mutation{{Path: []uint32{1}, Value: abi.String("rewritten")}}}, nil
}

func BenchmarkNativeEventHost(b *testing.B) {
	bus := plugin.NewBus(b.Context(), time.Second)
	if err := bus.Attach(benchmarkInstance{}); err != nil {
		b.Fatal(err)
	}
	p := player.New([16]byte{1}, "Alex", player.ClientEditionJava)
	for _, count := range []int{1, 100, 500, 600, 1000} {
		b.Run(fmt.Sprintf("warmed_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for range count {
					message := "original"
					bus.EmitPlayerChat(p, &message)
				}
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*count), "ns/event")
		})
	}
}
