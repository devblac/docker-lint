package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

func TestParse(t *testing.T) {
	cfg, err := Parse([]byte(`
ignore: [DKL103]
severity:
  DKL404: warning
  DKL202: "off"
  DKL303: Error
strict: true
exclude:
  - vendor
  - "examples/**"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !reflect.DeepEqual(cfg.Ignore, []string{"DKL103"}) {
		t.Errorf("Ignore = %q", cfg.Ignore)
	}
	if cfg.Strict == nil || !*cfg.Strict {
		t.Errorf("Strict = %v, want true", cfg.Strict)
	}
	if cfg.Quiet != nil {
		t.Errorf("Quiet = %v, want unset", *cfg.Quiet)
	}
	if !reflect.DeepEqual(cfg.Exclude, []string{"vendor", "examples/**"}) {
		t.Errorf("Exclude = %q", cfg.Exclude)
	}
	if got := cfg.DisabledRules(); !reflect.DeepEqual(got, []string{"DKL202"}) {
		t.Errorf("DisabledRules() = %q", got)
	}
	wantOverrides := map[string]ast.Severity{"DKL404": ast.SeverityWarning, "DKL303": ast.SeverityError}
	if got := cfg.SeverityOverrides(); !reflect.DeepEqual(got, wantOverrides) {
		t.Errorf("SeverityOverrides() = %v, want %v", got, wantOverrides)
	}
	if got := cfg.RuleIDs(); !reflect.DeepEqual(got, []string{"DKL103", "DKL202", "DKL303", "DKL404"}) {
		t.Errorf("RuleIDs() = %q", got)
	}
}

// TestParseUnquotedOff verifies a bare off is read as the string "off", not as
// a YAML 1.1 boolean.
func TestParseUnquotedOff(t *testing.T) {
	cfg, err := Parse([]byte("severity:\n  DKL404: off\n  DKL202: OFF\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := cfg.DisabledRules(); !reflect.DeepEqual(got, []string{"DKL202", "DKL404"}) {
		t.Errorf("DisabledRules() = %q", got)
	}
}

func TestParseEmpty(t *testing.T) {
	for _, input := range []string{"", "# only a comment\n"} {
		cfg, err := Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", input, err)
		}
		if cfg.Strict != nil || len(cfg.Ignore) != 0 {
			t.Errorf("Parse(%q) = %+v, want an empty config", input, cfg)
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ignores: [DKL101]\n", "field ignores not found"}, // typo is not silently ignored
		{"severity:\n  DKL101: critical\n", `unknown value "critical"`},
		{"strict: maybe\n", "cannot unmarshal"},
		{"ignore: DKL101\n", "cannot unmarshal"},
	}
	for _, tt := range tests {
		_, err := Parse([]byte(tt.input))
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("Parse(%q) error = %v, want it to mention %q", tt.input, err, tt.want)
		}
	}
}

func TestFind(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	deep := filepath.Join(repo, "services", "api")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("strict: true\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A config above the repository root is not picked up.
	write(filepath.Join(root, ".docker-lint.yaml"))
	if got, err := Find(deep); err != nil || got != "" {
		t.Errorf("Find() = %q, %v; want no config outside the repository", got, err)
	}

	write(filepath.Join(repo, ".docker-lint.yml"))
	if got, _ := Find(deep); got != filepath.Join(repo, ".docker-lint.yml") {
		t.Errorf("Find() = %q, want the repository root config", got)
	}

	// The nearest file wins, and .yaml is preferred over .yml.
	write(filepath.Join(deep, ".docker-lint.yml"))
	write(filepath.Join(deep, ".docker-lint.yaml"))
	if got, _ := Find(deep); got != filepath.Join(deep, ".docker-lint.yaml") {
		t.Errorf("Find() = %q, want the nearest .yaml config", got)
	}
}

func TestLoadReportsPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".docker-lint.yaml")
	if err := os.WriteFile(path, []byte("strict: maybe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("Load() error = %v, want it to name %s", err, path)
	}
}
