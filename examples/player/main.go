package main

import (
	"fmt"
	"os"

	"github.com/ui-sound-system/uss-go/examples/player/internal/app"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
