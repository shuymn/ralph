package main

import (
	"fmt"
	"os"
	"time"

	ralphinit "github.com/shuymn/ralph/internal/init"
)

const (
	usage       = "usage: ralph init"
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

func main() {
	exitCode := run(os.Args[1:])
	os.Exit(exitCode)
}

func run(args []string) int {
	if len(args) != 1 || args[0] != "init" {
		_, _ = fmt.Fprintln(os.Stderr, usage)
		return exitUsage
	}

	if err := ralphinit.Scaffold(".", time.Now(), os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ralph init failed: %v\n", err)
		return exitFailure
	}

	return exitOK
}
