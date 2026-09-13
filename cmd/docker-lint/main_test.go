// Package main provides the CLI entry point for docker-lint.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTempDockerfile creates a temporary Dockerfile with the given content.
func createTempDockerfile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp Dockerfile: %v", err)
	}
	return path
}

// TestHelpOutput tests that --help flag displays usage information.
// Requirements: 6.1
func TestHelpOutput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"long flag", []string{"--help"}},
		{"short flag", []string{"-h"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := run(tt.args, nil, &stdout, &stderr)

			if exitCode != ExitSuccess {
				t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
			}

			output := stdout.String()

			// Check for key elements in help output
			expectedElements := []string{
				"docker-lint",
				"Usage:",
				"--help",
				"--version",
				"--json",
				"--quiet",
				"--strict",
				"--ignore",
				"--rules",
				"Exit Codes:",
			}

			for _, elem := range expectedElements {
				if !strings.Contains(output, elem) {
					t.Errorf("help output missing expected element: %s", elem)
				}
			}
		})
	}
}

// TestVersionOutput tests that --version flag displays version information.
// Requirements: 6.1
func TestVersionOutput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"long flag", []string{"--version"}},
		{"short flag", []string{"-v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := run(tt.args, nil, &stdout, &stderr)

			if exitCode != ExitSuccess {
				t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
			}

			output := stdout.String()
			if !strings.Contains(output, "docker-lint version") {
				t.Errorf("version output missing 'docker-lint version': %s", output)
			}
		})
	}
}

// TestRunReadsFromStdin ensures stdin is used when no file path is provided.
func TestRunReadsFromStdin(t *testing.T) {
	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	dockerfile := "FROM alpine:3.18\nUSER nobody\nHEALTHCHECK CMD echo ok\n"
	if _, err := w.Write([]byte(dockerfile)); err != nil {
		t.Fatalf("failed to write Dockerfile to pipe: %v", err)
	}
	w.Close()

	os.Stdin = r

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{}, r, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d (stderr: %s)", ExitSuccess, exitCode, stderr.String())
	}
}

// TestRulesListing tests that --rules flag lists all available rules.
// Requirements: 8.2
func TestRulesListing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"--rules"}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	output := stdout.String()

	// Check for header
	if !strings.Contains(output, "Available rules:") {
		t.Error("rules output missing 'Available rules:' header")
	}

	// Check for some known rule IDs
	expectedRules := []string{
		"DL3006", // Missing explicit image tag
		"DL3007", // Using 'latest' tag
		"DL4002", // No USER instruction
		"DL5000", // Missing HEALTHCHECK
	}

	for _, rule := range expectedRules {
		if !strings.Contains(output, rule) {
			t.Errorf("rules output missing expected rule: %s", rule)
		}
	}
}

// TestFileInput tests reading Dockerfile from file path argument.
// Requirements: 6.2
func TestFileInput(t *testing.T) {
	dockerfile := `FROM alpine:3.18
RUN echo "hello"
USER nobody
HEALTHCHECK CMD echo ok
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	// Should succeed with no errors (this is a valid Dockerfile)
	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d; stderr: %s", ExitSuccess, exitCode, stderr.String())
	}
}

// TestFileInputWithWarnings tests file input that produces warnings.
// Requirements: 6.2, 6.5
func TestFileInputWithWarnings(t *testing.T) {
	// Dockerfile with issues that should produce warnings
	dockerfile := `FROM alpine
RUN echo "hello"
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	// Should succeed (warnings only, non-strict mode)
	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	output := stdout.String()
	// Should contain warning about missing tag
	if !strings.Contains(output, "DL3006") {
		t.Errorf("expected warning about image tag (DL3006), got: %s", output)
	}
}

