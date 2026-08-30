package ipc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// maximumSocketPath is what AF_UNIX accepts: the POSIX sun_path[108] minus its
// terminator. Windows applies the same limit, measured — 107 binds, 108 fails.
//
// The kernel's own refusal is "bind: invalid argument", which names neither the
// limit nor the path, so the length is checked here instead.
const maximumSocketPath = 107

// staleProbeTimeout bounds the dial that tells a leftover socket file from one a
// live server is listening on. It is a local connect that either answers at once
// or not at all.
const staleProbeTimeout = 250 * time.Millisecond

// SocketPath builds the path a runtime will connect back on.
//
// Short by design: the directory is the caller's, but the name is not, and the
// budget is only 107 bytes for both. Including the process id keeps two servers
// on one machine from picking the same path.
func SocketPath(directory, runtime string) (string, error) {
	path := filepath.Join(directory, fmt.Sprintf("gc-%s-%d.sock", runtime, os.Getpid()))
	if len(path) > maximumSocketPath {
		return "", fmt.Errorf(
			"ipc: socket path %q is %d bytes, over the %d byte limit; use a shorter directory than %q",
			path, len(path), maximumSocketPath, directory)
	}
	return path, nil
}

// Listen opens the socket, clearing a leftover file from a host that was killed
// before it could clean up.
//
// A stale file is not distinguishable from a live one by looking at it, so the
// path is dialled instead: something that answers is a running server and its
// socket is left alone. Only a path that binds nothing is removed — stealing a
// live socket would take a working runtime away from another server.
func Listen(path string) (net.Listener, error) {
	if len(path) > maximumSocketPath {
		return nil, fmt.Errorf("ipc: socket path %q is %d bytes, over the %d byte limit",
			path, len(path), maximumSocketPath)
	}
	listener, err := net.Listen("unix", path)
	if err == nil {
		return listener, nil
	}
	if _, statErr := os.Stat(path); statErr != nil {
		// Nothing in the way: the failure is about something else entirely.
		return nil, fmt.Errorf("ipc: listen on %s: %w", path, err)
	}
	if probe, dialErr := net.DialTimeout("unix", path, staleProbeTimeout); dialErr == nil {
		probe.Close()
		return nil, fmt.Errorf("ipc: %s is already served by a live process", path)
	}
	if removeErr := os.Remove(path); removeErr != nil {
		return nil, fmt.Errorf("ipc: remove stale socket %s: %w", path, removeErr)
	}
	listener, err = net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("ipc: listen on %s after clearing a stale socket: %w", path, err)
	}
	return listener, nil
}
