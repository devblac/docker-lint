// Package analyzer provides the orchestration layer for running lint rules against Dockerfiles.
package analyzer

import (
	"sort"

	"github.com/docker-lint/docker-lint/internal/ast"
	"github.com/docker-lint/docker-lint/internal/rules"
)

// Config holds configuration options for the analyzer.
type Config struct {
	// IgnoreRules is a list of rule IDs to skip during analysis.
	IgnoreRules []string
}

// Analyzer orchestrates the execution of lint rules against a Dockerfile AST.
type Analyzer struct {
	registry *rules.RuleRegistry
	config   Config
}

// New creates a new Analyzer with the given registry and configuration.
func New(registry *rules.RuleRegistry, config Config) *Analyzer {
	return &Analyzer{
		registry: registry,
		config:   config,
	}
}

// NewWithDefaults creates a new Analyzer using the default rule registry.
func NewWithDefaults(config Config) *Analyzer {
	return New(rules.DefaultRegistry, config)
}

// Analyze runs all registered rules against the Dockerfile and returns findings.
// It respects both the global ignore configuration and inline ignore comments.
func (a *Analyzer) Analyze(dockerfile *ast.Dockerfile) []ast.Finding {
	if dockerfile == nil {
		return nil
	}

	// Build a set of globally ignored rules for fast lookup
	ignoredRules := make(map[string]bool)
	for _, ruleID := range a.config.IgnoreRules {
		ignoredRules[ruleID] = true
	}

	var allFindings []ast.Finding

	// Run each registered rule
	for _, rule := range a.registry.All() {
		// Skip globally ignored rules
		if ignoredRules[rule.ID()] {
			continue
		}

		// Execute the rule
		findings := rule.Check(dockerfile)

		// Filter findings based on inline ignores
		for _, finding := range findings {
			if a.isIgnoredByInlineComment(dockerfile, finding) {
				continue
			}
			allFindings = append(allFindings, finding)
		}
	}

	// Sort findings by line number, then by rule ID for deterministic output
	sort.Slice(allFindings, func(i, j int) bool {
		if allFindings[i].Line != allFindings[j].Line {
			return allFindings[i].Line < allFindings[j].Line
		}
		return allFindings[i].RuleID < allFindings[j].RuleID
	})

	return allFindings
}


// isIgnoredByInlineComment checks if a finding should be ignored based on inline comments.
// Inline ignore comments apply to the line immediately following the comment.
func (a *Analyzer) isIgnoredByInlineComment(dockerfile *ast.Dockerfile, finding ast.Finding) bool {
	if dockerfile.InlineIgnores == nil {
		return false
	}

	// Check if there's an inline ignore for this line
	ignoredRules, exists := dockerfile.InlineIgnores[finding.Line]
	if !exists {
		return false
	}

	// Check if the finding's rule ID is in the ignored list
	for _, ruleID := range ignoredRules {
		if ruleID == finding.RuleID {
			return true
		}
	}

	return false
}

// AnalyzeWithRules runs only the specified rules against the Dockerfile.
// This is useful for testing or when only specific rules should be applied.
func (a *Analyzer) AnalyzeWithRules(dockerfile *ast.Dockerfile, ruleIDs []string) []ast.Finding {
	if dockerfile == nil {
		return nil
	}

	// Build a set of globally ignored rules for fast lookup
	ignoredRules := make(map[string]bool)
	for _, ruleID := range a.config.IgnoreRules {
		ignoredRules[ruleID] = true
	}

	// Build a set of requested rules
	requestedRules := make(map[string]bool)
	for _, ruleID := range ruleIDs {
		requestedRules[ruleID] = true
	}

	var allFindings []ast.Finding

	// Run only the requested rules
	for _, rule := range a.registry.All() {
		// Skip if not in requested rules
		if !requestedRules[rule.ID()] {
			continue
		}

		// Skip globally ignored rules
		if ignoredRules[rule.ID()] {
			continue
		}

		// Execute the rule
		findings := rule.Check(dockerfile)

		// Filter findings based on inline ignores
		for _, finding := range findings {
			if a.isIgnoredByInlineComment(dockerfile, finding) {
				continue
			}
			allFindings = append(allFindings, finding)
		}
	}

	// Sort findings by line number, then by rule ID for deterministic output
	sort.Slice(allFindings, func(i, j int) bool {
		if allFindings[i].Line != allFindings[j].Line {
			return allFindings[i].Line < allFindings[j].Line
		}
		return allFindings[i].RuleID < allFindings[j].RuleID
	})

	return allFindings
}

// Registry returns the rule registry used by this analyzer.
func (a *Analyzer) Registry() *rules.RuleRegistry {
	return a.registry
}
