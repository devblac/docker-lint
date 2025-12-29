// Package rules provides lint rule implementations for docker-lint.
package rules

import (
	"regexp"
	"strings"

	"github.com/devblac/docker-lint/internal/ast"
)

// secretPatterns contains regex patterns for detecting potential secrets in ENV/ARG keys.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)password`),
	regexp.MustCompile(`(?i)passwd`),
	regexp.MustCompile(`(?i)secret`),
	regexp.MustCompile(`(?i)token`),
	regexp.MustCompile(`(?i)api[_-]?key`),
	regexp.MustCompile(`(?i)apikey`),
	regexp.MustCompile(`(?i)private[_-]?key`),
	regexp.MustCompile(`(?i)privatekey`),
	regexp.MustCompile(`(?i)access[_-]?key`),
	regexp.MustCompile(`(?i)accesskey`),
	regexp.MustCompile(`(?i)auth[_-]?token`),
	regexp.MustCompile(`(?i)credentials?`),
	regexp.MustCompile(`(?i)ssh[_-]?key`),
	regexp.MustCompile(`(?i)encryption[_-]?key`),
}

// urlPattern matches URLs in ADD sources
var urlPattern = regexp.MustCompile(`^https?://`)

// archiveExtensions contains file extensions that indicate archive files
var archiveExtensions = []string{
	".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".txz",
	".zip", ".gz", ".bz2", ".xz",
}

// SecretInEnvRule checks for potential secrets in ENV instructions (DL4000).
type SecretInEnvRule struct{}

func (r *SecretInEnvRule) ID() string             { return RuleSecretInEnv }
func (r *SecretInEnvRule) Name() string           { return "Potential secret in ENV" }
func (r *SecretInEnvRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *SecretInEnvRule) Description() string {
	return "Avoid storing secrets in ENV instructions as they persist in the image layers"
}

func (r *SecretInEnvRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		env, ok := instr.(*ast.EnvInstruction)
		if !ok {
			continue
		}

		// Check if the key matches any secret pattern
		if isSecretKey(env.Key) {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       env.Line(),
				Column:     1,
				Message:    "ENV instruction contains key '" + env.Key + "' which may contain a secret",
				Suggestion: "Use Docker secrets, build-time secrets (--secret), or runtime environment variables instead",
			})
		}
	}

	return findings
}

// SecretInArgRule checks for potential secrets in ARG instructions (DL4001).
type SecretInArgRule struct{}

func (r *SecretInArgRule) ID() string             { return RuleSecretInArg }
func (r *SecretInArgRule) Name() string           { return "Potential secret in ARG" }
func (r *SecretInArgRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *SecretInArgRule) Description() string {
	return "Avoid storing secrets in ARG instructions as they are visible in image history"
}

func (r *SecretInArgRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		arg, ok := instr.(*ast.ArgInstruction)
		if !ok {
			continue
		}

		// Check if the name matches any secret pattern
		if isSecretKey(arg.Name) {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       arg.Line(),
				Column:     1,
				Message:    "ARG instruction contains name '" + arg.Name + "' which may contain a secret",
				Suggestion: "Use Docker secrets or build-time secrets (--secret) instead of ARG for sensitive values",
			})
		}
	}

	return findings
}

// NoUserRule checks for Dockerfiles without USER instruction (DL4002).
type NoUserRule struct{}

func (r *NoUserRule) ID() string             { return RuleNoUser }
func (r *NoUserRule) Name() string           { return "No USER instruction" }
func (r *NoUserRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *NoUserRule) Description() string {
	return "Containers should not run as root; specify a USER instruction"
}

func (r *NoUserRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	// Check each stage separately for USER instruction
	for _, stage := range dockerfile.Stages {
		hasUser := false
		var lastInstrLine int

		for _, instr := range stage.Instructions {
			if _, ok := instr.(*ast.UserInstruction); ok {
				hasUser = true
			}
			lastInstrLine = instr.Line()
		}

		// If no USER instruction in this stage, report at the FROM line
		if !hasUser && stage.FromInstr != nil {
			// Use the FROM instruction line for the finding
			line := stage.FromInstr.Line()
			if lastInstrLine > 0 {
				line = lastInstrLine
			}

			stageName := stage.Name
			if stageName == "" {
				stageName = "stage " + intToString(stage.Index)
			}

			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       line,
				Column:     1,
				Message:    "No USER instruction in " + stageName + "; container will run as root",
				Suggestion: "Add 'USER <username>' instruction to run container as non-root user",
			})
		}
	}

	return findings
}

// AddWithURLRule checks for ADD instructions with URL sources (DL4003).
type AddWithURLRule struct{}

func (r *AddWithURLRule) ID() string             { return RuleAddWithURL }
func (r *AddWithURLRule) Name() string           { return "ADD with URL" }
func (r *AddWithURLRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *AddWithURLRule) Description() string {
	return "Using ADD with URLs is discouraged; use curl or wget in RUN for better control"
}

func (r *AddWithURLRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		add, ok := instr.(*ast.AddInstruction)
		if !ok {
			continue
		}

		// Check if any source is a URL
		for _, source := range add.Sources {
			if urlPattern.MatchString(source) {
				findings = append(findings, ast.Finding{
					RuleID:     r.ID(),
					Severity:   r.Severity(),
					Line:       add.Line(),
					Column:     1,
					Message:    "ADD with URL source is not recommended",
					Suggestion: "Use 'RUN curl -o <dest> <url>' or 'RUN wget -O <dest> <url>' for better caching and security",
				})
				break // Only report once per ADD instruction
			}
		}
	}

	return findings
}

// AddOverCopyRule checks for ADD where COPY would suffice (DL4004).
type AddOverCopyRule struct{}

func (r *AddOverCopyRule) ID() string             { return RuleAddOverCopy }
func (r *AddOverCopyRule) Name() string           { return "ADD where COPY would suffice" }
func (r *AddOverCopyRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *AddOverCopyRule) Description() string {
	return "Use COPY instead of ADD when not extracting archives or fetching URLs"
}

func (r *AddOverCopyRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		add, ok := instr.(*ast.AddInstruction)
		if !ok {
			continue
		}

		// Check if ADD is being used for URL (handled by DL4003)
		hasURL := false
		hasArchive := false

		for _, source := range add.Sources {
			// Check for URL
			if urlPattern.MatchString(source) {
				hasURL = true
				break
			}

			// Check for archive extension
			if isArchiveFile(source) {
				hasArchive = true
				break
			}
		}

		// If no URL and no archive, COPY would suffice
		if !hasURL && !hasArchive {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       add.Line(),
				Column:     1,
				Message:    "ADD used where COPY would suffice",
				Suggestion: "Use COPY instead of ADD for simple file copying; ADD should only be used for URL fetching or archive extraction",
			})
		}
	}

	return findings
}

// isSecretKey checks if a key name matches common secret patterns.
func isSecretKey(key string) bool {
	for _, pattern := range secretPatterns {
		if pattern.MatchString(key) {
			return true
		}
	}
	return false
}

// isArchiveFile checks if a filename has an archive extension.
func isArchiveFile(filename string) bool {
	lower := strings.ToLower(filename)
	for _, ext := range archiveExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// init registers the security rules with the default registry.
func init() {
	RegisterDefault(&SecretInEnvRule{})
	RegisterDefault(&SecretInArgRule{})
	RegisterDefault(&NoUserRule{})
	RegisterDefault(&AddWithURLRule{})
	RegisterDefault(&AddOverCopyRule{})
}
