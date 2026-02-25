package ralphrunner_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphrunner "github.com/shuymn/ralph/internal/runner"
)

const (
	reviewPromptMarker = "ROLE_REVIEW"
	judgePromptMarker  = "ROLE_JUDGE"
)

func TestReviewSchedulerRules(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		"prompt-run\n",
	)

	writeFile(t, filepath.Join(root, ".ralph", "prompt.review.md"), reviewPromptMarker+"\n")
	writeFile(t, filepath.Join(root, ".ralph", "prompt.judge.md"), judgePromptMarker+"\n")

	roleLog := filepath.Join(root, ".ralph", "role.log")
	cfg := testConfig(
		fmt.Sprintf(
			"input=$(cat); printf '%%s\\n' \"$input\" >> %s; "+
				"if [ \"$input\" = %s ]; then "+
				"printf '{\"signal\":\"continue\",\"new_findings\":1,\"new_finding_keys\":[\"A\"]}\\n'; "+
				"else printf 'iteration\\n'; fi",
			shQuote(roleLog),
			shQuote(judgePromptMarker),
		),
		nil,
		nil,
	)
	cfg.Agent.Command = ""
	cfg.Agent.MaxIterations = 6
	cfg.Completion.Review.ReviewConvergence = ralphconfig.ReviewConvergenceMode{
		MinReviews:   2,
		MaxReviews:   6,
		JudgeEvery:   2,
		StableRounds: 2,
	}

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
		Mode:       ralphrunner.ModeReview,
	})
	if code != ralphrunner.ExitCodeMaxIterations {
		t.Fatalf("expected exit code %d, got %d", ralphrunner.ExitCodeMaxIterations, code)
	}

	got := readLoggedRoles(t, roleLog)
	want := []string{
		reviewPromptMarker,
		reviewPromptMarker,
		judgePromptMarker,
		reviewPromptMarker,
		reviewPromptMarker,
		judgePromptMarker,
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected role count: got=%d want=%d roles=%v", len(got), len(want), got)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf(
				"unexpected role at index %d: got=%q want=%q full=%v",
				idx,
				got[idx],
				want[idx],
				got,
			)
		}
	}
}

func TestReviewSchedulerMaxReviewsBoundary(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		"prompt-run\n",
	)

	writeFile(t, filepath.Join(root, ".ralph", "prompt.review.md"), reviewPromptMarker+"\n")
	writeFile(t, filepath.Join(root, ".ralph", "prompt.judge.md"), judgePromptMarker+"\n")

	roleLog := filepath.Join(root, ".ralph", "role.log")
	cfg := testConfig(
		fmt.Sprintf(
			"input=$(cat); printf '%%s\\n' \"$input\" >> %s; "+
				"if [ \"$input\" = %s ]; then "+
				"printf '{\"signal\":\"continue\",\"new_findings\":1,\"new_finding_keys\":[\"A\"]}\\n'; "+
				"else printf 'iteration\\n'; fi",
			shQuote(roleLog),
			shQuote(judgePromptMarker),
		),
		nil,
		nil,
	)
	cfg.Agent.Command = ""
	cfg.Agent.MaxIterations = 4
	cfg.Completion.Review.ReviewConvergence = ralphconfig.ReviewConvergenceMode{
		MinReviews:   1,
		MaxReviews:   2,
		JudgeEvery:   5,
		StableRounds: 2,
	}

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
		Mode:       ralphrunner.ModeReview,
	})
	if code != ralphrunner.ExitCodeMaxIterations {
		t.Fatalf("expected exit code %d, got %d", ralphrunner.ExitCodeMaxIterations, code)
	}

	got := readLoggedRoles(t, roleLog)
	if len(got) < 3 {
		t.Fatalf("expected at least 3 scheduled roles, got %v", got)
	}
	if got[2] != judgePromptMarker {
		t.Fatalf("expected third role to be judge at max_reviews boundary, got %v", got)
	}

	reviewCount := 0
	for idx, role := range got {
		if role != reviewPromptMarker {
			continue
		}
		reviewCount++
		if idx >= 2 {
			t.Fatalf("review role should not run after max_reviews boundary, got %v", got)
		}
	}
	if reviewCount != 2 {
		t.Fatalf("expected exactly 2 review runs, got %d roles=%v", reviewCount, got)
	}
}

func TestReviewRunIDFormatAndArtifactNaming(t *testing.T) {
	t.Parallel()

	root, tmpDir := setupWorkspace(
		t,
		`{"branchName":"main","stories":[{"id":"TASK-1","passes":false,"deps":[]}]}`,
		"prompt-run\n",
	)

	writeFile(t, filepath.Join(root, ".ralph", "prompt.review.md"), "review output\n")
	writeFile(
		t,
		filepath.Join(root, ".ralph", "prompt.judge.md"),
		`{"signal":"continue","new_findings":1,"new_finding_keys":["A"]}`+"\n",
	)

	cfg := testConfig("cat", nil, nil)
	cfg.Agent.Command = ""
	cfg.Agent.MaxIterations = 2
	cfg.Completion.Review.ReviewConvergence = ralphconfig.ReviewConvergenceMode{
		MinReviews:   1,
		MaxReviews:   2,
		JudgeEvery:   1,
		StableRounds: 1,
	}

	code := ralphrunner.Run(context.Background(), cfg, ralphrunner.Options{
		WorkingDir: root,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		TempDir:    tmpDir,
		Sleep:      func(time.Duration) {},
		Mode:       ralphrunner.ModeReview,
	})
	if code != ralphrunner.ExitCodeMaxIterations {
		t.Fatalf("expected exit code %d, got %d", ralphrunner.ExitCodeMaxIterations, code)
	}

	reviewsDir := filepath.Join(root, ".ralph", "reviews")
	runDirs, err := os.ReadDir(reviewsDir)
	if err != nil {
		t.Fatalf("read reviews dir failed: %v", err)
	}
	if len(runDirs) != 1 {
		t.Fatalf("expected one run directory, got %d", len(runDirs))
	}

	runID := runDirs[0].Name()
	if !regexp.MustCompile(`^\d{8}T\d{6}Z$`).MatchString(runID) {
		t.Fatalf("unexpected run_id format: %q", runID)
	}

	artifactDir := filepath.Join(reviewsDir, runID)
	artifacts, err := os.ReadDir(artifactDir)
	if err != nil {
		t.Fatalf("read artifact dir failed: %v", err)
	}

	names := make(map[string]struct{}, len(artifacts))
	for _, entry := range artifacts {
		names[entry.Name()] = struct{}{}
	}
	if _, ok := names["REVIEW_0001.md"]; !ok {
		t.Fatalf("expected REVIEW_0001.md, got files: %v", sortedKeys(names))
	}
	if _, ok := names["JUDGE_0001.json"]; !ok {
		t.Fatalf("expected JUDGE_0001.json, got files: %v", sortedKeys(names))
	}
}

func readLoggedRoles(t *testing.T, path string) []string {
	t.Helper()

	raw := strings.TrimSpace(readFile(t, path))
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
