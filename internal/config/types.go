package config

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrorKindParse      ErrorKind = "parse"
	ErrorKindValidation ErrorKind = "validation"
)

const (
	DefaultIfExpr          = "success()"
	DefaultOnFail          = "stop_loop"
	DefaultAgentIterations = 60
	DefaultAgentSleep      = 5
	DefaultCompletionMode  = "tail_match"
	DefaultCompletionTail  = 20
	DefaultCompletionSig   = "<promise>COMPLETE</promise>"
	DefaultCommitMode      = "split"
	DefaultFallbackMessage = "feat: implement task (auto-commit)"
	SupportedVersion       = "1"
	BuiltinAutoCommit      = "auto_commit"
	OnFailContinue         = "continue"
	CommitModeTogether     = "together"
)

type Config struct {
	Version    string
	Agent      AgentConfig
	Completion CompletionConfig
	Git        GitConfig
	Phases     PhasesConfig
}

type AgentConfig struct {
	Command       string
	MaxIterations int
	SleepSeconds  int
}

type CompletionConfig struct {
	Strategy  string
	Signal    string
	TailLines int
}

type GitConfig struct {
	Commit          string
	FallbackMessage string
}

type PhasesConfig struct {
	Pre  PhaseConfig
	Post PhaseConfig
}

type PhaseConfig struct {
	Steps []StepConfig
}

type StepConfig struct {
	Name   string
	Run    string
	Uses   string
	If     string
	OnFail string
}

type Error struct {
	kind ErrorKind
	msg  string
	err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %v", e.msg, e.err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *Error) Kind() ErrorKind {
	if e == nil {
		return ""
	}
	return e.kind
}

func IsParseError(err error) bool {
	var typed *Error
	if !errors.As(err, &typed) {
		return false
	}
	return typed.Kind() == ErrorKindParse
}

func IsValidationError(err error) bool {
	var typed *Error
	if !errors.As(err, &typed) {
		return false
	}
	return typed.Kind() == ErrorKindValidation
}

func newParseError(msg string, err error) error {
	return &Error{kind: ErrorKindParse, msg: msg, err: err}
}

func newValidationError(msg string) error {
	return &Error{kind: ErrorKindValidation, msg: msg}
}
