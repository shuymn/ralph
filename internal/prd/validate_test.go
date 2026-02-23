package ralphprd_test

import (
	"errors"
	"testing"

	ralphprd "github.com/shuymn/ralph/internal/prd"
)

func TestValidateBytesRejectsMalformedStoryID(t *testing.T) {
	t.Parallel()

	_, err := ralphprd.ValidateBytes([]byte(`{
  "stories": [
    {"id": 123, "passes": false, "deps": []}
  ]
}`))
	assertErrorCode(t, err, ralphprd.ErrCodePRDStoryMalformed)
}

func TestValidateBytesRejectsDuplicateStoryID(t *testing.T) {
	t.Parallel()

	_, err := ralphprd.ValidateBytes([]byte(`{
  "stories": [
    {"id": "TASK-1", "passes": false, "deps": []},
    {"id": "TASK-1", "passes": true, "deps": []}
  ]
}`))
	assertErrorCode(t, err, ralphprd.ErrCodePRDStoryIDDuplicate)
}

func TestValidateBytesRejectsEmptyStoryID(t *testing.T) {
	t.Parallel()

	_, err := ralphprd.ValidateBytes([]byte(`{
  "stories": [
    {"id": "   ", "passes": false, "deps": []}
  ]
}`))
	assertErrorCode(t, err, ralphprd.ErrCodePRDStoryIDEmpty)
}

func TestValidateBytesRejectsMissingStories(t *testing.T) {
	t.Parallel()

	_, err := ralphprd.ValidateBytes([]byte(`{
  "project": "sample"
}`))
	assertErrorCode(t, err, ralphprd.ErrCodePRDStoriesRequired)
}

func TestValidateBytesRejectsEmptyStories(t *testing.T) {
	t.Parallel()

	_, err := ralphprd.ValidateBytes([]byte(`{
  "stories": []
}`))
	assertErrorCode(t, err, ralphprd.ErrCodePRDStoriesEmpty)
}

func TestValidateBytesRejectsNonBooleanPasses(t *testing.T) {
	t.Parallel()

	_, err := ralphprd.ValidateBytes([]byte(`{
  "stories": [
    {"id": "TASK-1", "passes": "false", "deps": []}
  ]
}`))
	assertErrorCode(t, err, ralphprd.ErrCodePRDStoryMalformed)
}

func TestValidateBytesRejectsNonStringDeps(t *testing.T) {
	t.Parallel()

	_, err := ralphprd.ValidateBytes([]byte(`{
  "stories": [
    {"id": "TASK-1", "passes": false, "deps": [1]}
  ]
}`))
	assertErrorCode(t, err, ralphprd.ErrCodePRDStoryMalformed)
}

func assertErrorCode(t *testing.T, err error, wantCode string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error code %q, got nil", wantCode)
	}

	var coded interface {
		Code() string
		ExitCode() int
	}
	if !errors.As(err, &coded) {
		t.Fatalf("expected coded error, got: %T %v", err, err)
	}

	if coded.Code() != wantCode {
		t.Fatalf("unexpected error code: want=%q got=%q", wantCode, coded.Code())
	}
	if coded.ExitCode() != 22 {
		t.Fatalf("unexpected exit code: want=22 got=%d", coded.ExitCode())
	}
}
