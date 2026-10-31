package rules

import (
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

func TestLayerRulesRegistered(t *testing.T) {
	// Verify all layer optimization rules are registered
	expectedRules := []string{
		RuleCacheNotCleaned,      // DKL201
		RuleConsecutiveRun,       // DKL202
		RuleSuboptimalOrdering,   // DKL203
		RuleUpdateWithoutInstall, // DKL204
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

	// The recommended caching pattern: copy only the manifests into the
	// working directory, install, then copy the rest.
	manifestCopies := map[string][]string{
		"go modules":        {"go.mod", "go.sum"},
		"npm wildcard":      {"package*.json"},
		"dotnet projects":   {"*.csproj"},
		"python and poetry": {"pyproject.toml", "poetry.lock"},
	}
	for name, sources := range manifestCopies {
		t.Run("manifest copy into ./ is not flagged: "+name, func(t *testing.T) {
			dockerfile := &ast.Dockerfile{
				Stages: []ast.Stage{
					{
						Index: 0,
						Instructions: []ast.Instruction{
							&ast.CopyInstruction{LineNum: 1, Sources: sources, Dest: "./"},
							&ast.RunInstruction{LineNum: 2, Command: "go mod download"},
							&ast.CopyInstruction{LineNum: 3, Sources: []string{"."}, Dest: "."},
						},
					},
				},
			}

			if findings := rule.Check(dockerfile); len(findings) != 0 {
				t.Fatalf("expected 0 findings, got %d", len(findings))
			}
		})
	}

	t.Run("broad wildcard copy is still flagged", func(t *testing.T) {
		dockerfile := &ast.Dockerfile{
			Stages: []ast.Stage{
				{
					Index: 0,
					Instructions: []ast.Instruction{
						&ast.CopyInstruction{LineNum: 1, Sources: []string{"*"}, Dest: "./"},
						&ast.RunInstruction{LineNum: 2, Command: "npm install"},
						&ast.CopyInstruction{LineNum: 3, Sources: []string{"."}, Dest: "."},
					},
				},
			},
		}

		if findings := rule.Check(dockerfile); len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
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
		// Dependency installers beyond the original list.
		{"npm ci", true},
		{"npm i --omit=dev", true},
		{"pnpm install --frozen-lockfile", true},
		{"bun install", true},
		{"yarn", true},
		{"yarn --frozen-lockfile && yarn build", true},
		{"python -m pip install -r requirements.txt", true},
		{"uv sync --frozen", true},
		{"poetry install --no-root", true},
		{"pipenv install --deploy", true},
		{"bundle install", true},
		{"composer install --no-dev", true},
		{"cargo fetch", true},
		{"dotnet restore", true},
		{"mvn -B dependency:go-offline", true},
		{"apt install -y curl", true},
		{"apt-get -y --no-install-recommends install curl", true},
		{"microdnf install -y curl", true},
		// Commands that build or run the project rather than install into it.
		{"npm run build", false},
		{"npm init -y", false},
		{"yarn build", false},
		{"go build ./...", false},
		{"cargo build --release", false},
		{"dotnet publish -c Release", false},
		{"mvn package", false},
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
		{"go.sum", true},
		{"pnpm-lock.yaml", true},
		{"requirements-dev.txt", true},
		{"MyApp.csproj", true},
		{"package*.json", true},
		{"*.lock", true},
		{"*", false},
		{"*.go", false},
		{".", false},
		{"./", false},
		{"package.json.bak", false},
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

func TestCacheNotCleanedModernPatterns(t *testing.T) {
	rule := &CacheNotCleanedRule{}

	tests := []struct {
		name          string
		run           *ast.RunInstruction
		expectedCount int
	}{
		{
			name:          "apt-get dist-clean cleans the lists",
			run:           &ast.RunInstruction{LineNum: 1, Command: "apt-get update; apt-get install -y curl; apt-get dist-clean"},
			expectedCount: 0,
		},
		{
			name:          "rm with flags in another order cleans the lists",
			run:           &ast.RunInstruction{LineNum: 1, Command: "apt-get update && apt-get install -y curl && rm -fr /var/lib/apt/lists/*"},
			expectedCount: 0,
		},
		{
			name: "apt cache mount over /var/lib/apt keeps lists out of the layer",
			run: &ast.RunInstruction{
				LineNum:     1,
				Command:     "apt-get update && apt-get install -y gcc",
				CacheMounts: []string{"/var/cache/apt", "/var/lib/apt"},
			},
			expectedCount: 0,
		},
		{
			name: "apt cache mount over the archives only still leaves the lists",
			run: &ast.RunInstruction{
				LineNum:     1,
				Command:     "apt-get update && apt-get install -y gcc",
				CacheMounts: []string{"/var/cache/apt"},
			},
			expectedCount: 1,
		},
		{
			name: "pip cache mount",
			run: &ast.RunInstruction{
				LineNum:     1,
				Command:     "pip install -r requirements.txt",
				CacheMounts: []string{"/root/.cache/pip"},
			},
			expectedCount: 0,
		},
		{
			name: "pip cache mount for a non-root user's home",
			run: &ast.RunInstruction{
				LineNum:     1,
				Command:     "pip install -r requirements.txt",
				CacheMounts: []string{"/home/app/.cache/"},
			},
			expectedCount: 0,
		},
		{
			name: "unrelated cache mount does not excuse pip",
			run: &ast.RunInstruction{
				LineNum:     1,
				Command:     "pip install -r requirements.txt",
				CacheMounts: []string{"/go/pkg/mod"},
			},
			expectedCount: 1,
		},
		{
			name: "apk cache mount",
			run: &ast.RunInstruction{
				LineNum:     1,
				Command:     "apk add git",
				CacheMounts: []string{"/var/cache/apk"},
			},
			expectedCount: 0,
		},
		{
			name:          "heredoc apk update and add without --no-cache",
			run:           &ast.RunInstruction{LineNum: 1, Command: "<<EOF\napk update\napk add git\n"},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dockerfile := &ast.Dockerfile{Instructions: []ast.Instruction{tt.run}}
			if findings := rule.Check(dockerfile); len(findings) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(findings))
			}
		})
	}
}

func TestSuboptimalOrderingRunDependsOnCopy(t *testing.T) {
	rule := &SuboptimalOrderingRule{}

	// The install step runs the script the COPY brought in, so the COPY cannot
	// move after it. Taken from the official php image.
	dockerfile := &ast.Dockerfile{
		Stages: []ast.Stage{
			{
				Index: 0,
				Instructions: []ast.Instruction{
					&ast.CopyInstruction{LineNum: 1, Sources: []string{"docker-php-source"}, Dest: "/usr/local/bin/"},
					&ast.RunInstruction{LineNum: 2, Command: "set -eux; apk add --no-cache gcc; docker-php-source extract"},
					&ast.CopyInstruction{LineNum: 3, Sources: []string{"docker-php-ext-enable"}, Dest: "/usr/local/bin/"},
				},
			},
		},
	}

	if findings := rule.Check(dockerfile); len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

func TestRunReferencesCopy(t *testing.T) {
	tests := []struct {
		command string
		sources []string
		want    bool
	}{
		{"./install.sh && apt-get install -y curl", []string{"scripts/install.sh"}, true},
		{"apt-get install -y curl", []string{"scripts/install.sh"}, false},
		// Short or wildcard names would match almost any command.
		{"apt-get install -y curl", []string{"."}, false},
		{"cd src && npm install", []string{"src/"}, false},
		{"docker-php-ext-enable opcache", []string{"docker-php-ext-*"}, false},
	}

	for _, tt := range tests {
		if got := runReferencesCopy(tt.command, tt.sources); got != tt.want {
			t.Errorf("runReferencesCopy(%q, %v) = %v, want %v", tt.command, tt.sources, got, tt.want)
		}
	}
}

// TestSuboptimalOrderingWholeContext verifies the classic anti-pattern, the
// whole build context copied before the dependencies are installed, is caught
// even when nothing else is copied after the install.
func TestSuboptimalOrderingWholeContext(t *testing.T) {
	rule := &SuboptimalOrderingRule{}

	stage := func(instrs ...ast.Instruction) *ast.Dockerfile {
		return &ast.Dockerfile{Stages: []ast.Stage{{Index: 0, Instructions: instrs}}}
	}

	tests := []struct {
		name string
		df   *ast.Dockerfile
		want int
	}{
		{
			name: "node: COPY . . then npm ci",
			df: stage(
				&ast.CopyInstruction{LineNum: 1, Sources: []string{"."}, Dest: "."},
				&ast.RunInstruction{LineNum: 2, Command: "npm ci"},
				&ast.RunInstruction{LineNum: 3, Command: "npm run build"},
			),
			want: 1,
		},
		{
			name: "python: COPY ./ /app/ then pip install -r",
			df: stage(
				&ast.CopyInstruction{LineNum: 1, Sources: []string{"./"}, Dest: "/app/"},
				&ast.RunInstruction{LineNum: 2, Command: "pip install --no-cache-dir -r requirements.txt"},
			),
			want: 1,
		},
		{
			name: "go: ADD * ./ then go mod download",
			df: stage(
				&ast.AddInstruction{LineNum: 1, Sources: []string{"*"}, Dest: "./"},
				&ast.RunInstruction{LineNum: 2, Command: "go mod download"},
			),
			want: 1,
		},
		{
			name: "reported once across several installs",
			df: stage(
				&ast.CopyInstruction{LineNum: 1, Sources: []string{"."}, Dest: "."},
				&ast.RunInstruction{LineNum: 2, Command: "apt-get install -y make"},
				&ast.RunInstruction{LineNum: 3, Command: "npm ci"},
			),
			want: 1,
		},
		{
			name: "manifests first, then the whole context after the install",
			df: stage(
				&ast.CopyInstruction{LineNum: 1, Sources: []string{"package.json", "package-lock.json"}, Dest: "./"},
				&ast.RunInstruction{LineNum: 2, Command: "npm ci"},
				&ast.CopyInstruction{LineNum: 3, Sources: []string{"."}, Dest: "."},
				&ast.RunInstruction{LineNum: 4, Command: "npm run build"},
			),
			want: 0,
		},
		{
			name: "whole context before a build step only",
			df: stage(
				&ast.CopyInstruction{LineNum: 1, Sources: []string{"."}, Dest: "."},
				&ast.RunInstruction{LineNum: 2, Command: "go build -o /app ./cmd/app"},
			),
			want: 0,
		},
		{
			name: "a single specific file before an install is not reported",
			df: stage(
				&ast.CopyInstruction{LineNum: 1, Sources: []string{"nginx.conf"}, Dest: "/etc/nginx/"},
				&ast.RunInstruction{LineNum: 2, Command: "apk add --no-cache curl"},
			),
			want: 0,
		},
		{
			name: "COPY --from is not the build context",
			df: stage(
				&ast.CopyInstruction{LineNum: 1, From: "builder", Sources: []string{"."}, Dest: "."},
				&ast.RunInstruction{LineNum: 2, Command: "npm ci"},
			),
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rule.Check(tt.df); len(got) != tt.want {
				t.Errorf("got %d findings, want %d: %+v", len(got), tt.want, got)
			}
		})
	}
}

// TestCacheNotCleanedPipEnv verifies PIP_NO_CACHE_DIR in the environment
// counts as --no-cache-dir, within its own stage only.
func TestCacheNotCleanedPipEnv(t *testing.T) {
	rule := &CacheNotCleanedRule{}
	from := func(line int) *ast.FromInstruction {
		return &ast.FromInstruction{LineNum: line, Image: "python", Tag: "3.12-slim"}
	}
	pip := func(line int, cmd string) *ast.RunInstruction {
		return &ast.RunInstruction{LineNum: line, Command: cmd}
	}

	tests := []struct {
		name   string
		instrs []ast.Instruction
		want   int
	}{
		{"ENV disables the cache", []ast.Instruction{from(1), &ast.EnvInstruction{LineNum: 2, Key: "PIP_NO_CACHE_DIR", Value: "1"}, pip(3, "pip install flask")}, 0},
		{"any non-empty value counts, as in pip", []ast.Instruction{from(1), &ast.EnvInstruction{LineNum: 2, Key: "PIP_NO_CACHE_DIR", Value: "off"}, pip(3, "pip install flask")}, 0},
		{"empty value does not", []ast.Instruction{from(1), &ast.EnvInstruction{LineNum: 2, Key: "PIP_NO_CACHE_DIR", Value: ""}, pip(3, "pip install flask")}, 1},
		{"set for the command", []ast.Instruction{from(1), pip(2, "PIP_NO_CACHE_DIR=1 pip install flask")}, 0},
		{"ENV of an earlier stage does not carry over", []ast.Instruction{from(1), &ast.EnvInstruction{LineNum: 2, Key: "PIP_NO_CACHE_DIR", Value: "1"}, from(3), pip(4, "pip install flask")}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &ast.Dockerfile{Instructions: tt.instrs}
			if got := rule.Check(df); len(got) != tt.want {
				t.Errorf("got %d findings, want %d", len(got), tt.want)
			}
		})
	}
}
