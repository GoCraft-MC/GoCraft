// Package plugin owns plugin discovery, lifecycle, and event dispatch.
package plugin

import "github.com/GoCraft-MC/gocraft-abi/gcpkg"

// Bundle is one archive this server found, and where it decided to keep that
// plugin's data.
//
// The archive half is the shared format: a build tool writes it and any host
// reads it. The data directory is not — it is this server's answer to where a
// plugin's files live, which no build tool has an opinion about and no other
// host has to agree with.
type Bundle struct {
	gcpkg.Bundle
	DataDirectory string
}
