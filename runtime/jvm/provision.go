package jvm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"GoCraft/core/plugin"
)

// Provision resolves the java binary this runtime will spawn, before any
// listener opens.
//
// System first, download second. An admin who already runs a JDK 25 should not
// get a second copy on disk, and provisioning is a step rather than an
// invisible side effect — which is the lesson both Gradle and uv arrived at
// after doing it the other way round.
//
// It never runs outside boot. If the JVM dies later and the cache has been
// purged in the meantime, fetching 45 MB while players are connected is not an
// improvement over leaving the runtime down until the next restart.
func (r *Runtime) Provision(ctx context.Context, provisioner plugin.Provisioner) error {
	if java := strings.TrimSpace(r.config.JavaPath); java != "" {
		version, err := r.probe(ctx, java)
		if err != nil {
			return fmt.Errorf("jvm: configured java %s: %w", java, err)
		}
		if version < minimumJavaVersion {
			return fmt.Errorf("jvm: configured java %s is Java %d, the runtime needs %d or newer",
				java, version, minimumJavaVersion)
		}
		r.setJava(java)
		return nil
	}
	if r.config.PreferSystem {
		java, err := r.detectSystem(ctx)
		if err == nil {
			r.setJava(java)
			return nil
		}
		return r.provisionJDK(ctx, provisioner, err)
	}
	return r.provisionJDK(ctx, provisioner, nil)
}

// provisionJDK downloads the pinned JDK. detected carries why the system search
// failed, so a single message tells the admin both what was looked at and what
// could not be fetched instead — two half-answers in two log lines is how this
// becomes a support ticket.
func (r *Runtime) provisionJDK(ctx context.Context, provisioner plugin.Provisioner, detected error) error {
	artifact, version, err := lookupPin(jdkPins, platformKey())
	if err != nil {
		return errors.Join(detected, err)
	}
	if provisioner == nil {
		return errors.Join(detected, fmt.Errorf(
			"jvm: automatic provisioning is unavailable in this build; install a JDK %d "+
				"and set JAVA_HOME, or set plugins runtime java_path", minimumJavaVersion))
	}
	key := "jdk-" + version + "-" + platformKey()
	directory, err := provisioner.Fetch(ctx, key, artifact)
	if err != nil {
		return errors.Join(detected, fmt.Errorf("jvm: fetch %s: %w", key, err))
	}
	java := filepath.Join(directory, filepath.FromSlash(artifact.Bin))
	// Extracted is not the same as runnable: a truncated archive, a missing
	// execute bit or a musl/glibc mismatch all survive extraction and only
	// surface when something tries to run it.
	if _, err := r.probe(ctx, java); err != nil {
		return fmt.Errorf("jvm: provisioned JDK at %s is not runnable: %w", java, err)
	}
	r.setJava(java)
	return nil
}

// errNoJavaFound is returned when nothing even looked like a java binary, as
// opposed to candidates that were found and rejected for their version.
var errNoJavaFound = errors.New("no java binary was found")

// detectSystem returns the first candidate that probes as a new enough JDK.
//
// Every rejection is reported, not just the last: an admin with JAVA_HOME on a
// Java 21 and a Java 25 further down PATH needs to see both verdicts to
// understand why the one they expected was not chosen.
func (r *Runtime) detectSystem(ctx context.Context) (string, error) {
	candidates := r.javaCandidates()
	if len(candidates) == 0 {
		return "", errNoJavaFound
	}
	rejected := make([]error, 0, len(candidates))
	for _, candidate := range candidates {
		version, err := r.probe(ctx, candidate)
		if err != nil {
			rejected = append(rejected, fmt.Errorf("%s: %w", candidate, err))
			continue
		}
		if version < minimumJavaVersion {
			rejected = append(rejected, fmt.Errorf(
				"%s: Java %d, the runtime needs %d or newer", candidate, version, minimumJavaVersion))
			continue
		}
		return candidate, nil
	}
	return "", errors.Join(rejected...)
}

// javaCandidates lists the java binaries worth probing, most authoritative
// first: what the admin configured, then what JAVA_HOME points at, then
// whatever PATH resolves.
//
// Configuration outranks the environment because it is the more deliberate of
// the two — an admin who wrote a path in server.yml meant that one, whereas
// JAVA_HOME is often set by something else entirely.
func (r *Runtime) javaCandidates() []string {
	var candidates []string
	add := func(path string) {
		if path == "" {
			return
		}
		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}

	add(strings.TrimSpace(r.config.JavaPath))
	if home := strings.TrimSpace(os.Getenv("JAVA_HOME")); home != "" {
		add(filepath.Join(home, "bin", javaExecutable()))
	}
	if found, err := exec.LookPath(javaExecutable()); err == nil {
		add(found)
	}
	return candidates
}

func javaExecutable() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}

// probe reports the major version of a candidate binary, running it when the
// configuration has not supplied a substitute.
func (r *Runtime) probe(ctx context.Context, java string) (int, error) {
	if r.config.Probe != nil {
		return r.config.Probe(ctx, java)
	}
	return probeJavaVersion(ctx, java)
}

// javaVersionPattern matches the first quoted version in `java -version`.
//
// Both spellings have to be handled. Modern releases print `openjdk version
// "25.0.3"`, and anything from the 8 era prints `java version "1.8.0_202"`,
// where the major version is the second component rather than the first. A
// server told "Java 1 is too old" would send its admin looking in the wrong
// place entirely.
var javaVersionPattern = regexp.MustCompile(`version "(\d+)(?:\.(\d+))?[^"]*"`)

func probeJavaVersion(ctx context.Context, java string) (int, error) {
	// -version writes to stderr, and has since long before anyone thought of
	// parsing it. CombinedOutput is what makes this work on every release.
	output, err := exec.CommandContext(ctx, java, "-version").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("running %s -version: %w", java, err)
	}
	return parseJavaVersion(string(output))
}

func parseJavaVersion(output string) (int, error) {
	match := javaVersionPattern.FindStringSubmatch(output)
	if match == nil {
		return 0, fmt.Errorf("no version in %q", strings.TrimSpace(firstLine(output)))
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("unreadable version %q", match[1])
	}
	if major == 1 {
		// The 1.x scheme: 1.8.0_202 is Java 8, and a bare 1 means nothing.
		if match[2] == "" {
			return 0, fmt.Errorf("unreadable legacy version %q", match[0])
		}
		return strconv.Atoi(match[2])
	}
	return major, nil
}

func firstLine(text string) string {
	if index := strings.IndexAny(text, "\r\n"); index >= 0 {
		return text[:index]
	}
	return text
}
