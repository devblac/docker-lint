// Package rules provides lint rule implementations for docker-lint.
package rules

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/devblac/docker-lint/internal/ast"
)

// secretWords are key words that name a secret on their own. Keys are matched
// word by word rather than by substring, so TOKENIZER_MODEL or BYPASS_CACHE do
// not look like secrets.
var secretWords = map[string]bool{
	"PASSWORD":    true,
	"PASSWD":      true,
	"PASS":        true,
	"SECRET":      true,
	"TOKEN":       true,
	"CREDENTIAL":  true,
	"CREDENTIALS": true,
	// Compound spellings written without a separator.
	"APIKEY":     true,
	"PRIVATEKEY": true,
	"ACCESSKEY":  true,
	"SECRETKEY":  true,
	"AUTHTOKEN":  true,
}

// keyQualifiers are words that turn a following KEY into a secret, as in
// API_KEY or SSH_KEY. KEY alone is too generic (CACHE_KEY, SORT_KEY).
var keyQualifiers = map[string]bool{
	"API":        true,
	"PRIVATE":    true,
	"ACCESS":     true,
	"SECRET":     true,
	"SSH":        true,
	"ENCRYPTION": true,
	"SIGNING":    true,
	"MASTER":     true,
	"LICENSE":    true,
}

// metadataSuffixes end keys that describe a secret instead of holding one:
// POSTGRES_PASSWORD_FILE (the Docker secrets convention), PASSWORD_MIN_LENGTH,
// TOKEN_URL.
var metadataSuffixes = map[string]bool{
	"FILE":       true,
	"PATH":       true,
	"DIR":        true,
	"LENGTH":     true,
	"LEN":        true,
	"MIN":        true,
	"MAX":        true,
	"SIZE":       true,
	"COUNT":      true,
	"ENABLED":    true,
	"DISABLED":   true,
	"REQUIRED":   true,
	"TYPE":       true,
	"MODE":       true,
	"POLICY":     true,
	"TTL":        true,
	"TIMEOUT":    true,
	"EXPIRY":     true,
	"EXPIRES":    true,
	"EXPIRATION": true,
	"URL":        true,
	"URI":        true,
	"ENDPOINT":   true,
	"HOST":       true,
	"PORT":       true,
	"NAME":       true,
}

// urlPattern matches URLs in ADD sources
var urlPattern = regexp.MustCompile(`^https?://`)

// archiveExtensions contains file extensions that indicate archive files
var archiveExtensions = []string{
	".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".txz",
	".zip", ".gz", ".bz2", ".xz",
}

// SecretInEnvRule checks for potential secrets in ENV instructions (DKL301).
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

		// An empty value bakes nothing into the image; it is a placeholder for
		// a value supplied at runtime.
		if env.Value == "" {
			continue
		}

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

// SecretInArgRule checks for potential secrets in ARG instructions (DKL302).
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

// NoUserRule checks that the final image does not run as root (DKL303).
//
// Only the final stage matters: earlier stages are discarded after the build,
// and they often need root to install packages. The final stage inherits the
// USER of any earlier stage it builds on ("FROM builder").
type NoUserRule struct{}

func (r *NoUserRule) ID() string             { return RuleNoUser }
func (r *NoUserRule) Name() string           { return "Final image runs as root" }
func (r *NoUserRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *NoUserRule) Description() string {
	return "The final image should not run as root; set a non-root USER in the final stage"
}

func (r *NoUserRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	if len(dockerfile.Stages) == 0 {
		return nil
	}

	finalIdx := len(dockerfile.Stages) - 1
	final := dockerfile.Stages[finalIdx]
	stageLabel := "the final stage"
	if final.Name != "" {
		stageLabel = "final stage '" + final.Name + "'"
	}

	user, base := effectiveUser(dockerfile.Stages, finalIdx)

	if user == nil {
		if runsAsNonRootByDefault(base) {
			return nil
		}
		return []ast.Finding{{
			RuleID:     r.ID(),
			Severity:   r.Severity(),
			Line:       lastLine(final),
			Column:     1,
			Message:    "No USER instruction in " + stageLabel + "; container will run as root",
			Suggestion: "Add 'USER <username>' to the final stage to run the container as a non-root user",
		}}
	}

	if isRootUser(user) {
		return []ast.Finding{{
			RuleID:     r.ID(),
			Severity:   r.Severity(),
			Line:       user.Line(),
			Column:     1,
			Message:    "Last USER in " + stageLabel + " is root; container will run as root",
			Suggestion: "Switch to a non-root user with 'USER <username>' after the steps that need root",
		}}
	}

	return nil
}

