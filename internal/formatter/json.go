package formatter

import (
	"encoding/json"
	"io"

	"github.com/devblac/docker-lint/internal/ast"
)

// JSONFinding represents a single finding in JSON output format.
type JSONFinding struct {
	RuleID     string `json:"rule_id"`
	Severity   string `json:"severity"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// JSONSummary represents the summary section of JSON output.
type JSONSummary struct {
	Total    int `json:"total"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

// JSONOutput represents the complete JSON output structure.
type JSONOutput struct {
	File     string        `json:"file"`
	Findings []JSONFinding `json:"findings"`
	Summary  JSONSummary   `json:"summary"`
}

// JSONFormatter formats findings as JSON for machine consumption.
type JSONFormatter struct {
	// Filename is the name of the file being analyzed.
	Filename string
	// Quiet suppresses informational findings in the output.
	Quiet bool
}

// NewJSONFormatter creates a new JSONFormatter with the given filename.
func NewJSONFormatter(filename string, quiet bool) *JSONFormatter {
	return &JSONFormatter{
		Filename: filename,
		Quiet:    quiet,
	}
}


// Format writes the findings to the given writer as valid JSON.
func (f *JSONFormatter) Format(findings []ast.Finding, w io.Writer) error {
	output := JSONOutput{
		File:     f.Filename,
		Findings: make([]JSONFinding, 0),
		Summary: JSONSummary{
			Total:    0,
			Errors:   0,
			Warnings: 0,
			Info:     0,
		},
	}

	for _, finding := range findings {
		// Skip info-level findings in quiet mode
		if f.Quiet && finding.Severity == ast.SeverityInfo {
			continue
		}

		jsonFinding := JSONFinding{
			RuleID:     finding.RuleID,
			Severity:   finding.Severity.String(),
			Line:       finding.Line,
			Column:     finding.Column,
			Message:    finding.Message,
			Suggestion: finding.Suggestion,
		}
		output.Findings = append(output.Findings, jsonFinding)

		// Update summary counts
		switch finding.Severity {
		case ast.SeverityError:
			output.Summary.Errors++
		case ast.SeverityWarning:
			output.Summary.Warnings++
		case ast.SeverityInfo:
			output.Summary.Info++
		}
		output.Summary.Total++
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
