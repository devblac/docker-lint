package rules

import (
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

func TestLayerRulesRegistered(t *testing.T) {
	// Verify all layer optimization rules are registered
	expectedRules := []string{
		RuleCacheNotCleaned,      // DL3009
		RuleConsecutiveRun,       // DL3010
		RuleSuboptimalOrdering,   // DL3011
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

func TestSuboptimalOrderingRule(t *testing.T) {
	rule := &SuboptimalOrderingRule{}

	t.Run("warns when copy precedes package install with later copy", func(t *testing.T) {
		dockerfile := &ast.Dockerfile{
			Stages: []ast.Stage{
				{
					Index: 0,
					Instructions: []ast.Instruction{
						&ast.CopyInstruction{LineNum: 1, Sources: []string{"app/"}, Dest: "/app/"},
						&ast.RunInstruction{LineNum: 2, Command: "apt-get install -y curl"},
						&ast.CopyInstruction{LineNum: 3, Sources: []string{"assets/"}, Dest: "/assets/"},
					},
				},
			},
		}

		findings := rule.Check(dockerfile)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].RuleID != RuleSuboptimalOrdering {
			t.Fatalf("expected rule %s, got %s", RuleSuboptimalOrdering, findings[0].RuleID)
		}
	})

	t.Run("package manifest copy is ignored before install", func(t *testing.T) {
		dockerfile := &ast.Dockerfile{
			Stages: []ast.Stage{
				{
					Index: 0,
					Instructions: []ast.Instruction{
						&ast.CopyInstruction{LineNum: 1, Sources: []string{"requirements.txt"}, Dest: "/app/requirements.txt"},
						&ast.RunInstruction{LineNum: 2, Command: "pip install -r requirements.txt"},
						&ast.CopyInstruction{LineNum: 3, Sources: []string{"src/"}, Dest: "/app/src/"},
					},
				},
			},
		}

		findings := rule.Check(dockerfile)
		if len(findings) != 0 {
			t.Fatalf("expected 0 findings, got %d", len(findings))
		}
	})
}

func TestIsPackageInstallCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"apt-get install -y curl", true},
		{"yum install -y curl", true},
		{"apk add --no-cache bash", true},
		{"pip install flask", true},
		{"npm install", true},
		{"yarn install", true},
		{"go mod download", true},
		{"echo hello", false},
	}

	for _, tt := range tests {
		if got := isPackageInstallCommand(tt.cmd); got != tt.expected {
			t.Errorf("isPackageInstallCommand(%q) = %v, want %v", tt.cmd, got, tt.expected)
		}
	}
}

func TestIsPackageFile(t *testing.T) {
	tests := []struct {
		dest     string
		expected bool
	}{
		{"/app/requirements.txt", true},
		{"/app/package.json", true},
		{"/app/go.mod", true},
		{"/app/Gemfile", true},
		{"/app/src/main.go", false},
		{"/data/readme.md", false},
	}

	for _, tt := range tests {
		if got := isPackageFile(tt.dest); got != tt.expected {
			t.Errorf("isPackageFile(%q) = %v, want %v", tt.dest, got, tt.expected)
		}
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
