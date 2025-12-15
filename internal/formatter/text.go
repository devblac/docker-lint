// Package formatter provides output formatters for lint findings.
package formatter

import (
	"fmt"
	"io"

	"github.com/devblac/docker-lint/internal/ast"
)

// TextFormatter formats findings as human-readable text.
type TextFormatter struct {
	// Filename is the name of the file being analyzed (for output).
	Filename string
	// Quiet suppresses informational messages, showing only warnings and errors.
	Quiet bool
}

// NewTextFormatter creates a new TextFormatter with the given filename.
func NewTextFormatter(filename string, quiet bool) *TextFormatter {
	return &TextFormatter{
		Filename: filename,
		Quiet:    quiet,
	}
}

// Format writes the findings to the given writer in human-readable text format.
// Format: file:line:column: [severity] rule_id: message
func (f *TextFormatter) Format(findings []ast.Finding, w io.Writer) error {
	for _, finding := range findings {
		// Skip info-level findings in quiet mode
		if f.Quiet && finding.Severity == ast.SeverityInfo {
			continue
		}

		// Format: file:line:column: [severity] rule_id: message
		line := fmt.Sprintf("%s:%d:%d: [%s] %s: %s",
			f.Filename,
			finding.Line,
			finding.Column,
			finding.Severity.String(),
			finding.RuleID,
			finding.Message,
		)

		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}

		// Include suggestion if available
		if finding.Suggestion != "" {
			suggestion := fmt.Sprintf("  Suggestion: %s", finding.Suggestion)
			if _, err := fmt.Fprintln(w, suggestion); err != nil {
				return err
			}
		}
	}

	return nil
}
