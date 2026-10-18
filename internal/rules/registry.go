// Package rules provides lint rule implementations and registry for docker-lint.
package rules

import (
	"sort"
	"sync"

	"github.com/devblac/docker-lint/internal/ast"
)

// Rule IDs use the DKL prefix and are grouped by category in hundreds:
//
//	DKL0xx  Dockerfile structure
//	DKL1xx  base image
//	DKL2xx  layers and build efficiency
//	DKL3xx  security
//	DKL4xx  best practices
//
// The prefix keeps them apart from hadolint's DLxxxx IDs.

// Rule IDs for Dockerfile structure rules (DKL0xx)
const (
	RuleNoFrom = "DKL001" // Instructions without a preceding FROM
)

// Rule IDs for base image rules (DKL1xx)
const (
	RuleMissingTag     = "DKL101" // Missing explicit image tag
	RuleLatestTag      = "DKL102" // Using 'latest' tag
	RuleLargeBaseImage = "DKL103" // Large base image for the final image
)

// Rule IDs for layer and build efficiency rules (DKL2xx)
const (
	RuleCacheNotCleaned      = "DKL201" // Package manager cache not cleaned
	RuleConsecutiveRun       = "DKL202" // Consecutive RUN instructions
	RuleSuboptimalOrdering   = "DKL203" // Suboptimal layer ordering
	RuleUpdateWithoutInstall = "DKL204" // Package update without install
)

// Rule IDs for security rules (DKL3xx)
const (
	RuleSecretInEnv = "DKL301" // Potential secret in ENV
	RuleSecretInArg = "DKL302" // Potential secret in ARG
	RuleNoUser      = "DKL303" // Final image runs as root
	RuleAddWithURL  = "DKL304" // ADD with URL
	RuleAddOverCopy = "DKL305" // ADD where COPY would suffice
)

// Rule IDs for best practice rules (DKL4xx)
const (
	RuleMultipleCMD        = "DKL401" // Multiple CMD instructions
	RuleMultipleEntrypoint = "DKL402" // Multiple ENTRYPOINT instructions
	RuleRelativeWorkdir    = "DKL403" // WORKDIR with relative path
	RuleMissingHealthcheck = "DKL404" // Missing HEALTHCHECK
	RuleWildcardCopy       = "DKL405" // Wildcard in COPY/ADD source
)

// LegacyRuleIDs maps the DLxxxx IDs used before v0.1.0 to their replacements.
//
// They are deliberately not accepted as aliases: DLxxxx are hadolint's IDs and
// name different rules there, which is why they were replaced. Knowing the
// replacement only lets the CLI turn a silently ineffective --ignore into an
// actionable message.
var LegacyRuleIDs = map[string]string{
	"DL3006": RuleMissingTag,
	"DL3007": RuleLatestTag,
	"DL3008": RuleLargeBaseImage,
	"DL3009": RuleCacheNotCleaned,
	"DL3010": RuleConsecutiveRun,
	"DL3011": RuleSuboptimalOrdering,
	"DL3012": RuleUpdateWithoutInstall,
	"DL4000": RuleSecretInEnv,
	"DL4001": RuleSecretInArg,
	"DL4002": RuleNoUser,
	"DL4003": RuleAddWithURL,
	"DL4004": RuleAddOverCopy,
	"DL3001": RuleMultipleCMD,
	"DL3002": RuleMultipleEntrypoint,
	"DL3003": RuleRelativeWorkdir,
	"DL5000": RuleMissingHealthcheck,
	"DL5001": RuleWildcardCopy,
}

// Rule defines the interface that all lint rules must implement.
type Rule interface {
	// ID returns the unique identifier for this rule (e.g., "DKL101").
	ID() string

	// Name returns a short human-readable name for this rule.
	Name() string

	// Description returns a detailed description of what this rule checks.
	Description() string

	// Severity returns the default severity level for findings from this rule.
	Severity() ast.Severity

	// Check analyzes the Dockerfile and returns any findings.
	Check(dockerfile *ast.Dockerfile) []ast.Finding
}

// RuleRegistry manages the collection of available lint rules.
type RuleRegistry struct {
	mu    sync.RWMutex
	rules map[string]Rule
}

// NewRegistry creates a new empty RuleRegistry.
func NewRegistry() *RuleRegistry {
	return &RuleRegistry{
		rules: make(map[string]Rule),
	}
}

// Register adds a rule to the registry.
// If a rule with the same ID already exists, it will be replaced.
func (r *RuleRegistry) Register(rule Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rule.ID()] = rule
}

// Get retrieves a rule by its ID.
// Returns nil if the rule is not found.
func (r *RuleRegistry) Get(id string) Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rules[id]
}

// All returns all registered rules sorted by ID.
func (r *RuleRegistry) All() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect all rule IDs
	ids := make([]string, 0, len(r.rules))
	for id := range r.rules {
		ids = append(ids, id)
	}

	// Sort IDs for consistent ordering
	sort.Strings(ids)

	// Build sorted rule slice
	result := make([]Rule, 0, len(r.rules))
	for _, id := range ids {
		result = append(result, r.rules[id])
	}

	return result
}

// Count returns the number of registered rules.
func (r *RuleRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.rules)
}

// DefaultRegistry is the global registry containing all built-in rules.
var DefaultRegistry = NewRegistry()

// RegisterDefault registers a rule with the default registry.
func RegisterDefault(rule Rule) {
	DefaultRegistry.Register(rule)
}
