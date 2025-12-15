// Package rules provides lint rule implementations for docker-lint.
package rules

import (
	"regexp"
	"strings"

	"github.com/devblac/docker-lint/internal/ast"
)

// Package manager patterns for cache cleanup detection
var (
	// aptGetPattern matches apt-get install commands
	aptGetInstallPattern = regexp.MustCompile(`apt-get\s+(install|upgrade)`)
	// aptGetCleanPattern matches apt-get clean or rm -rf /var/lib/apt/lists
	aptGetCleanPattern = regexp.MustCompile(`(apt-get\s+clean|rm\s+-rf?\s+/var/lib/apt/lists)`)

	// yumInstallPattern matches yum/dnf install commands
	yumInstallPattern = regexp.MustCompile(`(yum|dnf)\s+install`)
	// yumCleanPattern matches yum/dnf clean all
	yumCleanPattern = regexp.MustCompile(`(yum|dnf)\s+clean\s+all`)

	// apkAddPattern matches apk add commands
	apkAddPattern = regexp.MustCompile(`apk\s+(add|update)`)
	// apkNoCachePattern matches apk add --no-cache or rm -rf /var/cache/apk
	apkNoCachePattern = regexp.MustCompile(`(apk\s+add\s+[^\n]*--no-cache|rm\s+-rf?\s+/var/cache/apk)`)

	// pipInstallPattern matches pip install commands
	pipInstallPattern = regexp.MustCompile(`pip[3]?\s+install`)
	// pipNoCachePattern matches pip install --no-cache-dir
	pipNoCachePattern = regexp.MustCompile(`pip[3]?\s+install\s+[^\n]*--no-cache-dir`)

	// Package update patterns (without install in same command)
	aptGetUpdatePattern = regexp.MustCompile(`apt-get\s+update`)
	yumUpdatePattern    = regexp.MustCompile(`(yum|dnf)\s+(update|upgrade)`)
)

// CacheNotCleanedRule checks for package manager installs without cache cleanup (DL3009).
type CacheNotCleanedRule struct{}

func (r *CacheNotCleanedRule) ID() string             { return RuleCacheNotCleaned }
func (r *CacheNotCleanedRule) Name() string           { return "Package manager cache not cleaned" }
func (r *CacheNotCleanedRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *CacheNotCleanedRule) Description() string {
	return "Clean package manager cache in the same RUN instruction to reduce image size"
}


func (r *CacheNotCleanedRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		run, ok := instr.(*ast.RunInstruction)
		if !ok {
			continue
		}

		cmd := run.Command

		// Check apt-get install without cleanup
		if aptGetInstallPattern.MatchString(cmd) && !aptGetCleanPattern.MatchString(cmd) {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       run.Line(),
				Column:     1,
				Message:    "apt-get install without cache cleanup increases image size",
				Suggestion: "Add 'apt-get clean && rm -rf /var/lib/apt/lists/*' in the same RUN instruction",
			})
			continue
		}

		// Check yum/dnf install without cleanup
		if yumInstallPattern.MatchString(cmd) && !yumCleanPattern.MatchString(cmd) {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       run.Line(),
				Column:     1,
				Message:    "yum/dnf install without cache cleanup increases image size",
				Suggestion: "Add 'yum clean all' or 'dnf clean all' in the same RUN instruction",
			})
			continue
		}

		// Check apk add without --no-cache
		if apkAddPattern.MatchString(cmd) && !apkNoCachePattern.MatchString(cmd) {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       run.Line(),
				Column:     1,
				Message:    "apk add without --no-cache increases image size",
				Suggestion: "Use 'apk add --no-cache' or add 'rm -rf /var/cache/apk/*'",
			})
			continue
		}

		// Check pip install without --no-cache-dir
		if pipInstallPattern.MatchString(cmd) && !pipNoCachePattern.MatchString(cmd) {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       run.Line(),
				Column:     1,
				Message:    "pip install without --no-cache-dir increases image size",
				Suggestion: "Use 'pip install --no-cache-dir' to avoid caching packages",
			})
		}
	}

	return findings
}

