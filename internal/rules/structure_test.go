package rules

import (
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

func TestNoFromRuleRegistered(t *testing.T) {
	if DefaultRegistry.Get(RuleNoFrom) == nil {
		t.Fatalf("rule %s not registered", RuleNoFrom)
	}
}

func TestNoFromRule(t *testing.T) {
	rule := &NoFromRule{}

	tests := []struct {
		name          string
		instructions  []ast.Instruction
		expectedCount int
		expectedLine  int
	}{
		{
			name: "starts with FROM",
			instructions: []ast.Instruction{
				&ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: "3.20"},
				&ast.RunInstruction{LineNum: 2, Command: "echo hi"},
			},
			expectedCount: 0,
		},
		{
			name: "ARG before FROM is allowed",
			instructions: []ast.Instruction{
				&ast.ArgInstruction{LineNum: 1, Name: "BASE", Default: "alpine:3.20"},
				&ast.FromInstruction{LineNum: 2, Image: "${BASE}"},
			},
			expectedCount: 0,
		},
		{
			name: "RUN before FROM",
			instructions: []ast.Instruction{
				&ast.ArgInstruction{LineNum: 1, Name: "VERSION"},
				&ast.RunInstruction{LineNum: 2, Command: "echo hi"},
				&ast.EnvInstruction{LineNum: 3, Key: "A", Value: "1"},
				&ast.FromInstruction{LineNum: 4, Image: "alpine", Tag: "3.20"},
			},
			expectedCount: 1,
			expectedLine:  2,
		},
		{
			name: "no FROM at all",
			instructions: []ast.Instruction{
				&ast.RunInstruction{LineNum: 3, Command: "echo hi"},
			},
			expectedCount: 1,
			expectedLine:  3,
		},
		{
			name: "only ARG instructions",
			instructions: []ast.Instruction{
				&ast.ArgInstruction{LineNum: 2, Name: "VERSION"},
			},
			expectedCount: 1,
			expectedLine:  2,
		},
		{
			name:          "empty Dockerfile",
			instructions:  nil,
			expectedCount: 1,
			expectedLine:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := rule.Check(&ast.Dockerfile{Instructions: tt.instructions})
			if len(findings) != tt.expectedCount {
				t.Fatalf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
			if tt.expectedCount == 0 {
				return
			}
			if findings[0].Line != tt.expectedLine {
				t.Errorf("finding line = %d, want %d", findings[0].Line, tt.expectedLine)
			}
			if findings[0].Severity != ast.SeverityError {
				t.Errorf("severity = %v, want error", findings[0].Severity)
			}
		})
	}
}