// TestJSONOutput tests that --json flag produces valid JSON output.
// Requirements: 7.2
func TestJSONOutput(t *testing.T) {
	dockerfile := `FROM alpine
RUN echo "hello"
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"--json", path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	// Verify output is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v\nOutput: %s", err, stdout.String())
	}

	// Check for expected JSON structure
	if _, ok := result["file"]; !ok {
		t.Error("JSON output missing 'file' field")
	}
	if _, ok := result["findings"]; !ok {
		t.Error("JSON output missing 'findings' field")
	}
	if _, ok := result["summary"]; !ok {
		t.Error("JSON output missing 'summary' field")
	}
}

// TestJSONOutputShortFlag tests that -j flag produces valid JSON output.
// Requirements: 7.2
func TestJSONOutputShortFlag(t *testing.T) {
	dockerfile := `FROM alpine:3.18
USER nobody
HEALTHCHECK CMD echo ok
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"-j", path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	// Verify output is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

// TestStrictMode tests that --strict flag treats warnings as errors.
// Requirements: 6.7
func TestStrictMode(t *testing.T) {
	// Dockerfile with warnings (missing tag)
	dockerfile := `FROM alpine
RUN echo "hello"
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"--strict", path}, nil, &stdout, &stderr)

	// Should fail in strict mode due to warnings
	if exitCode != ExitLintErrors {
		t.Errorf("expected exit code %d in strict mode with warnings, got %d", ExitLintErrors, exitCode)
	}
}

// TestStrictModeShortFlag tests that -s flag treats warnings as errors.
// Requirements: 6.7
func TestStrictModeShortFlag(t *testing.T) {
	dockerfile := `FROM alpine
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"-s", path}, nil, &stdout, &stderr)

	// Should fail in strict mode due to warnings
	if exitCode != ExitLintErrors {
		t.Errorf("expected exit code %d in strict mode with warnings, got %d", ExitLintErrors, exitCode)
	}
}

// TestStrictModeNoWarnings tests strict mode with clean Dockerfile.
// Requirements: 6.4, 6.7
func TestStrictModeNoWarnings(t *testing.T) {
	// Clean Dockerfile with no issues
	dockerfile := `FROM alpine:3.18
USER nobody
HEALTHCHECK CMD wget -q --spider http://localhost/ || exit 1
CMD ["echo", "hello"]
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"--strict", path}, nil, &stdout, &stderr)

	// Should succeed even in strict mode with no warnings
	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d with clean Dockerfile in strict mode, got %d; output: %s", ExitSuccess, exitCode, stdout.String())
	}
}

// TestIgnoreFlag tests that --ignore flag excludes specified rules.
// Requirements: 8.1
func TestIgnoreFlag(t *testing.T) {
	// Dockerfile that would trigger DL3006 (missing tag)
	dockerfile := `FROM alpine
RUN echo "hello"
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"--ignore", "DL3006", path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	output := stdout.String()
	// DL3006 should not appear in output
	if strings.Contains(output, "DL3006") {
		t.Errorf("DL3006 should be ignored but appeared in output: %s", output)
	}
}

// TestIgnoreMultipleRules tests ignoring multiple rules.
// Requirements: 8.1
func TestIgnoreMultipleRules(t *testing.T) {
	dockerfile := `FROM alpine
RUN echo "hello"
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"--ignore", "DL3006,DL4002,DL5000", path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	output := stdout.String()
	ignoredRules := []string{"DL3006", "DL4002", "DL5000"}
	for _, rule := range ignoredRules {
		if strings.Contains(output, rule) {
			t.Errorf("rule %s should be ignored but appeared in output", rule)
		}
	}
}

// TestQuietMode tests that --quiet flag suppresses informational messages.
// Requirements: 7.3
func TestQuietMode(t *testing.T) {
	// Dockerfile that might produce info-level findings
	dockerfile := `FROM alpine:3.18
COPY *.txt /app/
USER nobody
HEALTHCHECK CMD echo ok
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"--quiet", path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	// In quiet mode, info-level messages should be suppressed
	output := stdout.String()
	if strings.Contains(output, "[info]") {
		t.Errorf("quiet mode should suppress info messages, got: %s", output)
	}
}

// TestQuietModeShortFlag tests that -q flag suppresses informational messages.
// Requirements: 7.3
func TestQuietModeShortFlag(t *testing.T) {
	dockerfile := `FROM alpine:3.18
COPY *.txt /app/
USER nobody
HEALTHCHECK CMD echo ok
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"-q", path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
}

// TestExitCodeNoFindings tests exit code 0 when no findings.
// Requirements: 6.4
func TestExitCodeNoFindings(t *testing.T) {
	// Clean Dockerfile
	dockerfile := `FROM alpine:3.18
USER nobody
HEALTHCHECK CMD wget -q --spider http://localhost/ || exit 1
CMD ["echo", "hello"]
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d for clean Dockerfile, got %d; output: %s", ExitSuccess, exitCode, stdout.String())
	}
}

