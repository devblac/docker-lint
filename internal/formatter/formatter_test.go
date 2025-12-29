// Package formatter provides output formatters for lint findings.
package formatter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

func TestTextFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		quiet    bool
		findings []ast.Finding
		want     []string
		notWant  []string
	}{
		{
			name:     "single warning finding",
			filename: "Dockerfile",
			quiet:    false,
			findings: []ast.Finding{
				{
					RuleID:   "DL3006",
					Severity: ast.SeverityWarning,
					Line:     1,
					Column:   1,
					Message:  "Missing explicit image tag",
				},
			},
			want: []string{
				"Dockerfile:1:1: [warning] DL3006: Missing explicit image tag",
			},
		},
		{
			name:     "finding with suggestion",
			filename: "Dockerfile",
			quiet:    false,
			findings: []ast.Finding{
				{
					RuleID:     "DL3007",
					Severity:   ast.SeverityWarning,
					Line:       5,
					Column:     1,
					Message:    "Using 'latest' tag",
					Suggestion: "Use explicit version tag like 'alpine:3.18'",
				},
			},
			want: []string{
				"Dockerfile:5:1: [warning] DL3007: Using 'latest' tag",
				"Suggestion: Use explicit version tag like 'alpine:3.18'",
			},
		},
		{
			name:     "quiet mode filters info",
			filename: "Dockerfile",
			quiet:    true,
			findings: []ast.Finding{
				{
					RuleID:   "DL5001",
					Severity: ast.SeverityInfo,
					Line:     3,
					Column:   1,
					Message:  "Wildcard in COPY source",
				},
				{
					RuleID:   "DL3006",
					Severity: ast.SeverityWarning,
					Line:     1,
					Column:   1,
					Message:  "Missing explicit image tag",
				},
			},
			want: []string{
				"Dockerfile:1:1: [warning] DL3006: Missing explicit image tag",
			},
			notWant: []string{
				"DL5001",
				"[info]",
			},
		},
		{
			name:     "error severity",
			filename: "Dockerfile.prod",
			quiet:    false,
			findings: []ast.Finding{
				{
					RuleID:   "DL3000",
					Severity: ast.SeverityError,
					Line:     10,
					Column:   5,
					Message:  "Invalid syntax",
				},
			},
			want: []string{
				"Dockerfile.prod:10:5: [error] DL3000: Invalid syntax",
			},
		},
		{
			name:     "empty findings",
			filename: "Dockerfile",
			quiet:    false,
			findings: []ast.Finding{},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewTextFormatter(tt.filename, tt.quiet)
			var buf bytes.Buffer

			err := formatter.Format(tt.findings, &buf)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			output := buf.String()

			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Format() output missing expected string: %q\nGot: %s", want, output)
				}
			}

			for _, notWant := range tt.notWant {
				if strings.Contains(output, notWant) {
					t.Errorf("Format() output contains unexpected string: %q\nGot: %s", notWant, output)
				}
			}
		})
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		quiet    bool
		findings []ast.Finding
		wantErr  bool
	}{
		{
			name:     "single finding",
			filename: "Dockerfile",
			quiet:    false,
			findings: []ast.Finding{
				{
					RuleID:   "DL3006",
					Severity: ast.SeverityWarning,
					Line:     1,
					Column:   1,
					Message:  "Missing explicit image tag",
				},
			},
		},
		{
			name:     "multiple findings with all severities",
			filename: "Dockerfile",
			quiet:    false,
			findings: []ast.Finding{
				{
					RuleID:   "DL3000",
					Severity: ast.SeverityError,
					Line:     1,
					Column:   1,
					Message:  "Error message",
				},
				{
					RuleID:   "DL3006",
					Severity: ast.SeverityWarning,
					Line:     2,
					Column:   1,
					Message:  "Warning message",
				},
				{
					RuleID:   "DL5001",
					Severity: ast.SeverityInfo,
					Line:     3,
					Column:   1,
					Message:  "Info message",
				},
			},
		},
		{
			name:     "quiet mode filters info",
			filename: "Dockerfile",
			quiet:    true,
			findings: []ast.Finding{
				{
					RuleID:   "DL5001",
					Severity: ast.SeverityInfo,
					Line:     3,
					Column:   1,
					Message:  "Info message",
				},
				{
					RuleID:   "DL3006",
					Severity: ast.SeverityWarning,
					Line:     1,
					Column:   1,
					Message:  "Warning message",
				},
			},
		},
		{
			name:     "empty findings",
			filename: "Dockerfile",
			quiet:    false,
			findings: []ast.Finding{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewJSONFormatter(tt.filename, tt.quiet)
			var buf bytes.Buffer

			err := formatter.Format(tt.findings, &buf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Format() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Verify output is valid JSON
			var output JSONOutput
			if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
				t.Fatalf("Format() produced invalid JSON: %v\nOutput: %s", err, buf.String())
			}

			// Verify filename
			if output.File != tt.filename {
				t.Errorf("Format() file = %q, want %q", output.File, tt.filename)
			}

			// Count expected findings (accounting for quiet mode)
			expectedCount := 0
			expectedErrors := 0
			expectedWarnings := 0
			expectedInfo := 0
			for _, f := range tt.findings {
				if tt.quiet && f.Severity == ast.SeverityInfo {
					continue
				}
				expectedCount++
				switch f.Severity {
				case ast.SeverityError:
					expectedErrors++
				case ast.SeverityWarning:
					expectedWarnings++
				case ast.SeverityInfo:
					expectedInfo++
				}
			}

			// Verify findings count
			if len(output.Findings) != expectedCount {
				t.Errorf("Format() findings count = %d, want %d", len(output.Findings), expectedCount)
			}

			// Verify summary
			if output.Summary.Total != expectedCount {
				t.Errorf("Format() summary.total = %d, want %d", output.Summary.Total, expectedCount)
			}
			if output.Summary.Errors != expectedErrors {
				t.Errorf("Format() summary.errors = %d, want %d", output.Summary.Errors, expectedErrors)
			}
			if output.Summary.Warnings != expectedWarnings {
				t.Errorf("Format() summary.warnings = %d, want %d", output.Summary.Warnings, expectedWarnings)
			}
			if output.Summary.Info != expectedInfo {
				t.Errorf("Format() summary.info = %d, want %d", output.Summary.Info, expectedInfo)
			}
		})
	}
}

