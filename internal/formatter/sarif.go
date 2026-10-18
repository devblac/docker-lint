package formatter

import (
	"encoding/json"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/devblac/docker-lint/internal/ast"
)

// SARIF 2.1.0 is the version GitHub code scanning and most other consumers read.
const (
	sarifVersion = "2.1.0"
	sarifSchema  = "https://json.schemastore.org/sarif-2.1.0.json"
)

// RuleInfo describes a lint rule for output formats that publish rule metadata.
type RuleInfo struct {
	ID          string
	Name        string
	Description string
	Severity    ast.Severity
}

// ToolInfo describes docker-lint itself in SARIF output.
type ToolInfo struct {
	Name           string
	Version        string
	InformationURI string
	Rules          []RuleInfo
}

// SARIFFormatter formats findings as a SARIF 2.1.0 log, the format GitHub code
// scanning ingests to show findings as alerts on pull requests.
type SARIFFormatter struct {
	// Filename is the path of the analyzed file, reported as the artifact URI.
	Filename string
	// Quiet suppresses informational findings.
	Quiet bool
	// Tool carries the tool version and the metadata of every rule.
	Tool ToolInfo
}

// NewSARIFFormatter creates a new SARIFFormatter.
func NewSARIFFormatter(filename string, quiet bool, tool ToolInfo) *SARIFFormatter {
	return &SARIFFormatter{Filename: filename, Quiet: quiet, Tool: tool}
}

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name"`
	ShortDescription     sarifMessage        `json:"shortDescription"`
	FullDescription      sarifMessage        `json:"fullDescription"`
	HelpURI              string              `json:"helpUri,omitempty"`
	DefaultConfiguration sarifConfiguration  `json:"defaultConfiguration"`
	Properties           sarifRuleProperties `json:"properties"`
}

type sarifRuleProperties struct {
	Tags []string `json:"tags"`
}

type sarifConfiguration struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	RuleIndex *int            `json:"ruleIndex,omitempty"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
}

// Format writes the findings to w as a SARIF log with a single run.
func (f *SARIFFormatter) Format(findings []ast.Finding, w io.Writer) error {
	return f.FormatReports([]Report{{Filename: f.Filename, Findings: findings}}, w)
}

// FormatReports writes the findings of several files as one SARIF run, the
// shape code scanning expects for a single analysis of a repository.
func (f *SARIFFormatter) FormatReports(reports []Report, w io.Writer) error {
	rules := make([]sarifRule, 0, len(f.Tool.Rules))
	ruleIndex := make(map[string]int, len(f.Tool.Rules))
	for i, rule := range f.Tool.Rules {
		ruleIndex[rule.ID] = i
		rules = append(rules, sarifRule{
			ID:                   rule.ID,
			Name:                 sarifRuleName(rule.Name),
			ShortDescription:     sarifMessage{Text: rule.Name},
			FullDescription:      sarifMessage{Text: rule.Description},
			HelpURI:              ruleHelpURI(f.Tool.InformationURI),
			DefaultConfiguration: sarifConfiguration{Level: sarifLevel(rule.Severity)},
			Properties:           sarifRuleProperties{Tags: []string{ruleCategory(rule.ID)}},
		})
	}

	results := make([]sarifResult, 0)
	for _, report := range reports {
		results = f.appendResults(results, report, ruleIndex)
	}

	log := sarifLog{
		Schema:  sarifSchema,
		Version: sarifVersion,
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           f.Tool.Name,
				Version:        f.Tool.Version,
				InformationURI: f.Tool.InformationURI,
				Rules:          rules,
			}},
			Results: results,
		}},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(log)
}

// appendResults converts the findings of one file into SARIF results.
func (f *SARIFFormatter) appendResults(results []sarifResult, report Report, ruleIndex map[string]int) []sarifResult {
	uri := artifactURI(report.Filename)
	for _, finding := range report.Findings {
		if f.Quiet && finding.Severity == ast.SeverityInfo {
			continue
		}

		text := finding.Message
		if finding.Suggestion != "" {
			text += "\nSuggestion: " + finding.Suggestion
		}

		result := sarifResult{
			RuleID:  finding.RuleID,
			Level:   sarifLevel(finding.Severity),
			Message: sarifMessage{Text: text},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: uri},
					Region:           sarifRegion{StartLine: max(finding.Line, 1), StartColumn: finding.Column},
				},
			}},
		}
		if idx, ok := ruleIndex[finding.RuleID]; ok {
			result.RuleIndex = &idx
		}
		results = append(results, result)
	}
	return results
}

// sarifLevel maps a severity to a SARIF result level.
func sarifLevel(severity ast.Severity) string {
	switch severity {
	case ast.SeverityError:
		return "error"
	case ast.SeverityWarning:
		return "warning"
	default:
		return "note"
	}
}

// sarifRuleName turns a human rule name into the PascalCase identifier SARIF
// recommends for reportingDescriptor.name: "Using 'latest' tag" -> "UsingLatestTag".
func sarifRuleName(name string) string {
	var b strings.Builder
	for _, word := range strings.FieldsFunc(name, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	}) {
		b.WriteString(strings.ToUpper(word[:1]) + word[1:])
	}
	return b.String()
}

// ruleCategory derives a rule's category tag from the hundreds digit of its ID.
func ruleCategory(id string) string {
	if len(id) == 6 && strings.HasPrefix(id, "DKL") {
		switch id[3] {
		case '0':
			return "correctness"
		case '1':
			return "base-image"
		case '2':
			return "efficiency"
		case '3':
			return "security"
		case '4':
			return "best-practice"
		}
	}
	return "docker"
}

// ruleHelpURI points at the rules reference in the project README.
func ruleHelpURI(informationURI string) string {
	if informationURI == "" {
		return ""
	}
	return strings.TrimSuffix(informationURI, "/") + "#rules"
}

// artifactURI converts a file path to a SARIF artifact URI. Relative paths stay
// relative, which code scanning resolves against the repository root; absolute
// paths become file:// URIs.
func artifactURI(path string) string {
	slashed := filepath.ToSlash(path)

	if filepath.IsAbs(path) || strings.HasPrefix(slashed, "/") {
		if !strings.HasPrefix(slashed, "/") {
			slashed = "/" + slashed // Windows drive paths: C:/x -> /C:/x
		}
		return (&url.URL{Scheme: "file", Path: slashed}).String()
	}

	slashed = strings.TrimPrefix(slashed, "./")
	segments := strings.Split(slashed, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
