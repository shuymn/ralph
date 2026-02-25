package condition_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuymn/ralph/internal/condition"
)

func TestEvaluateSupportedFunctions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		expr    string
		ctx     condition.Context
		want    bool
		wantErr bool
	}{
		{
			name: "always returns true",
			expr: "always()",
			ctx: condition.Context{
				Success: false,
				Changed: func() (bool, error) { return false, nil },
			},
			want: true,
		},
		{
			name: "success reflects context",
			expr: "success()",
			ctx: condition.Context{
				Success: true,
				Changed: func() (bool, error) { return false, nil },
			},
			want: true,
		},
		{
			name: "failure reflects inverse success",
			expr: "failure()",
			ctx: condition.Context{
				Success: true,
				Changed: func() (bool, error) { return false, nil },
			},
			want: false,
		},
		{
			name: "changed delegates callback",
			expr: "changed()",
			ctx: condition.Context{
				Success: true,
				Changed: func() (bool, error) { return true, nil },
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := condition.Evaluate(tc.expr, tc.ctx)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Evaluate() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Evaluate() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Fatalf("Evaluate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEvaluateBooleanOperatorsAndParentheses(t *testing.T) {
	t.Parallel()

	got, err := condition.Evaluate(
		"success() && (changed() || !failure())",
		condition.Context{
			Success: true,
			Changed: func() (bool, error) {
				return false, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("Evaluate() error = %v, want nil", err)
	}
	if !got {
		t.Fatalf("Evaluate() = %v, want true", got)
	}
}

func TestEvaluateRejectsUnknownSymbols(t *testing.T) {
	t.Parallel()

	_, err := condition.Evaluate("unknown()", condition.Context{Success: true})
	if err == nil {
		t.Fatalf("Evaluate() error = nil, want parse error")
	}
	if !condition.IsParseError(err) {
		t.Fatalf("Evaluate() error kind = %T (%v), want parse", err, err)
	}
}

func TestEvaluateChangedFailureReturnsEvaluationError(t *testing.T) {
	t.Parallel()

	nonExistentDir := filepath.Join(t.TempDir(), "missing")

	_, err := condition.Evaluate(
		"changed()",
		condition.Context{
			Success: true,
			Changed: condition.GitChanged(nonExistentDir),
		},
	)
	if err == nil {
		t.Fatalf("Evaluate() error = nil, want evaluation error")
	}
	if !condition.IsEvaluationError(err) {
		t.Fatalf("Evaluate() error kind = %T (%v), want evaluation", err, err)
	}
	if !strings.Contains(err.Error(), "changed()") {
		t.Fatalf("Evaluate() error = %q, want message mentioning changed()", err.Error())
	}
}
