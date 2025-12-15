// Package rules provides lint rule implementations for docker-lint.
package rules

import (
	"reflect"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: docker-lint, Property 9: Latest Tag Detection Completeness**
// **Validates: Requirements 2.1, 2.2**
//
// Property: For any FROM instruction using "latest" tag explicitly or implicitly
// (no tag), the findings SHALL contain exactly one warning about tag usage.
func TestLatestTagDetectionCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	missingTagRule := &MissingTagRule{}
	latestTagRule := &LatestTagRule{}

	// Property: FROM with no tag triggers exactly one warning (DL3006)
	properties.Property("FROM with no tag triggers exactly one missing tag warning", prop.ForAll(
		func(image string) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{
						LineNum: 1,
						Image:   image,
						Tag:     "", // No tag - implicitly latest
					},
				},
			}

			findings := missingTagRule.Check(dockerfile)

			// Should have exactly one finding for missing tag
			if len(findings) != 1 {
				return false
			}

			// Finding should be DL3006
			if findings[0].RuleID != RuleMissingTag {
				return false
			}

			// Finding should be a warning
			if findings[0].Severity != ast.SeverityWarning {
				return false
			}

			return true
		},
		genNonScratchImageName(),
	))

	// Property: FROM with explicit "latest" tag triggers exactly one warning (DL3007)
	properties.Property("FROM with explicit latest tag triggers exactly one latest tag warning", prop.ForAll(
		func(image string, latestVariant string) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{
						LineNum: 1,
						Image:   image,
						Tag:     latestVariant, // Explicit latest tag
					},
				},
			}

			findings := latestTagRule.Check(dockerfile)

			// Should have exactly one finding for latest tag
			if len(findings) != 1 {
				return false
			}

			// Finding should be DL3007
			if findings[0].RuleID != RuleLatestTag {
				return false
			}

			// Finding should be a warning
			if findings[0].Severity != ast.SeverityWarning {
				return false
			}

			return true
		},
		genNonScratchImageName(),
		genLatestTagVariant(),
	))

	// Property: FROM with specific version tag triggers no warnings from either rule
	properties.Property("FROM with specific version tag triggers no tag warnings", prop.ForAll(
		func(image string, tag string) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{
						LineNum: 1,
						Image:   image,
						Tag:     tag,
					},
				},
			}

			missingFindings := missingTagRule.Check(dockerfile)
			latestFindings := latestTagRule.Check(dockerfile)

			// Should have no findings from either rule
			return len(missingFindings) == 0 && len(latestFindings) == 0
		},
		genNonScratchImageName(),
		genSpecificVersionTag(),
	))

	// Property: FROM with digest triggers no tag warnings (digest is more specific)
	properties.Property("FROM with digest triggers no tag warnings", prop.ForAll(
		func(image string, digest string) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{
						LineNum: 1,
						Image:   image,
						Tag:     "", // No tag
						Digest:  digest,
					},
				},
			}

			missingFindings := missingTagRule.Check(dockerfile)
			latestFindings := latestTagRule.Check(dockerfile)

			// Should have no findings from either rule when digest is present
			return len(missingFindings) == 0 && len(latestFindings) == 0
		},
		genNonScratchImageName(),
		genDigest(),
	))

	// Property: scratch image triggers no tag warnings
	properties.Property("scratch image triggers no tag warnings", prop.ForAll(
		func(scratchVariant string, tag string) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.FromInstruction{
						LineNum: 1,
						Image:   scratchVariant,
						Tag:     tag,
					},
				},
			}

			missingFindings := missingTagRule.Check(dockerfile)
			latestFindings := latestTagRule.Check(dockerfile)

			// Should have no findings for scratch image
			return len(missingFindings) == 0 && len(latestFindings) == 0
		},
		genScratchImageVariant(),
		gen.OneConstOf("", "latest"),
	))

	// Property: Multiple FROM instructions each get their own warning
	properties.Property("multiple FROM with no tag each trigger one warning", prop.ForAll(
		func(images []string) bool {
			instructions := make([]ast.Instruction, len(images))
			for i, img := range images {
				instructions[i] = &ast.FromInstruction{
					LineNum: i + 1,
					Image:   img,
					Tag:     "", // No tag
				}
			}

			dockerfile := &ast.Dockerfile{
				Instructions: instructions,
			}

			findings := missingTagRule.Check(dockerfile)

			// Should have exactly one finding per FROM instruction
			return len(findings) == len(images)
		},
		genNonScratchImageSlice(1, 5),
	))

	properties.TestingRun(t)
}

// Generator helpers

// genNonScratchImageName generates image names that are not "scratch"
func genNonScratchImageName() gopter.Gen {
	return gen.OneConstOf(
		"alpine",
		"ubuntu",
		"debian",
		"nginx",
		"node",
		"golang",
		"python",
		"redis",
		"postgres",
		"mysql",
		"busybox",
	)
}

// genLatestTagVariant generates variations of the "latest" tag
func genLatestTagVariant() gopter.Gen {
	return gen.OneConstOf(
		"latest",
		"LATEST",
		"Latest",
	)
}

// genSpecificVersionTag generates specific version tags (not "latest")
func genSpecificVersionTag() gopter.Gen {
	return gen.OneConstOf(
		"3.18",
		"22.04",
		"1.21",
		"18-alpine",
		"3.11-slim",
		"14.2",
		"8.0",
		"stable",
		"bullseye",
		"bookworm",
	)
}

// genDigest generates valid-looking SHA256 digests
func genDigest() gopter.Gen {
	return gen.OneConstOf(
		"sha256:abc123def456",
		"sha256:1234567890abcdef",
		"sha256:fedcba0987654321",
	)
}

// genScratchImageVariant generates variations of the "scratch" image name
func genScratchImageVariant() gopter.Gen {
	return gen.OneConstOf(
		"scratch",
		"SCRATCH",
		"Scratch",
	)
}

// genNonScratchImageSlice generates a slice of non-scratch image names
func genNonScratchImageSlice(minSize, maxSize int) gopter.Gen {
	return gen.IntRange(minSize, maxSize).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		gens := make([]gopter.Gen, count)
		for i := 0; i < count; i++ {
			gens[i] = genNonScratchImageName()
		}
		return gopter.CombineGens(gens...).Map(func(vals []interface{}) []string {
			result := make([]string, len(vals))
			for i, v := range vals {
				result[i] = v.(string)
			}
			return result
		})
	}, reflect.TypeOf([]string{}))
}
