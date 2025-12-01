package rules

import (
	"testing"

	"github.com/docker-lint/docker-lint/internal/ast"
)

func TestBestPracticeRulesRegistered(t *testing.T) {
	// Verify all best practice rules are registered
	expectedRules := []string{
		RuleMultipleCMD,        // DL3001
		RuleMultipleEntrypoint, // DL3002
		RuleRelativeWorkdir,    // DL3003
		RuleMissingHealthcheck, // DL5000
		RuleWildcardCopy,       // DL5001
	}

	for _, ruleID := range expectedRules {
		rule := DefaultRegistry.Get(ruleID)
		if rule == nil {
			t.Errorf("Rule %s not registered in DefaultRegistry", ruleID)
		}
	}
}

func TestMultipleCMDRule(t *testing.T) {
	rule := &MultipleCMDRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "single CMD - no warning",
			dockerfile: &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						Instructions: []ast.Instruction{
							&ast.CmdInstruction{LineNum: 2, Command: []string{"echo", "hello"}},
						},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "multiple CMD - warning for all but last",
			dockerfile: &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						Instructions: []ast.Instruction{
							&ast.CmdInstruction{LineNum: 2, Command: []string{"echo", "first"}},
							&ast.CmdInstruction{LineNum: 3, Command: []string{"echo", "second"}},
							&ast.CmdInstruction{LineNum: 4, Command: []string{"echo", "third"}},
						},
					},
				},
			},
			expectedCount: 2, // warnings for first two CMDs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := rule.Check(tt.dockerfile)
			if len(findings) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
		})
	}
}


func TestMultipleEntrypointRule(t *testing.T) {
	rule := &MultipleEntrypointRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "single ENTRYPOINT - no warning",
			dockerfile: &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						Instructions: []ast.Instruction{
							&ast.EntrypointInstruction{LineNum: 2, Command: []string{"/app/start.sh"}},
						},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "multiple ENTRYPOINT - warning for all but last",
			dockerfile: &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						Instructions: []ast.Instruction{
							&ast.EntrypointInstruction{LineNum: 2, Command: []string{"/first"}},
							&ast.EntrypointInstruction{LineNum: 3, Command: []string{"/second"}},
						},
					},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := rule.Check(tt.dockerfile)
			if len(findings) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
		})
	}
}

func TestRelativeWorkdirRule(t *testing.T) {
	rule := &RelativeWorkdirRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "absolute path - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.WorkdirInstruction{LineNum: 2, Path: "/app"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "relative path - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.WorkdirInstruction{LineNum: 2, Path: "app"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "variable path - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.WorkdirInstruction{LineNum: 2, Path: "$APP_DIR"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "Windows absolute path - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.WorkdirInstruction{LineNum: 2, Path: "C:\\app"},
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
		})
	}
}

func TestMissingHealthcheckRule(t *testing.T) {
	rule := &MissingHealthcheckRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "has HEALTHCHECK - no warning",
			dockerfile: &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						FromInstr: &ast.FromInstruction{LineNum: 1, Image: "alpine"},
						Instructions: []ast.Instruction{
							&ast.HealthcheckInstruction{LineNum: 2, Command: []string{"CMD", "curl", "-f", "http://localhost/"}},
						},
					},
				},
				Instructions: []ast.Instruction{
					&ast.HealthcheckInstruction{LineNum: 2, Command: []string{"CMD", "curl", "-f", "http://localhost/"}},
				},
			},
			expectedCount: 0,
		},
		{
			name: "missing HEALTHCHECK - warning",
			dockerfile: &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						FromInstr: &ast.FromInstruction{LineNum: 1, Image: "alpine"},
						Instructions: []ast.Instruction{
							&ast.RunInstruction{LineNum: 2, Command: "echo hello"},
						},
					},
				},
				Instructions: []ast.Instruction{
					&ast.RunInstruction{LineNum: 2, Command: "echo hello"},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := rule.Check(tt.dockerfile)
			if len(findings) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
		})
	}
}

func TestWildcardCopyRule(t *testing.T) {
	rule := &WildcardCopyRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "explicit file - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.CopyInstruction{LineNum: 2, Sources: []string{"app.go"}, Dest: "/app/"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "wildcard in COPY - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.CopyInstruction{LineNum: 2, Sources: []string{"*.go"}, Dest: "/app/"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "wildcard in ADD - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.AddInstruction{LineNum: 2, Sources: []string{"src/*"}, Dest: "/app/"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "COPY --from (multi-stage) with wildcard - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.CopyInstruction{LineNum: 2, Sources: []string{"*.go"}, Dest: "/app/", From: "builder"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "question mark wildcard - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.CopyInstruction{LineNum: 2, Sources: []string{"file?.txt"}, Dest: "/app/"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "bracket wildcard - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.CopyInstruction{LineNum: 2, Sources: []string{"file[0-9].txt"}, Dest: "/app/"},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := rule.Check(tt.dockerfile)
			if len(findings) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
		})
	}
}
