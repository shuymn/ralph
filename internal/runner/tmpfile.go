package ralphrunner

import (
	"fmt"
	"os"
	"sync"
)

type tmpTracker struct {
	mu    sync.Mutex
	paths map[string]struct{}
}

func newTmpTracker() *tmpTracker {
	return &tmpTracker{
		paths: make(map[string]struct{}),
	}
}

func (t *tmpTracker) create(tempDir string) (*os.File, error) {
	file, err := os.CreateTemp(tempDir, "ralph-main-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	t.mu.Lock()
	t.paths[file.Name()] = struct{}{}
	t.mu.Unlock()

	return file, nil
}

func (t *tmpTracker) cleanup(path string) {
	if path == "" {
		return
	}

	t.mu.Lock()
	delete(t.paths, path)
	t.mu.Unlock()

	_ = os.Remove(path)
}

func (t *tmpTracker) cleanupAll() {
	t.mu.Lock()
	paths := make([]string, 0, len(t.paths))
	for path := range t.paths {
		paths = append(paths, path)
	}
	t.paths = make(map[string]struct{})
	t.mu.Unlock()

	for _, path := range paths {
		_ = os.Remove(path)
	}
}
