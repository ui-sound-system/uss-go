// The latency tester builds as a standalone Windows console executable.
package main

import (
	"fmt"
	"os"

	"github.com/ui-sound-system/uss-go/internal/latency"
)

func main() {
	if err := latency.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
