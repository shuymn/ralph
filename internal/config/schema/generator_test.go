package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ralphconfig "github.com/shuymn/ralph/internal/config"
	ralphschema "github.com/shuymn/ralph/internal/config/schema"
)

const (
	gitCommitTogetherValue = "together"
	stepOnFailContinue     = "continue"
	stepUsesAutoCommit     = "auto_commit"
)

func TestGenerateIncludesHeaderAndTopLevelKeys(t *testing.T) {
	t.Parallel()

	output, err := ralphschema.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(output, &doc); err != nil {
		t.Fatalf("schema JSON must be valid: %v", err)
	}

	comment, _ := doc["$comment"].(string)
	if comment != ralphschema.GeneratedComment {
		t.Fatalf("$comment = %q, want %q", comment, ralphschema.GeneratedComment)
	}

	props, _ := doc["properties"].(map[string]any)
	if len(props) == 0 {
		t.Fatalf("properties must not be empty")
	}

	for _, key := range []string{"version", "agent", "git", "phases"} {
		if _, ok := props[key]; !ok {
			t.Fatalf("missing top-level property %q", key)
		}
	}
}

func TestArtifactExistsAndMatchesGenerate(t *testing.T) {
	t.Parallel()

	artifactPath := filepath.Clean(filepath.Join("..", "..", "..", ralphschema.ArtifactPath))
	got, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("read artifact %q: %v", artifactPath, err)
	}

	want, err := ralphschema.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("artifact differs from Generate() output")
	}
}

func TestTemplateSchemaReferenceMatchesArtifactPath(t *testing.T) {
	t.Parallel()

	templatePath := filepath.Clean(
		filepath.Join("..", "..", "..", "internal", "init", "templates", "config.tmpl"),
	)
	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read template %q: %v", templatePath, err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		t.Fatalf("config template first line must be schema directive")
	}

	wantSuffix := "/" + filepath.ToSlash(ralphschema.ArtifactPath)
	if !strings.Contains(lines[0], wantSuffix) {
		t.Fatalf("schema directive %q must contain %q", lines[0], wantSuffix)
	}
}

func TestSchemaConstraints(t *testing.T) {
	t.Parallel()

	doc := mustSchemaDoc(t)

	version := mustMapAtPath(t, doc, "properties", "version")
	if gotConst, ok := version["const"].(string); !ok || gotConst != ralphconfig.SupportedVersion {
		t.Fatalf("version.const = %v, want %q", version["const"], ralphconfig.SupportedVersion)
	}
	assertAdditionalPropertiesFalse(t, "root", doc)

	git := mustMapAtPath(t, doc, "properties", "git")
	assertAdditionalPropertiesFalse(t, "git", git)
	assertEnumValues(
		t,
		mustMapAtPath(t, git, "properties", "commit"),
		[]string{ralphconfig.DefaultGitCommitMode, gitCommitTogetherValue},
	)

	phases := mustMapAtPath(t, doc, "properties", "phases")
	assertAdditionalPropertiesFalse(t, "phases", phases)

	for _, phase := range []string{"pre", "post"} {
		phaseSchema := mustMapAtPath(t, phases, "properties", phase)
		assertAdditionalPropertiesFalse(t, "phases."+phase, phaseSchema)

		step := mustStepSchema(t, phaseSchema)
		assertAdditionalPropertiesFalse(t, "phases."+phase+".steps.items", step)
		assertEnumValues(
			t,
			mustMapAtPath(t, step, "properties", "on_fail"),
			[]string{stepOnFailContinue, ralphconfig.DefaultStepOnFail},
		)
		assertEnumValues(
			t,
			mustMapAtPath(t, step, "properties", "uses"),
			[]string{stepUsesAutoCommit},
		)
		assertStepRunUsesXOR(t, step)
	}
}

func mustSchemaDoc(t *testing.T) map[string]any {
	t.Helper()

	output, err := ralphschema.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(output, &doc); err != nil {
		t.Fatalf("schema JSON must be valid: %v", err)
	}

	return doc
}

func mustStepSchema(t *testing.T, phaseSchema map[string]any) map[string]any {
	t.Helper()

	return mustMapAtPath(t, phaseSchema, "properties", "steps", "items")
}

func mustMapAtPath(t *testing.T, root map[string]any, path ...string) map[string]any {
	t.Helper()

	current := any(root)
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("path %q has non-object segment %T", strings.Join(path, "."), current)
		}
		next, ok := obj[key]
		if !ok {
			t.Fatalf("path %q is missing key %q", strings.Join(path, "."), key)
		}
		current = next
	}

	result, ok := current.(map[string]any)
	if !ok {
		t.Fatalf("path %q must resolve to object, got %T", strings.Join(path, "."), current)
	}

	return result
}

func assertAdditionalPropertiesFalse(t *testing.T, name string, schema map[string]any) {
	t.Helper()

	value, ok := schema["additionalProperties"].(bool)
	if !ok || value {
		t.Fatalf("%s.additionalProperties = %v, want false", name, schema["additionalProperties"])
	}
}

func assertEnumValues(t *testing.T, schema map[string]any, want []string) {
	t.Helper()

	rawEnum, ok := schema["enum"].([]any)
	if !ok {
		t.Fatalf("enum is missing: %#v", schema)
	}
	if len(rawEnum) != len(want) {
		t.Fatalf("enum length = %d, want %d (%v)", len(rawEnum), len(want), want)
	}

	remaining := make(map[string]struct{}, len(want))
	for _, value := range want {
		remaining[value] = struct{}{}
	}
	for _, raw := range rawEnum {
		value, ok := raw.(string)
		if !ok {
			t.Fatalf("enum value must be string, got %T", raw)
		}
		if _, exists := remaining[value]; !exists {
			t.Fatalf("unexpected enum value %q (want %v)", value, want)
		}
		delete(remaining, value)
	}
	if len(remaining) > 0 {
		t.Fatalf("missing enum values: %v", mapKeys(remaining))
	}
}

func assertStepRunUsesXOR(t *testing.T, stepSchema map[string]any) {
	t.Helper()

	branches, ok := stepSchema["oneOf"].([]any)
	if !ok {
		t.Fatalf("oneOf is missing on step schema")
	}
	if len(branches) != 2 {
		t.Fatalf("step oneOf branch count = %d, want 2", len(branches))
	}

	hasRunRequired := false
	hasUsesRequired := false
	for _, branchRaw := range branches {
		branch, ok := branchRaw.(map[string]any)
		if !ok {
			t.Fatalf("oneOf branch must be object, got %T", branchRaw)
		}
		required := toStringSet(t, branch["required"])
		if _, exists := required["run"]; exists {
			hasRunRequired = true
		}
		if _, exists := required["uses"]; exists {
			hasUsesRequired = true
		}
	}

	if !hasRunRequired || !hasUsesRequired {
		t.Fatalf("step oneOf must contain required branches for run and uses")
	}
}

func toStringSet(t *testing.T, value any) map[string]struct{} {
	t.Helper()

	list, ok := value.([]any)
	if !ok {
		t.Fatalf("required must be an array, got %T", value)
	}
	set := make(map[string]struct{}, len(list))
	for _, item := range list {
		str, ok := item.(string)
		if !ok {
			t.Fatalf("required value must be string, got %T", item)
		}
		set[str] = struct{}{}
	}
	return set
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
