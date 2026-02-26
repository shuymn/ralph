package ralphrunner

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphprd "github.com/shuymn/ralph/internal/prd"
)

var errInvalidCompletionTailLines = errors.New("completion.run.tail_lines must be > 0")

func evaluateCompletion(
	outputPath string,
	doc ralphprd.Document,
	completionCfg ralphconfig.RunCompletionProfile,
) (bool, bool, error) {
	allPassed := true
	for _, story := range doc.Stories {
		if !story.Passes {
			allPassed = false
			break
		}
	}

	if !allPassed {
		return false, false, nil
	}

	if completionCfg.TailLines <= 0 {
		return false, false, errInvalidCompletionTailLines
	}

	lines, err := readLines(outputPath)
	if err != nil {
		return false, false, err
	}

	start := 0
	if completionCfg.TailLines < len(lines) {
		start = len(lines) - completionCfg.TailLines
	}

	if slices.Contains(lines[start:], completionCfg.Signal) {
		return true, false, nil
	}

	return false, true, nil
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open main output: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	lines := make([]string, 0)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan main output: %w", err)
	}

	return lines, nil
}
