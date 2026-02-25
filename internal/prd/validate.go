package prd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type ErrorKind string

const (
	ErrorKindParse      ErrorKind = "parse"
	ErrorKindValidation ErrorKind = "validation"
)

type Story struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Deps   []string `json:"deps"`
	Passes bool     `json:"passes"`
}

type Document struct {
	Stories []Story `json:"stories"`
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

func Load(path string) (Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Document{}, newParseError("read prd file", err)
	}

	var rawTop map[string]json.RawMessage
	if err := json.Unmarshal(content, &rawTop); err != nil {
		return Document{}, newParseError("decode json", err)
	}

	storiesRaw, ok := rawTop["stories"]
	if !ok {
		return Document{}, newValidationError("stories is required")
	}

	var rawStories []json.RawMessage
	if err := json.Unmarshal(storiesRaw, &rawStories); err != nil {
		return Document{}, newParseError("decode stories", err)
	}

	doc := Document{Stories: make([]Story, 0, len(rawStories))}
	for i, rawStory := range rawStories {
		story, err := parseStory(rawStory, i)
		if err != nil {
			return Document{}, err
		}
		doc.Stories = append(doc.Stories, story)
	}

	if err := Validate(&doc); err != nil {
		return Document{}, err
	}

	return doc, nil
}

func parseStory(rawStory json.RawMessage, index int) (Story, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawStory, &fields); err != nil {
		return Story{}, newParseError(fmt.Sprintf("decode story[%d]", index), err)
	}

	idRaw, ok := fields["id"]
	if !ok {
		return Story{}, newValidationError(fmt.Sprintf("stories[%d].id is required", index))
	}

	var id string
	if err := json.Unmarshal(idRaw, &id); err != nil {
		return Story{}, newParseError(fmt.Sprintf("decode stories[%d].id", index), err)
	}

	depsRaw, ok := fields["deps"]
	if !ok {
		return Story{}, newValidationError(fmt.Sprintf("stories[%d].deps is required", index))
	}

	var deps []string
	if err := json.Unmarshal(depsRaw, &deps); err != nil {
		return Story{}, newParseError(fmt.Sprintf("decode stories[%d].deps", index), err)
	}
	if deps == nil {
		return Story{}, newValidationError(
			fmt.Sprintf("stories[%d].deps must be an array of strings", index),
		)
	}

	passesRaw, ok := fields["passes"]
	if !ok {
		return Story{}, newValidationError(fmt.Sprintf("stories[%d].passes is required", index))
	}

	var passes bool
	if err := json.Unmarshal(passesRaw, &passes); err != nil {
		return Story{}, newParseError(fmt.Sprintf("decode stories[%d].passes", index), err)
	}

	story := Story{
		ID:     strings.TrimSpace(id),
		Deps:   deps,
		Passes: passes,
	}

	if titleRaw, ok := fields["title"]; ok {
		var title string
		if err := json.Unmarshal(titleRaw, &title); err != nil {
			return Story{}, newParseError(fmt.Sprintf("decode stories[%d].title", index), err)
		}
		story.Title = title
	}

	return story, nil
}

func Validate(doc *Document) error {
	if doc == nil {
		return newValidationError("prd is required")
	}

	if len(doc.Stories) == 0 {
		return newValidationError("stories must contain at least one entry")
	}

	seen := make(map[string]struct{}, len(doc.Stories))
	for i, story := range doc.Stories {
		if story.ID == "" {
			return newValidationError(fmt.Sprintf("stories[%d].id must be non-empty", i))
		}
		if _, exists := seen[story.ID]; exists {
			return newValidationError(fmt.Sprintf("duplicate story id %q", story.ID))
		}
		seen[story.ID] = struct{}{}
	}

	return nil
}