// TestExitCodeWarningsOnly tests exit code 0 when only warnings (non-strict).
// Requirements: 6.5
func TestExitCodeWarningsOnly(t *testing.T) {
	dockerfile := `FROM alpine
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	// Warnings only should exit 0 in non-strict mode
	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d for warnings only (non-strict), got %d", ExitSuccess, exitCode)
	}
}

// TestFileNotFound tests exit code 2 for file not found.
// Requirements: 6.8
func TestFileNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"nonexistent-dockerfile.txt"}, nil, &stdout, &stderr)

	if exitCode != ExitFatal {
		t.Errorf("expected exit code %d for file not found, got %d", ExitFatal, exitCode)
	}

	if !strings.Contains(stderr.String(), "Error") {
		t.Error("expected error message in stderr for file not found")
	}
}

// TestParseError tests exit code 2 for parse errors.
// Requirements: 6.8
func TestParseError(t *testing.T) {
	// Invalid Dockerfile content
	dockerfile := `INVALID_INSTRUCTION something
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	if exitCode != ExitFatal {
		t.Errorf("expected exit code %d for parse error, got %d; stderr: %s", ExitFatal, exitCode, stderr.String())
	}
}

// TestTextOutputFormat tests the default text output format.
// Requirements: 7.1
func TestTextOutputFormat(t *testing.T) {
	dockerfile := `FROM alpine
RUN echo "hello"
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	output := stdout.String()

	// Text output should contain line numbers and rule IDs
	// Format: file:line:column: [severity] rule_id: message
	if len(output) > 0 {
		// Should contain colon-separated format elements
		if !strings.Contains(output, ":") {
			t.Errorf("text output should contain colon-separated format, got: %s", output)
		}
		// Should contain severity indicator
		if !strings.Contains(output, "[warning]") && !strings.Contains(output, "[error]") && !strings.Contains(output, "[info]") {
			t.Errorf("text output should contain severity indicator, got: %s", output)
		}
	}
}

// TestCombinedFlags tests combining multiple flags.
// Requirements: 6.1-6.8, 7.1-7.4
func TestCombinedFlags(t *testing.T) {
	dockerfile := `FROM alpine
RUN echo "hello"
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	// Combine --json and --quiet
	exitCode := run([]string{"--json", "--quiet", path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	// Should still be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

// TestJSONWithStrict tests combining --json and --strict flags.
// Requirements: 6.7, 7.2
func TestJSONWithStrict(t *testing.T) {
	dockerfile := `FROM alpine
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"--json", "--strict", path}, nil, &stdout, &stderr)

	// Should fail due to warnings in strict mode
	if exitCode != ExitLintErrors {
		t.Errorf("expected exit code %d in strict mode with warnings, got %d", ExitLintErrors, exitCode)
	}

	// Output should still be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON even with non-zero exit: %v", err)
	}
}

// TestIgnoreWithStrict tests combining --ignore and --strict flags.
// Requirements: 6.7, 8.1
func TestIgnoreWithStrict(t *testing.T) {
	dockerfile := `FROM alpine
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	// Ignore all rules that would trigger warnings
	exitCode := run([]string{"--strict", "--ignore", "DL3006,DL4002,DL5000", path}, nil, &stdout, &stderr)

	// Should succeed since all warning-producing rules are ignored
	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d when ignoring all warning rules in strict mode, got %d; output: %s", ExitSuccess, exitCode, stdout.String())
	}
}

// TestMultiStageDockerfile tests parsing multi-stage Dockerfiles.
// Requirements: 1.3
func TestMultiStageDockerfile(t *testing.T) {
	dockerfile := `FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

FROM alpine:3.18
COPY --from=builder /app/main /main
USER nobody
HEALTHCHECK CMD /main --health
CMD ["/main"]
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	// Should parse successfully
	if exitCode == ExitFatal {
		t.Errorf("failed to parse multi-stage Dockerfile: %s", stderr.String())
	}
}

// TestInlineIgnoreComment tests inline ignore comments.
// Requirements: 8.3
func TestInlineIgnoreComment(t *testing.T) {
	dockerfile := `# docker-lint ignore: DL3006
FROM alpine
USER nobody
HEALTHCHECK CMD echo ok
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	output := stdout.String()
	// DL3006 should be ignored for the FROM line
	if strings.Contains(output, "DL3006") {
		t.Errorf("DL3006 should be ignored via inline comment, got: %s", output)
	}
}

// TestEmptyDockerfile tests handling of empty Dockerfile.
// Requirements: 1.4
func TestEmptyDockerfile(t *testing.T) {
	dockerfile := ``
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	// Empty Dockerfile should be handled (may produce warnings or errors)
	// The important thing is it doesn't crash
	if exitCode == ExitFatal && !strings.Contains(stderr.String(), "Error") {
		t.Error("empty Dockerfile should produce an error message")
	}
}

// TestDockerfileWithComments tests that comments are handled correctly.
// Requirements: 1.2
func TestDockerfileWithComments(t *testing.T) {
	dockerfile := `# This is a comment
FROM alpine:3.18
# Another comment
RUN echo "hello"
USER nobody
HEALTHCHECK CMD echo ok
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	if exitCode == ExitFatal {
		t.Errorf("failed to parse Dockerfile with comments: %s", stderr.String())
	}
}

// TestJSONOutputStructure tests the JSON output structure in detail.
// Requirements: 7.2
func TestJSONOutputStructure(t *testing.T) {
	dockerfile := `FROM alpine
RUN echo "hello"
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{"--json", path}, nil, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	var result struct {
		File     string `json:"file"`
		Findings []struct {
			RuleID     string `json:"rule_id"`
			Severity   string `json:"severity"`
			Line       int    `json:"line"`
			Column     int    `json:"column"`
			Message    string `json:"message"`
			Suggestion string `json:"suggestion,omitempty"`
		} `json:"findings"`
		Summary struct {
			Total    int `json:"total"`
			Errors   int `json:"errors"`
			Warnings int `json:"warnings"`
			Info     int `json:"info"`
		} `json:"summary"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	// Verify file field
	if result.File == "" {
		t.Error("JSON output 'file' field should not be empty")
	}

	// Verify findings have required fields
	for i, finding := range result.Findings {
		if finding.RuleID == "" {
			t.Errorf("finding %d missing rule_id", i)
		}
		if finding.Severity == "" {
			t.Errorf("finding %d missing severity", i)
		}
		if finding.Line == 0 {
			t.Errorf("finding %d missing line number", i)
		}
		if finding.Message == "" {
			t.Errorf("finding %d missing message", i)
		}
	}

	// Verify summary totals match findings count
	if result.Summary.Total != len(result.Findings) {
		t.Errorf("summary total (%d) doesn't match findings count (%d)", result.Summary.Total, len(result.Findings))
	}
}

// TestAllInstructionTypes tests parsing of various Dockerfile instructions.
// Requirements: 1.1
func TestAllInstructionTypes(t *testing.T) {
	dockerfile := `FROM alpine:3.18 AS base
ARG VERSION=1.0
ENV APP_HOME=/app
LABEL maintainer="test@example.com"
WORKDIR /app
COPY --chown=nobody:nobody . .
ADD https://example.com/file.tar.gz /tmp/
RUN echo "building"
EXPOSE 8080/tcp
VOLUME ["/data"]
USER nobody
HEALTHCHECK --interval=30s CMD wget -q --spider http://localhost/ || exit 1
STOPSIGNAL SIGTERM
SHELL ["/bin/bash", "-c"]
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["--help"]
`
	path := createTempDockerfile(t, dockerfile)
	var stdout, stderr bytes.Buffer

	exitCode := run([]string{path}, nil, &stdout, &stderr)

	// Should parse successfully (not a fatal error)
	if exitCode == ExitFatal {
		t.Errorf("failed to parse Dockerfile with all instruction types: %s", stderr.String())
	}
}

func TestPrintVersionWithBuildInfo(t *testing.T) {
	origVersion, origCommit, origDate := version, commit, date
	version, commit, date = "1.2.3", "abc123", "2025-01-01"
	defer func() {
		version, commit, date = origVersion, origCommit, origDate
	}()

	var buf bytes.Buffer
	printVersion(&buf)
	output := buf.String()

	if !strings.Contains(output, "1.2.3") {
		t.Errorf("version output should include version, got: %s", output)
	}
	if !strings.Contains(output, "abc123") {
		t.Errorf("version output should include commit, got: %s", output)
	}
	if !strings.Contains(output, "2025-01-01") {
		t.Errorf("version output should include date, got: %s", output)
	}
}
