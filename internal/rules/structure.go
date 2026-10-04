package rules

import (
	"github.com/devblac/docker-lint/internal/ast"
)

// NoFromRule reports a Dockerfile whose instructions do not start a build stage
// with FROM (DKL001).
//
// Only ARG may appear before the first FROM. Anything else, or a Dockerfile
// with no FROM at all, fails `docker build` with "no build stage in current
// context". BuildKit's parser accepts such files - the check happens later, in
// the build - so it is reported here as a lint error.
type NoFromRule struct{}

func (r *NoFromRule) ID() string             { return RuleNoFrom }
func (r *NoFromRule) Name() string           { return "Instruction before FROM" }
func (r *NoFromRule) Severity() ast.Severity { return ast.SeverityError }

func (r *NoFromRule) Description() string {
	return "Every build must start with FROM; only ARG may appear before the first FROM"
}

func (r *NoFromRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	for _, instr := range dockerfile.Instructions {
		switch instr.(type) {
		case *ast.FromInstruction:
			return nil
		case *ast.ArgInstruction:
			continue
		}

		// Report only the first offending instruction: every later one fails
		// for the same reason.
		return []ast.Finding{{
			RuleID:     r.ID(),
			Severity:   r.Severity(),
			Line:       instr.Line(),
			Column:     1,
			Message:    string(instr.Type()) + " appears before the first FROM; the build will fail",
			Suggestion: "Start the Dockerfile with 'FROM <image>'; only ARG may come before it",
		}}
	}

	// No FROM at all: either empty or only ARG instructions.
	line := 1
	if len(dockerfile.Instructions) > 0 {
		line = dockerfile.Instructions[0].Line()
	}
	return []ast.Finding{{
		RuleID:     r.ID(),
		Severity:   r.Severity(),
		Line:       line,
		Column:     1,
		Message:    "Dockerfile has no FROM instruction; the build will fail",
		Suggestion: "Add 'FROM <image>' to start a build stage",
	}}
}

// init registers the structure rules with the default registry.
func init() {
	RegisterDefault(&NoFromRule{})
}
