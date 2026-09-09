// Package main is the entrypoint for the gh-pr-pro GitHub CLI extension.
// It initializes and executes the root command hierarchy, handling any top-level execution errors.
package main

import (
	"fmt"
	"os"

	"github.com/brad/gh-pr-pro/pkg/cmd"
)

// main is the application entrypoint. It invokes the root Cobra command runner
// and terminates the process with exit code 1 if an error is returned during execution.
func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
