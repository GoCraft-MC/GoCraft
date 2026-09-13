package consolelog

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func record(level slog.Level, msg string, attrs ...slog.Attr) slog.Record {
	r := slog.NewRecord(time.Date(2026, 9, 13, 23, 23, 19, 0, time.UTC), level, msg, 0)
	for _, a := range attrs {
		r.AddAttrs(a)
	}
	return r
}

func TestFormatLinePlain(t *testing.T) {
	tests := []struct {
		name   string
		record slog.Record
		attrs  []slog.Attr
		want   string
	}{
		{
			name:   "info with attributes",
			record: record(slog.LevelInfo, "java listener enabled", slog.String("addr", "0.0.0.0:25565")),
			want:   "[23:23:19 INFO]: Java listener enabled addr=0.0.0.0:25565\n",
		},
		{
			name:   "warn level label",
			record: record(slog.LevelWarn, "plugins: unclean shutdown", slog.Any("err", errors.New("boom"))),
			want:   "[23:23:19 WARN]: Plugins: unclean shutdown err=boom\n",
		},
		{
			name:   "error message with spaces is quoted",
			record: record(slog.LevelError, "server stopped", slog.String("err", "bind: address already in use")),
			want:   "[23:23:19 ERROR]: Server stopped err=\"bind: address already in use\"\n",
		},
		{
			name:   "preformatted attrs are appended",
			record: record(slog.LevelInfo, "plugins: loaded", slog.Int("count", 3)),
			attrs:  []slog.Attr{slog.String("directory", "plugins")},
			want:   "[23:23:19 INFO]: Plugins: loaded directory=plugins count=3\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(formatLine(tt.record, tt.attrs, false))
			if got != tt.want {
				t.Fatalf("formatLine mismatch\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestFormatLineColoredHasNoAnsiInFile(t *testing.T) {
	r := record(slog.LevelWarn, "plugins: unclean shutdown", slog.Any("err", errors.New("boom")))

	plain := string(formatLine(r, nil, false))
	colored := string(formatLine(r, nil, true))
	if plain == colored {
		t.Fatal("coloured output should differ from plain output")
	}
	if !strings.Contains(colored, ansiYellow) || !strings.Contains(colored, ansiReset) {
		t.Errorf("coloured warn line missing ANSI sequences: %q", colored)
	}
	if strings.Contains(plain, "\x1b") {
		t.Errorf("plain line must not contain ANSI codes: %q", plain)
	}
}

func TestHandlerWritesToConsoleAndFile(t *testing.T) {
	var console, file bytes.Buffer
	h := New(&console, Options{Color: true, File: &file})
	logger := slog.New(h)

	logger.Warn("could not save player data", slog.String("uuid", "abc"), slog.Any("err", errors.New("disk full")))

	if strings.Contains(file.String(), "\x1b") {
		t.Errorf("file sink must stay plain, got %q", file.String())
	}
	if !strings.Contains(console.String(), ansiYellow) {
		t.Errorf("console sink should be coloured, got %q", console.String())
	}
	wantFile := "["
	if !strings.HasPrefix(file.String(), wantFile) {
		t.Errorf("file line should start with timestamp bracket, got %q", file.String())
	}
	if !strings.Contains(file.String(), "Could not save player data uuid=abc err=disk full") {
		t.Errorf("file line should carry message and attributes, got %q", file.String())
	}
}

func TestHandlerLevelFilter(t *testing.T) {
	var console bytes.Buffer
	h := New(&console, Options{})
	logger := slog.New(h)

	if logger.Enabled(nil, slog.LevelDebug) {
		t.Error("debug should be disabled by default")
	}
	logger.Debug("hidden")
	logger.Info("shown")
	if strings.Contains(console.String(), "hidden") {
		t.Error("debug record should have been filtered out")
	}
	if !strings.Contains(console.String(), "Shown") {
		t.Error("info record should have been written")
	}
}

func TestWithAttrsCarriedToLines(t *testing.T) {
	var console bytes.Buffer
	logger := slog.New(New(&console, Options{})).With(slog.String("world", "overworld"))

	logger.Info("dimension worlds ready")
	if !strings.Contains(console.String(), "world=overworld") {
		t.Errorf("With attributes should appear on every line, got %q", console.String())
	}
}
