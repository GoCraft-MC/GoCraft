package ipc

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"google.golang.org/protobuf/proto"

	wire "GoCraft/abi/v1/wire"
)

// maximumFrameSize bounds what a peer may declare it is about to send. Without
// it, a runtime announcing a four-gigabyte frame would have the host allocate
// four gigabytes before discovering the lie.
//
// Envelopes carry events, verdicts and paths — never a bundle — so the limit is
// generous by a wide margin.
const maximumFrameSize = 4 << 20

// Codec frames envelopes on a byte stream: a varint length, then the encoded
// envelope. A stream carries no message boundaries of its own, so the length is
// what tells the reader where one envelope ends and the next begins.
//
// Reads are single-consumer by contract — one read loop owns the stream. Writes
// are serialised here, because two goroutines writing at once would interleave
// their bytes and corrupt every frame that follows.
type Codec struct {
	reader *bufio.Reader

	mu      sync.Mutex
	writer  io.Writer
	payload []byte
	frame   []byte
}

func NewCodec(stream io.ReadWriter) *Codec {
	return &Codec{reader: bufio.NewReader(stream), writer: stream}
}

// Send writes one envelope. Header and payload go out in a single Write: a
// header that reached the peer without its payload would desynchronise the
// stream permanently, and there is no way to take it back.
func (c *Codec) Send(envelope *wire.Envelope) error {
	if envelope == nil {
		return fmt.Errorf("ipc: missing envelope")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	payload, err := proto.MarshalOptions{}.MarshalAppend(c.payload[:0], envelope)
	if err != nil {
		return fmt.Errorf("ipc: encode envelope: %w", err)
	}
	c.payload = payload
	if len(payload) > maximumFrameSize {
		return fmt.Errorf("ipc: envelope of %d bytes exceeds the %d byte limit", len(payload), maximumFrameSize)
	}
	c.frame = append(binary.AppendUvarint(c.frame[:0], uint64(len(payload))), payload...)
	if _, err := c.writer.Write(c.frame); err != nil {
		return fmt.Errorf("ipc: write envelope: %w", err)
	}
	return nil
}

// Close closes the underlying stream when it can be closed, which is what
// unblocks a read loop parked in Receive. A stream with nothing to close — a
// buffer in a test — is not an error.
func (c *Codec) Close() error {
	if closer, ok := c.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Receive reads the next envelope.
//
// A clean io.EOF means the peer closed between frames, which is how an orderly
// shutdown ends and is not an error for the caller to log as one. Anything cut
// short mid-frame surfaces as io.ErrUnexpectedEOF instead, so the two cases stay
// distinguishable with errors.Is.
func (c *Codec) Receive() (*wire.Envelope, error) {
	length, err := binary.ReadUvarint(c.reader)
	if err != nil {
		return nil, err
	}
	if length > maximumFrameSize {
		return nil, fmt.Errorf("ipc: peer declared a %d byte frame, over the %d byte limit", length, maximumFrameSize)
	}
	frame := make([]byte, length)
	if _, err := io.ReadFull(c.reader, frame); err != nil {
		return nil, err
	}
	var envelope wire.Envelope
	if err := proto.Unmarshal(frame, &envelope); err != nil {
		return nil, fmt.Errorf("ipc: decode envelope: %w", err)
	}
	return &envelope, nil
}