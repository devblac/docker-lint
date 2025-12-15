package analyzer

import (
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
	"github.com/devblac/docker-lint/internal/parser"
	"github.com/devblac/docker-lint/internal/rules"
)

func TestAnalyzer_Analyze_BasicFindings(t *testing.T) {
	// Dockerfile with issues: missing tag, no USER instruction
	dockerfile := `FROM ubuntu
RUN apt-get update
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// Should have findings for missing tag and no USER
	if len(findings) == 0 {
		t.Error("Expected findings but got none")
	}

	// Check that we have a missing tag finding
	hasMissingTag := false
	for _, f := range findings {
		if f.RuleID == rules.RuleMissingTag {
			hasMissingTag = true
			break
		}
	}
	if !hasMissingTag {
		t.Error("Expected DL3006 (missing tag) finding")
	}
}

func TestAnalyzer_Analyze_GlobalIgnore(t *testing.T) {
	dockerfile := `FROM ubuntu
RUN apt-get update
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	// Ignore the missing tag rule
	analyzer := NewWithDefaults(Config{
		IgnoreRules: []string{rules.RuleMissingTag},
	})
	findings := analyzer.Analyze(df)

	// Should NOT have the missing tag finding
	for _, f := range findings {
		if f.RuleID == rules.RuleMissingTag {
			t.Error("Expected DL3006 to be ignored but it was reported")
		}
	}
}


func TestAnalyzer_Analyze_InlineIgnore(t *testing.T) {
	// Dockerfile with inline ignore comment
	dockerfile := `# docker-lint ignore: DL3006
FROM ubuntu
RUN apt-get update
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// Should NOT have the missing tag finding for line 2 (ubuntu)
	for _, f := range findings {
		if f.RuleID == rules.RuleMissingTag && f.Line == 2 {
			t.Error("Expected DL3006 to be ignored via inline comment but it was reported")
		}
	}
}

func TestAnalyzer_Analyze_InlineIgnoreMultipleRules(t *testing.T) {
	// Dockerfile with inline ignore for multiple rules
	dockerfile := `# docker-lint ignore: DL3006, DL3008
FROM ubuntu
RUN apt-get update
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// Should NOT have DL3006 or DL3008 for line 2
	for _, f := range findings {
		if f.Line == 2 && (f.RuleID == rules.RuleMissingTag || f.RuleID == rules.RuleLargeBaseImage) {
			t.Errorf("Expected %s to be ignored via inline comment but it was reported", f.RuleID)
		}
	}
}

func TestAnalyzer_Analyze_NilDockerfile(t *testing.T) {
	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(nil)

	if findings != nil {
		t.Error("Expected nil findings for nil Dockerfile")
	}
}

func TestAnalyzer_Analyze_EmptyDockerfile(t *testing.T) {
	df := &ast.Dockerfile{
		Instructions:  []ast.Instruction{},
		InlineIgnores: make(map[int][]string),
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// Empty dockerfile should not cause panic
	if findings == nil {
		findings = []ast.Finding{}
	}
}

func TestAnalyzer_Analyze_DeterministicOrder(t *testing.T) {
	dockerfile := `FROM ubuntu
FROM debian
RUN apt-get update
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})

	// Run analysis multiple times
	var prevFindings []ast.Finding
	for i := 0; i < 5; i++ {
		findings := analyzer.Analyze(df)

		if prevFindings != nil {
			// Compare with previous run
			if len(findings) != len(prevFindings) {
				t.Errorf("Run %d: findings count changed from %d to %d", i, len(prevFindings), len(findings))
			}

			for j := range findings {
				if j < len(prevFindings) {
					if findings[j].RuleID != prevFindings[j].RuleID || findings[j].Line != prevFindings[j].Line {
						t.Errorf("Run %d: finding %d differs from previous run", i, j)
					}
				}
			}
		}
		prevFindings = findings
	}
}

func TestAnalyzer_AnalyzeWithRules(t *testing.T) {
	dockerfile := `FROM ubuntu
RUN apt-get update
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})

	// Only run the missing tag rule
	findings := analyzer.AnalyzeWithRules(df, []string{rules.RuleMissingTag})

	// Should only have findings from the specified rule
	for _, f := range findings {
		if f.RuleID != rules.RuleMissingTag {
			t.Errorf("Expected only DL3006 findings, got %s", f.RuleID)
		}
	}

	// Should have at least one finding
	hasMissingTag := false
	for _, f := range findings {
		if f.RuleID == rules.RuleMissingTag {
			hasMissingTag = true
			break
		}
	}
	if !hasMissingTag {
		t.Error("Expected DL3006 finding")
	}
}

