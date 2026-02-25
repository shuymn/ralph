package ralphrunner

import ralphconfig "github.com/shuymn/ralph/internal/config"

func nextReviewRole(state reviewState, reviewCfg ralphconfig.ReviewConvergenceMode) Role {
	minReviews := max(reviewCfg.MinReviews, 1)
	maxReviews := max(reviewCfg.MaxReviews, minReviews)
	judgeEvery := max(reviewCfg.JudgeEvery, 1)

	if state.reviewCount < minReviews {
		return RoleReview
	}
	if state.reviewCount >= maxReviews {
		return RoleJudge
	}
	if state.reviewsSinceJudge >= judgeEvery {
		return RoleJudge
	}

	return RoleReview
}
