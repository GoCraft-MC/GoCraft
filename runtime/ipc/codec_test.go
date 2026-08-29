package ipc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"google.golang.org/protobuf/proto"

	wire "GoCraft/abi/v1/wire"
)

// stream is a bytes.Buffer seen as a ReadWriter, which is enough for framing:
// the codec never needs a real socket to be exercised.
func testCodec() (*Codec, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	return NewCodec(buffer), buffer
}

func pingEnvelope(seq uint64) *wire.Envelope {
	return &wire.Envelope{Seq: seq, Body: &wire.Envelope_Ping{Ping: &wire.Ping{}}}
}

func TestCodecRoundTripsAnEnvelope(t *testing.T) {
	codec, _ := testCodec()
	sent := &wire.Envelope{
		Seq:  7,
		Body: &wire.Envelope_Hello{Hello: &wire.Hello{Abi: 1, Runtime: "jvm 25.0.3"}},
	}
	if err := codec.Send(sent); err != nil {
		t.Fatal(err)
	}
	received, err := codec.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(received, sent) {
		t.Fatalf("round trip = %v, want %v", received, sent)
	}
}

// Framing exists for exactly this: a stream has no message boundaries, so the
// reader has to be told where each envelope ends.
func TestCodecSeparatesBackToBackEnvelopes(t *testing.T) {
	codec, _ := testCodec()
	for seq := uint64(1); seq <= 4; seq++ {
		if err := codec.Send(pingEnvelope(seq)); err != nil {
			t.Fatal(err)
		}
	}
	for seq := uint64(1); seq <= 4; seq++ {
		received, err := codec.Receive()
		if err != nil {
			t.Fatal(err)
		}
		if received.GetSeq() != seq {
			t.Fatalf("envelope %d has seq %d", seq, received.GetSeq())
		}
		if received.GetPing() == nil {
			t.Fatalf("envelope %d lost its body", seq)
		}
	}
}

// A peer that closes between frames has shut down cleanly. The caller must be
// able to tell that from a connection cut mid-message.
func TestCodecReportsACleanCloseAsEOF(t *testing.T) {
	codec, _ := testCodec()
	_, err := codec.Receive()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Receive() error = %v, want io.EOF", err)
	}
}

func TestCodecReportsATruncatedFrame(t *testing.T) {
	codec, buffer := testCodec()
	if err := codec.Send(pingEnvelope(1)); err != nil {
		t.Fatal(err)
	}
	full := buffer.Bytes()
	truncated := full[:len(full)-1]

	cut, _ := testCodec()
	cut.reader.Reset(bytes.NewReader(truncated))
	if _, err := cut.Receive(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Receive() error = %v, want io.ErrUnexpectedEOF", err)
	}
}

// The limit has to be checked against the declared length, before allocating.
func TestCodecRejectsAnOversizedDeclaredFrame(t *testing.T) {
	codec, buffer := testCodec()
	buffer.Write(binary.AppendUvarint(nil, uint64(maximumFrameSize)+1))
	_, err := codec.Receive()
	if err == nil || !strings.Contains(err.Error(), "over the") {
		t.Fatalf("Receive() error = %v, want a frame size rejection", err)
	}
}

func TestCodecRejectsAMalformedPayload(t *testing.T) {
	codec, buffer := testCodec()
	garbage := []byte{0xff, 0xff, 0xff}
	buffer.Write(binary.AppendUvarint(nil, uint64(len(garbage))))
	buffer.Write(garbage)
	if _, err := codec.Receive(); err == nil || !strings.Contains(err.Error(), "decode envelope") {
		t.Fatalf("Receive() error = %v, want a decode failure", err)
	}
}

func TestCodecRejectsANilEnvelope(t *testing.T) {
	codec, _ := testCodec()
	if err := codec.Send(nil); err == nil {
		t.Fatal("Send() accepted nil")
	}
}

// Send reuses its buffers between calls, so a caller that keeps a reference to
// an envelope must not see it corrupted by the next send.
func TestCodecReuseDoesNotCorruptEarlierFrames(t *testing.T) {
	codec, _ := testCodec()
	first := &wire.Envelope{Seq: 1, Body: &wire.Envelope_Fail{
		Fail: &wire.Fail{PluginId: "fr.oreo.hello", Reason: strings.Repeat("x", 512)},
	}}
	second := pingEnvelope(2)
	if err := codec.Send(first); err != nil {
		t.Fatal(err)
	}
	if err := codec.Send(second); err != nil {
		t.Fatal(err)
	}
	received, err := codec.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(received, first) {
		t.Fatalf("first envelope = %v, want %v", received, first)
	}
	received, err = codec.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(received, second) {
		t.Fatalf("second envelope = %v, want %v", received, second)
	}
}

// recordingStream keeps every Write call apart instead of concatenating them,
// which is what makes the invariant below observable.
type recordingStream struct {
	mu     sync.Mutex
	writes [][]byte
}

func (s *recordingStream) Write(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes = append(s.writes, append([]byte(nil), data...))
	return len(data), nil
}

func (s *recordingStream) Read([]byte) (int, error) { return 0, io.EOF }

// Concurrent sends must not interleave. Two failure modes are covered at once:
// a Send split across several Write calls would let another goroutine slip
// bytes between them, and unsynchronised access to the codec's reusable
// marshalling buffer would corrupt the payloads themselves.
//
// The assertion is that each recorded Write is exactly one whole envelope, and
// that every sequence number arrives intact.
func TestCodecSerialisesConcurrentSends(t *testing.T) {
	const senders = 16
	stream := &recordingStream{}
	codec := NewCodec(stream)

	var wait sync.WaitGroup
	wait.Add(senders)
	for seq := 1; seq <= senders; seq++ {
		go func() {
			defer wait.Done()
			envelope := &wire.Envelope{Seq: uint64(seq), Body: &wire.Envelope_Fail{
				Fail: &wire.Fail{PluginId: "fr.oreo.hello", Reason: strings.Repeat("x", seq*64)},
			}}
			if err := codec.Send(envelope); err != nil {
				t.Error(err)
			}
		}()
	}
	wait.Wait()

	if len(stream.writes) != senders {
		t.Fatalf("%d Write calls for %d envelopes: a frame was split", len(stream.writes), senders)
	}
	seen := make(map[uint64]int, senders)
	for _, frame := range stream.writes {
		single := NewCodec(bytes.NewBuffer(frame))
		envelope, err := single.Receive()
		if err != nil {
			t.Fatalf("a recorded frame did not decode: %v", err)
		}
		if _, err := single.Receive(); !errors.Is(err, io.EOF) {
			t.Fatal("a single Write carried more than one frame")
		}
		seen[envelope.GetSeq()] = len(envelope.GetFail().GetReason())
	}
	if len(seen) != senders {
		t.Fatalf("recovered %d distinct envelopes, want %d", len(seen), senders)
	}
	for seq := uint64(1); seq <= senders; seq++ {
		if want := int(seq) * 64; seen[seq] != want {
			t.Fatalf("envelope %d carried %d bytes of payload, want %d", seq, seen[seq], want)
		}
	}
}
