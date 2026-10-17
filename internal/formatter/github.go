package formatter

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/devblac/docker-lint/internal/ast"
)

// GitHubFormatter writes findings as GitHub Actions workflow commands, which
// GitHub renders as annotations on the lines of a pull request's diff.
//
// Unlike SARIF upload, annotations need no code scanning (GitHub Advanced
// Security on private repositories), so they work in any repository.
type GitHubFormatter struct {
	// Filename is the path of the analyzed file, relative to the repository root.
	Filename string
	// Quiet suppresses informational findings.
	Quiet bool
}

// NewGitHubFormatter creates a new GitHubFormatter.
func NewGitHubFormatter(filename string, quiet bool) *GitHubFormatter {
	return &GitHubFormatter{Filename: filename, Quiet: quiet}
}

// Format writes one workflow command per finding:
// ::warning file=Dockerfile,line=3,col=1,title=DKL101::message
func (f *GitHubFormatter) Format(findings []ast.Finding, w io.Writer) error {
	file := strings.TrimPrefix(filepath.ToSlash(f.Filename), "./")

	for _, finding := range findings {
		if f.Quiet && finding.Severity == ast.SeverityInfo {
			continue
		}

		message := finding.Message
		if finding.Suggestion != "" {
			message += "\nSuggestion: " + finding.Suggestion
		}

		// Input read from stdin has no file to annotate; the command still
		// shows up in the workflow run summary.
		properties := "title=" + escapeProperty(finding.RuleID)
		if file != "stdin" {
			properties = fmt.Sprintf("file=%s,line=%d,col=%d,%s",
				escapeProperty(file), max(finding.Line, 1), max(finding.Column, 1), properties)
		}

		if _, err := fmt.Fprintf(w, "::%s %s::%s\n",
			annotationLevel(finding.Severity), properties, escapeData(message)); err != nil {
			return err
		}
	}

	return nil
}

// annotationLevel maps a severity to a workflow command.
func annotationLevel(severity ast.Severity) string {
	switch severity {
	case ast.SeverityError:
		return "error"
	case ast.SeverityWarning:
		return "warning"
	default:
		return "notice"
	}
}

// escapeData escapes a workflow command's message, as @actions/core does.
func escapeData(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	return strings.ReplaceAll(s, "\n", "%0A")
}

// escapeProperty escapes a workflow command's property value, as @actions/core does.
func escapeProperty(s string) string {
	s = escapeData(s)
	s = strings.ReplaceAll(s, ":", "%3A")
	return strings.ReplaceAll(s, ",", "%2C")
}
