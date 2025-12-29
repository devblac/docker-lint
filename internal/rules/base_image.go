// Package rules provides lint rule implementations for docker-lint.
package rules

import (
	"strings"

	"github.com/devblac/docker-lint/internal/ast"
)

// largeBaseImages is a set of known large base images that have smaller alternatives.
var largeBaseImages = map[string]string{
	"ubuntu":      "ubuntu:*-slim or alpine",
	"debian":      "debian:*-slim or alpine",
	"centos":      "alpine or distroless",
	"fedora":      "alpine or distroless",
	"amazonlinux": "alpine or distroless",
	"oraclelinux": "alpine or distroless",
	"python":      "python:*-slim or python:*-alpine",
	"node":        "node:*-slim or node:*-alpine",
	"ruby":        "ruby:*-slim or ruby:*-alpine",
	"golang":      "golang:*-alpine or distroless",
	"openjdk":     "openjdk:*-slim or eclipse-temurin:*-alpine",
	"java":        "eclipse-temurin:*-alpine or distroless",
	"php":         "php:*-alpine",
	"perl":        "perl:*-slim",
	"rust":        "rust:*-slim or rust:*-alpine",
}

// MissingTagRule checks for FROM instructions without explicit image tags (DL3006).
type MissingTagRule struct{}

func (r *MissingTagRule) ID() string             { return RuleMissingTag }
func (r *MissingTagRule) Name() string           { return "Missing explicit image tag" }
func (r *MissingTagRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *MissingTagRule) Description() string {
	return "Always tag the version of an image explicitly to ensure reproducible builds"
}

func (r *MissingTagRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		from, ok := instr.(*ast.FromInstruction)
		if !ok {
			continue
		}

		// Skip scratch image (special case, no tag needed)
		if strings.ToLower(from.Image) == "scratch" {
			continue
		}

		// Skip if using digest (more specific than tag)
		if from.Digest != "" {
			continue
		}

		// Check if tag is missing (empty)
		if from.Tag == "" {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       from.Line(),
				Column:     1,
				Message:    "Image '" + from.Image + "' does not have an explicit tag, defaulting to 'latest'",
				Suggestion: "Use explicit tag like '" + from.Image + ":<version>' for reproducible builds",
			})
		}
	}

	return findings
}

// LatestTagRule checks for FROM instructions using the 'latest' tag (DL3007).
type LatestTagRule struct{}

func (r *LatestTagRule) ID() string             { return RuleLatestTag }
func (r *LatestTagRule) Name() string           { return "Using 'latest' tag" }
func (r *LatestTagRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *LatestTagRule) Description() string {
	return "Using 'latest' tag can lead to unpredictable builds as the image may change"
}

func (r *LatestTagRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		from, ok := instr.(*ast.FromInstruction)
		if !ok {
			continue
		}

		// Skip scratch image
		if strings.ToLower(from.Image) == "scratch" {
			continue
		}

		// Skip if using digest (more specific than tag)
		if from.Digest != "" {
			continue
		}

		// Check if tag is explicitly 'latest'
		if strings.ToLower(from.Tag) == "latest" {
			findings = append(findings, ast.Finding{
				RuleID:     r.ID(),
				Severity:   r.Severity(),
				Line:       from.Line(),
				Column:     1,
				Message:    "Using 'latest' tag for image '" + from.Image + "' is not recommended",
				Suggestion: "Pin to a specific version like '" + from.Image + ":<version>' for reproducible builds",
			})
		}
	}

	return findings
}

// LargeBaseImageRule checks for large base images without slim variants (DL3008).
type LargeBaseImageRule struct{}

func (r *LargeBaseImageRule) ID() string             { return RuleLargeBaseImage }
func (r *LargeBaseImageRule) Name() string           { return "Large base image" }
func (r *LargeBaseImageRule) Severity() ast.Severity { return ast.SeverityWarning }

func (r *LargeBaseImageRule) Description() string {
	return "Consider using a smaller base image variant to reduce image size"
}

func (r *LargeBaseImageRule) Check(dockerfile *ast.Dockerfile) []ast.Finding {
	var findings []ast.Finding

	for _, instr := range dockerfile.Instructions {
		from, ok := instr.(*ast.FromInstruction)
		if !ok {
			continue
		}

		// Extract base image name (without registry prefix)
		imageName := extractBaseImageName(from.Image)

		// Check if it's a known large base image
		alternative, isLarge := largeBaseImages[imageName]
		if !isLarge {
			continue
		}

		// Check if already using a slim/alpine variant
		tag := strings.ToLower(from.Tag)
		if isSlimVariant(tag) {
			continue
		}

		findings = append(findings, ast.Finding{
			RuleID:     r.ID(),
			Severity:   r.Severity(),
			Line:       from.Line(),
			Column:     1,
			Message:    "Image '" + from.Image + "' is a large base image",
			Suggestion: "Consider using " + alternative + " for smaller image size",
		})
	}

	return findings
}

// extractBaseImageName extracts the image name without registry prefix.
// e.g., "docker.io/library/ubuntu" -> "ubuntu"
// e.g., "gcr.io/project/ubuntu" -> "ubuntu"
func extractBaseImageName(image string) string {
	// Remove registry prefix if present
	parts := strings.Split(image, "/")
	name := parts[len(parts)-1]
	return strings.ToLower(name)
}

// isSlimVariant checks if the tag indicates a slim/alpine/minimal variant.
func isSlimVariant(tag string) bool {
	if tag == "" {
		return false
	}
	tag = strings.ToLower(tag)
	slimIndicators := []string{"slim", "alpine", "minimal", "distroless", "scratch", "tiny", "micro"}
	for _, indicator := range slimIndicators {
		if strings.Contains(tag, indicator) {
			return true
		}
	}
	return false
}

// init registers the base image rules with the default registry.
func init() {
	RegisterDefault(&MissingTagRule{})
	RegisterDefault(&LatestTagRule{})
	RegisterDefault(&LargeBaseImageRule{})
}
