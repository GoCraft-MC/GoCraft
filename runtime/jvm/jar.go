package jvm

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// embeddedJar is gocraft-runtime.jar, carried inside the server binary.
//
// It is empty in this build. The jar is produced by the gocraft-java
// repository, which does not exist yet (deliverable 07), and the Go repository
// must never require a JVM to build or release — so the embed lives behind the
// gocraft_jvm_jar tag in jar_embedded.go and this file supplies the stub the
// default build compiles. ensureJar says exactly that when asked for a path.
//
// Nothing about this is a placeholder for the mechanism: extractJar below is
// the real thing, and filling this in is one tagged file plus the artifact.
var embeddedJar []byte

// ensureJar returns the path to the runtime jar, extracting it on first use.
func (r *Runtime) ensureJar() (string, error) {
	// An explicit path is what a developer building against a local
	// gocraft-java checkout uses, and it skips the cache entirely: extracting a
	// jar the admin already put on disk would only add a copy that can go
	// stale.
	if r.config.JarPath != "" {
		if _, err := os.Stat(r.config.JarPath); err != nil {
			return "", fmt.Errorf("jvm: configured runtime jar: %w", err)
		}
		return r.config.JarPath, nil
	}
	if len(embeddedJar) == 0 {
		return "", fmt.Errorf(
			"jvm: this build carries no runtime jar, so no download can fix it; " +
				"build with -tags gocraft_jvm_jar against a tree containing " +
				"gocraft-runtime.jar, or set the runtime's JarPath")
	}
	return extractJar(r.jarDirectory(), embeddedJar)
}

// extractJar writes the jar under a name derived from its own content hash.
//
// The name is the whole point. A jar dropped next to the executable, or fetched
// at runtime, reintroduces the most common support ticket for this
// architecture: somebody updates the server, keeps the old jar, and collects
// incomprehensible ABI mismatch errors. A content-addressed name cannot
// desynchronise from the binary that carries it — a different binary asks for a
// different file.
func extractJar(directory string, jar []byte) (string, error) {
	sum := sha256.Sum256(jar)
	path := filepath.Join(directory, fmt.Sprintf("gocraft-runtime-%x.jar", sum[:8]))

	if info, err := os.Stat(path); err == nil && info.Size() == int64(len(jar)) {
		return path, nil
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("jvm: create %s: %w", directory, err)
	}
	if err := atomicWrite(path, jar); err != nil {
		return "", err
	}
	return path, nil
}

// atomicWrite writes through a temporary file in the same directory and
// renames.
//
// Two servers starting at once on one machine share this cache, and a reader
// must never see a half-written jar: rename is what makes the file appear
// complete or not at all. Same directory, because rename is only atomic within
// a filesystem.
func atomicWrite(path string, contents []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("jvm: create %s: %w", path, err)
	}
	name := temporary.Name()
	_, writeErr := temporary.Write(contents)
	closeErr := temporary.Close()
	if err := firstError(writeErr, closeErr); err != nil {
		os.Remove(name)
		return fmt.Errorf("jvm: write %s: %w", path, err)
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		// Losing the race against another server that extracted the same
		// content is not a failure: the file is named after its bytes, so
		// whatever is there is what we were about to write.
		if info, statErr := os.Stat(path); statErr == nil && info.Size() == int64(len(contents)) {
			return nil
		}
		return fmt.Errorf("jvm: install %s: %w", path, err)
	}
	return nil
}

// jarDirectory is where extracted jars live: the configured directory, then the
// user cache, then the temporary directory for containers that have no HOME.
func (r *Runtime) jarDirectory() string {
	if r.config.ExtractDirectory != "" {
		return r.config.ExtractDirectory
	}
	if cache, err := os.UserCacheDir(); err == nil {
		return filepath.Join(cache, "gocraft")
	}
	return filepath.Join(os.TempDir(), "gocraft")
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
