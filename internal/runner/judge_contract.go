package ralphrunner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	errJudgeSignalRequired         = errors.New("judge contract signal is required")
	errJudgeNewFindingsRequired    = errors.New("judge contract new_findings is required")
	errJudgeNewFindingKeysRequired = errors.New("judge contract new_finding_keys is required")
	errJudgeNewFindingsNegative    = errors.New("judge contract new_findings must be >= 0")
	errJudgeFindingsCountMismatch  = errors.New(
		"judge contract new_findings must match len(new_finding_keys)",
	)
	errJudgeTrailingJSONValue = errors.New("judge contract has unexpected trailing json value")
	errJudgeTrailingContent   = errors.New("judge contract has unexpected trailing content")
)

type judgeContract struct {
	Signal         string
	NewFindings    int
	NewFindingKeys []string
}

type judgeContractPayload struct {
	Signal *string `json:"signal"`
	//nolint:tagliatelle // judge JSON contract is intentionally snake_case.
	NewFindings *int `json:"new_findings"`
	//nolint:tagliatelle // judge JSON contract is intentionally snake_case.
	NewFindingKeys *[]string `json:"new_finding_keys"`
}

func parseJudgeContract(path string) (judgeContract, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return judgeContract{}, fmt.Errorf("read judge artifact: %w", err)
	}
	content = stripJudgeMachineContext(content)

	var payload judgeContractPayload
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return judgeContract{}, fmt.Errorf("decode judge artifact json: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return judgeContract{}, fmt.Errorf(
				"decode judge artifact json: %w",
				errJudgeTrailingJSONValue,
			)
		}
		return judgeContract{}, fmt.Errorf(
			"decode judge artifact json: %w",
			errors.Join(errJudgeTrailingContent, err),
		)
	}
	if payload.Signal == nil || strings.TrimSpace(*payload.Signal) == "" {
		return judgeContract{}, errJudgeSignalRequired
	}
	if payload.NewFindings == nil {
		return judgeContract{}, errJudgeNewFindingsRequired
	}
	if *payload.NewFindings < 0 {
		return judgeContract{}, errJudgeNewFindingsNegative
	}
	if payload.NewFindingKeys == nil {
		return judgeContract{}, errJudgeNewFindingKeysRequired
	}
	if err := validateJudgeFindingConsistency(
		*payload.NewFindings,
		*payload.NewFindingKeys,
	); err != nil {
		return judgeContract{}, err
	}

	return judgeContract{
		Signal:         *payload.Signal,
		NewFindings:    *payload.NewFindings,
		NewFindingKeys: append([]string(nil), (*payload.NewFindingKeys)...),
	}, nil
}

func validateJudgeFindingConsistency(newFindings int, newFindingKeys []string) error {
	if newFindings != len(newFindingKeys) {
		return fmt.Errorf(
			"%w: new_findings=%d new_finding_keys=%d",
			errJudgeFindingsCountMismatch,
			newFindings,
			len(newFindingKeys),
		)
	}
	return nil
}

func stripJudgeMachineContext(content []byte) []byte {
	trimmed := bytes.TrimLeft(content, " \t\r\n")
	startMarker := []byte(judgeContextStartMarker)
	endMarker := []byte(judgeContextEndMarker)

	if !bytes.HasPrefix(trimmed, startMarker) {
		return content
	}

	afterStart := trimmed[len(startMarker):]
	_, afterEnd, found := bytes.Cut(afterStart, endMarker)
	if !found {
		return content
	}

	return bytes.TrimLeft(afterEnd, " \t\r\n")
}
