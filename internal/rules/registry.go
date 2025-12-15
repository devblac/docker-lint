// Package rules provides lint rule implementations and registry for docker-lint.
package rules

import (
	"sort"
	"sync"

	"github.com/devblac/docker-lint/internal/ast"
)

// Rule IDs for base image rules (DL3xxx)
const (
	RuleMissingTag         = "DL3006" // Missing explicit image tag
	RuleLatestTag          = "DL3007" // Using 'latest' tag
	RuleLargeBaseImage     = "DL3008" // Large base image without slim variant
	RuleCacheNotCleaned    = "DL3009" // Package manager cache not cleaned
	RuleConsecutiveRun     = "DL3010" // Consecutive RUN instructions
	RuleSuboptimalOrdering = "DL3011" // Suboptimal layer ordering
	RuleUpdateWithoutInstall = "DL3012" // Package update without install
)

// Rule IDs for best practice rules (DL3xxx continued)
const (
	RuleMultipleCMD        = "DL3001" // Multiple CMD instructions
	RuleMultipleEntrypoint = "DL3002" // Multiple ENTRYPOINT instructions
	RuleRelativeWorkdir    = "DL3003" // WORKDIR with relative path
)

// Rule IDs for security rules (DL4xxx)
const (
	RuleSecretInEnv  = "DL4000" // Potential secret in ENV
	RuleSecretInArg  = "DL4001" // Potential secret in ARG
	RuleNoUser       = "DL4002" // No USER instruction (running as root)
	RuleAddWithURL   = "DL4003" // ADD with URL
	RuleAddOverCopy  = "DL4004" // ADD where COPY would suffice
)

// Rule IDs for best practice rules (DL5xxx)
const (
	RuleMissingHealthcheck = "DL5000" // Missing HEALTHCHECK
	RuleWildcardCopy       = "DL5001" // Wildcard in COPY/ADD source
)

// Rule defines the interface that all lint rules must implement.
type Rule interface {
	// ID returns the unique identifier for this rule (e.g., "DL3006").
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
