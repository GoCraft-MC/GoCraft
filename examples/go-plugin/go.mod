// The example is its own module, and that is the point of it.
//
// It requires a published gocraft-api-go rather than reaching into a sibling
// directory, so building it proves what a plugin author's build does — that
// the SDK is usable from outside this repository, by someone who only has the
// module path.
module github.com/GoCraft-MC/GoCraft/examples/go-plugin

go 1.26.0

require github.com/GoCraft-MC/gocraft-api-go v0.1.1-0.20260901135734-6803a318739c

require (
	github.com/GoCraft-MC/gocraft-abi v0.2.1-0.20260901135001-5a8ed28a84c4 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
