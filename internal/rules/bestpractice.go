// Package rules provides lint rule implementations for docker-lint.
package rules

import (
	"path/filepath"
	"strings"

	"github.com/devblac/docker-lint/internal/ast"
)

// wildcardChars contains characters that indicate wildcard patterns
var wildcardChars = []string{"*", "?", "["}

// MultipleCMDRule checks for multiple CMD instructions in a Dockerfile (DL3001).
type MultipleCMDRule struct{}

func (r *MultipleCMDRule) ID() string             { return RuleMultipleCMD }
func (r *MultipleCMDRule) Name() string           { return "Multiple CMD instructions" }
func (r *MultipleCMDRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *MultipleCMDRule) Description() string {
	return "Only the last CMD instruction takes effect; multiple CMD instructions are likely a mistake"
}

func (r *MultipleCMDRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	// Check each stage separately
	for _, stage := range dockerfile.Stages {
		var cmdInstrs []*ast.CmdInstruction

		for _, instr := range stage.Instructions {
			if cmd, ok := instr.(*ast.CmdInstruction); ok {
				cmdInstrs = append(cmdInstrs, cmd)
			}
		}

		// If more than one CMD, report all but the last
		if len(cmdInstrs) > 1 {
			for i := 0; i < len(cmdInstrs)-1; i++ {
				findings = append(findings, ast.Finding{
					RuleID:     r.ID(),
					Severity:   r.Severity(),
					Line:       cmdInstrs[i].Line(),
					Column:     1,
					Message:    "Multiple CMD instructions found; only the last one will take effect",
					Suggestion: "Remove duplicate CMD instructions and keep only the final one",
				})
			}
		}
	}

	return findings
}

// MultipleEntrypointRule checks for multiple ENTRYPOINT instructions in a Dockerfile (DL3002).
type MultipleEntrypointRule struct{}

func (r *MultipleEntrypointRule) ID() string             { return RuleMultipleEntrypoint }
func (r *MultipleEntrypointRule) Name() string           { return "Multiple ENTRYPOINT instructions" }
func (r *MultipleEntrypointRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *MultipleEntrypointRule) Description() string {
	return "Only the last ENTRYPOINT instruction takes effect; multiple ENTRYPOINT instructions are likely a mistake"
}

func (r *MultipleEntrypointRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	// Check each stage separately
	for _, stage := range dockerfile.Stages {
		var entrypointInstrs []*ast.EntrypointInstruction

		for _, instr := range stage.Instructions {
			if ep, ok := instr.(*ast.EntrypointInstruction); ok {
				entrypointInstrs = append(entrypointInstrs, ep)
			}
		}

		// If more than one ENTRYPOINT, report all but the last
		if len(entrypointInstrs) > 1 {
			for i := 0; i < len(entrypointInstrs)-1; i++ {
				findings = append(findings, ast.Finding{
					RuleID:     r.ID(),
					Severity:   r.Severity(),
					Line:       entrypointInstrs[i].Line(),
					Column:     1,
					Message:    "Multiple ENTRYPOINT instructions found; only the last one will take effect",
					Suggestion: "Remove duplicate ENTRYPOINT instructions and keep only the final one",
				})
			}
		}
	}

	return findings
}

// RelativeWorkdirRule checks for WORKDIR instructions with relative paths (DL3003).
type RelativeWorkdirRule struct{}

func (r *RelativeWorkdirRule) ID() string             { return RuleRelativeWorkdir }
func (r *RelativeWorkdirRule) Name() string           { return "WORKDIR with relative path" }
func (r *RelativeWorkdirRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *RelativeWorkdirRule) Description() string {
	return "Use absolute paths in WORKDIR to avoid confusion about the current directory"
}

func (r *RelativeWorkdirRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		workdir, ok := instr.(*ast.WorkdirInstruction)
		if !ok {
			continue
		}

		path := workdir.Path

		// Check if path is relative (doesn't start with / or a variable)
		if !isAbsolutePath(path) {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       workdir.Line(),
				Column:     1,
				Message:    "WORKDIR uses relative path '" + path + "'",
				Suggestion: "Use an absolute path like '/" + path + "' for clarity",
			})
		}
	}

	return findings
}

