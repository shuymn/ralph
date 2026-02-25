package ralphcmd

import (
	"fmt"
	"io"

	"github.com/shuymn/ralph/internal/runner"
)

const runUsage = "usage: ralph run [--dry-run]"

func RunRun(rootDir string, args []string, stdout, stderr io.Writer) int {
	dryRun := false
	switch len(args) {
	case 0:
	case 1:
		if args[0] == "--dry-run" {
			dryRun = true
			break
		}
		fallthrough
	default:
		fmt.Fprintln(stderr, runUsage)
		return 1
	}

	options := runner.Options{
		WorkDir: rootDir,
		Stdout:  stdout,
		Stderr:  stderr,
	}
	if dryRun {
		return runner.DryRun(options)
	}

	return runner.Run(options)
}
