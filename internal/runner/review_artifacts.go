package ralphrunner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type reviewArtifacts struct {
	reviewsDir string
	runID      string
}

var errUnsupportedReviewArtifactRole = errors.New("unsupported review artifact role")

func newReviewArtifacts(paths fixedPaths, runID string) reviewArtifacts {
	return reviewArtifacts{
		reviewsDir: filepath.Join(filepath.Dir(paths.PRD), "reviews"),
		runID:      runID,
	}
}

func (artifacts reviewArtifacts) runDir() string {
	return filepath.Join(artifacts.reviewsDir, artifacts.runID)
}

func (artifacts reviewArtifacts) nextPath(role Role, state reviewState) (string, error) {
	switch role {
	case RoleRun:
		return "", fmt.Errorf("%w: %q", errUnsupportedReviewArtifactRole, role)
	case RoleReview:
		return filepath.Join(
			artifacts.runDir(),
			fmt.Sprintf("REVIEW_%04d.md", state.nextReviewIndex()),
		), nil
	case RoleJudge:
		return filepath.Join(
			artifacts.runDir(),
			fmt.Sprintf("JUDGE_%04d.json", state.nextJudgeIndex()),
		), nil
	default:
		return "", fmt.Errorf("%w: %q", errUnsupportedReviewArtifactRole, role)
	}
}

func (artifacts reviewArtifacts) persist(path, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create review artifact dir: %w", err)
	}

	srcFile, err := os.Open(outputPath)
	if err != nil {
		return fmt.Errorf("open main output for review artifact: %w", err)
	}
	defer func() {
		_ = srcFile.Close()
	}()

	dstFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create review artifact file: %w", err)
	}
	defer func() {
		_ = dstFile.Close()
	}()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("write review artifact: %w", err)
	}

	return nil
}
