package prd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuymn/ralph/internal/prd"
)

func TestLoadReturnsValidationErrorForInvalidStoryIDs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		content     string
		wantMessage string
	}{
		{
			name:        "empty story id is rejected",
			content:     `{"stories":[{"id":"","deps":[],"passes":false}]}`,
			wantMessage: "id must be non-empty",
		},
		{
			name:        "duplicate story id is rejected",
			content:     `{"stories":[{"id":"task-1","deps":[],"passes":false},{"id":"task-1","deps":[],"passes":true}]}`,
			wantMessage: "duplicate story id",
		},
		{
			name:        "stories must be non-empty",
			content:     `{"stories":[]}`,
			wantMessage: "stories must contain at least one entry",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			prdPath := writePRDFile(t, tc.content)
			_, err := prd.Load(prdPath)
			if err == nil {
				t.Fatalf("Load() error = nil, want validation error")
			}
			if !prd.IsValidationError(err) {
				t.Fatalf("Load() error kind = %T (%v), want validation", err, err)
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("Load() error = %q, want substring %q", err.Error(), tc.wantMessage)
			}
		})
	}
}

func TestLoadReturnsParseErrorForMalformedStoryIDShape(t *testing.T) {
	t.Parallel()

	prdPath := writePRDFile(t, `{"stories":[{"id":123,"deps":[],"passes":false}]}`)
	_, err := prd.Load(prdPath)
	if err == nil {
		t.Fatalf("Load() error = nil, want parse error")
	}
	if !prd.IsParseError(err) {
		t.Fatalf("Load() error kind = %T (%v), want parse", err, err)
	}
}

func TestLoadAcceptsUnknownTopLevelFields(t *testing.T) {
	t.Parallel()

	prdPath := writePRDFile(
		t,
		`{"project":"demo","plan":"docs/plans/example.md","stories":[{"id":"task-1","deps":[],"passes":false}],"extra":"value"}`,
	)
	doc, err := prd.Load(prdPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(doc.Stories) != 1 {
		t.Fatalf("len(doc.Stories) = %d, want 1", len(doc.Stories))
	}
}

func writePRDFile(t *testing.T, content string) string {
	t.Helper()

	rootDir := t.TempDir()
	prdPath := filepath.Join(rootDir, "prd.json")
	if err := os.WriteFile(prdPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile(prd.json) error = %v", err)
	}
	return prdPath
}
