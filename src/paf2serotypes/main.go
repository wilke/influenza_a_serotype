package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/me/influenza_a_serotype/src/paf2serotypes/cmd"
)

func main() {
	// Set maximum number of CPUs to use
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Execute the root command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
