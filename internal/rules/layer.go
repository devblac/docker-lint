// Package rules provides lint rule implementations for docker-lint.
package rules

import (
	"path"
	"regexp"
	"strings"

	"github.com/devblac/docker-lint/internal/ast"
)

// Package manager patterns for cache cleanup detection
var (
	// aptGetPattern matches apt-get install commands
	aptGetInstallPattern = regexp.MustCompile(`apt-get\s+(install|upgrade)`)
	// aptGetCleanPattern matches the ways apt's package lists get removed:
	// apt-get clean, apt dist-clean (apt >= 2.7, used by Debian trixie images)
	// and rm -rf /var/lib/apt/lists in any flag order.
	aptGetCleanPattern = regexp.MustCompile(`(apt(-get)?\s+(dist-)?clean|rm\s+-[a-zA-Z]+\s+/var/lib/apt/lists)`)

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

// CacheNotCleanedRule checks for package manager installs without cache cleanup (DKL201).
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

		// A cache mount keeps the package manager's cache out of the layer
		// entirely, which is better than cleaning it afterwards.

		// Check apt-get install without cleanup
		if aptGetInstallPattern.MatchString(cmd) && !aptGetCleanPattern.MatchString(cmd) &&
			!hasCacheMountFor(run, "/var/lib/apt/lists") {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       run.Line(),
				Column:     1,
				Message:    "apt-get install without cache cleanup increases image size",
				Suggestion: "Add 'rm -rf /var/lib/apt/lists/*' (or 'apt-get dist-clean') in the same RUN instruction, or use '--mount=type=cache,target=/var/lib/apt'",
			})
			continue
		}

		// Check yum/dnf install without cleanup
		if yumInstallPattern.MatchString(cmd) && !yumCleanPattern.MatchString(cmd) &&
			!hasCacheMountFor(run, "/var/cache/yum", "/var/cache/dnf") {
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
		if apkAddPattern.MatchString(cmd) && !apkNoCachePattern.MatchString(cmd) &&
			!hasCacheMountFor(run, "/var/cache/apk", "/etc/apk/cache") {
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
		if pipInstallPattern.MatchString(cmd) && !pipNoCachePattern.MatchString(cmd) && !hasPipCacheMount(run) {
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

// hasCacheMountFor reports whether the RUN mounts a build cache over any of
// the given paths, directly or through a parent directory.
func hasCacheMountFor(run *ast.RunInstruction, paths ...string) bool {
	for _, target := range run.CacheMounts {
		target = strings.TrimRight(target, "/")
		for _, p := range paths {
			if target == p || strings.HasPrefix(p, target+"/") {
				return true
			}
		}
	}
	return false
}

// hasPipCacheMount reports whether the RUN mounts a build cache over pip's
// cache, which lives in ~/.cache/pip for whichever user runs pip.
func hasPipCacheMount(run *ast.RunInstruction) bool {
	for _, target := range run.CacheMounts {
		target = strings.TrimRight(target, "/")
		if strings.HasSuffix(target, "/.cache/pip") || strings.HasSuffix(target, "/.cache") {
			return true
		}
	}
	return false
}

// ConsecutiveRunRule checks for consecutive RUN instructions that could be combined (DKL202).
//
// It is informational: with BuildKit, separate RUN steps are often deliberate,
// trading a few layers for clearer, independently cached build steps. The
// regression corpus showed it firing on three files out of four, including
// Docker's own official images.
type ConsecutiveRunRule struct{}

func (r *ConsecutiveRunRule) ID() string             { return RuleConsecutiveRun }
func (r *ConsecutiveRunRule) Name() string           { return "Consecutive RUN instructions" }
func (r *ConsecutiveRunRule) Severity() ast.Severity { return ast.SeverityInfo }

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

// SuboptimalOrderingRule checks for COPY/ADD before RUN that doesn't depend on copied files (DKL203).
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

	// Track COPY/ADD instructions and their sources
	type copyInfo struct {
		instr   ast.Instruction
		sources []string
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
			copies = append(copies, copyInfo{instr: v, sources: v.Sources})

		case *ast.AddInstruction:
			copies = append(copies, copyInfo{instr: v, sources: v.Sources})

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
						// Copying only dependency manifests before the install
						// step is the recommended caching pattern.
						if isDependencyManifestCopy(cp.sources) {
							continue
						}
						// The RUN uses what this COPY brought in (a script, a
						// config file), so the COPY has to stay before it.
						if runReferencesCopy(v.Command, cp.sources) {
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

// runReferencesCopy reports whether a command mentions a file brought in by a
// COPY/ADD, which means the RUN depends on it. Only distinctive names count:
// short names such as "." or "src" would match almost any command.
func runReferencesCopy(command string, sources []string) bool {
	for _, source := range sources {
		name := path.Base(strings.TrimRight(source, "/"))
		if len(name) < 4 || strings.ContainsAny(name, "*?[") {
			continue
		}
		if strings.Contains(command, name) {
			return true
		}
	}
	return false
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

// packageFiles are dependency manifests and lock files, lower-cased.
var packageFiles = []string{
	// Python
	"requirements.txt", "pyproject.toml", "poetry.lock", "pipfile", "pipfile.lock",
	"setup.py", "setup.cfg", "uv.lock",
	// JavaScript
	"package.json", "package-lock.json", "npm-shrinkwrap.json", "yarn.lock",
	"pnpm-lock.yaml", ".npmrc", ".yarnrc.yml",
	// Go
	"go.mod", "go.sum", "go.work", "go.work.sum",
	// Ruby, Rust, PHP, Elixir
	"gemfile", "gemfile.lock", "cargo.toml", "cargo.lock",
	"composer.json", "composer.lock", "mix.exs", "mix.lock",
	// JVM
	"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle",
	"settings.gradle.kts", "gradle.properties",
	// .NET
	"packages.config", "nuget.config",
}

// packageFileSuffixes are extensions of dependency manifests named per project.
var packageFileSuffixes = []string{".csproj", ".fsproj", ".vbproj", ".sln"}

// wildcardManifestCandidates are the names a wildcard source is matched
// against: every known manifest plus examples of the per-project ones.
var wildcardManifestCandidates = append(
	append([]string{}, packageFiles...),
	"requirements-dev.txt", "app.csproj", "app.fsproj", "app.sln",
)

// nonManifestProbes are ordinary source files and directories. A wildcard
// source that matches any of them ("*", "*.go") copies more than manifests.
var nonManifestProbes = []string{"main.go", "index.js", "app.py", "readme.md", "dockerfile", "src"}

// isDependencyManifestCopy reports whether every source of a COPY/ADD is a
// dependency manifest, as in "COPY go.mod go.sum ./".
func isDependencyManifestCopy(sources []string) bool {
	if len(sources) == 0 {
		return false
	}
	for _, source := range sources {
		if !isPackageFile(source) {
			return false
		}
	}
	return true
}

// isPackageFile reports whether a path names a dependency manifest or lock
// file. Wildcard patterns count when they select manifests only, as
// "package*.json" or "*.csproj" do.
func isPackageFile(p string) bool {
	base := strings.ToLower(path.Base(strings.TrimRight(p, "/")))

	if strings.ContainsAny(base, "*?[") {
		for _, probe := range nonManifestProbes {
			if ok, _ := path.Match(base, probe); ok {
				return false
			}
		}
		for _, candidate := range wildcardManifestCandidates {
			if ok, _ := path.Match(base, candidate); ok {
				return true
			}
		}
		return false
	}

	for _, pf := range packageFiles {
		if base == pf {
			return true
		}
	}
	for _, suffix := range packageFileSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}

	// requirements-dev.txt, requirements.prod.txt and the like
	ok, _ := path.Match("requirements*.txt", base)
	return ok
}

// UpdateWithoutInstallRule checks for package update without install in same command (DKL204).
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
