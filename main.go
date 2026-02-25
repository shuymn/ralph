package main

import (
	"os"

	ralphcmd "github.com/shuymn/ralph/cmd/ralph"
)

func main() {
	os.Exit(ralphcmd.Run(os.Args[1:], os.Stdout, os.Stderr))
}
