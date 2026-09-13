package main

import (
	"GoCraft/internal/cli"
	"fmt"
	"os"
)

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
