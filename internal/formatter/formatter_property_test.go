// Package formatter provides output formatters for lint findings.
package formatter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/docker-lint/docker-lint/internal/ast"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: docker-lint, Property 7: JSON Output Validity**
// **Validates: Requirements 7.2**
//
// Property: For any analysis run with --json flag, the output SHALL be valid JSON
// that can be parsed without error and conforms to the defined schema.
func TestJSONOutputValidity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 20

	properties := gopter.NewProperties(parameters)

	properties.Property("JSON output is always valid and parseable", prop.ForAll(
		func(findings []ast.Finding, filename string, quiet bool) bool {
			formatter := NewJSONFormatter(filename, quiet)
			var buf bytes.Buffer

			err := formatter.Format(findings, &buf)
			if err != nil {
				return false
			}

			// Verify output is valid JSON
			var output JSONOutput
			if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
				return false
			}

			// Verify schema conformance
			if output.File != filename {
				return false
			}

			// Verify summary counts match findings
			expectedTotal := 0
			expectedErrors := 0
			expectedWarnings := 0
			expectedInfo := 0

			for _, f := range findings {
				if quiet && f.Severity == ast.SeverityInfo {
					continue
				}
				expectedTotal++
				switch f.Severity {
				case ast.SeverityError:
					expectedErrors++
				case ast.SeverityWarning:
					expectedWarnings++
				case ast.SeverityInfo:
					expectedInfo++
				}
			}

			if output.Summary.Total != expectedTotal {
				return false
			}
			if output.Summary.Errors != expectedErrors {
				return false
			}
			if output.Summary.Warnings != expectedWarnings {
				return false
			}
			if output.Summary.Info != expectedInfo {
				return false
			}

			// Verify findings count matches
			if len(output.Findings) != expectedTotal {
				return false
			}

			return true
		},
		genFindingsSlice(),
		genFilename(),
		gen.Bool(),
	))

	properties.TestingRun(t)
}

// genFindingsSlice generates a slice of findings for property testing.
func genFindingsSlice() gopter.Gen {
	return gen.IntRange(0, 10).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		if count == 0 {
			return gen.Const([]ast.Finding{})
		}
		gens := make([]gopter.Gen, count)
		for i := 0; i < count; i++ {
			gens[i] = genFinding()
		}
		return gopter.CombineGens(gens...).Map(func(vals []interface{}) []ast.Finding {
			result := make([]ast.Finding, len(vals))
			for i, v := range vals {
				result[i] = v.(ast.Finding)
			}
			return result
		})
	}, nil)
}

// genFinding generates a random Finding for property testing.
func genFinding() gopter.Gen {
	return gopter.CombineGens(
		genRuleID(),
		genSeverity(),
		gen.IntRange(1, 100),
		gen.IntRange(1, 80),
		genMessage(),
		genSuggestion(),
	).Map(func(vals []interface{}) ast.Finding {
		return ast.Finding{
			RuleID:     vals[0].(string),
			Severity:   vals[1].(ast.Severity),
			Line:       vals[2].(int),
			Column:     vals[3].(int),
			Message:    vals[4].(string),
			Suggestion: vals[5].(string),
		}
	})
}

func genRuleID() gopter.Gen {
	return gen.OneConstOf(
		"DL3000", "DL3001", "DL3002", "DL3003", "DL3006", "DL3007",
		"DL3008", "DL3009", "DL3010", "DL3011", "DL3012",
		"DL4000", "DL4001", "DL4002", "DL4003", "DL4004",
		"DL5000", "DL5001",
	)
}

func genSeverity() gopter.Gen {
	return gen.OneConstOf(ast.SeverityInfo, ast.SeverityWarning, ast.SeverityError)
}

func genMessage() gopter.Gen {
	return gen.OneConstOf(
		"Missing explicit image tag",
		"Using 'latest' tag",
		"Consecutive RUN instructions",
		"Potential secret in ENV",
		"No USER instruction",
		"Missing HEALTHCHECK",
		"WORKDIR with relative path",
	)
}

func genSuggestion() gopter.Gen {
	return gen.OneConstOf(
		"",
		"Use explicit version tag",
		"Combine RUN instructions",
		"Add USER instruction",
		"Add HEALTHCHECK instruction",
		"Use absolute path",
	)
}

func genFilename() gopter.Gen {
	return gen.OneConstOf(
		"Dockerfile",
		"Dockerfile.dev",
		"Dockerfile.prod",
		"docker/Dockerfile",
		"build/Dockerfile",
	)
}
