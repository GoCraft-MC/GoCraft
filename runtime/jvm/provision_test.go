package jvm

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// probeReturning answers from a table keyed by path, so a test can describe a
// machine with several JDKs on it without any of them existing. Nothing in this
// package's suite runs a JVM: §16 keeps the Java toolchain out of this
// repository's CI, and a test that spawned one would put it back.
func probeReturning(versions map[string]int) func(context.Context, string) (int, error) {
	return func(_ context.Context, java string) (int, error) {
		version, ok := versions[java]
		if !ok {
			return 0, fmt.Errorf("no such file")
		}
		return version, nil
	}
}

func TestParseJavaVersion(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		output string
		want   int
		fails  bool
	}{
		{
			name:   "modern",
			output: "openjdk version \"25.0.3\" 2026-01-20\nOpenJDK Runtime Environment\n",
			want:   25,
		},
		{
			name:   "oracle",
			output: "java version \"25.0.1\" 2025-10-21 LTS\n",
			want:   25,
		},
		{
			name:   "no minor",
			output: "openjdk version \"21\" 2023-09-19\n",
			want:   21,
		},
		{
			// 1.8.0_202 is Java 8. Reading the major component naively gives 1,
			// and an admin told "Java 1 is too old" looks in the wrong place.
			name:   "legacy 1.x scheme",
			output: "java version \"1.8.0_202\"\nJava(TM) SE Runtime Environment\n",
			want:   8,
		},
		{
			name:   "early access",
			output: "openjdk version \"26-ea\" 2026-03-17\n",
			want:   26,
		},
		{
			name:   "not java at all",
			output: "bash: java: command not found\n",
			fails:  true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			version, err := parseJavaVersion(testCase.output)
			if testCase.fails {
				if err == nil {
					t.Fatalf("parseJavaVersion(%q) = %d, want an error", testCase.output, version)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseJavaVersion(%q) = %v", testCase.output, err)
			}
			if version != testCase.want {
				t.Fatalf("parseJavaVersion(%q) = %d, want %d", testCase.output, version, testCase.want)
			}
		})
	}
}

// An admin who wrote a path in the configuration meant that one; JAVA_HOME is
// often set by something else entirely.
func TestJavaCandidatesPrefersConfigurationOverTheEnvironment(t *testing.T) {
	t.Setenv("JAVA_HOME", filepath.FromSlash("/opt/jdk21"))
	t.Setenv("PATH", "")

	runtime := New(Config{JavaPath: filepath.FromSlash("/opt/jdk25/bin/java")})
	candidates := runtime.javaCandidates()

	if len(candidates) != 2 {
		t.Fatalf("javaCandidates() = %v, want the configured path then JAVA_HOME", candidates)
	}
	if candidates[0] != filepath.FromSlash("/opt/jdk25/bin/java") {
		t.Fatalf("javaCandidates()[0] = %q, want the configured path first", candidates[0])
	}
	if candidates[1] != filepath.Join(filepath.FromSlash("/opt/jdk21"), "bin", javaExecutable()) {
		t.Fatalf("javaCandidates()[1] = %q, want JAVA_HOME second", candidates[1])
	}
}

func TestJavaCandidatesDeduplicates(t *testing.T) {
	home := filepath.FromSlash("/opt/jdk25")
	t.Setenv("JAVA_HOME", home)
	t.Setenv("PATH", "")

	runtime := New(Config{JavaPath: filepath.Join(home, "bin", javaExecutable())})
	if candidates := runtime.javaCandidates(); len(candidates) != 1 {
		t.Fatalf("javaCandidates() = %v, want one entry for one binary", candidates)
	}
}

func TestDetectSystemSkipsAJDKThatIsTooOld(t *testing.T) {
	home := filepath.FromSlash("/opt/jdk21")
	old := filepath.Join(home, "bin", javaExecutable())
	t.Setenv("JAVA_HOME", home)
	t.Setenv("PATH", "")

	runtime := New(Config{Probe: probeReturning(map[string]int{old: 21})})

	if _, err := runtime.detectSystem(t.Context()); err == nil {
		t.Fatal("detectSystem() accepted a JDK below the baseline")
	} else if !strings.Contains(err.Error(), "Java 21") ||
		!strings.Contains(err.Error(), fmt.Sprint(minimumJavaVersion)) {
		t.Fatalf("detectSystem() error = %v, want both versions named", err)
	}
}