// isAbsolutePath checks if a path is absolute or starts with a variable.
func isAbsolutePath(path string) bool {
	// Empty path is not absolute
	if path == "" {
		return false
	}

	// Unix absolute path
	if strings.HasPrefix(path, "/") {
		return true
	}

	// Windows absolute path (e.g., C:\)
	if len(path) >= 2 && path[1] == ':' {
		return true
	}

	// Path starting with variable (e.g., $HOME, ${APP_DIR})
	if strings.HasPrefix(path, "$") {
		return true
	}

	return false
}

// MissingHealthcheckRule checks for Dockerfiles without HEALTHCHECK instruction (DL5000).
type MissingHealthcheckRule struct{}

func (r *MissingHealthcheckRule) ID() string             { return RuleMissingHealthcheck }
func (r *MissingHealthcheckRule) Name() string           { return "Missing HEALTHCHECK" }
func (r *MissingHealthcheckRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *MissingHealthcheckRule) Description() string {
	return "Add a HEALTHCHECK instruction to enable container health monitoring"
}

func (r *MissingHealthcheckRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	// Check if any stage has a HEALTHCHECK
	hasHealthcheck := false
	for _, instr := range dockerfile.Instructions {
		if _, ok := instr.(*ast.HealthcheckInstruction); ok {
			hasHealthcheck = true
			break
		}
	}

	// If no HEALTHCHECK found, report at the last stage's FROM or last instruction
	if !hasHealthcheck && len(dockerfile.Stages) > 0 {
		// Report on the last stage (the one that produces the final image)
		lastStage := dockerfile.Stages[len(dockerfile.Stages)-1]
		line := 1
		if lastStage.FromInstr != nil {
			line = lastStage.FromInstr.Line()
		}
		if len(lastStage.Instructions) > 0 {
			line = lastStage.Instructions[len(lastStage.Instructions)-1].Line()
		}

		findings = append(findings, ast.Finding{
			RuleID:     r.ID(),
			Severity:   r.Severity(),
			Line:       line,
			Column:     1,
			Message:    "No HEALTHCHECK instruction found",
			Suggestion: "Add 'HEALTHCHECK CMD <command>' to enable container health monitoring",
		})
	}

	return findings
}

// WildcardCopyRule checks for wildcard patterns in COPY/ADD sources (DL5001).
type WildcardCopyRule struct{}

func (r *WildcardCopyRule) ID() string             { return RuleWildcardCopy }
func (r *WildcardCopyRule) Name() string           { return "Wildcard in COPY/ADD source" }
func (r *WildcardCopyRule) Severity() ast.Severity { return ast.SeverityInfo }

func (r *WildcardCopyRule) Description() string {
	return "Wildcard patterns in COPY/ADD may include unnecessary files, increasing build context size"
}

func (r *WildcardCopyRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		switch v := instr.(type) {
		case *ast.CopyInstruction:
			// Skip COPY --from (multi-stage copies from other stages)
			if v.From != "" {
				continue
			}
			if hasWildcard(v.Sources) {
				findings = append(findings, ast.Finding{
					RuleID:     r.ID(),
					Severity:   r.Severity(),
					Line:       v.Line(),
					Column:     1,
					Message:    "COPY uses wildcard pattern which may include unnecessary files",
					Suggestion: "Consider using explicit file paths or a .dockerignore file to exclude unnecessary files",
				})
			}

		case *ast.AddInstruction:
			if hasWildcard(v.Sources) {
				findings = append(findings, ast.Finding{
					RuleID:     r.ID(),
					Severity:   r.Severity(),
					Line:       v.Line(),
					Column:     1,
					Message:    "ADD uses wildcard pattern which may include unnecessary files",
					Suggestion: "Consider using explicit file paths or a .dockerignore file to exclude unnecessary files",
				})
			}
		}
	}

	return findings
}

// hasWildcard checks if any source path contains wildcard characters.
func hasWildcard(sources []string) bool {
	for _, source := range sources {
		// Get just the filename part for checking
		base := filepath.Base(source)
		for _, wc := range wildcardChars {
			if strings.Contains(base, wc) {
				return true
			}
		}
	}
	return false
}

// init registers the best practice rules with the default registry.
func init() {
	RegisterDefault(&MultipleCMDRule{})
	RegisterDefault(&MultipleEntrypointRule{})
	RegisterDefault(&RelativeWorkdirRule{})
	RegisterDefault(&MissingHealthcheckRule{})
	RegisterDefault(&WildcardCopyRule{})
}
