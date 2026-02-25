package git

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/shuymn/ralph/internal/prd"
)

var errTaskIDAmbiguous = errors.New("exactly one task must transition from false to true")

func ExtractTaskID(before, after prd.Document) (string, error) {
	beforePassesByID := make(map[string]bool, len(before.Stories))
	for _, story := range before.Stories {
		beforePassesByID[story.ID] = story.Passes
	}

	transitioned := make([]string, 0, 1)
	for _, story := range after.Stories {
		beforePasses, exists := beforePassesByID[story.ID]
		if !exists {
			continue
		}
		if !beforePasses && story.Passes {
			transitioned = append(transitioned, story.ID)
		}
	}

	if len(transitioned) != 1 {
		return "", fmt.Errorf("%w: %s", errTaskIDAmbiguous, formatTaskIDs(transitioned))
	}

	return transitioned[0], nil
}

func formatTaskIDs(taskIDs []string) string {
	if len(taskIDs) == 0 {
		return "none"
	}

	copyIDs := append([]string(nil), taskIDs...)
	slices.Sort(copyIDs)
	return strings.Join(copyIDs, ",")
}
