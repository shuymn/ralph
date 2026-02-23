package ralphconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

var errMultipleYAMLDocuments = errors.New("multiple YAML documents are not supported")

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	return LoadBytes(content)
}

func LoadBytes(content []byte) (Config, error) {
	var cfg Config

	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, newParseError("parse config YAML", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return Config{}, newParseError("parse config YAML", errMultipleYAMLDocuments)
	} else if !errors.Is(err, io.EOF) {
		return Config{}, newParseError("parse config YAML", err)
	}

	if err := Validate(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
