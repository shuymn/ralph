package main

import (
	"fmt"
	"io"
	"time"

	ralphinit "github.com/shuymn/ralph/internal/init"
)

func RunInit(root string, stderr io.Writer, now func() time.Time) error {
	if now == nil {
		now = time.Now
	}
	if err := ralphinit.Scaffold(root, now(), stderr); err != nil {
		return fmt.Errorf("init scaffold: %w", err)
	}
	return nil
}
