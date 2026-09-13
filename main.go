package main

import (
	"GoCraft/internal/cli"
	"fmt"
	"os"
)

// version is overridden at build time via -ldflags:
//
//	go build -ldflags="-X main.version=v1.2.3" .
var version = "dev"

func main() {
	if err := cli.RootCommand.Execute(); err != nil {
		fmt.Println()
		fmt.Printf("error: %v", err)
		fmt.Println()
		os.Exit(1)
	}

	fmt.Println()
	// normal exit
}
