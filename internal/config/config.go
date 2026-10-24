// Package config loads docker-lint's project configuration file.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/devblac/docker-lint/internal/ast"
)

// FileNames are the configuration file names looked up, in order of preference.
var FileNames = []string{".docker-lint.yaml", ".docker-lint.yml"}

// SeverityOff disables a rule from the severity map.
const SeverityOff = "off"

// Config is the content of a .docker-lint.yaml file.
type Config struct {
	// Ignore lists rule IDs that are never reported.
	Ignore []string `yaml:"ignore"`
	// Severity overrides the severity of individual rules: error, warning,
	// info, or off to disable the rule.
	Severity map[string]string `yaml:"severity"`
	// Strict fails on warnings, not only on errors. Nil means unset.
	Strict *bool `yaml:"strict"`
	// Quiet hides informational findings. Nil means unset.
	Quiet *bool `yaml:"quiet"`
	// Exclude lists path patterns skipped when a directory or glob is expanded.
	Exclude []string `yaml:"exclude"`
}

// Load reads and validates the configuration file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Parse decodes and validates configuration from YAML. Unknown keys are an
// error, so a misspelled option fails loudly instead of being ignored.
func Parse(data []byte) (*Config, error) {
	cfg := &Config{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	ids := make([]string, 0, len(c.Severity))
	for id := range c.Severity {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		value := strings.ToLower(strings.TrimSpace(c.Severity[id]))
		if value == SeverityOff {
			continue
		}
		if _, ok := parseSeverity(value); !ok {
			return fmt.Errorf("severity of %s: unknown value %q (want error, warning, info or off)", id, c.Severity[id])
		}
	}
	return nil
}

// DisabledRules returns the rules set to "off" in the severity map, sorted.
func (c *Config) DisabledRules() []string {
	var ids []string
	for id, value := range c.Severity {
		if strings.EqualFold(strings.TrimSpace(value), SeverityOff) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// SeverityOverrides returns the severity map without the disabled rules.
func (c *Config) SeverityOverrides() map[string]ast.Severity {
	overrides := make(map[string]ast.Severity)
	for id, value := range c.Severity {
		if severity, ok := parseSeverity(strings.ToLower(strings.TrimSpace(value))); ok {
			overrides[id] = severity
		}
	}
	return overrides
}

// RuleIDs returns every rule ID the configuration mentions, sorted, so callers
// can warn about IDs that match no rule.
func (c *Config) RuleIDs() []string {
	seen := make(map[string]bool)
	var ids []string
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, id := range c.Ignore {
		add(id)
	}
	for id := range c.Severity {
		add(id)
	}
	sort.Strings(ids)
	return ids
}

func parseSeverity(value string) (ast.Severity, bool) {
	switch value {
	case "error":
		return ast.SeverityError, true
	case "warning":
		return ast.SeverityWarning, true
	case "info":
		return ast.SeverityInfo, true
	}
	return 0, false
}

// Find looks for a configuration file in dir and then in each parent
// directory, stopping after the repository root (the first directory that
// contains .git) or the filesystem root. It returns "" when there is none.
func Find(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		for _, name := range FileNames {
			candidate := filepath.Join(dir, name)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return "", nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}
