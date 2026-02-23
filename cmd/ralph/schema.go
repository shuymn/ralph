package main

import (
	"fmt"
	"os"

	ralphschema "github.com/shuymn/ralph/internal/config/schema"
)

const schemaCommandFailureExitCode = 1

func main() {
	if err := RunSchema("."); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ralph schema failed: %v\n", err)
		os.Exit(schemaCommandFailureExitCode)
	}
}

func RunSchema(root string) error {
	if err := ralphschema.WriteRoot(root); err != nil {
		return fmt.Errorf("write schema artifact: %w", err)
	}
	return nil
}