func TestAnalyzer_Registry(t *testing.T) {
	registry := rules.NewRegistry()
	analyzer := New(registry, Config{})

	if analyzer.Registry() != registry {
		t.Error("Registry() should return the same registry passed to New()")
	}
}

func TestAnalyzer_Analyze_CleanDockerfile(t *testing.T) {
	// A well-formed Dockerfile that should have minimal findings
	dockerfile := `FROM alpine:3.18
WORKDIR /app
COPY . .
RUN apk add --no-cache curl && rm -rf /var/cache/apk/*
USER nobody
HEALTHCHECK CMD curl -f http://localhost/ || exit 1
CMD ["./app"]
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// Should have very few or no critical findings
	errorCount := 0
	for _, f := range findings {
		if f.Severity == ast.SeverityError {
			errorCount++
		}
	}

	if errorCount > 0 {
		t.Errorf("Expected no errors for clean Dockerfile, got %d", errorCount)
	}
}

func TestNew_WithCustomRegistry(t *testing.T) {
	// Create a custom registry with only one rule
	registry := rules.NewRegistry()
	registry.Register(&rules.MissingTagRule{})

	analyzer := New(registry, Config{})

	dockerfile := `FROM ubuntu
RUN apt-get update
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	findings := analyzer.Analyze(df)

	// Should only have findings from the registered rule
	for _, f := range findings {
		if f.RuleID != rules.RuleMissingTag {
			t.Errorf("Expected only DL3006 findings from custom registry, got %s", f.RuleID)
		}
	}
}

func TestAnalyzer_Analyze_SortedByLine(t *testing.T) {
	dockerfile := `FROM ubuntu
FROM debian
FROM alpine
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// Verify findings are sorted by line number
	for i := 1; i < len(findings); i++ {
		if findings[i].Line < findings[i-1].Line {
			t.Errorf("Findings not sorted by line: line %d came after line %d",
				findings[i].Line, findings[i-1].Line)
		}
	}
}

func TestAnalyzer_Analyze_InlineIgnoreOnlyAffectsNextLine(t *testing.T) {
	// The inline ignore should only affect line 2, not line 3
	dockerfile := `# docker-lint ignore: DL3006
FROM ubuntu
FROM debian
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// Line 2 (ubuntu) should be ignored
	// Line 3 (debian) should NOT be ignored
	hasLine3Finding := false
	for _, f := range findings {
		if f.RuleID == rules.RuleMissingTag {
			if f.Line == 2 {
				t.Error("Line 2 should be ignored via inline comment")
			}
			if f.Line == 3 {
				hasLine3Finding = true
			}
		}
	}

	if !hasLine3Finding {
		t.Error("Line 3 should NOT be ignored - inline ignore only affects next line")
	}
}

