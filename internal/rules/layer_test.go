package rules

import (
	"testing"

	"github.com/docker-lint/docker-lint/internal/ast"
)

func TestLayerRulesRegistered(t *testing.T) {
	// Verify all layer optimization rules are registered
	expectedRules := []string{
		RuleCacheNotCleaned,     // DL3009
		RuleConsecutiveRun,      // DL3010
		RuleSuboptimalOrdering,  // DL3011
		RuleUpdateWithoutInstall, // DL3012
	}

	for _, ruleID := range expectedRules {
		rule := DefaultRegistry.Get(ruleID)
		if rule == nil {
			t.Errorf("Rule %s not registered in DefaultRegistry", ruleID)
		}
	}
}

func TestCacheNotCleanedRule(t *testing.T) {
	rule := &CacheNotCleanedRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "apt-get install with cleanup - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "apt-get update && apt-get install -y curl && apt-get clean && rm -rf /var/lib/apt/lists/*"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "apt-get install without cleanup - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "apt-get update && apt-get install -y curl"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "yum install with cleanup - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "yum install -y curl && yum clean all"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "yum install without cleanup - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "yum install -y curl"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "apk add with --no-cache - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "apk add --no-cache curl"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "apk add without --no-cache - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "apk add curl"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "pip install with --no-cache-dir - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "pip install --no-cache-dir flask"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "pip install without --no-cache-dir - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "pip install flask"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "non-package-manager command - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "echo hello"},
				},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := rule.Check(tt.dockerfile)
			if len(findings) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
			if tt.expectedCount > 0 && findings[0].RuleID != RuleCacheNotCleaned {
				t.Errorf("expected rule ID %s, got %s", RuleCacheNotCleaned, findings[0].RuleID)
			}
		})
	}
}

func TestConsecutiveRunRule(t *testing.T) {
	rule := &ConsecutiveRunRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "single RUN - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "echo hello"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "two consecutive RUNs - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "echo hello"},
					&ast.RunInstruction{LineNum: 2, Command: "echo world"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "RUNs separated by other instruction - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "echo hello"},
					&ast.WorkdirInstruction{LineNum: 2, Path: "/app"},
					&ast.RunInstruction{LineNum: 3, Command: "echo world"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "three consecutive RUNs - one warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "echo one"},
					&ast.RunInstruction{LineNum: 2, Command: "echo two"},
					&ast.RunInstruction{LineNum: 3, Command: "echo three"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "empty dockerfile - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := rule.Check(tt.dockerfile)
			if len(findings) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
			if tt.expectedCount > 0 && findings[0].RuleID != RuleConsecutiveRun {
				t.Errorf("expected rule ID %s, got %s", RuleConsecutiveRun, findings[0].RuleID)
			}
		})
	}
}

func TestUpdateWithoutInstallRule(t *testing.T) {
	rule := &UpdateWithoutInstallRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "apt-get update with install - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "apt-get update && apt-get install -y curl"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "apt-get update without install - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "apt-get update"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "yum update with makecache without install - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "yum update && yum makecache"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "yum update with install - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "yum update && yum install -y curl"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "non-update command - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 1, Command: "echo hello"},
				},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := rule.Check(tt.dockerfile)
			if len(findings) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
			if tt.expectedCount > 0 && findings[0].RuleID != RuleUpdateWithoutInstall {
				t.Errorf("expected rule ID %s, got %s", RuleUpdateWithoutInstall, findings[0].RuleID)
			}
		})
	}
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-5, "-5"},
	}

	for _, tt := range tests {
		result := intToString(tt.input)
		if result != tt.expected {
			t.Errorf("intToString(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}
