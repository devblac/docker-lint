package formatter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

func testToolInfo() ToolInfo {
	return ToolInfo{
		Name:           "docker-lint",
		Version:        "1.2.3",
		InformationURI: "https://github.com/devblac/docker-lint",
		Rules: []RuleInfo{
			{ID: "DKL101", Name: "Missing explicit image tag", Description: "Always tag images", Severity: ast.SeverityWarning},
			{ID: "DKL303", Name: "Final image runs as root", Description: "Set a non-root USER", Severity: ast.SeverityWarning},
			{ID: "DKL404", Name: "Missing HEALTHCHECK", Description: "Add a HEALTHCHECK", Severity: ast.SeverityInfo},
			{ID: "DKL001", Name: "Instruction before FROM", Description: "Start with FROM", Severity: ast.SeverityError},
		},
	}
}

// decodeSARIF formats findings and decodes the result into a generic map.
func decodeSARIF(t *testing.T, f *SARIFFormatter, findings []ast.Finding) map[string]interface{} {
	t.Helper()
	var buf bytes.Buffer
	if err := f.Format(findings, &buf); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	var log map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	return log
}

func TestSARIFFormatterStructure(t *testing.T) {
	findings := []ast.Finding{
		{RuleID: "DKL101", Severity: ast.SeverityWarning, Line: 3, Column: 1, Message: "Image 'alpine' has no tag", Suggestion: "Pin a version"},
		{RuleID: "DKL001", Severity: ast.SeverityError, Line: 1, Column: 1, Message: "RUN appears before FROM"},
		{RuleID: "DKL404", Severity: ast.SeverityInfo, Line: 9, Column: 1, Message: "No HEALTHCHECK"},
	}

	log := decodeSARIF(t, NewSARIFFormatter("services/api/Dockerfile", false, testToolInfo()), findings)

	if log["version"] != "2.1.0" {
		t.Errorf("version = %v, want 2.1.0", log["version"])
	}
	if log["$schema"] != sarifSchema {
		t.Errorf("$schema = %v, want %s", log["$schema"], sarifSchema)
	}

	runs := log["runs"].([]interface{})
	if len(runs) != 1 {
		t.Fatalf("len(runs) = %d, want 1", len(runs))
	}
	run := runs[0].(map[string]interface{})

	driver := run["tool"].(map[string]interface{})["driver"].(map[string]interface{})
	if driver["name"] != "docker-lint" || driver["version"] != "1.2.3" {
		t.Errorf("driver = %v", driver)
	}
	rules := driver["rules"].([]interface{})
	if len(rules) != 4 {
		t.Fatalf("len(rules) = %d, want 4", len(rules))
	}
	first := rules[0].(map[string]interface{})
	if first["id"] != "DKL101" || first["name"] != "MissingExplicitImageTag" {
		t.Errorf("rules[0] = %v", first)
	}
	if first["helpUri"] != "https://github.com/devblac/docker-lint#rules" {
		t.Errorf("helpUri = %v", first["helpUri"])
	}

	results := run["results"].([]interface{})
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	wantLevels := []string{"warning", "error", "note"}
	wantIndexes := []float64{0, 3, 2}
	for i, r := range results {
		result := r.(map[string]interface{})
		if result["level"] != wantLevels[i] {
			t.Errorf("results[%d].level = %v, want %s", i, result["level"], wantLevels[i])
		}
		if result["ruleIndex"] != wantIndexes[i] {
			t.Errorf("results[%d].ruleIndex = %v, want %v", i, result["ruleIndex"], wantIndexes[i])
		}
	}

	location := results[0].(map[string]interface{})["locations"].([]interface{})[0].(map[string]interface{})["physicalLocation"].(map[string]interface{})
	if uri := location["artifactLocation"].(map[string]interface{})["uri"]; uri != "services/api/Dockerfile" {
		t.Errorf("uri = %v, want services/api/Dockerfile", uri)
	}
	if line := location["region"].(map[string]interface{})["startLine"]; line != float64(3) {
		t.Errorf("startLine = %v, want 3", line)
	}

	message := results[0].(map[string]interface{})["message"].(map[string]interface{})["text"].(string)
	if !strings.Contains(message, "Image 'alpine' has no tag") || !strings.Contains(message, "Suggestion: Pin a version") {
		t.Errorf("message = %q", message)
	}
}

func TestSARIFFormatterNoFindings(t *testing.T) {
	var buf bytes.Buffer
	if err := NewSARIFFormatter("Dockerfile", false, testToolInfo()).Format(nil, &buf); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	// Consumers expect an empty array, not null, when nothing was found.
	if !strings.Contains(buf.String(), `"results": []`) {
		t.Errorf("expected an empty results array:\n%s", buf.String())
	}
}

func TestSARIFFormatterQuiet(t *testing.T) {
	findings := []ast.Finding{
		{RuleID: "DKL404", Severity: ast.SeverityInfo, Line: 1, Column: 1, Message: "No HEALTHCHECK"},
		{RuleID: "DKL101", Severity: ast.SeverityWarning, Line: 1, Column: 1, Message: "No tag"},
	}
	log := decodeSARIF(t, NewSARIFFormatter("Dockerfile", true, testToolInfo()), findings)
	results := log["runs"].([]interface{})[0].(map[string]interface{})["results"].([]interface{})
	if len(results) != 1 {
		t.Fatalf("quiet mode: len(results) = %d, want 1", len(results))
	}
}

func TestArtifactURI(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"Dockerfile", "Dockerfile"},
		{"./Dockerfile", "Dockerfile"},
		{"services/api/Dockerfile", "services/api/Dockerfile"},
		{"my app/Dockerfile.prod", "my%20app/Dockerfile.prod"},
		{"/home/ci/repo/Dockerfile", "file:///home/ci/repo/Dockerfile"},
	}
	for _, tt := range tests {
		if got := artifactURI(tt.path); got != tt.want {
			t.Errorf("artifactURI(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestSARIFRuleName(t *testing.T) {
	tests := map[string]string{
		"Using 'latest' tag":           "UsingLatestTag",
		"ADD where COPY would suffice": "ADDWhereCOPYWouldSuffice",
		"Wildcard in COPY/ADD source":  "WildcardInCOPYADDSource",
	}
	for in, want := range tests {
		if got := sarifRuleName(in); got != want {
			t.Errorf("sarifRuleName(%q) = %q, want %q", in, got, want)
		}
	}
}
