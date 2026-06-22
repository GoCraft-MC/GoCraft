// Command genpumpkinblocks compresses Pumpkin's current Bedrock block-state
// stream for embedding in GoCraft's Bedrock palette resolver.
package main

import (
	"compress/gzip"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"
)

func main() {
	input := flag.String("input", "", "Pumpkin assets/bedrock/block_states.nbt")
	output := flag.String("output", "bedrock/world/block_states.nbt.gz", "compressed embedded block-state stream")
	flag.Parse()
	if *input == "" {
		panic("-input is required")
	}

	in, err := os.Open(*input)
	if err != nil {
		panic(err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		panic(err)
	}
	out, err := os.Create(*output)
	if err != nil {
		panic(err)
	}

	compressed, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		_ = out.Close()
		panic(err)
	}
	// A fixed header makes regeneration byte-for-byte reproducible.
	compressed.Name = "block_states.nbt"
	compressed.ModTime = time.Unix(0, 0)
	if _, err := io.Copy(compressed, in); err != nil {
		_ = compressed.Close()
		_ = out.Close()
		panic(err)
	}
	if err := compressed.Close(); err != nil {
		_ = out.Close()
		panic(err)
	}
	if err := out.Close(); err != nil {
		panic(err)
	}
}
