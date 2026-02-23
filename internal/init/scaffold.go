package ralphinit

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const (
	ralphDirName = ".ralph"
	configSchema = "# yaml-language-server: $schema=" +
		"https://raw.githubusercontent.com/shuymn/ralph/main/schemas/config.schema.json"
	filePerm      = 0o644
	directoryPerm = 0o755
)

//go:embed templates/config.tmpl
var configTemplate string

//go:embed templates/prompt.tmpl
var promptTemplate string

//go:embed templates/prd.tmpl
var prdTemplate string

//go:embed templates/progress.tmpl
var progressTemplate string

type templateSpec struct {
	name     string
	filename string
	body     string
}

type templateData struct {
	Today string
}

func Scaffold(root string, now time.Time, stderr io.Writer) error {
	if stderr == nil {
		stderr = io.Discard
	}

	ralphDirPath := filepath.Join(root, ralphDirName)
	if err := os.MkdirAll(ralphDirPath, directoryPerm); err != nil {
		return fmt.Errorf("create %s: %w", ralphDirName, err)
	}

	data := templateData{
		Today: now.Format("2006-01-02"),
	}

	specs := []templateSpec{
		{name: "config", filename: "config.yml", body: configTemplate},
		{name: "prompt", filename: "prompt.md", body: promptTemplate},
		{name: "prd", filename: "prd.json", body: prdTemplate},
		{name: "progress", filename: "progress.md", body: progressTemplate},
	}

	for _, spec := range specs {
		if err := createFromTemplate(ralphDirPath, spec, data, stderr); err != nil {
			return err
		}
	}

	return nil
}

func createFromTemplate(
	baseDir string,
	spec templateSpec,
	data templateData,
	stderr io.Writer,
) error {
	outputPath := filepath.Join(baseDir, spec.filename)
	ok, err := exists(outputPath)
	if err != nil {
		return fmt.Errorf("check .ralph/%s: %w", spec.filename, err)
	}
	if ok {
		_, _ = fmt.Fprintf(stderr, "skipped existing file: .ralph/%s\n", spec.filename)
		return nil
	}

	rendered, err := renderTemplate(spec.name, spec.body, data)
	if err != nil {
		return fmt.Errorf("render %s: %w", spec.filename, err)
	}

	if spec.filename == "config.yml" {
		rendered = ensureSchemaHeader(rendered)
	}

	if err := os.WriteFile(outputPath, []byte(rendered), filePerm); err != nil {
		return fmt.Errorf("write .ralph/%s: %w", spec.filename, err)
	}

	return nil
}

func renderTemplate(name, body string, data templateData) (string, error) {
	tmpl, err := template.New(name).Parse(body)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}

	return b.String(), nil
}

func ensureSchemaHeader(content string) string {
	if strings.Contains(content, configSchema) {
		return content
	}
	if content == "" {
		return configSchema + "\n"
	}
	return configSchema + "\n" + content
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("stat %s: %w", path, err)
}
