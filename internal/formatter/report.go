package formatter

import (
	"io"

	"github.com/devblac/docker-lint/internal/ast"
)

// Report holds the findings of one linted file.
type Report struct {
	// Filename is the path of the file, as shown in the output.
	Filename string
	// Findings are the file's findings, sorted by line.
	Findings []ast.Finding
}

// FormatReports writes the findings of several files as consecutive text
// blocks; each line already names its file.
func (f *TextFormatter) FormatReports(reports []Report, w io.Writer) error {
	for _, r := range reports {
		if err := NewTextFormatter(r.Filename, f.Quiet).Format(r.Findings, w); err != nil {
			return err
		}
	}
	return nil
}

// FormatReports writes the workflow commands of several files.
func (f *GitHubFormatter) FormatReports(reports []Report, w io.Writer) error {
	for _, r := range reports {
		if err := NewGitHubFormatter(r.Filename, f.Quiet).Format(r.Findings, w); err != nil {
			return err
		}
	}
	return nil
}