// ConsecutiveRunRule checks for consecutive RUN instructions that could be combined (DL3010).
type ConsecutiveRunRule struct{}

func (r *ConsecutiveRunRule) ID() string             { return RuleConsecutiveRun }
func (r *ConsecutiveRunRule) Name() string           { return "Consecutive RUN instructions" }
func (r *ConsecutiveRunRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *ConsecutiveRunRule) Description() string {
	return "Combine consecutive RUN instructions to reduce the number of layers"
}

func (r *ConsecutiveRunRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	// Track consecutive RUN instructions
	var consecutiveRuns []*ast.RunInstruction

	for _, instr := range dockerfile.Instructions {
		run, ok := instr.(*ast.RunInstruction)
		if ok {
			consecutiveRuns = append(consecutiveRuns, run)
		} else {
			// Non-RUN instruction breaks the sequence
			if len(consecutiveRuns) >= 2 {
				// Report finding at the first RUN of the consecutive sequence
				findings = append(findings, ast.Finding{
					RuleID:     r.ID(),
					Severity:   r.Severity(),
					Line:       consecutiveRuns[0].Line(),
					Column:     1,
					Message:    formatConsecutiveRunMessage(len(consecutiveRuns)),
					Suggestion: "Combine RUN instructions using '&&' to reduce layers",
				})
			}
			consecutiveRuns = nil
		}
	}

	// Check for trailing consecutive RUNs
	if len(consecutiveRuns) >= 2 {
		findings = append(findings, ast.Finding{
			RuleID:     r.ID(),
			Severity:   r.Severity(),
			Line:       consecutiveRuns[0].Line(),
			Column:     1,
			Message:    formatConsecutiveRunMessage(len(consecutiveRuns)),
			Suggestion: "Combine RUN instructions using '&&' to reduce layers",
		})
	}

	return findings
}

func formatConsecutiveRunMessage(count int) string {
	return "Found " + intToString(count) + " consecutive RUN instructions that could be combined"
}

// intToString converts an int to string without importing strconv
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}


// SuboptimalOrderingRule checks for COPY/ADD before RUN that doesn't depend on copied files (DL3011).
type SuboptimalOrderingRule struct{}

func (r *SuboptimalOrderingRule) ID() string             { return RuleSuboptimalOrdering }
func (r *SuboptimalOrderingRule) Name() string           { return "Suboptimal layer ordering" }
func (r *SuboptimalOrderingRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *SuboptimalOrderingRule) Description() string {
	return "Place instructions that change less frequently earlier to optimize layer caching"
}

func (r *SuboptimalOrderingRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	// Process each stage separately
	for _, stage := range dockerfile.Stages {
		findings = append(findings, r.checkStage(stage)...)
	}

	return findings
}

func (r *SuboptimalOrderingRule) checkStage(stage ast.Stage) []ast.Finding {
	var findings []ast.Finding

	// Track COPY/ADD instructions and their destinations
	type copyInfo struct {
		instr ast.Instruction
		dest  string
	}
	var copies []copyInfo

	// Track if we've seen a RUN that installs packages (likely doesn't depend on copied files)
	for i, instr := range stage.Instructions {
		switch v := instr.(type) {
		case *ast.CopyInstruction:
			// Skip COPY --from (multi-stage copies)
			if v.From != "" {
				continue
			}
			copies = append(copies, copyInfo{instr: v, dest: v.Dest})

		case *ast.AddInstruction:
			copies = append(copies, copyInfo{instr: v, dest: v.Dest})

		case *ast.RunInstruction:
			// Check if this RUN is a package install that likely doesn't depend on copied files
			if isPackageInstallCommand(v.Command) && len(copies) > 0 {
				// Look ahead to see if there are more COPY/ADD after this RUN
				hasMoreCopies := false
				for j := i + 1; j < len(stage.Instructions); j++ {
					switch stage.Instructions[j].(type) {
					case *ast.CopyInstruction, *ast.AddInstruction:
						hasMoreCopies = true
						break
					}
				}

				// If there are copies before package install and more copies after,
				// the early copies might be suboptimal
				if hasMoreCopies {
					for _, cp := range copies {
						// Skip if copying package files (requirements.txt, package.json, etc.)
						if isPackageFile(cp.dest) {
							continue
						}
						findings = append(findings, ast.Finding{
							RuleID:     r.ID(),
							Severity:   r.Severity(),
							Line:       cp.instr.Line(),
							Column:     1,
							Message:    "COPY/ADD before package installation may reduce cache efficiency",
							Suggestion: "Move COPY/ADD after RUN instructions that don't depend on copied files",
						})
					}
				}
			}
		}
	}

	return findings
}

