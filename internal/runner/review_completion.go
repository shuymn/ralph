package ralphrunner

import (
	"strings"

	ralphconfig "github.com/shuymn/ralph/internal/config"
)

type reviewCompletion struct {
	signal       string
	minReviews   int
	maxReviews   int
	stableRounds int
	stableCount  int
}

func newReviewCompletion(profile ralphconfig.ReviewCompletionProfile) reviewCompletion {
	signal := strings.TrimSpace(profile.Signal)
	if signal == "" {
		signal = ralphconfig.DefaultCompletionSignal
	}

	minReviews := max(profile.ReviewConvergence.MinReviews, 1)
	maxReviews := max(profile.ReviewConvergence.MaxReviews, minReviews)
	stableRounds := max(profile.ReviewConvergence.StableRounds, 1)

	return reviewCompletion{
		signal:       signal,
		minReviews:   minReviews,
		maxReviews:   maxReviews,
		stableRounds: stableRounds,
	}
}

func (completion *reviewCompletion) recordJudge(result judgeContract) {
	if result.Signal == completion.signal && result.NewFindings == 0 {
		completion.stableCount++
		return
	}
	completion.stableCount = 0
}

func (completion *reviewCompletion) converged(reviewCount int) bool {
	return reviewCount >= completion.minReviews &&
		completion.stableCount >= completion.stableRounds
}

func (completion *reviewCompletion) nonConvergedAtMax(reviewCount int) bool {
	return reviewCount >= completion.maxReviews &&
		completion.stableCount < completion.stableRounds
}
