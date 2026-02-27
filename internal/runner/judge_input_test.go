//nolint:testpackage // This file validates unexported judge input construction helpers.
package ralphrunner

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	ralphconfig "github.com/shuymn/ralph/internal/config"
)

func TestBuildJudgeInputPrefixPartitionsReviewArtifacts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	paths := resolvePaths(root)
	runtime := reviewRuntime{
		state: reviewState{
			reviewCount:       5,
			reviewsSinceJudge: 2,
			judgeCount:        2,
			runID:             "20260226T120000Z",
		},
		artifacts: newReviewArtifacts(paths, "20260226T120000Z"),
		completion: reviewCompletion{
			signal: "READY",
		},
	}

	prefix := buildJudgeInputPrefix(runtime)
	ctx := mustParseJudgeMachineContext(t, prefix)
	runDir := filepath.Join(root, ".ralph", "reviews", "20260226T120000Z")

	if ctx.RunID != "20260226T120000Z" {
		t.Fatalf("unexpected run_id: got=%q", ctx.RunID)
	}
	if ctx.ReviewCount != 5 {
		t.Fatalf("unexpected review_count: got=%d", ctx.ReviewCount)
	}
	if ctx.ReviewsSinceJudge != 2 {
		t.Fatalf("unexpected reviews_since_judge: got=%d", ctx.ReviewsSinceJudge)
	}
	if ctx.JudgeCount != 2 {
		t.Fatalf("unexpected judge_count: got=%d", ctx.JudgeCount)
	}
	if ctx.CompletionSignal != "READY" {
		t.Fatalf("unexpected completion_signal: got=%q", ctx.CompletionSignal)
	}

	assertStringSliceEqual(
		t,
		"new_review_files",
		ctx.NewReviewFiles,
		[]string{
			filepath.Join(runDir, "REVIEW_0004.md"),
			filepath.Join(runDir, "REVIEW_0005.md"),
		},
	)
	assertStringSliceEqual(
		t,
		"previously_judged_review_files",
		ctx.PreviouslyJudgedReviewFiles,
		[]string{
			filepath.Join(runDir, "REVIEW_0001.md"),
			filepath.Join(runDir, "REVIEW_0002.md"),
			filepath.Join(runDir, "REVIEW_0003.md"),
		},
	)
	assertStringSliceEqual(
		t,
		"all_review_files",
		ctx.AllReviewFiles,
		[]string{
			filepath.Join(runDir, "REVIEW_0001.md"),
			filepath.Join(runDir, "REVIEW_0002.md"),
			filepath.Join(runDir, "REVIEW_0003.md"),
			filepath.Join(runDir, "REVIEW_0004.md"),
			filepath.Join(runDir, "REVIEW_0005.md"),
		},
	)
	if ctx.CurrentJudgeArtifact != filepath.Join(runDir, "JUDGE_0003.json") {
		t.Fatalf("unexpected current_judge_artifact: got=%q", ctx.CurrentJudgeArtifact)
	}
	assertStringSliceEqual(
		t,
		"previous_judge_artifacts",
		ctx.PreviousJudgeArtifacts,
		[]string{
			filepath.Join(runDir, "JUDGE_0001.json"),
			filepath.Join(runDir, "JUDGE_0002.json"),
		},
	)
}

func TestBuildJudgeInputPrefixDefaultsCompletionSignal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	paths := resolvePaths(root)
	runtime := reviewRuntime{
		state: reviewState{
			reviewCount:       1,
			reviewsSinceJudge: 1,
			judgeCount:        0,
			runID:             "20260226T120000Z",
		},
		artifacts:  newReviewArtifacts(paths, "20260226T120000Z"),
		completion: reviewCompletion{signal: " \n\t "},
	}

	prefix := buildJudgeInputPrefix(runtime)
	ctx := mustParseJudgeMachineContext(t, prefix)
	if ctx.CompletionSignal != ralphconfig.DefaultCompletionSignal {
		t.Fatalf(
			"unexpected default completion signal: got=%q want=%q",
			ctx.CompletionSignal,
			ralphconfig.DefaultCompletionSignal,
		)
	}
}

func mustParseJudgeMachineContext(t *testing.T, prefix string) judgeMachineContext {
	t.Helper()

	start := strings.Index(prefix, judgeContextStartMarker)
	if start < 0 {
		t.Fatalf("missing start marker in prefix: %q", prefix)
	}
	end := strings.Index(prefix, judgeContextEndMarker)
	if end < 0 {
		t.Fatalf("missing end marker in prefix: %q", prefix)
	}
	jsonBody := strings.TrimSpace(prefix[start+len(judgeContextStartMarker) : end])
	if jsonBody == "" {
		t.Fatalf("machine context JSON body must not be empty")
	}

	var ctx judgeMachineContext
	if err := json.Unmarshal([]byte(jsonBody), &ctx); err != nil {
		t.Fatalf("unmarshal machine context: %v", err)
	}
	return ctx
}

func assertStringSliceEqual(t *testing.T, label string, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s length mismatch: got=%d want=%d got=%v", label, len(got), len(want), got)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf(
				"%s[%d] mismatch: got=%q want=%q full=%v",
				label,
				idx,
				got[idx],
				want[idx],
				got,
			)
		}
	}
}
