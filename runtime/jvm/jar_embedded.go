//go:build gocraft_jvm_jar

// This file is what turns the stub in jar.go into the real embedded runtime.
//
// It is behind a tag because the jar is built by the gocraft-java repository
// and the Go repository must never require a JVM to build or release. A default
// `go build` compiles jar.go's empty slice and reports, honestly, that this
// binary carries no runtime jar; a release build produced from a tree that has
// the artifact adds -tags gocraft_jvm_jar and gets the embed.
//
// Building with the tag in a tree without gocraft-runtime.jar fails at compile
// time, which is the correct outcome: it is not a build that can work.

package jvm

import _ "embed"

//go:embed gocraft-runtime.jar
var embeddedRuntimeJar []byte

func init() { embeddedJar = embeddedRuntimeJar }
