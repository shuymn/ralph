package ralphprd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	ExitCodeValidation = 22

	ErrCodePRDParse            = "PRD_PARSE"
	ErrCodePRDStoriesRequired  = "PRD_STORIES_REQUIRED"
	ErrCodePRDStoriesEmpty     = "PRD_STORIES_EMPTY"
	ErrCodePRDStoryMalformed   = "PRD_STORY_MALFORMED"
	ErrCodePRDStoryIDEmpty     = "PRD_STORY_ID_EMPTY"
	ErrCodePRDStoryIDDuplicate = "PRD_STORY_ID_DUPLICATE"
)

type Story struct {
	ID     string   `json:"id"`
	Passes bool     `json:"passes"`
	Deps   []string `json:"deps"`
}

type Document struct {
	Stories []Story `json:"stories"`
}

type ValidationError struct {
	code string
	msg  string
}

func (e *ValidationError) Error() string {
	return e.msg
}

func (e *ValidationError) Code() string {
	return e.code
}

func (e *ValidationError) ExitCode() int {
	return ExitCodeValidation
}

func newValidationError(code, msg string) error {
	return &ValidationError{
		code: code,
		msg:  msg,
	}
}

func ValidateBytes(content []byte) (Document, error) {
	topLevel, err := parseTopLevel(content)
	if err != nil {
		return Document{}, err
	}

	rawStories, ok := topLevel["stories"]
	if !ok {
		return Document{}, newValidationError(
			ErrCodePRDStoriesRequired,
			"stories field is required",
		)
	}

	var rawStoryList []json.RawMessage
	if err := json.Unmarshal(rawStories, &rawStoryList); err != nil {
		return Document{}, newValidationError(ErrCodePRDStoryMalformed, "stories must be an array")
	}

	doc := Document{
		Stories: make([]Story, 0, len(rawStoryList)),
	}
	for idx, rawStory := range rawStoryList {
		story, err := parseStory(rawStory, idx)
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

func parseTopLevel(content []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	var topLevel map[string]json.RawMessage
	if err := decoder.Decode(&topLevel); err != nil {
		return nil, newValidationError(ErrCodePRDParse, err.Error())
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, newValidationError(
				ErrCodePRDParse,
				"multiple JSON documents are not supported",
			)
		}
		return nil, newValidationError(ErrCodePRDParse, err.Error())
	}

	return topLevel, nil
}

func parseStory(rawStory json.RawMessage, idx int) (Story, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawStory, &fields); err != nil {
		return Story{}, newValidationError(
			ErrCodePRDStoryMalformed,
			fmt.Sprintf("story[%d] must be an object: %v", idx, err),
		)
	}

	rawID, ok := fields["id"]
	if !ok {
		return Story{}, newValidationError(
			ErrCodePRDStoryMalformed,
			fmt.Sprintf("story[%d] id is required", idx),
		)
	}
	var id string
	if err := json.Unmarshal(rawID, &id); err != nil {
		return Story{}, newValidationError(
			ErrCodePRDStoryMalformed,
			fmt.Sprintf("story[%d] id must be a string: %v", idx, err),
		)
	}

	rawPasses, ok := fields["passes"]
	if !ok {
		return Story{}, newValidationError(
			ErrCodePRDStoryMalformed,
			fmt.Sprintf("story[%d] passes is required", idx),
		)
	}
	var passes bool
	if err := json.Unmarshal(rawPasses, &passes); err != nil {
		return Story{}, newValidationError(
			ErrCodePRDStoryMalformed,
			fmt.Sprintf("story[%d] passes must be boolean: %v", idx, err),
		)
	}

	rawDeps, ok := fields["deps"]
	if !ok {
		return Story{}, newValidationError(
			ErrCodePRDStoryMalformed,
			fmt.Sprintf("story[%d] deps is required", idx),
		)
	}
	var deps []string
	if err := json.Unmarshal(rawDeps, &deps); err != nil {
		return Story{}, newValidationError(
			ErrCodePRDStoryMalformed,
			fmt.Sprintf("story[%d] deps must be string array: %v", idx, err),
		)
	}

	return Story{
		ID:     id,
		Passes: passes,
		Deps:   deps,
	}, nil
}

func Validate(doc *Document) error {
	if doc == nil {
		return newValidationError(ErrCodePRDStoriesRequired, "stories field is required")
	}
	if doc.Stories == nil {
		return newValidationError(ErrCodePRDStoriesRequired, "stories field is required")
	}
	if len(doc.Stories) == 0 {
		return newValidationError(ErrCodePRDStoriesEmpty, "stories must not be empty")
	}

	seen := make(map[string]struct{}, len(doc.Stories))
	for idx := range doc.Stories {
		story := &doc.Stories[idx]
		id := strings.TrimSpace(story.ID)
		if id == "" {
			return newValidationError(
				ErrCodePRDStoryIDEmpty,
				fmt.Sprintf("story[%d] id must be non-empty", idx),
			)
		}
		if _, exists := seen[id]; exists {
			return newValidationError(
				ErrCodePRDStoryIDDuplicate,
				fmt.Sprintf("duplicate story id: %q", id),
			)
		}
		if story.Deps == nil {
			return newValidationError(
				ErrCodePRDStoryMalformed,
				fmt.Sprintf("story[%d] deps is required", idx),
			)
		}

		story.ID = id
		seen[id] = struct{}{}
	}

	return nil
}
