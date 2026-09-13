// Package main provides the CLI entry point for docker-lint.
package main

import (
	"reflect"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: docker-lint, Property 6: Exit Code Consistency**
// **Validates: Requirements 6.4, 6.5, 6.6, 6.7, 6.8**
//
// Property: For any analysis result, the exit code SHALL be:
// - 0 when no errors exist (and not strict mode with warnings)
// - 1 when errors exist or strict mode with warnings
// - 2 for fatal errors (tested separately via unit tests since it requires I/O errors)
func TestExitCodeConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 20

	properties := gopter.NewProperties(parameters)

	properties.Property("exit code is 0 when no errors and not strict with warnings", prop.ForAll(
		func(findings []ast.Finding, strict bool) bool {
			hasErrors := false
			hasWarnings := false

			for _, f := range findings {
				switch f.Severity {
				case ast.SeverityError:
					hasErrors = true
				case ast.SeverityWarning:
					hasWarnings = true
				}
			}

			exitCode := determineExitCode(findings, strict)

			// Case 1: No errors and (no warnings OR not strict) -> exit 0
			if !hasErrors && (!hasWarnings || !strict) {
				return exitCode == ExitSuccess
			}

			// Case 2: Has errors -> exit 1
			if hasErrors {
				return exitCode == ExitLintErrors
			}

			// Case 3: Has warnings and strict mode -> exit 1
			if hasWarnings && strict {
				return exitCode == ExitLintErrors
			}

			// Default case: exit 0
			return exitCode == ExitSuccess
		},
		genFindings(),
		gen.Bool(),
	))

	properties.Property("exit code is 1 when errors exist regardless of strict mode", prop.ForAll(
		func(findings []ast.Finding, strict bool) bool {
			// Ensure at least one error exists
			hasError := false
			for _, f := range findings {
				if f.Severity == ast.SeverityError {
					hasError = true
					break
				}
			}
			if !hasError {
				return true // Skip if no errors (precondition not met)
			}

			exitCode := determineExitCode(findings, strict)
			return exitCode == ExitLintErrors
		},
		genFindingsWithErrors(),
		gen.Bool(),
	))

	properties.Property("exit code is 1 when warnings exist in strict mode", prop.ForAll(
		func(findings []ast.Finding) bool {
			// Ensure at least one warning exists and no errors
			hasWarning := false
			hasError := false
			for _, f := range findings {
				if f.Severity == ast.SeverityWarning {
					hasWarning = true
				}
				if f.Severity == ast.SeverityError {
					hasError = true
				}
			}
			if !hasWarning || hasError {
				return true // Skip if no warnings or has errors (precondition not met)
			}

			exitCode := determineExitCode(findings, true) // strict mode
			return exitCode == ExitLintErrors
		},
		genFindingsWithWarnings(),
	))

	properties.Property("exit code is 0 when warnings exist in non-strict mode", prop.ForAll(
		func(findings []ast.Finding) bool {
			// Ensure at least one warning exists and no errors
			hasWarning := false
			hasError := false
			for _, f := range findings {
				if f.Severity == ast.SeverityWarning {
					hasWarning = true
				}
				if f.Severity == ast.SeverityError {
					hasError = true
				}
			}
			if !hasWarning || hasError {
				return true // Skip if no warnings or has errors (precondition not met)
			}

			exitCode := determineExitCode(findings, false) // non-strict mode
			return exitCode == ExitSuccess
		},
		genFindingsWithWarnings(),
	))

	properties.Property("exit code is 0 when no findings exist", prop.ForAll(
		func(strict bool) bool {
			exitCode := determineExitCode([]ast.Finding{}, strict)
			return exitCode == ExitSuccess
		},
		gen.Bool(),
	))

	properties.TestingRun(t)
}

// genFindings generates a random slice of findings with various severities.
func genFindings() gopter.Gen {
	return gen.IntRange(0, 10).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		if count == 0 {
			return gen.Const([]ast.Finding{})
		}
		return gen.SliceOfN(count, genFinding())
	}, reflect.TypeOf([]ast.Finding{}))
}

// genFinding generates a random finding.
func genFinding() gopter.Gen {
	return gopter.CombineGens(
		genRuleID(),
		genSeverity(),
		gen.IntRange(1, 100),
		gen.IntRange(1, 80),
		genMessage(),
	).Map(func(vals []interface{}) ast.Finding {
		return ast.Finding{
			RuleID:     vals[0].(string),
			Severity:   vals[1].(ast.Severity),
			Line:       vals[2].(int),
			Column:     vals[3].(int),
			Message:    vals[4].(string),
			Suggestion: "",
		}
	})
}

// genFindingsWithErrors generates findings that include at least one error.
func genFindingsWithErrors() gopter.Gen {
	return gen.IntRange(1, 10).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		return gen.SliceOfN(count, genFinding()).Map(func(findings []ast.Finding) []ast.Finding {
			// Ensure at least one error exists
			hasError := false
			for _, f := range findings {
				if f.Severity == ast.SeverityError {
					hasError = true
					break
				}
			}
			if !hasError && len(findings) > 0 {
				findings[0].Severity = ast.SeverityError
			}
			return findings
		})
	}, reflect.TypeOf([]ast.Finding{}))
}

// genFindingsWithWarnings generates findings that include at least one warning but no errors.
func genFindingsWithWarnings() gopter.Gen {
	return gen.IntRange(1, 10).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		return gen.SliceOfN(count, genFindingNoError()).Map(func(findings []ast.Finding) []ast.Finding {
			// Ensure at least one warning exists
			hasWarning := false
			for _, f := range findings {
				if f.Severity == ast.SeverityWarning {
					hasWarning = true
					break
				}
			}
			if !hasWarning && len(findings) > 0 {
				findings[0].Severity = ast.SeverityWarning
			}
			return findings
		})
	}, reflect.TypeOf([]ast.Finding{}))
}

// genFindingNoError generates a finding that is not an error (info or warning only).
func genFindingNoError() gopter.Gen {
	return gopter.CombineGens(
		genRuleID(),
		genSeverityNoError(),
		gen.IntRange(1, 100),
		gen.IntRange(1, 80),
		genMessage(),
	).Map(func(vals []interface{}) ast.Finding {
		return ast.Finding{
			RuleID:     vals[0].(string),
			Severity:   vals[1].(ast.Severity),
			Line:       vals[2].(int),
			Column:     vals[3].(int),
			Message:    vals[4].(string),
			Suggestion: "",
		}
	})
}

// genRuleID generates a random rule ID.
func genRuleID() gopter.Gen {
	return gen.OneConstOf(
		"DL3006", "DL3007", "DL3008", "DL3009", "DL3010",
		"DL4000", "DL4001", "DL4002", "DL4003", "DL4004",
		"DL5000", "DL5001", "DL3001", "DL3002", "DL3003",
	)
}

// genSeverity generates a random severity level.
func genSeverity() gopter.Gen {
	return gen.IntRange(0, 2).Map(func(i int) ast.Severity {
		return ast.Severity(i)
	})
}

// genSeverityNoError generates a severity that is not an error (info or warning).
func genSeverityNoError() gopter.Gen {
	return gen.IntRange(0, 1).Map(func(i int) ast.Severity {
		return ast.Severity(i)
	})
}

// genMessage generates a random lint message.
func genMessage() gopter.Gen {
	return gen.OneConstOf(
		"Missing explicit image tag",
		"Using 'latest' tag",
		"Potential secret in ENV",
		"No USER instruction",
		"Consecutive RUN instructions",
		"Package manager cache not cleaned",
		"WORKDIR with relative path",
	)
}