func TestDetectSystemTakesTheFirstSuitableCandidate(t *testing.T) {
	old := filepath.FromSlash("/opt/jdk21/bin/" + javaExecutable())
	current := filepath.FromSlash("/opt/jdk25/bin/" + javaExecutable())
	t.Setenv("JAVA_HOME", filepath.FromSlash("/opt/jdk25"))
	t.Setenv("PATH", "")

	runtime := New(Config{
		JavaPath: old,
		Probe:    probeReturning(map[string]int{old: 21, current: 25}),
	})
	found, err := runtime.detectSystem(t.Context())
	if err != nil {
		t.Fatalf("detectSystem() = %v", err)
	}
	if found != current {
		t.Fatalf("detectSystem() = %q, want the JDK that meets the baseline", found)
	}
}

func TestDetectSystemReportsThatNothingWasFound(t *testing.T) {
	t.Setenv("JAVA_HOME", "")
	t.Setenv("PATH", "")

	_, err := New(Config{}).detectSystem(t.Context())
	if !errors.Is(err, errNoJavaFound) {
		t.Fatalf("detectSystem() = %v, want errNoJavaFound", err)
	}
}

func TestProvisionAcceptsASystemJDK(t *testing.T) {
	java := filepath.FromSlash("/opt/jdk25/bin/" + javaExecutable())
	runtime := New(Config{
		JavaPath:     java,
		PreferSystem: true,
		Probe:        probeReturning(map[string]int{java: 25}),
	})
	if err := runtime.Provision(t.Context(), nil); err != nil {
		t.Fatalf("Provision() = %v", err)
	}
	if runtime.Java() != java {
		t.Fatalf("Java() = %q, want the system JDK", runtime.Java())
	}
}

func TestProvisionDoesNotReplaceAnInvalidConfiguredJava(t *testing.T) {
	configured := filepath.FromSlash("/opt/jdk21/bin/" + javaExecutable())
	system := filepath.FromSlash("/opt/jdk25/bin/" + javaExecutable())
	t.Setenv("JAVA_HOME", filepath.FromSlash("/opt/jdk25"))
	t.Setenv("PATH", "")
	runtime := New(Config{
		JavaPath: configured, PreferSystem: true,
		Probe: probeReturning(map[string]int{configured: 21, system: 25}),
	})

	err := runtime.Provision(t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), "Java 21") {
		t.Fatalf("Provision() = %v, want the configured JDK rejected", err)
	}
	if runtime.Java() != "" {
		t.Fatalf("Java() = %q after rejecting the configured JDK", runtime.Java())
	}
}

// The failure an admin actually hits today: no system JDK, and a pin table with
// no rows in it. One message has to carry both halves, or it becomes two
// support questions.
func TestProvisionExplainsBothHalvesOfTheFailure(t *testing.T) {
	t.Setenv("JAVA_HOME", "")
	t.Setenv("PATH", "")

	runtime := New(Config{PreferSystem: true})
	err := runtime.Provision(t.Context(), nil)
	if err == nil {
		t.Fatal("Provision() succeeded with no JDK anywhere")
	}
	if !strings.Contains(err.Error(), "no java binary was found") {
		t.Fatalf("Provision() error = %v, want the system search reported", err)
	}
	if !strings.Contains(err.Error(), "no JDK pinned for") {
		t.Fatalf("Provision() error = %v, want the missing pin reported", err)
	}
	if runtime.Java() != "" {
		t.Fatalf("Java() = %q after a failed Provision", runtime.Java())
	}
}

// An explicit path is more authoritative than the provisioning preference.
func TestProvisionUsesConfiguredJavaWhenSystemIsNotPreferred(t *testing.T) {
	java := filepath.FromSlash("/opt/jdk25/bin/" + javaExecutable())
	runtime := New(Config{
		JavaPath:     java,
		PreferSystem: false,
		Probe:        probeReturning(map[string]int{java: 25}),
	})
	if err := runtime.Provision(t.Context(), nil); err != nil {
		t.Fatalf("Provision() = %v", err)
	}
	if runtime.Java() != java {
		t.Fatalf("Java() = %q, want the configured JDK", runtime.Java())
	}
}
