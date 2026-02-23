package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ralphschema "github.com/shuymn/ralph/internal/config/schema"
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
