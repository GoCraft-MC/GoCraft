package server

import (
	"strings"
	"testing"
	"time"
)

func TestTimingsReportIncludesRuntimeAndPlayerStats(t *testing.T) {
	timings := newTickTimings(func() (int, int) { return 3, 20 })
	timings.commit(10 * time.Millisecond)
	report := timings.Report()
	for _, expected := range []string{`RAM:`, `CPU:`, `Players:`, `3/20`} {
		if !strings.Contains(report, expected) {
			t.Errorf(`timings report does not contain %q: %s`, expected, report)
		}
	}
	if plain := stripMinecraftFormatting(report); strings.ContainsRune(plain, '§') {
		t.Fatalf(`plain console report still contains formatting: %q`, plain)
	}
}

func TestMSPTReportsPaperStyleRollingWindows(t *testing.T) {
	timings := newTickTimings()
	for tick := 0; tick < 1200; tick++ {
		timings.commit(time.Duration(10+tick%3) * time.Millisecond)
	}
	report := timings.MSPT()
	for _, expected := range []string{"avg/min/max", "5s:", "10s:", "1m:"} {
		if !strings.Contains(report, expected) {
			t.Errorf("MSPT report does not contain %q: %s", expected, report)
		}
	}
	if !strings.Contains(report, "§a") {
		t.Fatalf("healthy MSPT report is not green: %s", report)
	}
}

func TestMSPTReportsMissingSamples(t *testing.T) {
	if report := newTickTimings().MSPT(); !strings.Contains(report, "No tick-time data") {
		t.Fatalf("empty MSPT report = %q", report)
	}
}
