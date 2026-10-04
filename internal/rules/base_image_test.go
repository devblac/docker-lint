package rules

import (
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

func TestBaseImageRulesRegistered(t *testing.T) {
	// Verify all base image rules are registered
	expectedRules := []string{
		RuleMissingTag,     // DL3006
		RuleLatestTag,      // DL3007
		RuleLargeBaseImage, // DL3008
	}

	for _, ruleID := range expectedRules {
		rule := DefaultRegistry.Get(ruleID)
		if rule == nil {
			t.Errorf("Rule %s not registered in DefaultRegistry", ruleID)
		}
	}
}

func TestMissingTagRule(t *testing.T) {
	rule := &MissingTagRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "image with tag - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: "3.18"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "image without tag - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: ""},
				},
			},
			expectedCount: 1,
		},
		{
			name: "scratch image without tag - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "scratch", Tag: ""},
				},
			},
			expectedCount: 0,
		},
		{
			name: "image with digest - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: "", Digest: "sha256:abc123"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "build argument image - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 2, Image: "${BASE}"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "reference to an earlier stage - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "golang", Tag: "1.22", Alias: "builder"},
					&ast.FromInstruction{LineNum: 2, Image: "builder", Alias: "test"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "untagged image before a stage of the same name is defined - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "builder"},
					&ast.FromInstruction{LineNum: 2, Image: "golang", Tag: "1.22", Alias: "builder"},
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
			if tt.expectedCount > 0 && findings[0].RuleID != RuleMissingTag {
				t.Errorf("expected rule ID %s, got %s", RuleMissingTag, findings[0].RuleID)
			}
		})
	}
}

func TestLatestTagRule(t *testing.T) {
	rule := &LatestTagRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "specific tag - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: "3.18"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "latest tag - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: "latest"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "LATEST tag (uppercase) - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: "LATEST"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "scratch with latest - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "scratch", Tag: "latest"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "image with digest - no warning even with latest",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: "latest", Digest: "sha256:abc123"},
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
			if tt.expectedCount > 0 && findings[0].RuleID != RuleLatestTag {
				t.Errorf("expected rule ID %s, got %s", RuleLatestTag, findings[0].RuleID)
			}
		})
	}
}

func TestLargeBaseImageRule(t *testing.T) {
	rule := &LargeBaseImageRule{}

	tests := []struct {
		name          string
		dockerfile    *ast.Dockerfile
		expectedCount int
	}{
		{
			name: "alpine - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: "3.18"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "ubuntu without slim - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "ubuntu", Tag: "22.04"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "ubuntu with slim - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "ubuntu", Tag: "22.04-slim"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "debian without slim - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "debian", Tag: "bullseye"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "python with alpine - no warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "python", Tag: "3.11-alpine"},
				},
			},
			expectedCount: 0,
		},
		{
			name: "node without slim - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "node", Tag: "18"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "registry prefix - warning",
			dockerfile: &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{LineNum: 1, Image: "docker.io/library/ubuntu", Tag: "22.04"},
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
			if tt.expectedCount > 0 && findings[0].RuleID != RuleLargeBaseImage {
				t.Errorf("expected rule ID %s, got %s", RuleLargeBaseImage, findings[0].RuleID)
			}
		})
	}
}

func TestExtractBaseImageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"alpine", "alpine"},
		{"ubuntu", "ubuntu"},
		{"docker.io/library/ubuntu", "ubuntu"},
		{"gcr.io/project/myimage", "myimage"},
		{"registry.example.com/org/app", "app"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractBaseImageName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsSlimVariant(t *testing.T) {
	tests := []struct {
		tag      string
		expected bool
	}{
		{"", false},
		{"latest", false},
		{"22.04", false},
		{"slim", true},
		{"22.04-slim", true},
		{"alpine", true},
		{"3.11-alpine", true},
		{"distroless", true},
		{"minimal", true},
		{"tiny", true},
		{"micro", true},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := isSlimVariant(tt.tag)
			if result != tt.expected {
				t.Errorf("isSlimVariant(%q) = %v, expected %v", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestLargeBaseImageFinalImageOnly(t *testing.T) {
	rule := &LargeBaseImageRule{}

	tests := []struct {
		name          string
		froms         []*ast.FromInstruction
		expectedCount int
		expectedLine  int
	}{
		{
			name: "large image only in a build stage",
			froms: []*ast.FromInstruction{
				{LineNum: 1, Image: "node", Tag: "lts", Alias: "build"},
				{LineNum: 5, Image: "nginx", Tag: "alpine"},
			},
			expectedCount: 0,
		},
		{
			name: "final stage built on a large build stage",
			froms: []*ast.FromInstruction{
				{LineNum: 1, Image: "node", Tag: "lts", Alias: "development"},
				{LineNum: 9, Image: "development", Alias: "dev-envs"},
			},
			expectedCount: 1,
			expectedLine:  1,
		},
		{
			name: "chain through several stages to a slim base",
			froms: []*ast.FromInstruction{
				{LineNum: 1, Image: "rust", Tag: "1.80", Alias: "builder"},
				{LineNum: 4, Image: "debian", Tag: "bookworm-slim", Alias: "base"},
				{LineNum: 6, Image: "base", Alias: "final"},
			},
			expectedCount: 0,
		},
		{
			name: "final stage from scratch",
			froms: []*ast.FromInstruction{
				{LineNum: 1, Image: "golang", Tag: "1.22", Alias: "build"},
				{LineNum: 6, Image: "scratch"},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dockerfile := &ast.Dockerfile{}
			for _, from := range tt.froms {
				dockerfile.Instructions = append(dockerfile.Instructions, from)
			}

			findings := rule.Check(dockerfile)
			if len(findings) != tt.expectedCount {
				t.Fatalf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
			if tt.expectedCount > 0 && findings[0].Line != tt.expectedLine {
				t.Errorf("finding line = %d, want %d", findings[0].Line, tt.expectedLine)
			}
		})
	}
}
