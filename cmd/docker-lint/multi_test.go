package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree creates files below a temporary directory and returns its path.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// chdir changes the working directory for the rest of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

const (
	cleanDockerfile   = "FROM alpine:3.20\nUSER nobody\n"
	warningDockerfile = "FROM alpine\nUSER nobody\n" // DKL101
	errorDockerfile   = "RUN echo hi\n"              // DKL001
)

func TestLintDirectory(t *testing.T) {
	chdir(t, writeTree(t, map[string]string{
		"services/api/Dockerfile":     warningDockerfile,
		"services/web/web.Dockerfile": errorDockerfile,
		"services/web/README.md":      "not a Dockerfile",
	}))

	var stdout, stderr bytes.Buffer
	code := run([]string{"-q", "services"}, nil, &stdout, &stderr)

	if code != ExitLintErrors {
		t.Errorf("exit code %d, want %d; stderr: %s", code, ExitLintErrors, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"services/api/Dockerfile:1:1: [warning] DKL101",
		"services/web/web.Dockerfile:1:1: [error] DKL001",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestLintGlobJSONIsArray(t *testing.T) {
	chdir(t, writeTree(t, map[string]string{
		"a/Dockerfile": cleanDockerfile,
		"b/Dockerfile": warningDockerfile,
	}))

	var stdout, stderr bytes.Buffer
	if code := run([]string{"--json", "**/Dockerfile"}, nil, &stdout, &stderr); code != ExitSuccess {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var outputs []struct {
		File    string `json:"file"`
		Summary struct {
			Warnings int `json:"warnings"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &outputs); err != nil {
		t.Fatalf("expected a JSON array: %v\n%s", err, stdout.String())
	}
	if len(outputs) != 2 || outputs[0].File != "a/Dockerfile" || outputs[1].File != "b/Dockerfile" {
		t.Fatalf("unexpected files: %+v", outputs)
	}
	if outputs[1].Summary.Warnings == 0 {
		t.Errorf("expected warnings for b/Dockerfile: %+v", outputs[1])
	}
}

func TestLintSingleFileJSONIsObject(t *testing.T) {
	path := createTempDockerfile(t, cleanDockerfile)
	var stdout, stderr bytes.Buffer
	run([]string{"--json", path}, nil, &stdout, &stderr)

	var output map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("expected a JSON object for a single file: %v\n%s", err, stdout.String())
	}
}

func TestLintMultipleFilesSARIFIsOneRun(t *testing.T) {
	chdir(t, writeTree(t, map[string]string{
		"a/Dockerfile": warningDockerfile,
		"b/Dockerfile": errorDockerfile,
	}))

	var stdout, stderr bytes.Buffer
	if code := run([]string{"-f", "sarif", "a/Dockerfile", "b/Dockerfile"}, nil, &stdout, &stderr); code != ExitLintErrors {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var log struct {
		Runs []struct {
			Results []struct {
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &log); err != nil {
		t.Fatal(err)
	}
	if len(log.Runs) != 1 {
		t.Fatalf("len(runs) = %d, want 1", len(log.Runs))
	}
	uris := make(map[string]bool)
	for _, r := range log.Runs[0].Results {
		uris[r.Locations[0].PhysicalLocation.ArtifactLocation.URI] = true
	}
	if !uris["a/Dockerfile"] || !uris["b/Dockerfile"] {
		t.Errorf("results should cover both files, got %v", uris)
	}
}

// TestParseErrorDoesNotStopOtherFiles verifies one broken file is reported and
// makes the run fatal, while the other files are still linted.
func TestParseErrorDoesNotStopOtherFiles(t *testing.T) {
	chdir(t, writeTree(t, map[string]string{
		"a/Dockerfile": "FROM alpine:3.20\nFOOBAR x\n",
		"b/Dockerfile": warningDockerfile,
	}))

	var stdout, stderr bytes.Buffer
	code := run([]string{"."}, nil, &stdout, &stderr)

	if code != ExitFatal {
		t.Errorf("exit code %d, want %d", code, ExitFatal)
	}
	if !strings.Contains(stderr.String(), "Error: a/Dockerfile: failed to parse Dockerfile") {
		t.Errorf("stderr should name the broken file: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "b/Dockerfile:1:1: [warning] DKL101") {
		t.Errorf("the other file should still be linted:\n%s", stdout.String())
	}
}

func TestLintPathErrors(t *testing.T) {
	chdir(t, writeTree(t, map[string]string{"docs/README.md": "x"}))

	for _, args := range [][]string{
		{"docs"},
		{"**/Dockerfile"},
		{"missing/Dockerfile"},
	} {
		var stdout, stderr bytes.Buffer
		if code := run(args, nil, &stdout, &stderr); code != ExitFatal {
			t.Errorf("%v: exit code %d, want %d", args, code, ExitFatal)
		}
		if stdout.Len() != 0 {
			t.Errorf("%v: expected no output, got %q", args, stdout.String())
		}
	}
}

func TestConfigFileDiscovered(t *testing.T) {
	chdir(t, writeTree(t, map[string]string{
		".git/HEAD": "",
		".docker-lint.yaml": `
ignore: [DKL101]
severity:
  DKL303: error
  DKL404: "off"
exclude: [vendor]
`,
		"Dockerfile":        "FROM alpine\n", // DKL101, DKL303, DKL404
		"vendor/Dockerfile": errorDockerfile,
	}))

	var stdout, stderr bytes.Buffer
	code := run([]string{"."}, nil, &stdout, &stderr)

	out := stdout.String()
	if code != ExitLintErrors {
		t.Errorf("exit code %d, want %d (DKL303 raised to error); output:\n%s", code, ExitLintErrors, out)
	}
	if !strings.Contains(out, "[error] DKL303") {
		t.Errorf("DKL303 should be reported as an error:\n%s", out)
	}
	for _, unwanted := range []string{"DKL101", "DKL404", "vendor/"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("output should not contain %q:\n%s", unwanted, out)
		}
	}

	// --no-config restores the defaults.
	stdout.Reset()
	run([]string{"--no-config", "Dockerfile"}, nil, &stdout, &stderr)
	if !strings.Contains(stdout.String(), "[warning] DKL303") || !strings.Contains(stdout.String(), "DKL101") {
		t.Errorf("--no-config should ignore the configuration:\n%s", stdout.String())
	}
}

func TestFlagsOverrideConfig(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"lint.yaml":  "strict: true\nquiet: true\n",
		"Dockerfile": warningDockerfile,
	})
	cfg := filepath.Join(dir, "lint.yaml")
	dockerfile := filepath.Join(dir, "Dockerfile")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"--config", cfg, dockerfile}, nil, &stdout, &stderr); code != ExitLintErrors {
		t.Errorf("strict from config: exit code %d, want %d", code, ExitLintErrors)
	}
	if strings.Contains(stdout.String(), "[info]") {
		t.Errorf("quiet from config should hide info findings:\n%s", stdout.String())
	}

	stdout.Reset()
	if code := run([]string{"-c", cfg, "--strict=false", "--quiet=false", dockerfile}, nil, &stdout, &stderr); code != ExitSuccess {
		t.Errorf("--strict=false should win over the config: exit code %d", code)
	}
	if !strings.Contains(stdout.String(), "[info]") {
		t.Errorf("--quiet=false should win over the config:\n%s", stdout.String())
	}
}

func TestConfigErrors(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"bad.yaml":   "severity:\n  DKL101: critical\n",
		"Dockerfile": cleanDockerfile,
	})
	dockerfile := filepath.Join(dir, "Dockerfile")

	for _, args := range [][]string{
		{"--config", filepath.Join(dir, "bad.yaml"), dockerfile},
		{"--config", filepath.Join(dir, "missing.yaml"), dockerfile},
		{"--config", filepath.Join(dir, "bad.yaml"), "--no-config", dockerfile},
	} {
		var stdout, stderr bytes.Buffer
		if code := run(args, nil, &stdout, &stderr); code != ExitFatal {
			t.Errorf("%v: exit code %d, want %d", args, code, ExitFatal)
		}
		if !strings.Contains(stderr.String(), "Error:") {
			t.Errorf("%v: expected an error, got %q", args, stderr.String())
		}
	}
}

func TestConfigUnknownRuleWarns(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"lint.yaml":  "ignore: [DL3006, DKL999]\n",
		"Dockerfile": cleanDockerfile,
	})

	var stdout, stderr bytes.Buffer
	run([]string{"-c", filepath.Join(dir, "lint.yaml"), filepath.Join(dir, "Dockerfile")}, nil, &stdout, &stderr)

	for _, want := range []string{"DL3006 was renamed to DKL101", "unknown rule DKL999 in "} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr missing %q: %q", want, stderr.String())
		}
	}
}

// TestFlagsAfterPaths verifies flags are accepted after the paths, the way
// most command-line tools allow.
func TestFlagsAfterPaths(t *testing.T) {
	path := createTempDockerfile(t, warningDockerfile)

	tests := []struct {
		args []string
		code int
		want string
	}{
		{[]string{path, "--json"}, ExitSuccess, `"rule_id": "DKL101"`},
		{[]string{path, "-f", "github", "--strict"}, ExitLintErrors, "::warning file="},
		{[]string{"-q", path, "--ignore", "DKL101"}, ExitSuccess, ""},
	}
	for _, tt := range tests {
		var stdout, stderr bytes.Buffer
		code := run(tt.args, nil, &stdout, &stderr)
		if code != tt.code {
			t.Errorf("%v: exit code %d, want %d; stderr: %s", tt.args, code, tt.code, stderr.String())
		}
		if !strings.Contains(stdout.String(), tt.want) {
			t.Errorf("%v: output missing %q:\n%s", tt.args, tt.want, stdout.String())
		}
		if tt.want == "" && strings.Contains(stdout.String(), "DKL101") {
			t.Errorf("%v: --ignore after the path was not applied:\n%s", tt.args, stdout.String())
		}
	}
}

// TestDoubleDashEndsFlags verifies a path that looks like a flag can follow "--".
func TestDoubleDashEndsFlags(t *testing.T) {
	dir := writeTree(t, map[string]string{"--json": cleanDockerfile})
	chdir(t, dir)

	var stdout, stderr bytes.Buffer
	if code := run([]string{"-q", "--", "--json"}, nil, &stdout, &stderr); code != ExitSuccess {
		t.Errorf("exit code %d; stderr: %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), `"file"`) {
		t.Errorf("--json after -- should be a path, not a flag:\n%s", stdout.String())
	}
}

func TestDashReadsStdin(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-", "--json"}, strings.NewReader(warningDockerfile), &stdout, &stderr)
	if code != ExitSuccess || !strings.Contains(stdout.String(), `"file": "stdin"`) {
		t.Errorf("exit code %d, output:\n%s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	path := createTempDockerfile(t, cleanDockerfile)
	if code := run([]string{"-", path}, strings.NewReader(cleanDockerfile), &stdout, &stderr); code != ExitFatal {
		t.Errorf("- with another path: exit code %d, want %d", code, ExitFatal)
	}
}
