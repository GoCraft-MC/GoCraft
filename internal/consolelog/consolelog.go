// Package consolelog renders Paper-style console log lines.
package consolelog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"unicode"
	"unicode/utf8"
)

type Options struct {
	Level slog.Leveler
	Color bool
	File  io.Writer
}

type Handler struct {
	opts    Options
	console io.Writer
	mu      *sync.Mutex
	attrs   []slog.Attr
	groups  string
}

// New returns a Handler writing to console and, when set, opts.File.
func New(console io.Writer, opts Options) *Handler {
	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}
	return &Handler{
		opts:    opts,
		console: console,
		mu:      &sync.Mutex{},
	}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var firstErr error
	sinks := []struct {
		out     io.Writer
		colored bool
	}{{h.console, h.opts.Color}, {h.opts.File, false}}
	for _, sink := range sinks {
		if sink.out == nil {
			continue
		}
		if _, err := sink.out.Write(formatLine(r, h.attrs, h.groups, sink.colored)); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	flat := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		flat = append(flat, flattenAttr(h.groups, a)...)
	}
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), flat...)
	return &clone
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	if clone.groups == "" {
		clone.groups = name
	} else {
		clone.groups += "." + name
	}
	return &clone
}

const (
	ansiReset  = "\x1b[0m"
	ansiGray   = "\x1b[90m"
	ansiCyan   = "\x1b[36m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
)

func formatLine(r slog.Record, preformatted []slog.Attr, groups string, colored bool) []byte {
	line := make([]byte, 0, 128)

	line = append(line, '[')
	line = append(line, r.Time.Format("15:04:05")...)
	line = append(line, ' ')

	level := r.Level.Level()
	var label string
	switch {
	case level < slog.LevelInfo:
		label = "DEBUG"
	case level < slog.LevelWarn:
		label = "INFO"
	case level < slog.LevelError:
		label = "WARN"
	default:
		label = "ERROR"
	}
	if colored {
		line = append(line, levelColor(level)...)
	}
	line = append(line, label...)
	if colored {
		line = append(line, ansiReset...)
	}
	line = append(line, ']', ':', ' ')

	if colored && level >= slog.LevelWarn {
		line = append(line, levelColor(level)...)
	}
	line = appendCapitalized(line, r.Message)
	if colored && level >= slog.LevelWarn {
		line = append(line, ansiReset...)
	}

	for _, a := range preformatted {
		line = appendAttr(line, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		for _, fa := range flattenAttr(groups, a) {
			line = appendAttr(line, fa)
		}
		return true
	})
	return append(line, '\n')
}

func flattenAttr(prefix string, a slog.Attr) []slog.Attr {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return nil
	}
	if prefix != "" {
		a.Key = prefix + "." + a.Key
	}
	if a.Value.Kind() != slog.KindGroup {
		return []slog.Attr{a}
	}
	group := a.Value.Group()
	if a.Key != "" {
		if prefix != "" {
			prefix = prefix + "." + a.Key
		} else {
			prefix = a.Key
		}
	}
	flat := make([]slog.Attr, 0, len(group))
	for _, ga := range group {
		flat = append(flat, flattenAttr(prefix, ga)...)
	}
	return flat
}

func appendAttr(line []byte, a slog.Attr) []byte {
	line = append(line, ' ')
	line = appendKey(line, a.Key)
	line = append(line, '=')
	return appendValue(line, a.Value.Resolve())
}

func appendCapitalized(line []byte, s string) []byte {
	if s == "" {
		return line
	}
	r, size := utf8.DecodeRuneInString(s)
	line = append(line, string(unicode.ToUpper(r))...)
	return append(line, s[size:]...)
}

func levelColor(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return ansiGray
	case level < slog.LevelWarn:
		return ansiCyan
	case level < slog.LevelError:
		return ansiYellow
	default:
		return ansiRed
	}
}

func appendKey(line []byte, key string) []byte {
	if needsQuoting(key) {
		return strconv.AppendQuote(line, key)
	}
	return append(line, key...)
}

func appendValue(line []byte, v slog.Value) []byte {
	switch v.Kind() {
	case slog.KindString:
		return appendString(line, v.String())
	case slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool, slog.KindDuration, slog.KindTime:
		return append(line, v.String()...)
	default:
		if err, ok := v.Any().(error); ok {
			return appendString(line, err.Error())
		}
		if s, ok := v.Any().(fmt.Stringer); ok {
			return appendString(line, s.String())
		}
		return appendString(line, fmt.Sprintf("%+v", v.Any()))
	}
}

func appendString(line []byte, s string) []byte {
	if needsQuoting(s) {
		return strconv.AppendQuote(line, s)
	}
	return append(line, s...)
}

func needsQuoting(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] <= ' ' || s[i] == '"' || s[i] == '=' || s[i] == 0x7f {
			return true
		}
	}
	return false
}
