package network

import (
	"context"
	"errors"
	"log/slog"
	"net"
)

// HandlerFunc is the callback invoked in a new goroutine for each accepted connection.
type HandlerFunc func(conn *ClientConn)

// Listener wraps a net.Listener and dispatches accepted connections to a handler.
type Listener struct {
	addr    string
	handler HandlerFunc
	ln      net.Listener
}

// NewListener creates a Listener that will accept connections on addr.
func NewListener(addr string, handler HandlerFunc) *Listener {
	return &Listener{
		addr:    addr,
		handler: handler,
	}
}

// Listen binds to the configured address and blocks, accepting connections
// until ctx is cancelled or the listener is closed.
func (l *Listener) Listen(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", l.addr)
	if err != nil {
		return err
	}
	l.ln = ln

	slog.Info("listening for connections", "addr", ln.Addr())

	// Close the listener when the context is cancelled so that Accept unblocks.
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		tc, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil // clean shutdown
			}
			// Transient accept errors (e.g. "too many open files") — log and continue.
			slog.Warn("accept error", "err", err)
			continue
		}

		conn := NewClientConn(tc)
		slog.Debug("new connection", "remote", tc.RemoteAddr())

		go func() {
			defer func() {
				_ = conn.Close()
				slog.Debug("connection closed", "remote", conn.RemoteAddr())
			}()
			l.handler(conn)
		}()
	}
}

// Close stops the listener immediately.
func (l *Listener) Close() error {
	if l.ln != nil {
		return l.ln.Close()
	}
	return nil
}
