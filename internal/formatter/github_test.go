package formatter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

func TestGitHubFormatter(t *testing.T) {
	findings := []ast.Finding{
		{RuleID: "DKL101", Severity: ast.SeverityWarning, Line: 3, Column: 1, Message: "Image 'alpine' has no tag", Suggestion: "Pin a version"},
		{RuleID: "DKL001", Severity: ast.SeverityError, Line: 1, Column: 1, Message: "RUN appears before FROM"},
		{RuleID: "DKL404", Severity: ast.SeverityInfo, Line: 9, Column: 1, Message: "No HEALTHCHECK"},
	}

	var buf bytes.Buffer
	if err := NewGitHubFormatter("./services/api/Dockerfile", false).Format(findings, &buf); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	want := []string{
		"::warning file=services/api/Dockerfile,line=3,col=1,title=DKL101::Image 'alpine' has no tag%0ASuggestion: Pin a version",
		"::error file=services/api/Dockerfile,line=1,col=1,title=DKL001::RUN appears before FROM",
		"::notice file=services/api/Dockerfile,line=9,col=1,title=DKL404::No HEALTHCHECK",
	}
	got := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestGitHubFormatterEscaping(t *testing.T) {
	findings := []ast.Finding{
		{RuleID: "DKL301", Severity: ast.SeverityWarning, Line: 2, Column: 1, Message: "100% sure: a,b\nnext"},
	}

	var buf bytes.Buffer
	if err := NewGitHubFormatter("dir,with:odd/Dockerfile", false).Format(findings, &buf); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Properties escape ':' and ','; the message only escapes '%' and newlines.
	want := "::warning file=dir%2Cwith%3Aodd/Dockerfile,line=2,col=1,title=DKL301::100%25 sure: a,b%0Anext\n"
	if buf.String() != want {
		t.Errorf("got  %q\nwant %q", buf.String(), want)
	}
}

func TestGitHubFormatterStdinAndQuiet(t *testing.T) {
	findings := []ast.Finding{
		{RuleID: "DKL404", Severity: ast.SeverityInfo, Line: 1, Column: 1, Message: "No HEALTHCHECK"},
		{RuleID: "DKL101", Severity: ast.SeverityWarning, Line: 1, Column: 1, Message: "No tag"},
	}

	var buf bytes.Buffer
	if err := NewGitHubFormatter("stdin", true).Format(findings, &buf); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// stdin has no file to annotate, and quiet mode drops the info finding.
	if want := "::warning title=DKL101::No tag\n"; buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}
