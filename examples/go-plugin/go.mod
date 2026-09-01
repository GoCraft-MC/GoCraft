// The example is its own module, and that is the point of it.
//
// It requires a published gocraft-api-go rather than reaching into a sibling
// directory, so building it proves what a plugin author's build does — that
// the SDK is usable from outside this repository, by someone who only has the
// module path.
module github.com/GoCraft-MC/GoCraft/examples/go-plugin

go 1.26.0

require github.com/GoCraft-MC/gocraft-api-go v0.0.0-20260901110705-6a7cc68dd318

require (
	github.com/GoCraft-MC/gocraft-abi v0.0.0-20260901110452-4a542296899d // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
