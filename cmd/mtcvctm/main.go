// Package main is the entry point for mtcvctm CLI
package main

import (
	"fmt"
	"os"

	"github.com/sirosfoundation/mtcvctm/cmd/mtcvctm/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
