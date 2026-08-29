package ipc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	wire "GoCraft/abi/v1/wire"
)

// testPair wires a Conn to a bare Codec over net.Pipe, which is a real
// full-duplex stream rather than a buffer: a write blocks until the far side
// reads it, so a test that forgets to drain deadlocks instead of passing.
func testPair(t *testing.T, handler func(*wire.Envelope)) (*Conn, *Codec) {
	t.Helper()
	hostSide, peerSide := net.Pipe()
	conn := NewConn(NewCodec(hostSide), handler)
	t.Cleanup(func() {
		conn.Close()
		peerSide.Close()
	})
	return conn, NewCodec(peerSide)
}

func loadEnvelope(pluginID string) *wire.Envelope {
	return &wire.Envelope{Body: &wire.Envelope_Load{Load: &wire.Load{PluginId: pluginID}}}
}

func TestConnRoundTripsARequest(t *testing.T) {
	conn, peer := testPair(t, nil)
	go func() {
		request, err := peer.Receive()
		if err != nil {
			return
		}
		peer.Send(&wire.Envelope{
			Seq:  request.GetSeq(),
			Body: &wire.Envelope_Loaded{Loaded: &wire.Loaded{PluginId: request.GetLoad().GetPluginId()}},
		})
	}()

	reply, err := conn.Request(t.Context(), loadEnvelope("fr.oreo.hello"))
	if err != nil {
		t.Fatal(err)
	}
	if got := reply.GetLoaded().GetPluginId(); got != "fr.oreo.hello" {
		t.Fatalf("reply plugin = %q", got)
	}
}

// The property the whole type exists for: one socket, several callers in
// flight, replies coming back in a different order than the requests went out.
// Each caller must get its own answer.
func TestConnCorrelatesOutOfOrderReplies(t *testing.T) {
	const requests = 12
	conn, peer := testPair(t, nil)

	go func() {
		received := make([]*wire.Envelope, 0, requests)
		for range requests {
			request, err := peer.Receive()
			if err != nil {
				return
			}
			received = append(received, request)
		}
		// Answer backwards, so a implementation that simply returns the next
		// reply to the next waiter is wrong for every request but the middle.
		for index := len(received) - 1; index >= 0; index-- {
			request := received[index]
			peer.Send(&wire.Envelope{
				Seq:  request.GetSeq(),
				Body: &wire.Envelope_Loaded{Loaded: &wire.Loaded{PluginId: request.GetLoad().GetPluginId()}},
			})
		}
	}()

	var wait sync.WaitGroup
	wait.Add(requests)
	for index := range requests {
		go func() {
			defer wait.Done()
			name := fmt.Sprintf("plugin.%d", index)
			reply, err := conn.Request(t.Context(), loadEnvelope(name))
			if err != nil {
				t.Errorf("%s: %v", name, err)
				return
			}
			if got := reply.GetLoaded().GetPluginId(); got != name {
				t.Errorf("%s received the reply meant for %s", name, got)
			}
		}()
	}
	wait.Wait()
}

// When the shared event budget expires the caller walks away. The runtime may
// still answer afterwards, so the pending entry has to go with it.
func TestConnGivesUpWhenTheContextExpires(t *testing.T) {
	conn, peer := testPair(t, nil)
	go func() {
		peer.Receive() // read the request, never answer it
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, err := conn.Request(ctx, loadEnvelope("fr.oreo.hello"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Request() error = %v, want a deadline", err)
	}

	conn.mu.Lock()
	waiting := len(conn.pending)
	conn.mu.Unlock()
	if waiting != 0 {
		t.Fatalf("%d pending entries left behind", waiting)
	}
}

// A runtime that dies must not leave callers parked until their own timeouts
// fire one by one: the read loop wakes them as it exits.
func TestConnWakesWaitersWhenThePeerDies(t *testing.T) {
	conn, peer := testPair(t, nil)
	go func() {
		peer.Receive()
		peer.Close()
	}()

	_, err := conn.Request(t.Context(), loadEnvelope("fr.oreo.hello"))
	if err == nil {
		t.Fatal("Request() returned no error after the peer closed")
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		t.Fatalf("Request() error = %v, want a connection failure", err)
	}
	select {
	case <-conn.Done():
	case <-time.After(time.Second):
		t.Fatal("Done() never closed")
	}
	if conn.Err() == nil {
		t.Fatal("Err() is nil after the peer closed")
	}
}

// Sequence numbers start at 1, so an envelope arriving with zero answers
// nothing and belongs to the handler.
func TestConnRoutesUnsolicitedEnvelopesToTheHandler(t *testing.T) {
	unsolicited := make(chan *wire.Envelope, 1)
	_, peer := testPair(t, func(envelope *wire.Envelope) { unsolicited <- envelope })

	go peer.Send(&wire.Envelope{
		Body: &wire.Envelope_Fail{Fail: &wire.Fail{PluginId: "fr.oreo.hello", Reason: "config.yml:12"}},
	})

	select {
	case envelope := <-unsolicited:
		if got := envelope.GetFail().GetReason(); got != "config.yml:12" {
			t.Fatalf("handler received %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("the handler was never called")
	}
}

func TestConnCloseWakesWaiters(t *testing.T) {
	conn, peer := testPair(t, nil)
	go peer.Receive()

	started := make(chan struct{})
	failed := make(chan error, 1)
	go func() {
		close(started)
		_, err := conn.Request(t.Context(), loadEnvelope("fr.oreo.hello"))
		failed <- err
	}()
	<-started
	time.Sleep(20 * time.Millisecond) // let the request reach the wait
	conn.Close()

	select {
	case err := <-failed:
		if err == nil {
			t.Fatal("Request() succeeded after Close()")
		}
	case <-time.After(time.Second):
		t.Fatal("Close() left a caller parked")
	}
}

func TestConnRefusesRequestsAfterClose(t *testing.T) {
	conn, _ := testPair(t, nil)
	conn.Close()
	if _, err := conn.Request(t.Context(), loadEnvelope("fr.oreo.hello")); !errors.Is(err, ErrConnClosed) {
		t.Fatalf("Request() error = %v, want ErrConnClosed", err)
	}
}

// Close and the read loop race to report why the connection ended: closing the
// stream is exactly what makes the read fail. Whichever wins, the answer must
// be ErrConnClosed — a supervisor that reads a deliberate stop as a crash
// respawns the process it just asked to leave.
//
// The loop is what makes the race show up; a single run passes either way.
func TestConnCloseIsNeverReportedAsAFailure(t *testing.T) {
	for attempt := range 50 {
		hostSide, peerSide := net.Pipe()
		conn := NewConn(NewCodec(hostSide), nil)
		conn.Close()
		<-conn.Done()
		if !errors.Is(conn.Err(), ErrConnClosed) {
			t.Fatalf("attempt %d: Err() = %v, want ErrConnClosed", attempt, conn.Err())
		}
		peerSide.Close()
	}
}

func TestConnRejectsANilEnvelope(t *testing.T) {
	conn, _ := testPair(t, nil)
	if _, err := conn.Request(t.Context(), nil); err == nil {
		t.Fatal("Request() accepted nil")
	}
}