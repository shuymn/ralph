package main

import "testing"

func TestRunParsesReviewCommand(t *testing.T) {
	t.Parallel()

	exitCode := run([]string{"review", "--dry-run"})
	if exitCode == exitUsage {
		t.Fatalf("expected review command to be parsed, got usage exit code %d", exitCode)
	}
}
