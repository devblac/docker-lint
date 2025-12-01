package rules

import (
	"testing"

	"github.com/docker-lint/docker-lint/internal/ast"
)

func TestSecurityRulesRegistered(t *testing.T) {
	// Verify all security rules are registered
	expectedRules := []string{
		RuleSecretInEnv, // DL4000
		RuleSecretInArg, // DL4001
		RuleNoUser,      // DL4002
		RuleAddWithURL,  // DL4003
		RuleAddOverCopy, // DL4004
	}

	for _, ruleID := range expectedRules {
		rule := DefaultRegistry.Get(ruleID)
		if rule == nil {
			t.Errorf("Rule %s not registered in DefaultRegistry", ruleID)
		}
	}
}

func TestSecretInEnvRule(t *testing.T) {
	rule := &SecretInEnvRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "non-secret ENV - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.EnvInstruction{LineNum: 1, Key: "APP_NAME", Value: "myapp"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "PASSWORD in ENV - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.EnvInstruction{LineNum: 1, Key: "DB_PASSWORD", Value: "secret123"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "SECRET in ENV - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.EnvInstruction{LineNum: 1, Key: "APP_SECRET", Value: "mysecret"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "TOKEN in ENV - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.EnvInstruction{LineNum: 1, Key: "AUTH_TOKEN", Value: "token123"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "API_KEY in ENV - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.EnvInstruction{LineNum: 1, Key: "API_KEY", Value: "key123"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "PRIVATE_KEY in ENV - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.EnvInstruction{LineNum: 1, Key: "PRIVATE_KEY", Value: "-----BEGIN RSA PRIVATE KEY-----"},
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
			if tt.expectedCount > 0 && findings[0].RuleID != RuleSecretInEnv {
				t.Errorf("expected rule ID %s, got %s", RuleSecretInEnv, findings[0].RuleID)
			}
		})
	}
}

func TestSecretInArgRule(t *testing.T) {
	rule := &SecretInArgRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "non-secret ARG - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.ArgInstruction{LineNum: 1, Name: "APP_VERSION", Default: "1.0.0"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "PASSWORD in ARG - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.ArgInstruction{LineNum: 1, Name: "DB_PASSWORD", Default: "secret123"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "SECRET in ARG - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.ArgInstruction{LineNum: 1, Name: "APP_SECRET", Default: ""},
				},
			},
			expectedCount: 1,
		},
		{
			name: "CREDENTIALS in ARG - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.ArgInstruction{LineNum: 1, Name: "AWS_CREDENTIALS", Default: ""},
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
			if tt.expectedCount > 0 && findings[0].RuleID != RuleSecretInArg {
				t.Errorf("expected rule ID %s, got %s", RuleSecretInArg, findings[0].RuleID)
			}
		})
	}
}

func TestNoUserRule(t *testing.T) {
	rule := &NoUserRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "has USER instruction - no warning",
			dockerfile: &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						Index:     0,
						FromInstr: &ast.FromInstruction{LineNum: 1, Image: "alpine"},
						Instructions: []ast.Instruction{
							&ast.UserInstruction{LineNum: 2, User: "appuser"},
						},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "no USER instruction - warning",
			dockerfile: &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						Index:     0,
						FromInstr: &ast.FromInstruction{LineNum: 1, Image: "alpine"},
						Instructions: []ast.Instruction{
							&ast.RunInstruction{LineNum: 2, Command: "echo hello"},
						},
					},
				},
			},
			expectedCount: 1,
		},
		{
			name: "multi-stage with USER in final stage - one warning for first stage",
			dockerfile: &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						Index:     0,
						Name:      "builder",
						FromInstr: &ast.FromInstruction{LineNum: 1, Image: "golang"},
						Instructions: []ast.Instruction{
							&ast.RunInstruction{LineNum: 2, Command: "go build"},
						},
					},
					{
						Index:     1,
						FromInstr: &ast.FromInstruction{LineNum: 3, Image: "alpine"},
						Instructions: []ast.Instruction{
							&ast.UserInstruction{LineNum: 4, User: "appuser"},
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
			if tt.expectedCount > 0 && findings[0].RuleID != RuleNoUser {
				t.Errorf("expected rule ID %s, got %s", RuleNoUser, findings[0].RuleID)
			}
		})
	}
}

func TestAddWithURLRule(t *testing.T) {
	rule := &AddWithURLRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "ADD with local file - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.AddInstruction{LineNum: 1, Sources: []string{"app.tar.gz"}, Dest: "/app/"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "ADD with HTTP URL - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.AddInstruction{LineNum: 1, Sources: []string{"http://example.com/file.tar.gz"}, Dest: "/app/"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "ADD with HTTPS URL - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.AddInstruction{LineNum: 1, Sources: []string{"https://example.com/file.tar.gz"}, Dest: "/app/"},
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
			if tt.expectedCount > 0 && findings[0].RuleID != RuleAddWithURL {
				t.Errorf("expected rule ID %s, got %s", RuleAddWithURL, findings[0].RuleID)
			}
		})
	}
}

func TestAddOverCopyRule(t *testing.T) {
	rule := &AddOverCopyRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "ADD with archive - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.AddInstruction{LineNum: 1, Sources: []string{"app.tar.gz"}, Dest: "/app/"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "ADD with URL - no warning (handled by DL4003)",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.AddInstruction{LineNum: 1, Sources: []string{"http://example.com/file"}, Dest: "/app/"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "ADD with regular file - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.AddInstruction{LineNum: 1, Sources: []string{"app.go"}, Dest: "/app/"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "ADD with directory - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.AddInstruction{LineNum: 1, Sources: []string{"src/"}, Dest: "/app/"},
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
			if tt.expectedCount > 0 && findings[0].RuleID != RuleAddOverCopy {
				t.Errorf("expected rule ID %s, got %s", RuleAddOverCopy, findings[0].RuleID)
			}
		})
	}
}

func TestIsSecretKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"APP_NAME", false},
		{"PORT", false},
		{"PASSWORD", true},
		{"DB_PASSWORD", true},
		{"SECRET", true},
		{"APP_SECRET", true},
		{"TOKEN", true},
		{"AUTH_TOKEN", true},
		{"API_KEY", true},
		{"APIKEY", true},
		{"PRIVATE_KEY", true},
		{"PRIVATEKEY", true},
		{"ACCESS_KEY", true},
		{"ACCESSKEY", true},
		{"CREDENTIALS", true},
		{"SSH_KEY", true},
		{"ENCRYPTION_KEY", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSecretKey(tt.key)
			if result != tt.expected {
				t.Errorf("isSecretKey(%q) = %v, expected %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestIsArchiveFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"app.go", false},
		{"README.md", false},
		{"app.tar", true},
		{"app.tar.gz", true},
		{"app.tgz", true},
		{"app.tar.bz2", true},
		{"app.tbz2", true},
		{"app.tar.xz", true},
		{"app.txz", true},
		{"app.zip", true},
		{"app.gz", true},
		{"app.bz2", true},
		{"app.xz", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isArchiveFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isArchiveFile(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}
