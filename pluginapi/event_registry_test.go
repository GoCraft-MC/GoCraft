package pluginapi

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func testLogger(output *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(output, nil))
}

func TestEventsDispatchInOrderAndRecoverPanics(t *testing.T) {
	var output bytes.Buffer
	events := newEvents(testLogger(&output))
	called := 0
	if err := events.OnBlockBreak(func(*BlockBreakEvent) {
		called++
		panic("broken listener")
	}); err != nil {
		t.Fatal(err)
	}
	if err := events.OnBlockBreak(func(event *BlockBreakEvent) {
		called++
		event.Cancel()
	}); err != nil {
		t.Fatal(err)
	}

	event := &BlockBreakEvent{}
	events.dispatch(event)
	if called != 2 || !event.Cancelled() {
		t.Fatalf("dispatch called %d listeners, cancelled=%v", called, event.Cancelled())
	}
	if log := output.String(); !strings.Contains(log, "broken listener") || !strings.Contains(log, "stack") {
		t.Fatalf("panic log = %q", log)
	}
	if registered := events.registeredTypes(); len(registered) != 1 || registered[0] != EventBlockBreak {
		t.Fatalf("registeredTypes() = %v", registered)
	}
}

func TestEventsClearPreventsFutureCallbacks(t *testing.T) {
	events := newEvents(slog.Default())
	called := false
	if err := events.OnPlayerJoin(func(*PlayerJoinEvent) { called = true }); err != nil {
		t.Fatal(err)
	}
	events.clear()
	events.dispatch(&PlayerJoinEvent{})
	if called {
		t.Fatal("disabled registry invoked a listener")
	}
	if err := events.OnPlayerJoin(func(*PlayerJoinEvent) {}); err == nil {
		t.Fatal("disabled registry accepted a listener")
	}
}
