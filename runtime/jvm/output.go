package jvm

import (
	"bytes"
	"context"
	"io"
	"log/slog"
)

// maximumLogLine bounds one line from the child.
//
// A JVM stack trace is long but not unbounded; a plugin printing a megabyte
// with no newline is a bug, and buffering it whole would make the server pay
// for it. Past this the line is cut and logged, so the output is truncated
// rather than the memory unbounded.
const maximumLogLine = 8 << 10

// logWriter turns a child process's output into server log lines.
//
// Without it the JVM inherits the server's own stdout, and that only looks
// right: slog writes to a MultiWriter over stdout *and* latest.log, but the
// child writes to the file descriptor directly. Its output reaches the console
// and never the log file — so a plugin's message is there while an admin is
// watching and gone when they go looking.
//
// Going through slog also gets attribution and a level for free, which a raw
// pipe cannot have.
type logWriter struct {
	level slog.Level
	// pending holds a line the child has not finished writing. A pipe splits
	// wherever it likes, so a write is not a line.
	pending []byte
}

func (w *logWriter) Write(p []byte) (int, error) {
	written := len(p)
	w.pending = append(w.pending, p...)
	for {
		index := bytes.IndexByte(w.pending, '\n')
		if index < 0 {
			break
		}
		w.emit(w.pending[:index])
		w.pending = w.pending[index+1:]
	}
	// A child that never writes a newline must not grow this forever.
	if len(w.pending) > maximumLogLine {
		w.emit(w.pending[:maximumLogLine])
		w.pending = w.pending[:0]
	}
	return written, nil
}

// emit logs one line, with the plugin's own text as the message.
//
// The text is what an admin is looking for, so it goes where they will read it
// rather than into an attribute they have to unpack. A blank line is dropped:
// the JVM emits them and they carry nothing.
func (w *logWriter) emit(line []byte) {
	text := string(bytes.TrimRight(line, "\r"))
	if len(bytes.TrimSpace([]byte(text))) == 0 {
		return
	}
	slog.Log(context.Background(), w.level, text, "runtime", RuntimeName)
}

// outputs returns where the child's two streams go.
//
// Configured writers win, which is what lets a test capture them. Otherwise
// stdout becomes INFO and stderr WARN — the conventional split, and the one
// that puts a stack trace where someone will notice it.
//
// A line the child never terminated is lost when it exits. exec closes the pipe
// and does not call Close on a writer, and inventing a lifecycle for that would
// cost more than the last unterminated line of a process that is already gone.
func (r *Runtime) outputs() (stdout, stderr io.Writer) {
	stdout, stderr = r.config.Stdout, r.config.Stderr
	if stdout == nil {
		stdout = &logWriter{level: slog.LevelInfo}
	}
	if stderr == nil {
		stderr = &logWriter{level: slog.LevelWarn}
	}
	return stdout, stderr
}
