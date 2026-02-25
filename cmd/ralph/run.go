package ralphcmd

import (
	"fmt"
	"io"

	"github.com/shuymn/ralph/internal/runner"
)

func RunRun(rootDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "usage: ralph run")
		return 1
	}

	return runner.Run(runner.Options{
		WorkDir: rootDir,
		Stdout:  stdout,
		Stderr:  stderr,
	})
}
