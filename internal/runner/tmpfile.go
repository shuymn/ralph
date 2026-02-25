package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const mainOutputFilePattern = "ralph-main-*"

type trackedTmpFile struct {
	Path   string
	remove func(string) error
}

func createTrackedTmpFile(workDir string) (*os.File, trackedTmpFile, error) {
	file, err := os.CreateTemp(workDir, mainOutputFilePattern)
	if err != nil {
		return nil, trackedTmpFile{}, fmt.Errorf("create temp file: %w", err)
	}

	tracked := trackedTmpFile{
		Path:   file.Name(),
		remove: os.Remove,
	}

	return file, tracked, nil
}

func (file trackedTmpFile) Cleanup(stderr io.Writer) {
	if file.Path == "" {
		return
	}

	remove := file.remove
	if remove == nil {
		remove = os.Remove
	}

	if err := remove(file.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		if stderr == nil {
			return
		}
		fmt.Fprintf(
			stderr,
			"[ralph] cleanup tmpfile %s: %s\n",
			file.Path,
			strings.TrimSpace(err.Error()),
		)
	}
}
