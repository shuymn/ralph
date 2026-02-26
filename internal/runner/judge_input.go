package ralphrunner

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	ralphconfig "github.com/shuymn/ralph/internal/config"
)

const (
	judgeContextStartMarker = "MACHINE_CONTEXT_JSON_START"
	judgeContextEndMarker   = "MACHINE_CONTEXT_JSON_END"
)

type judgeMachineContext struct {
	RunID string `json:"run_id"` //nolint:tagliatelle // Judge context schema is intentionally snake_case.
	//nolint:tagliatelle // Judge context schema is intentionally snake_case.
	ReviewCount int `json:"review_count"`
	//nolint:tagliatelle // Judge context schema is intentionally snake_case.
	JudgeCount int `json:"judge_count"`
	//nolint:tagliatelle // Judge context schema is intentionally snake_case.
	ReviewsSinceJudge int `json:"reviews_since_judge"`
	//nolint:tagliatelle // Judge context schema is intentionally snake_case.
	CompletionSignal string `json:"completion_signal"`
	//nolint:tagliatelle // Judge context schema is intentionally snake_case.
	NewReviewFiles []string `json:"new_review_files"`
	//nolint:tagliatelle // Judge context schema is intentionally snake_case.
	PreviouslyJudgedReviewFiles []string `json:"previously_judged_review_files"`
	//nolint:tagliatelle // Judge context schema is intentionally snake_case.
	AllReviewFiles []string `json:"all_review_files"`
	//nolint:tagliatelle // Judge context schema is intentionally snake_case.
	CurrentJudgeArtifact string `json:"current_judge_artifact"`
	//nolint:tagliatelle // Judge context schema is intentionally snake_case.
	PreviousJudgeArtifacts []string `json:"previous_judge_artifacts"`
}

func buildJudgeInputPrefix(runtime reviewRuntime) string {
	reviewCount := max(runtime.state.reviewCount, 0)
	reviewsSinceJudge := max(min(runtime.state.reviewsSinceJudge, reviewCount), 0)
	judgeCount := max(runtime.state.judgeCount, 0)
	baselineReviewCount := reviewCount - reviewsSinceJudge
	runDir := runtime.artifacts.runDir()

	context := judgeMachineContext{
		RunID:             runtime.state.runID,
		ReviewCount:       reviewCount,
		JudgeCount:        judgeCount,
		ReviewsSinceJudge: reviewsSinceJudge,
		CompletionSignal:  strings.TrimSpace(runtime.completion.signal),
		NewReviewFiles: reviewArtifactPaths(
			runDir,
			baselineReviewCount+1,
			reviewCount,
		),
		PreviouslyJudgedReviewFiles: reviewArtifactPaths(runDir, 1, baselineReviewCount),
		AllReviewFiles:              reviewArtifactPaths(runDir, 1, reviewCount),
		CurrentJudgeArtifact: filepath.Join(
			runDir,
			fmt.Sprintf("JUDGE_%04d.json", runtime.state.nextJudgeIndex()),
		),
		PreviousJudgeArtifacts: judgeArtifactPaths(runDir, 1, judgeCount),
	}
	if context.CompletionSignal == "" {
		context.CompletionSignal = ralphconfig.DefaultCompletionSignal
	}

	payload, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		payload = []byte("{}")
	}
	return fmt.Sprintf(
		"%s\n%s\n%s\n\n",
		judgeContextStartMarker,
		string(payload),
		judgeContextEndMarker,
	)
}

func reviewArtifactPaths(runDir string, start, end int) []string {
	return numberedArtifactPaths(runDir, "REVIEW_%04d.md", start, end)
}

func judgeArtifactPaths(runDir string, start, end int) []string {
	return numberedArtifactPaths(runDir, "JUDGE_%04d.json", start, end)
}

func numberedArtifactPaths(runDir, pattern string, start, end int) []string {
	if start < 1 {
		start = 1
	}
	if end < start {
		return nil
	}
	paths := make([]string, 0, end-start+1)
	for idx := start; idx <= end; idx++ {
		paths = append(paths, filepath.Join(runDir, fmt.Sprintf(pattern, idx)))
	}
	return paths
}