func TestJSONFormatter_SchemaConformance(t *testing.T) {
	findings := []ast.Finding{
		{
			RuleID:     "DL3006",
			Severity:   ast.SeverityWarning,
			Line:       1,
			Column:     1,
			Message:    "Missing explicit image tag",
			Suggestion: "Use 'FROM alpine:3.18' instead of 'FROM alpine'",
		},
	}

	formatter := NewJSONFormatter("Dockerfile", false)
	var buf bytes.Buffer

	err := formatter.Format(findings, &buf)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var output JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Format() produced invalid JSON: %v", err)
	}

	// Verify finding fields
	if len(output.Findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(output.Findings))
	}

	f := output.Findings[0]
	if f.RuleID != "DL3006" {
		t.Errorf("Finding rule_id = %q, want %q", f.RuleID, "DL3006")
	}
	if f.Severity != "warning" {
		t.Errorf("Finding severity = %q, want %q", f.Severity, "warning")
	}
	if f.Line != 1 {
		t.Errorf("Finding line = %d, want %d", f.Line, 1)
	}
	if f.Column != 1 {
		t.Errorf("Finding column = %d, want %d", f.Column, 1)
	}
	if f.Message != "Missing explicit image tag" {
		t.Errorf("Finding message = %q, want %q", f.Message, "Missing explicit image tag")
	}
	if f.Suggestion != "Use 'FROM alpine:3.18' instead of 'FROM alpine'" {
		t.Errorf("Finding suggestion = %q, want %q", f.Suggestion, "Use 'FROM alpine:3.18' instead of 'FROM alpine'")
	}
}
