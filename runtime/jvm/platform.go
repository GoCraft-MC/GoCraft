package jvm

import (
	"os"
	"runtime"
)

// platformKey names the row of the pin table this machine needs.
//
// GOOS-GOARCH is not enough on Linux. A glibc tarball does not run on Alpine,
// and Alpine images are common on the hosts this server is deployed to, so musl
// gets its own key rather than a download that unpacks cleanly and then fails
// to exec with a message about a missing loader.
func platformKey() string {
	key := runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "linux" && isMusl() {
		key += "-musl"
	}
	return key
}

// muslLoaderPath is where musl's dynamic loader lives for a Go architecture, or
// empty for an architecture this build has no name for.
//
// The loader is named after the machine, not after GOARCH, and the two spellings
// disagree on every architecture that matters — which is the whole reason this
// mapping is written out rather than interpolated.
func muslLoaderPath(goarch string) string {
	machine := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
		"386":   "i386",
		"arm":   "armhf",
		"s390x": "s390x",
		"ppc64": "powerpc64",
	}[goarch]
	if machine == "" {
		return ""
	}
	return "/lib/ld-musl-" + machine + ".so.1"
}

// isMusl reports whether this is a musl system.
//
// The test is a file lookup rather than a cgo call: linking cgo into the server
// to answer one question at boot would cost cross-compilation, which is a far
// worse trade than a stat that is wrong only on a system that has musl's loader
// installed without using it.
func isMusl() bool {
	path := muslLoaderPath(runtime.GOARCH)
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
