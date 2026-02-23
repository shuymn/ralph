package ralphgit

import (
	"errors"
	"fmt"

	ralphprd "github.com/shuymn/ralph/internal/prd"
)

var errTaskIDTransitionCount = errors.New(
	"expected exactly one story to transition passes false->true",
)

func ExtractTaskID(before ralphprd.Document, after ralphprd.Document) (string, error) {
	afterPasses := make(map[string]bool, len(after.Stories))
	for _, story := range after.Stories {
		afterPasses[story.ID] = story.Passes
	}

	candidates := make([]string, 0, 1)
	for _, story := range before.Stories {
		if story.Passes {
			continue
		}
		afterPass, exists := afterPasses[story.ID]
		if !exists {
			continue
		}
		if afterPass {
			candidates = append(candidates, story.ID)
		}
	}

	if len(candidates) != 1 {
		return "", fmt.Errorf(
			"%w: got=%d",
			errTaskIDTransitionCount,
			len(candidates),
		)
	}

	return candidates[0], nil
}
