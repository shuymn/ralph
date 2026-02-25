package ralphinit

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

const (
	ralphDirName = ".ralph"
	dateLayout   = "2006-01-02"
	fileMode     = 0o644
	dirMode      = 0o755
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type templateFile struct {
	OutputPath   string
	TemplatePath string
	Render       bool
}

var templateFiles = []templateFile{
	{
		OutputPath:   "config.yml",
		TemplatePath: "templates/config.tmpl",
		Render:       false,
	},
	{
		OutputPath:   "prompt.md",
		TemplatePath: "templates/prompt.tmpl",
		Render:       false,
	},
	{
		OutputPath:   "prd.json",
		TemplatePath: "templates/prd.tmpl",
		Render:       false,
	},
	{
		OutputPath:   "progress.md",
		TemplatePath: "templates/progress.tmpl",
		Render:       true,
	},
}

type Options struct {
	RootDir string
	Now     time.Time
	Stderr  io.Writer
}

func Scaffold(opts Options) error {
	rootDir := opts.RootDir
	if rootDir == "" {
		rootDir = "."
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	stderr := opts.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	ralphDir := filepath.Join(rootDir, ralphDirName)
	if err := os.MkdirAll(ralphDir, dirMode); err != nil {
		return fmt.Errorf(
			"create %s directory: %w",
			filepath.ToSlash(ralphDirName),
			err,
		)
	}

	data := map[string]string{
		"Today": now.Format(dateLayout),
	}

	for _, file := range templateFiles {
		created, err := writeTemplateFile(ralphDir, file, data)
		if err != nil {
			return err
		}
		if !created {
			fmt.Fprintf(
				stderr,
				"skip existing file: %s\n",
				filepath.ToSlash(filepath.Join(ralphDirName, file.OutputPath)),
			)
		}
	}

	return nil
}

func writeTemplateFile(baseDir string, file templateFile, data map[string]string) (bool, error) {
	outputPath := filepath.Join(baseDir, file.OutputPath)
	if exists, err := pathExists(outputPath); err != nil {
		return false, fmt.Errorf(
			"inspect %s: %w",
			filepath.ToSlash(filepath.Join(ralphDirName, file.OutputPath)),
			err,
		)
	} else if exists {
		return false, nil
	}

	content, err := loadTemplateContent(file, data)
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(outputPath, content, fileMode); err != nil {
		return false, fmt.Errorf(
			"write %s: %w",
			filepath.ToSlash(filepath.Join(ralphDirName, file.OutputPath)),
			err,
		)
	}

	return true, nil
}

func loadTemplateContent(file templateFile, data map[string]string) ([]byte, error) {
	rawContent, err := fs.ReadFile(templateFS, file.TemplatePath)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", file.TemplatePath, err)
	}

	if !file.Render {
		return rawContent, nil
	}

	parsedTemplate, err := template.New(file.OutputPath).Parse(string(rawContent))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", file.TemplatePath, err)
	}

	var rendered bytes.Buffer
	if err := parsedTemplate.Execute(&rendered, data); err != nil {
		return nil, fmt.Errorf("render template %s: %w", file.TemplatePath, err)
	}

	return rendered.Bytes(), nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("stat %s: %w", path, err)
}