// isPackageInstallCommand checks if a command is a package manager install
func isPackageInstallCommand(cmd string) bool {
	return aptGetInstallPattern.MatchString(cmd) ||
		yumInstallPattern.MatchString(cmd) ||
		apkAddPattern.MatchString(cmd) ||
		pipInstallPattern.MatchString(cmd) ||
		strings.Contains(cmd, "npm install") ||
		strings.Contains(cmd, "yarn install") ||
		strings.Contains(cmd, "go mod download")
}

// isPackageFile checks if the destination is a package dependency file
func isPackageFile(dest string) bool {
	packageFiles := []string{
		"requirements.txt",
		"package.json",
		"package-lock.json",
		"yarn.lock",
		"go.mod",
		"go.sum",
		"Gemfile",
		"Gemfile.lock",
		"Cargo.toml",
		"Cargo.lock",
		"pom.xml",
		"build.gradle",
		"composer.json",
		"composer.lock",
	}
	dest = strings.ToLower(dest)
	for _, pf := range packageFiles {
		if strings.HasSuffix(dest, pf) || strings.Contains(dest, pf) {
			return true
		}
	}
	return false
}


// UpdateWithoutInstallRule checks for package update without install in same command (DL3012).
type UpdateWithoutInstallRule struct{}

func (r *UpdateWithoutInstallRule) ID() string             { return RuleUpdateWithoutInstall }
func (r *UpdateWithoutInstallRule) Name() string           { return "Package update without install" }
func (r *UpdateWithoutInstallRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *UpdateWithoutInstallRule) Description() string {
	return "Combine package update with install in the same RUN instruction to avoid cache issues"
}

func (r *UpdateWithoutInstallRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		run, ok := instr.(*ast.RunInstruction)
		if !ok {
			continue
		}

		cmd := run.Command

		// Check apt-get update without install in same command
		if aptGetUpdatePattern.MatchString(cmd) && !aptGetInstallPattern.MatchString(cmd) {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       run.Line(),
				Column:     1,
				Message:    "apt-get update without install in same RUN instruction",
				Suggestion: "Combine 'apt-get update' with 'apt-get install' in the same RUN instruction",
			})
			continue
		}

		// Check yum/dnf update without install in same command
		if yumUpdatePattern.MatchString(cmd) && !yumInstallPattern.MatchString(cmd) {
			// yum update alone is valid for updating packages, but yum makecache without install is not
			// Only flag if it looks like a cache refresh pattern
			if strings.Contains(cmd, "makecache") {
				findings = append(findings, ast.Finding{
					RuleID:     r.ID(),
					Severity:   r.Severity(),
					Line:       run.Line(),
					Column:     1,
					Message:    "yum/dnf makecache without install in same RUN instruction",
					Suggestion: "Combine cache refresh with install in the same RUN instruction",
				})
			}
		}
	}

	return findings
}

// init registers the layer optimization rules with the default registry.
func init() {
	RegisterDefault(&CacheNotCleanedRule{})
	RegisterDefault(&ConsecutiveRunRule{})
	RegisterDefault(&SuboptimalOrderingRule{})
	RegisterDefault(&UpdateWithoutInstallRule{})
}
