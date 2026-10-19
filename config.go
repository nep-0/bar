package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Config is per-user, never loaded from the source or invocation directory.
type config struct {
	Runtimes map[string][]string `json:"runtimes"`
	Tools    map[string]string   `json:"tools"`
}

func loadConfig() (config, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return config{}, fmt.Errorf("locate user config directory: %w", err)
	}
	path := filepath.Join(dir, "bar", "config.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return config{}, nil
	}
	if err != nil {
		return config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	cfg, err := parseConfig(data)
	if err != nil {
		return config{}, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return cfg, nil
}

func parseConfig(data []byte) (config, error) {
	var cfg config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return cfg, fmt.Errorf("expected a JSON object")
	}
	if err := decoder.Decode(&cfg); err != nil {
		return cfg, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return cfg, fmt.Errorf("expected exactly one JSON object")
	}
	for language, chain := range cfg.Runtimes {
		options, ok := recipes[language]
		if !ok {
			return cfg, fmt.Errorf("unknown language %q", language)
		}
		if len(chain) == 0 {
			return cfg, fmt.Errorf("runtime chain for %s must not be empty", language)
		}
		seen := make(map[string]bool)
		for _, name := range chain {
			if _, ok := findRecipe(options, name); !ok {
				return cfg, fmt.Errorf("unknown runtime %q for %s", name, language)
			}
			if seen[name] {
				return cfg, fmt.Errorf("duplicate runtime %q for %s", name, language)
			}
			seen[name] = true
		}
	}
	for name, path := range cfg.Tools {
		known := false
		for _, options := range recipes {
			for _, option := range options {
				for _, tool := range option.tools {
					if tool == name {
						known = true
					}
				}
			}
		}
		if !known {
			return cfg, fmt.Errorf("unknown tool %q", name)
		}
		if !filepath.IsAbs(path) {
			return cfg, fmt.Errorf("tool %q requires an absolute path", name)
		}
	}
	return cfg, nil
}
