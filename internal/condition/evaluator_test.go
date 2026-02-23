package ralphcondition_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	ralphcondition "github.com/shuymn/ralph/internal/condition"
)

func TestEvalSupportsBuiltinsOperatorsAndParentheses(t *testing.T) {
	t.Parallel()

	ctx := ralphcondition.NewContext(
		true,
		false,
		func() (bool, error) {
			return true, nil
		},
	)

	result, err := ralphcondition.Eval("success() && (!failure() && changed())", ctx)
	if err != nil {
		t.Fatalf("Eval returned error: %v", err)
	}
	if !result {
		t.Fatalf("expected expression to be true")
	}

	result, err = ralphcondition.Eval("always() && (false || success())", ctx)
	if err != nil {
		t.Fatalf("Eval returned error: %v", err)
	}
	if !result {
		t.Fatalf("expected expression with parentheses to be true")
	}
}

func TestEvalRejectsUnknownSymbol(t *testing.T) {
	t.Parallel()

	_, err := ralphcondition.Eval(
		"steps.main.success",
		ralphcondition.NewContext(true, false, nil),
	)

	assertErrorCode(t, err, ralphcondition.ErrCodeConditionUnknownSymbol)
}

func TestEvalMapsChangedFailureToExit22(t *testing.T) {
	t.Parallel()

	changedErr := errors.New("git failed")
	_, err := ralphcondition.Eval(
		"changed()",
		ralphcondition.NewContext(
			true,
			false,
			func() (bool, error) {
				return false, changedErr
			},
		),
	)

	assertErrorCode(t, err, ralphcondition.ErrCodeConditionRuntime)
	if !errors.Is(err, changedErr) {
		t.Fatalf("expected underlying changed error to be preserved")
	}
}

func TestNewGitChangedFuncFailure(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	changed := ralphcondition.NewGitChangedFunc(cwd)

	_, err := changed()
	if err == nil {
		t.Fatalf("expected changed() to fail outside git repository")
	}
	assertErrorCode(t, err, ralphcondition.ErrCodeConditionRuntime)
}

func TestNewGitChangedFuncReturnsTrueWhenRepositoryHasChanges(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	cwd := t.TempDir()
	runGit(t, cwd, "init")

	changed := ralphcondition.NewGitChangedFunc(cwd)
	result, err := changed()
	if err != nil {
		t.Fatalf("changed() returned error: %v", err)
	}
	if result {
		t.Fatalf("expected no changes in a fresh repository")
	}

	writeFile(t, filepath.Join(cwd, "README.md"), "hello\n")

	result, err = changed()
	if err != nil {
		t.Fatalf("changed() returned error after file creation: %v", err)
	}
	if !result {
		t.Fatalf("expected changed()=true after untracked file creation")
	}
}

func runGit(t *testing.T, cwd string, args ...string) {
	t.Helper()

	_, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git binary not found")
	}

	//nolint:gosec // Arguments are fixed test inputs scoped to a temporary directory.
	cmd := exec.Command("git", append([]string{"-C", cwd}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v; output=%s", args, err, string(output))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
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