// effectiveUser returns the USER instruction in effect at the end of the stage
// at idx, following FROM references to earlier stages, whose USER is inherited.
// It also returns the FROM instruction of the external image the chain starts
// from, or of the stage where a USER was found.
func effectiveUser(stages []ast.Stage, idx int) (*ast.UserInstruction, *ast.FromInstruction) {
	var base *ast.FromInstruction

	// Each step moves to a strictly earlier stage, so this terminates.
	for idx >= 0 {
		stage := stages[idx]
		base = stage.FromInstr

		var user *ast.UserInstruction
		for _, instr := range stage.Instructions {
			if u, ok := instr.(*ast.UserInstruction); ok {
				user = u // the last USER wins
			}
		}
		if user != nil {
			return user, base
		}

		idx = parentStage(stages[:idx], stage.FromInstr)
	}

	return nil, base
}

// parentStage returns the index of the latest earlier stage that from builds
// on, or -1 when from pulls an external image.
func parentStage(earlier []ast.Stage, from *ast.FromInstruction) int {
	if from == nil || from.Tag != "" || from.Digest != "" {
		return -1
	}
	for i := len(earlier) - 1; i >= 0; i-- {
		if earlier[i].Name != "" && strings.EqualFold(earlier[i].Name, from.Image) {
			return i
		}
	}
	return -1
}

// isRootUser reports whether a USER instruction selects the root user.
func isRootUser(user *ast.UserInstruction) bool {
	name := strings.TrimSpace(user.User)
	return name == "root" || name == "0"
}

// runsAsNonRootByDefault reports whether the base image is published to run as
// a non-root user without an explicit USER, as distroless ":nonroot" images are.
func runsAsNonRootByDefault(from *ast.FromInstruction) bool {
	if from == nil {
		return false
	}
	return strings.Contains(strings.ToLower(from.Tag), "nonroot") ||
		strings.Contains(strings.ToLower(from.Image), "nonroot")
}

// lastLine returns the line of the last instruction in a stage.
func lastLine(stage ast.Stage) int {
	if n := len(stage.Instructions); n > 0 {
		return stage.Instructions[n-1].Line()
	}
	if stage.FromInstr != nil {
		return stage.FromInstr.Line()
	}
	return 1
}

// AddWithURLRule checks for ADD instructions with URL sources (DKL304).
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

// AddOverCopyRule checks for ADD where COPY would suffice (DKL305).
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

		// Check if ADD is being used for URL (handled by DKL304)
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

// isSecretKey reports whether a variable name looks like it holds a secret.
func isSecretKey(key string) bool {
	words := keyWords(key)
	if len(words) == 0 || metadataSuffixes[words[len(words)-1]] {
		return false
	}

	for i, word := range words {
		if secretWords[word] {
			return true
		}
		if word == "KEY" && i > 0 && keyQualifiers[words[i-1]] {
			return true
		}
	}
	return false
}

// keyWords splits a variable name into upper-cased words on separators and
// camelCase boundaries: "dbPassword" and "DB_PASSWORD" both give [DB PASSWORD].
func keyWords(key string) []string {
	var words []string
	var current []rune

	flush := func() {
		if len(current) > 0 {
			words = append(words, strings.ToUpper(string(current)))
			current = current[:0]
		}
	}

	runes := []rune(key)
	for i, r := range runes {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
		case unicode.IsUpper(r) && i > 0 && unicode.IsLower(runes[i-1]):
			flush()
			current = append(current, r)
		default:
			current = append(current, r)
		}
	}
	flush()

	return words
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
