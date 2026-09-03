package link

import (
	"context"
	"errors"
	"fmt"
	"time"

	wire "github.com/GoCraft-MC/gocraft-abi/abi/v1/wire"
)

var (
	// ErrUnresponsive means the runtime held its socket open but stopped
	// answering. It was killed.
	ErrUnresponsive = errors.New("ipc: runtime stopped answering")

	// ErrProtocol means the runtime answered something it should not have. That
	// does not heal on its own, so it is reported at once rather than counted
	// against the heartbeat budget.
	ErrProtocol = errors.New("ipc: runtime broke the protocol")
)

// Liveness tunes the heartbeat. Zero values take the defaults from §13: one
// ping per second, three missed pongs before the runtime is killed.
type Liveness struct {
	Every   time.Duration
	Timeout time.Duration
	Missed  int
}

func (l Liveness) resolve() (every, timeout time.Duration, missed int) {
	every, timeout, missed = l.Every, l.Timeout, l.Missed
	if every <= 0 {
		every = time.Second
	}
	if timeout <= 0 {
		timeout = every
	}
	if missed <= 0 {
		missed = 3
	}
	return every, timeout, missed
}

// Watch pings the runtime until it stops answering, exits, or ctx ends.
//
// It exists for the failure the rest of this package cannot see: a process that
// is alive and holding its socket open, but no longer doing anything. Nothing
// else notices — Conn sees an open connection, the process never exits, and
// every event quietly burns its whole budget waiting for a verdict that is not
// coming. A crash is loud; this is not, which makes it worse.
//
// A ping is answered on the runtime's reader thread rather than by a handler,
// so a runtime busy running plugin code still pongs. Silence means it is stuck,
// not merely busy.
//
// Returns nil when ctx ends, which is the caller deciding to stop watching. Any
// other return means the runtime is gone.
func (c *Child) Watch(ctx context.Context, liveness Liveness) error {
	every, timeout, missed := liveness.resolve()
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	consecutive := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.exited:
			return fmt.Errorf("ipc: runtime exited: %w", c.exitErr)
		case <-c.conn.Done():
			return fmt.Errorf("ipc: connection lost: %w", c.conn.Err())
		case <-ticker.C:
		}

		pingCtx, cancel := context.WithTimeout(ctx, timeout)
		reply, err := c.conn.Request(pingCtx, &wire.Envelope{Body: &wire.Envelope_Ping{Ping: &wire.Ping{}}})
		cancel()

		switch {
		case err == nil && reply.GetPong() != nil:
			consecutive = 0
		case err == nil:
			return fmt.Errorf("%w: answered a ping with %T", ErrProtocol, reply.GetBody())
		case ctx.Err() != nil:
			// The caller cancelled us mid-ping; the runtime is not at fault.
			return nil
		case !errors.Is(err, context.DeadlineExceeded):
			return fmt.Errorf("ipc: heartbeat failed: %w", err)
		default:
			consecutive++
			if consecutive >= missed {
				// Killing is the point. A runtime that cannot answer a ping
				// cannot answer an event either, and leaving it running would
				// keep a socket, a process and a plugin set alive for nothing.
				kill(c.command)
				<-c.exited
				return fmt.Errorf("%w after %d missed pings", ErrUnresponsive, consecutive)
			}
		}
	}
}
