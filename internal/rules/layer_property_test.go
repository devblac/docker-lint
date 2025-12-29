// Package rules provides lint rule implementations for docker-lint.
package rules

import (
	"reflect"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: docker-lint, Property 10: Consecutive RUN Detection**
// **Validates: Requirements 3.1**
//
// Property: For any sequence of N consecutive RUN instructions (N >= 2),
// the findings SHALL contain exactly one warning about combining RUN instructions.
func TestConsecutiveRunDetection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	rule := &ConsecutiveRunRule{}

	// Property: N consecutive RUN instructions (N >= 2) produce exactly one warning
	properties.Property("N consecutive RUN instructions produce exactly one warning", prop.ForAll(
		func(runCount int, commands []string) bool {
			// Build a Dockerfile with N consecutive RUN instructions
			instructions := make([]ast.Instruction, runCount)
			for i := 0; i < runCount; i++ {
				instructions[i] = &ast.RunInstruction{
					LineNum: i + 1,
					Command: commands[i%len(commands)],
				}
			}

			dockerfile := &ast.Dockerfile{
				Instructions: instructions,
			}

			findings := rule.Check(dockerfile)

			// Should have exactly one finding for the consecutive sequence
			if len(findings) != 1 {
				return false
			}

			// Finding should be DL3010
			if findings[0].RuleID != RuleConsecutiveRun {
				return false
			}

			// Finding should be a warning
			if findings[0].Severity != ast.SeverityWarning {
				return false
			}

			// Finding should be on line 1 (first RUN of the sequence)
			if findings[0].Line != 1 {
				return false
			}

			return true
		},
		gen.IntRange(2, 10), // N >= 2 consecutive RUNs
		genRunCommands(),
	))

	// Property: Single RUN instruction produces no warning
	properties.Property("single RUN instruction produces no warning", prop.ForAll(
		func(command string) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{
						LineNum: 1,
						Command: command,
					},
				},
			}

			findings := rule.Check(dockerfile)

			// Should have no findings for a single RUN
			return len(findings) == 0
		},
		genRunCommand(),
	))

	// Property: RUN instructions separated by non-RUN produce no warning
	properties.Property("RUN instructions separated by non-RUN produce no warning", prop.ForAll(
		func(command1, command2 string, separatorType int) bool {
			// Create separator based on type
			var separator ast.Instruction
			switch separatorType {
			case 0:
				separator = &ast.WorkdirInstruction{LineNum: 2, Path: "/app"}
			case 1:
				separator = &ast.EnvInstruction{LineNum: 2, Key: "APP_ENV", Value: "production"}
			case 2:
				separator = &ast.CopyInstruction{LineNum: 2, Sources: []string{"."}, Dest: "/app"}
			case 3:
				separator = &ast.ExposeInstruction{LineNum: 2, Ports: []string{"8080"}}
			case 4:
				separator = &ast.UserInstruction{LineNum: 2, User: "appuser"}
			default:
				separator = &ast.LabelInstruction{LineNum: 2, Labels: map[string]string{"version": "1.0"}}
			}

			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{
						LineNum: 1,
						Command: command1,
					},
					separator,
					&ast.RunInstruction{
						LineNum: 3,
						Command: command2,
					},
				},
			}

			findings := rule.Check(dockerfile)

			// Should have no findings when RUNs are separated
			return len(findings) == 0
		},
		genRunCommand(),
		genRunCommand(),
		gen.IntRange(0, 5),
	))

	// Property: Multiple separate consecutive sequences produce one warning each
	properties.Property("multiple separate consecutive sequences produce one warning each", prop.ForAll(
		func(seq1Count, seq2Count int, commands []string) bool {
			// Build: seq1 RUNs, separator, seq2 RUNs
			instructions := make([]ast.Instruction, 0, seq1Count+1+seq2Count)

			// First sequence of consecutive RUNs
			for i := 0; i < seq1Count; i++ {
				instructions = append(instructions, &ast.RunInstruction{
					LineNum: i + 1,
					Command: commands[i%len(commands)],
				})
			}

			// Separator (non-RUN instruction)
			instructions = append(instructions, &ast.WorkdirInstruction{
				LineNum: seq1Count + 1,
				Path:    "/app",
			})

			// Second sequence of consecutive RUNs
			for i := 0; i < seq2Count; i++ {
				instructions = append(instructions, &ast.RunInstruction{
					LineNum: seq1Count + 2 + i,
					Command: commands[i%len(commands)],
				})
			}

			dockerfile := &ast.Dockerfile{
				Instructions: instructions,
			}

			findings := rule.Check(dockerfile)

			// Should have exactly 2 findings (one per sequence)
			if len(findings) != 2 {
				return false
			}

			// Both findings should be DL3010
			for _, f := range findings {
				if f.RuleID != RuleConsecutiveRun {
					return false
				}
				if f.Severity != ast.SeverityWarning {
					return false
				}
			}

			// First finding should be on line 1
			if findings[0].Line != 1 {
				return false
			}

			// Second finding should be on line seq1Count + 2 (first RUN of second sequence)
			if findings[1].Line != seq1Count+2 {
				return false
			}

			return true
		},
		gen.IntRange(2, 5), // First sequence size (>= 2)
		gen.IntRange(2, 5), // Second sequence size (>= 2)
		genRunCommands(),
	))

	// Property: Empty Dockerfile produces no warning
	properties.Property("empty Dockerfile produces no warning", prop.ForAll(
		func(_ bool) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{},
			}

			findings := rule.Check(dockerfile)

			return len(findings) == 0
		},
		gen.Bool(),
	))

	properties.TestingRun(t)
}

// Generator helpers

// genRunCommand generates a single RUN command string
func genRunCommand() gopter.Gen {
	return gen.OneConstOf(
		"echo hello",
		"apt-get update",
		"npm install",
		"pip install flask",
		"mkdir -p /app",
		"chmod +x /entrypoint.sh",
		"useradd -m appuser",
		"go build -o /app/main .",
	)
}

// genRunCommands generates a slice of RUN command strings
func genRunCommands() gopter.Gen {
	return gen.IntRange(1, 5).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		gens := make([]gopter.Gen, count)
		for i := 0; i < count; i++ {
			gens[i] = genRunCommand()
		}
		return gopter.CombineGens(gens...).Map(func(vals []interface{}) []string {
			result := make([]string, len(vals))
			for i, v := range vals {
				result[i] = v.(string)
			}
			return result
		})
	}, reflect.TypeOf([]string{}))
}
