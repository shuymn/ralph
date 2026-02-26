package ralphrunner

import "time"

type reviewState struct {
	reviewCount       int
	reviewsSinceJudge int
	judgeCount        int
	runID             string
}

func newReviewState(now time.Time) reviewState {
	return reviewState{runID: formatRunID(now)}
}

func formatRunID(now time.Time) string {
	return now.UTC().Format("20060102T150405Z")
}

func (state *reviewState) nextReviewIndex() int {
	return state.reviewCount + 1
}

func (state *reviewState) nextJudgeIndex() int {
	return state.judgeCount + 1
}

func (state *reviewState) recordRole(role Role) {
	switch role {
	case RoleRun:
		return
	case RoleReview:
		state.reviewCount++
		state.reviewsSinceJudge++
	case RoleJudge:
		state.judgeCount++
		state.reviewsSinceJudge = 0
	}
}
