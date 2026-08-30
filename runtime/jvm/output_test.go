package jvm

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// captureLogs points the default logger at a buffer for one test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var captured bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&captured, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &captured
}

// The failure this exists for: a child writing to the server's own stdout
// reaches the console and never latest.log, because slog tees to the file and
// the child writes to the descriptor underneath it.
func TestChildOutputBecomesServerLogLines(t *testing.T) {
	captured := captureLogs(t)
	writer := &logWriter{level: slog.LevelInfo}

	writer.Write([]byte("[fr.oreo.hello] open for business\n"))

	logged := captured.String()
	if !strings.Contains(logged, "[fr.oreo.hello] open for business") {
		t.Fatalf("logged = %q, want the plugin's own text as the message", logged)
	}
	if !strings.Contains(logged, "runtime=jvm") {
		t.Fatalf("logged = %q, want the runtime named", logged)
	}
	if !strings.Contains(logged, "level=INFO") {
		t.Fatalf("logged = %q, want stdout at INFO", logged)
	}
}

// A pipe splits wherever it likes, so a write is not a line. Reassembling is
// the whole job.
func TestPartialWritesBecomeOneLine(t *testing.T) {
	captured := captureLogs(t)
	writer := &logWriter{level: slog.LevelInfo}

	writer.Write([]byte("half a "))
	if captured.Len() != 0 {
		t.Fatalf("logged %q before the line was finished", captured.String())
	}
	writer.Write([]byte("line\nand another\n"))

	logged := captured.String()
	if !strings.Contains(logged, "half a line") {
		t.Fatalf("logged = %q, want the halves joined", logged)
	}
	if strings.Count(logged, "msg=") != 2 {
		t.Fatalf("logged = %q, want exactly two lines", logged)
	}
}

// A JVM on Windows ends its lines with CRLF, and a stray carriage return in a
// log file is the kind of thing that makes grep miss.
func TestCarriageReturnsAreStripped(t *testing.T) {
	captured := captureLogs(t)
	writer := &logWriter{level: slog.LevelWarn}

	writer.Write([]byte("a warning\r\n"))

	if strings.Contains(captured.String(), "\r") {
		t.Fatalf("logged = %q, want no carriage return", captured.String())
	}
	if !strings.Contains(captured.String(), "level=WARN") {
		t.Fatalf("logged = %q, want stderr at WARN", captured.String())
	}
}

// The JVM emits blank lines and they carry nothing.
func TestBlankLinesAreDropped(t *testing.T) {
	captured := captureLogs(t)
	writer := &logWriter{level: slog.LevelInfo}

	writer.Write([]byte("\n\n   \n"))

	if captured.Len() != 0 {
		t.Fatalf("logged %q for blank lines", captured.String())
	}
}

// A plugin printing without ever writing a newline must not grow the buffer
// until the server runs out of memory.
func TestAnUnterminatedFloodIsCut(t *testing.T) {
	captured := captureLogs(t)
	writer := &logWriter{level: slog.LevelInfo}

	writer.Write(bytes.Repeat([]byte("x"), maximumLogLine*3))

	if captured.Len() == 0 {
		t.Fatal("nothing was logged for a flood with no newline")
	}
	if len(writer.pending) > maximumLogLine {
		t.Fatalf("pending = %d bytes, want the buffer bounded", len(writer.pending))
	}
}

// A configured writer wins, which is what lets a test capture the raw stream.
func TestConfiguredWritersWin(t *testing.T) {
	var out, errs bytes.Buffer
	runtime := New(Config{Stdout: &out, Stderr: &errs})

	stdout, stderr := runtime.outputs()

	if stdout != &out || stderr != &errs {
		t.Fatal("outputs() ignored the configured writers")
	}
}
