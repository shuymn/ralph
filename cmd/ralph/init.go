package ralphcmd

import (
	"fmt"
	"io"
	"time"

	ralphinit "github.com/shuymn/ralph/internal/init"
)

func RunInit(rootDir string, stderr io.Writer) error {
	if err := ralphinit.Scaffold(ralphinit.Options{
		RootDir: rootDir,
		Now:     now(),
		Stderr:  stderr,
	}); err != nil {
		return fmt.Errorf("scaffold .ralph: %w", err)
	}

	return nil
}

func now() time.Time {
	return time.Now()
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: ralph <command>")
		return 1
	}

	switch args[0] {
	case "init":
		if err := RunInit(".", stderr); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintln(stdout, "initialized .ralph scaffolding")
		return 0
	case "run":
		return RunRun(".", args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return 1
	}
}
