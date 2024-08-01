package main

import (
	"fmt"
	"os"
)

func main() {
	rootCmd := NewRootCmd()
	if err := Execute(rootCmd); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