func TestAnalyzer_Analyze_CombinedGlobalAndInlineIgnore(t *testing.T) {
	dockerfile := `# docker-lint ignore: DL3008
FROM ubuntu
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	// Global ignore for DL3006, inline ignore for DL3008
	analyzer := NewWithDefaults(Config{
		IgnoreRules: []string{rules.RuleMissingTag},
	})
	findings := analyzer.Analyze(df)

	// Neither DL3006 nor DL3008 should appear for line 2
	for _, f := range findings {
		if f.Line == 2 {
			if f.RuleID == rules.RuleMissingTag {
				t.Error("DL3006 should be globally ignored")
			}
			if f.RuleID == rules.RuleLargeBaseImage {
				t.Error("DL3008 should be ignored via inline comment")
			}
		}
	}
}

func TestAnalyzer_Analyze_MultiStageDockerfile(t *testing.T) {
	dockerfile := `FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o app

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/app .
USER nobody
CMD ["./app"]
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// Should parse and analyze without errors
	// The second stage has USER instruction, so DL4002 should not fire for it
	// (depending on rule implementation - this tests that multi-stage works)
	if df == nil {
		t.Error("Expected parsed Dockerfile")
	}

	// Just verify we got some findings (the exact rules depend on implementation)
	_ = findings
}

func TestAnalyzer_Analyze_InlineIgnoreCaseInsensitive(t *testing.T) {
	// Test that inline ignore parsing is case-insensitive
	dockerfile := `# DOCKER-LINT IGNORE: DL3006
FROM ubuntu
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// DL3006 should be ignored even with uppercase comment
	for _, f := range findings {
		if f.RuleID == rules.RuleMissingTag && f.Line == 2 {
			t.Error("DL3006 should be ignored via case-insensitive inline comment")
		}
	}
}

func TestAnalyzer_AnalyzeWithRules_EmptyRuleList(t *testing.T) {
	dockerfile := `FROM ubuntu
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.AnalyzeWithRules(df, []string{})

	// With empty rule list, should have no findings
	if len(findings) != 0 {
		t.Errorf("Expected no findings with empty rule list, got %d", len(findings))
	}
}

func TestAnalyzer_AnalyzeWithRules_NonExistentRule(t *testing.T) {
	dockerfile := `FROM ubuntu
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.AnalyzeWithRules(df, []string{"NONEXISTENT"})

	// Non-existent rule should result in no findings
	if len(findings) != 0 {
		t.Errorf("Expected no findings for non-existent rule, got %d", len(findings))
	}
}

func TestAnalyzer_AnalyzeWithRules_RespectsGlobalIgnore(t *testing.T) {
	dockerfile := `FROM ubuntu
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{
		IgnoreRules: []string{rules.RuleMissingTag},
	})

	// Request the rule that's globally ignored
	findings := analyzer.AnalyzeWithRules(df, []string{rules.RuleMissingTag})

	// Should have no findings because the rule is globally ignored
	if len(findings) != 0 {
		t.Errorf("Expected no findings when rule is globally ignored, got %d", len(findings))
	}
}

func TestAnalyzer_Analyze_RealWorldDockerfile(t *testing.T) {
	// A more realistic Dockerfile with various issues
	dockerfile := `FROM node:latest
WORKDIR app
ENV API_KEY=secret123
RUN npm install
RUN npm run build
ADD https://example.com/file.tar.gz /tmp/
COPY . .
CMD ["node", "server.js"]
`
	df, err := parser.ParseString(dockerfile)
	if err != nil {
		t.Fatalf("Failed to parse Dockerfile: %v", err)
	}

	analyzer := NewWithDefaults(Config{})
	findings := analyzer.Analyze(df)

	// Should detect multiple issues
	ruleIDs := make(map[string]bool)
	for _, f := range findings {
		ruleIDs[f.RuleID] = true
	}

	// Expected findings:
	// - DL3007: latest tag
	// - DL3003: relative WORKDIR
	// - DL4000: secret in ENV
	// - DL3010: consecutive RUN
	// - DL4003: ADD with URL
	// - DL4002: no USER
	// - DL5000: no HEALTHCHECK

	expectedRules := []string{
		rules.RuleLatestTag,
		rules.RuleRelativeWorkdir,
	}

	for _, expected := range expectedRules {
		if !ruleIDs[expected] {
			t.Errorf("Expected finding for rule %s", expected)
		}
	}
}

// Helper to check if a string contains a substring (for message validation)
func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}
