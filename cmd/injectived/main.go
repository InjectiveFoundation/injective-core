package main

import (
	"fmt"
	"os"
)

// main is the entry point for the injectived command.
func main() {
	rootCmd := NewRootCmd()
	if err := Execute(rootCmd); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
