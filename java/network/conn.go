// Package network manages raw TCP connections to Minecraft Java Edition clients.
package network

import (
	"bufio"
	"crypto/cipher"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"GoCraft/java/auth"
	"GoCraft/java/protocol"
)

// State represents the connection's current protocol phase.
type State int

const (
	// StateHandshaking is the initial state: the client has connected but has
	// not yet indicated whether it wants to ping the server or log in.
	StateHandshaking State = iota
	// StateStatus means the client is performing a server-list ping.
	StateStatus
	// StateLogin means the client has started the login sequence.
	StateLogin
	// StateConfiguration means the client is in the post-login configuration phase.
	StateConfiguration
	// StatePlay means the client is in active gameplay.
	StatePlay
)

const (
	// readDeadline is the maximum idle time before a connection is closed.
	readDeadline = 30 * time.Second
	// writeDeadline is the maximum time allowed to flush a packet to the client.
	writeDeadline = 10 * time.Second
	// readBufSize is the size of the buffered reader per connection.
	readBufSize = 8192
)

// ClientConn wraps a raw TCP connection with Minecraft protocol helpers.
//
// Reading is owned by a single goroutine (no concurrent reads). Writing is
// protected by writeMu so that the login handler and other concurrent senders
// do not interleave bytes on the wire.
//
// After EnableEncryption is called, all reads go through an AES-128-CFB8
// decrypter and all writes go through an AES-128-CFB8 encrypter.
type ClientConn struct {
	conn   net.Conn
	reader *bufio.Reader
	writer io.Writer // net.Conn before encryption, encryptedWriter after

	State State

	writeMu   sync.Mutex
	closeOnce sync.Once
}

// NewClientConn wraps conn and returns a ClientConn ready to use.
func NewClientConn(conn net.Conn) *ClientConn {
	return &ClientConn{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, readBufSize),
		writer: conn,
		State:  StateHandshaking,
	}
}

// RemoteAddr returns the remote address of the underlying TCP connection.
func (c *ClientConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// ReadPacket reads the next packet from the connection.
// It sets a read deadline before each read to prevent slow-loris attacks.
func (c *ClientConn) ReadPacket() (*protocol.Packet, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		return nil, fmt.Errorf("conn: setting read deadline: %w", err)
	}
	return protocol.ReadPacket(c.reader)
}

// WritePacket serialises pkt and sends it to the client.
// The write mutex ensures concurrent callers do not interleave bytes.
func (c *ClientConn) WritePacket(pkt *protocol.Packet) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
		return fmt.Errorf("conn: setting write deadline: %w", err)
	}
	return protocol.WritePacket(c.writer, pkt)
}

// EnableEncryption switches the connection into AES-128-CFB8 encrypted mode.
//
// sharedSecret is the 16-byte AES key negotiated during login. Minecraft uses
// the same value for both the cipher key and the IV (initial vector).
//
// Any bytes the bufio.Reader had already buffered from the socket (which are
// encrypted data the client sent ahead of time) are preserved and fed through
// the decrypter transparently.
func (c *ClientConn) EnableEncryption(sharedSecret []byte) error {
	encStream, err := auth.NewCFB8Encrypter(sharedSecret, sharedSecret)
	if err != nil {
		return fmt.Errorf("conn: creating encrypter: %w", err)
	}
	decStream, err := auth.NewCFB8Decrypter(sharedSecret, sharedSecret)
	if err != nil {
		return fmt.Errorf("conn: creating decrypter: %w", err)
	}

	// Drain any bytes the bufio.Reader has already pulled from the socket.
	// These bytes arrived over the wire already encrypted and must go through
	// the decrypter just like future socket reads.
	buffered := make([]byte, c.reader.Buffered())
	if len(buffered) > 0 {
		if _, err := io.ReadFull(c.reader, buffered); err != nil {
			return fmt.Errorf("conn: draining read buffer: %w", err)
		}
	}

	decReader := &encryptedReader{
		conn:    c.conn,
		stream:  decStream,
		pending: buffered,
	}

	c.reader = bufio.NewReaderSize(decReader, readBufSize)
	c.writer = &encryptedWriter{conn: c.conn, stream: encStream}
	return nil
}

// Close closes the underlying TCP connection exactly once.
func (c *ClientConn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.conn.Close()
	})
	return err
}

// ─── encrypted I/O helpers ───────────────────────────────────────────────────

// encryptedReader wraps a net.Conn with a CFB8 decrypter.
// It first serves any bytes that were buffered before encryption was enabled.

type encryptedReader struct {
	conn    net.Conn
	stream  cipher.Stream
	pending []byte // encrypted bytes already read from the socket
}

func (r *encryptedReader) Read(p []byte) (int, error) {
	var n int
	if len(r.pending) > 0 {
		n = copy(p, r.pending)
		r.stream.XORKeyStream(p[:n], p[:n])
		r.pending = r.pending[n:]
		return n, nil
	}
	n, err := r.conn.Read(p)
	if n > 0 {
		r.stream.XORKeyStream(p[:n], p[:n])
	}
	return n, err
}

// encryptedWriter wraps a net.Conn with a CFB8 encrypter.
// It encrypts each Write in place before sending to the socket.
type encryptedWriter struct {
	conn   net.Conn
	stream cipher.Stream
}

func (w *encryptedWriter) Write(p []byte) (int, error) {
	// Allocate a separate buffer so we do not mutate the caller's slice.
	encrypted := make([]byte, len(p))
	w.stream.XORKeyStream(encrypted, p)
	return w.conn.Write(encrypted)
}
