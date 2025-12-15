// Package analyzer provides the orchestration layer for running lint rules against Dockerfiles.
package analyzer

import (
	"reflect"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
	"github.com/devblac/docker-lint/internal/rules"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: docker-lint, Property 2: Rule Determinism**
// **Validates: Requirements 1.5**
//
// Property: For any Dockerfile input and fixed configuration, running the
// analyzer multiple times SHALL produce identical findings in the same order.
func TestRuleDeterminism(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("analyzer produces identical findings on repeated runs", prop.ForAll(
		func(df *ast.Dockerfile, config Config) bool {
			analyzer := NewWithDefaults(config)

			// Run analysis multiple times
			const numRuns = 5
			var firstFindings []ast.Finding

			for i := 0; i < numRuns; i++ {
				findings := analyzer.Analyze(df)

				if i == 0 {
					firstFindings = findings
					continue
				}

				// Compare with first run
				if !findingsEqual(firstFindings, findings) {
					return false
				}
			}

			return true
		},
		genDockerfile(),
		genConfig(),
	))

	properties.TestingRun(t)
}

// findingsEqual compares two slices of findings for equality.
func findingsEqual(a, b []ast.Finding) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].RuleID != b[i].RuleID ||
			a[i].Severity != b[i].Severity ||
			a[i].Line != b[i].Line ||
			a[i].Column != b[i].Column ||
			a[i].Message != b[i].Message ||
			a[i].Suggestion != b[i].Suggestion {
			return false
		}
	}

	return true
}

// genDockerfile generates random valid Dockerfile ASTs for property testing.
func genDockerfile() gopter.Gen {
	return gen.IntRange(1, 5).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		return genInstructionSlice(count).FlatMap(func(instrs interface{}) gopter.Gen {
			instructions := instrs.([]ast.Instruction)
			// Ensure first instruction is FROM
			if len(instructions) > 0 {
				if _, ok := instructions[0].(*ast.FromInstruction); !ok {
					instructions[0] = &ast.FromInstruction{
						LineNum: 1,
						Image:   "alpine",
						Tag:     "3.18",
					}
				}
			}
			// Update line numbers
			for i := range instructions {
				setLineNum(instructions[i], i+1)
			}
			return gen.Const(&ast.Dockerfile{
				Instructions:  instructions,
				Stages:        []ast.Stage{},
				Comments:      []ast.Comment{},
				InlineIgnores: make(map[int][]string),
			})
		}, reflect.TypeOf(&ast.Dockerfile{}))
	}, reflect.TypeOf(&ast.Dockerfile{}))
}

// setLineNum sets the line number for an instruction.
func setLineNum(instr ast.Instruction, line int) {
	switch i := instr.(type) {
	case *ast.FromInstruction:
		i.LineNum = line
	case *ast.RunInstruction:
		i.LineNum = line
	case *ast.CopyInstruction:
		i.LineNum = line
	case *ast.AddInstruction:
		i.LineNum = line
	case *ast.EnvInstruction:
		i.LineNum = line
	case *ast.ArgInstruction:
		i.LineNum = line
	case *ast.WorkdirInstruction:
		i.LineNum = line
	case *ast.UserInstruction:
		i.LineNum = line
	case *ast.ExposeInstruction:
		i.LineNum = line
	case *ast.CmdInstruction:
		i.LineNum = line
	case *ast.EntrypointInstruction:
		i.LineNum = line
	case *ast.HealthcheckInstruction:
		i.LineNum = line
	case *ast.LabelInstruction:
		i.LineNum = line
	case *ast.VolumeInstruction:
		i.LineNum = line
	case *ast.ShellInstruction:
		i.LineNum = line
	case *ast.StopsignalInstruction:
		i.LineNum = line
	case *ast.OnbuildInstruction:
		i.LineNum = line
	}
}

// genInstructionSlice generates a slice of instructions of the given size.
func genInstructionSlice(n int) gopter.Gen {
	gens := make([]gopter.Gen, n)
	for i := 0; i < n; i++ {
		gens[i] = genInstruction()
	}
	return gopter.CombineGens(gens...).Map(func(vals []interface{}) []ast.Instruction {
		result := make([]ast.Instruction, len(vals))
		for i, v := range vals {
			result[i] = v.(ast.Instruction)
		}
		return result
	})
}

// genInstruction generates a random Dockerfile instruction.
func genInstruction() gopter.Gen {
	return gen.OneGenOf(
		genFromInstruction(),
		genRunInstruction(),
		genCopyInstruction(),
		genAddInstruction(),
		genEnvInstruction(),
		genArgInstruction(),
		genWorkdirInstruction(),
		genUserInstruction(),
		genCmdInstruction(),
		genEntrypointInstruction(),
		genHealthcheckInstruction(),
	)
}

// genFromInstruction generates a random FROM instruction.
func genFromInstruction() gopter.Gen {
	return gopter.CombineGens(
		genImageName(),
		genOptionalTag(),
		genOptionalAlias(),
	).Map(func(vals []interface{}) ast.Instruction {
		return &ast.FromInstruction{
			LineNum: 1,
			Image:   vals[0].(string),
			Tag:     vals[1].(string),
			Alias:   vals[2].(string),
		}
	})
}

// genRunInstruction generates a random RUN instruction.
func genRunInstruction() gopter.Gen {
	return genCommand().Map(func(cmd string) ast.Instruction {
		return &ast.RunInstruction{
			LineNum: 1,
			Command: cmd,
			Shell:   true,
		}
	})
}

// genCopyInstruction generates a random COPY instruction.
func genCopyInstruction() gopter.Gen {
	return gopter.CombineGens(
		genPath(),
		genPath(),
	).Map(func(vals []interface{}) ast.Instruction {
		return &ast.CopyInstruction{
			LineNum: 1,
			Sources: []string{vals[0].(string)},
			Dest:    vals[1].(string),
		}
	})
}

// genAddInstruction generates a random ADD instruction.
func genAddInstruction() gopter.Gen {
	return gopter.CombineGens(
		genAddSource(),
		genPath(),
	).Map(func(vals []interface{}) ast.Instruction {
		return &ast.AddInstruction{
			LineNum: 1,
			Sources: []string{vals[0].(string)},
			Dest:    vals[1].(string),
		}
	})
}

// genEnvInstruction generates a random ENV instruction.
func genEnvInstruction() gopter.Gen {
	return gopter.CombineGens(
		genEnvKey(),
		genEnvValue(),
	).Map(func(vals []interface{}) ast.Instruction {
		return &ast.EnvInstruction{
			LineNum: 1,
			Key:     vals[0].(string),
			Value:   vals[1].(string),
		}
	})
}

// genArgInstruction generates a random ARG instruction.
func genArgInstruction() gopter.Gen {
	return gopter.CombineGens(
		genArgName(),
		gen.OneConstOf("", "default_value", "1.0"),
	).Map(func(vals []interface{}) ast.Instruction {
		return &ast.ArgInstruction{
			LineNum: 1,
			Name:    vals[0].(string),
			Default: vals[1].(string),
		}
	})
}

// genWorkdirInstruction generates a random WORKDIR instruction.
func genWorkdirInstruction() gopter.Gen {
	return genWorkdirPath().Map(func(path string) ast.Instruction {
		return &ast.WorkdirInstruction{
			LineNum: 1,
			Path:    path,
		}
	})
}

// genUserInstruction generates a random USER instruction.
func genUserInstruction() gopter.Gen {
	return gopter.CombineGens(
		genUsername(),
		gen.OneConstOf("", "nogroup", "users"),
	).Map(func(vals []interface{}) ast.Instruction {
		return &ast.UserInstruction{
			LineNum: 1,
			User:    vals[0].(string),
			Group:   vals[1].(string),
		}
	})
}

// genCmdInstruction generates a random CMD instruction.
func genCmdInstruction() gopter.Gen {
	return gen.OneConstOf("echo hello", "./app", "node server.js").Map(func(cmd string) ast.Instruction {
		return &ast.CmdInstruction{
			LineNum: 1,
			Command: []string{cmd},
			Shell:   true,
		}
	})
}

// genEntrypointInstruction generates a random ENTRYPOINT instruction.
func genEntrypointInstruction() gopter.Gen {
	return gen.OneConstOf("./app", "/entrypoint.sh", "python main.py").Map(func(cmd string) ast.Instruction {
		return &ast.EntrypointInstruction{
			LineNum: 1,
			Command: []string{cmd},
			Shell:   true,
		}
	})
}

// genHealthcheckInstruction generates a random HEALTHCHECK instruction.
func genHealthcheckInstruction() gopter.Gen {
	return gen.Bool().Map(func(none bool) ast.Instruction {
		if none {
			return &ast.HealthcheckInstruction{
				LineNum: 1,
				None:    true,
			}
		}
		return &ast.HealthcheckInstruction{
			LineNum:  1,
			None:     false,
			Interval: "30s",
			Timeout:  "10s",
			Command:  []string{"CMD", "curl", "-f", "http://localhost/"},
		}
	})
}

// genConfig generates a random analyzer configuration.
func genConfig() gopter.Gen {
	return genIgnoreRules().Map(func(ignoreRules []string) Config {
		return Config{
			IgnoreRules: ignoreRules,
		}
	})
}

// genIgnoreRules generates a random list of rule IDs to ignore.
func genIgnoreRules() gopter.Gen {
	allRules := []string{
		rules.RuleMissingTag,
		rules.RuleLatestTag,
		rules.RuleLargeBaseImage,
		rules.RuleCacheNotCleaned,
		rules.RuleConsecutiveRun,
		rules.RuleSuboptimalOrdering,
		rules.RuleUpdateWithoutInstall,
		rules.RuleMultipleCMD,
		rules.RuleMultipleEntrypoint,
		rules.RuleRelativeWorkdir,
		rules.RuleSecretInEnv,
		rules.RuleSecretInArg,
		rules.RuleNoUser,
		rules.RuleAddWithURL,
		rules.RuleAddOverCopy,
		rules.RuleMissingHealthcheck,
		rules.RuleWildcardCopy,
	}

	return gen.IntRange(0, 3).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		if count == 0 {
			return gen.Const([]string{})
		}
		return gen.SliceOfN(count, gen.OneConstOf(
			allRules[0], allRules[1], allRules[2], allRules[3],
			allRules[4], allRules[5], allRules[6], allRules[7],
		)).Map(func(rules []string) []string {
			// Deduplicate
			seen := make(map[string]bool)
			result := make([]string, 0, len(rules))
			for _, r := range rules {
				if !seen[r] {
					seen[r] = true
					result = append(result, r)
				}
			}
			return result
		})
	}, reflect.TypeOf([]string{}))
}

// Helper generators for primitive values

func genImageName() gopter.Gen {
	return gen.OneConstOf("alpine", "ubuntu", "debian", "nginx", "node", "golang", "python")
}

func genOptionalTag() gopter.Gen {
	return gen.OneConstOf("", "latest", "3.18", "22.04", "1.21", "18", "3.11")
}

func genOptionalAlias() gopter.Gen {
	return gen.OneConstOf("", "builder", "base", "runtime", "final")
}

func genCommand() gopter.Gen {
	return gen.OneConstOf(
		"echo hello",
		"apt-get update",
		"apt-get install -y curl",
		"apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*",
		"apk add --no-cache curl",
		"npm install",
		"go build",
	)
}

func genPath() gopter.Gen {
	return gen.OneConstOf(".", "src", "app", "/app", "/usr/src/app", "package.json", "*.go")
}

func genAddSource() gopter.Gen {
	return gen.OneConstOf(
		".",
		"src",
		"app.tar.gz",
		"https://example.com/file.tar.gz",
		"package.json",
	)
}

func genWorkdirPath() gopter.Gen {
	return gen.OneConstOf("/app", "/usr/src/app", "/home/user", "app", "relative/path")
}

func genEnvKey() gopter.Gen {
	return gen.OneConstOf("PATH", "HOME", "NODE_ENV", "GO_VERSION", "APP_PORT", "API_KEY", "PASSWORD")
}

func genEnvValue() gopter.Gen {
	return gen.OneConstOf("/usr/local/bin", "/home/user", "production", "1.21", "8080", "secret123")
}

func genArgName() gopter.Gen {
	return gen.OneConstOf("VERSION", "BUILD_DATE", "APP_NAME", "SECRET_TOKEN", "API_KEY")
}

func genUsername() gopter.Gen {
	return gen.OneConstOf("nobody", "root", "app", "node", "www-data")
}

// **Feature: docker-lint, Property 3: Ignore Rule Exclusion**
// **Validates: Requirements 8.1**
//
// Property: For any Dockerfile and any set of ignored rule IDs, the findings
// SHALL NOT contain any finding with a rule ID in the ignored set.
func TestIgnoreRuleExclusion(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("findings never contain ignored rule IDs", prop.ForAll(
		func(df *ast.Dockerfile, ignoreRules []string) bool {
			// Create analyzer with the ignore rules
			config := Config{
				IgnoreRules: ignoreRules,
			}
			analyzer := NewWithDefaults(config)

			// Run analysis
			findings := analyzer.Analyze(df)

			// Build a set of ignored rules for fast lookup
			ignoredSet := make(map[string]bool)
			for _, ruleID := range ignoreRules {
				ignoredSet[ruleID] = true
			}

			// Verify no finding has a rule ID in the ignored set
			for _, finding := range findings {
				if ignoredSet[finding.RuleID] {
					t.Logf("Found ignored rule %s in findings", finding.RuleID)
					return false
				}
			}

			return true
		},
		genDockerfileWithIssues(),
		genNonEmptyIgnoreRules(),
	))

	properties.TestingRun(t)
}

// genDockerfileWithIssues generates Dockerfiles that are likely to trigger lint rules.
// This ensures we have meaningful test cases where rules would fire if not ignored.
func genDockerfileWithIssues() gopter.Gen {
	return gen.IntRange(2, 6).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		return genIssueInstructionSlice(count).FlatMap(func(instrs interface{}) gopter.Gen {
			instructions := instrs.([]ast.Instruction)
			// Ensure first instruction is FROM (potentially with issues)
			if len(instructions) > 0 {
				if _, ok := instructions[0].(*ast.FromInstruction); !ok {
					// Use FROM without tag to trigger DL3006
					instructions[0] = &ast.FromInstruction{
						LineNum: 1,
						Image:   "ubuntu",
						Tag:     "", // Missing tag triggers DL3006
					}
				}
			}
			// Update line numbers
			for i := range instructions {
				setLineNum(instructions[i], i+1)
			}
			return gen.Const(&ast.Dockerfile{
				Instructions:  instructions,
				Stages:        []ast.Stage{},
				Comments:      []ast.Comment{},
				InlineIgnores: make(map[int][]string),
			})
		}, reflect.TypeOf(&ast.Dockerfile{}))
	}, reflect.TypeOf(&ast.Dockerfile{}))
}

// genIssueInstructionSlice generates a slice of instructions likely to trigger rules.
func genIssueInstructionSlice(n int) gopter.Gen {
	gens := make([]gopter.Gen, n)
	for i := 0; i < n; i++ {
		gens[i] = genIssueInstruction()
	}
	return gopter.CombineGens(gens...).Map(func(vals []interface{}) []ast.Instruction {
		result := make([]ast.Instruction, len(vals))
		for i, v := range vals {
			result[i] = v.(ast.Instruction)
		}
		return result
	})
}

// genIssueInstruction generates instructions that are likely to trigger lint rules.
func genIssueInstruction() gopter.Gen {
	return gen.OneGenOf(
		genFromWithIssues(),
		genRunWithIssues(),
		genEnvWithSecrets(),
		genArgWithSecrets(),
		genWorkdirRelative(),
		genAddWithIssues(),
		genCopyWithWildcard(),
	)
}

// genFromWithIssues generates FROM instructions that trigger rules.
func genFromWithIssues() gopter.Gen {
	return gen.IntRange(0, 3).Map(func(choice int) ast.Instruction {
		switch choice {
		case 0:
			// Missing tag (DL3006)
			return &ast.FromInstruction{LineNum: 1, Image: "ubuntu", Tag: ""}
		case 1:
			// Latest tag (DL3007)
			return &ast.FromInstruction{LineNum: 1, Image: "alpine", Tag: "latest"}
		case 2:
			// Large base image (DL3008)
			return &ast.FromInstruction{LineNum: 1, Image: "ubuntu", Tag: "22.04"}
		default:
			return &ast.FromInstruction{LineNum: 1, Image: "debian", Tag: "bullseye"}
		}
	})
}

// genRunWithIssues generates RUN instructions that trigger rules.
func genRunWithIssues() gopter.Gen {
	commands := []string{
		// Cache not cleaned (DL3009)
		"apt-get install -y curl",
		// Update without install (DL3012)
		"apt-get update",
		// Simple command (may trigger consecutive RUN if multiple)
		"echo hello",
	}
	return gen.IntRange(0, len(commands)-1).Map(func(idx int) ast.Instruction {
		return &ast.RunInstruction{
			LineNum: 1,
			Command: commands[idx],
			Shell:   true,
		}
	})
}

// genEnvWithSecrets generates ENV instructions with secret-like keys.
func genEnvWithSecrets() gopter.Gen {
	pairs := [][]string{
		// Secret patterns (DL4000)
		{"PASSWORD", "secret123"},
		{"API_KEY", "abc123"},
		{"SECRET_TOKEN", "xyz789"},
		{"PRIVATE_KEY", "key_data"},
		// Normal env vars
		{"NODE_ENV", "production"},
	}
	return gen.IntRange(0, len(pairs)-1).Map(func(idx int) ast.Instruction {
		return &ast.EnvInstruction{
			LineNum: 1,
			Key:     pairs[idx][0],
			Value:   pairs[idx][1],
		}
	})
}

// genArgWithSecrets generates ARG instructions with secret-like names.
func genArgWithSecrets() gopter.Gen {
	pairs := [][]string{
		// Secret patterns (DL4001)
		{"PASSWORD", "default_pass"},
		{"API_KEY", ""},
		{"SECRET", "mysecret"},
		// Normal args
		{"VERSION", "1.0"},
	}
	return gen.IntRange(0, len(pairs)-1).Map(func(idx int) ast.Instruction {
		return &ast.ArgInstruction{
			LineNum: 1,
			Name:    pairs[idx][0],
			Default: pairs[idx][1],
		}
	})
}

// genWorkdirRelative generates WORKDIR with relative paths.
func genWorkdirRelative() gopter.Gen {
	paths := []string{
		// Relative paths (DL3003)
		"app",
		"src/app",
		"./build",
		// Absolute paths (no issue)
		"/app",
		"/usr/src/app",
	}
	return gen.IntRange(0, len(paths)-1).Map(func(idx int) ast.Instruction {
		return &ast.WorkdirInstruction{
			LineNum: 1,
			Path:    paths[idx],
		}
	})
}

// genAddWithIssues generates ADD instructions that trigger rules.
func genAddWithIssues() gopter.Gen {
	pairs := [][]string{
		// ADD with URL (DL4003)
		{"https://example.com/file.tar.gz", "/app/"},
		// ADD where COPY would suffice (DL4004)
		{"package.json", "/app/"},
		{"src/", "/app/src/"},
	}
	return gen.IntRange(0, len(pairs)-1).Map(func(idx int) ast.Instruction {
		return &ast.AddInstruction{
			LineNum: 1,
			Sources: []string{pairs[idx][0]},
			Dest:    pairs[idx][1],
		}
	})
}

// genCopyWithWildcard generates COPY instructions with wildcards.
func genCopyWithWildcard() gopter.Gen {
	pairs := [][]string{
		// Wildcard patterns (DL5001)
		{"*.go", "/app/"},
		{"src/*", "/app/src/"},
		// Normal COPY
		{"package.json", "/app/"},
	}
	return gen.IntRange(0, len(pairs)-1).Map(func(idx int) ast.Instruction {
		return &ast.CopyInstruction{
			LineNum: 1,
			Sources: []string{pairs[idx][0]},
			Dest:    pairs[idx][1],
		}
	})
}

// genNonEmptyIgnoreRules generates a non-empty list of rule IDs to ignore.
func genNonEmptyIgnoreRules() gopter.Gen {
	allRules := []string{
		rules.RuleMissingTag,
		rules.RuleLatestTag,
		rules.RuleLargeBaseImage,
		rules.RuleCacheNotCleaned,
		rules.RuleConsecutiveRun,
		rules.RuleSuboptimalOrdering,
		rules.RuleUpdateWithoutInstall,
		rules.RuleMultipleCMD,
		rules.RuleMultipleEntrypoint,
		rules.RuleRelativeWorkdir,
		rules.RuleSecretInEnv,
		rules.RuleSecretInArg,
		rules.RuleNoUser,
		rules.RuleAddWithURL,
		rules.RuleAddOverCopy,
		rules.RuleMissingHealthcheck,
		rules.RuleWildcardCopy,
	}

	return gen.IntRange(1, 5).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		return gen.SliceOfN(count, gen.OneConstOf(
			allRules[0], allRules[1], allRules[2], allRules[3],
			allRules[4], allRules[5], allRules[6], allRules[7],
			allRules[8], allRules[9], allRules[10], allRules[11],
			allRules[12], allRules[13], allRules[14], allRules[15],
			allRules[16],
		)).Map(func(rules []string) []string {
			// Deduplicate
			seen := make(map[string]bool)
			result := make([]string, 0, len(rules))
			for _, r := range rules {
				if !seen[r] {
					seen[r] = true
					result = append(result, r)
				}
			}
			return result
		})
	}, reflect.TypeOf([]string{}))
}


// **Feature: docker-lint, Property 4: Inline Ignore Effectiveness**
// **Validates: Requirements 8.3**
//
// Property: For any Dockerfile line preceded by an inline ignore comment for rule R,
// the findings SHALL NOT contain a finding for rule R on that specific line.
func TestInlineIgnoreEffectiveness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("inline ignore comments suppress findings for specified rules on the following line", prop.ForAll(
		func(df *ast.Dockerfile, inlineIgnores map[int][]string) bool {
			// Apply the inline ignores to the Dockerfile
			df.InlineIgnores = inlineIgnores

			// Create analyzer with no global ignores
			config := Config{
				IgnoreRules: []string{},
			}
			analyzer := NewWithDefaults(config)

			// Run analysis
			findings := analyzer.Analyze(df)

			// Verify that no finding exists for a rule that is inline-ignored on that line
			for _, finding := range findings {
				ignoredRules, exists := inlineIgnores[finding.Line]
				if !exists {
					continue
				}

				// Check if this finding's rule is in the ignored list for this line
				for _, ignoredRule := range ignoredRules {
					if finding.RuleID == ignoredRule {
						t.Logf("Found finding for rule %s on line %d, but it should be ignored by inline comment",
							finding.RuleID, finding.Line)
						return false
					}
				}
			}

			return true
		},
		genDockerfileWithIssues(),
		genInlineIgnoresForDockerfile(),
	))

	properties.TestingRun(t)
}

// genInlineIgnoresForDockerfile generates a map of line numbers to rule IDs to ignore.
// The inline ignores are generated for lines that are likely to have findings.
func genInlineIgnoresForDockerfile() gopter.Gen {
	// Generate inline ignores for lines 1-10 (typical Dockerfile size in tests)
	return gen.IntRange(0, 5).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		if count == 0 {
			return gen.Const(make(map[int][]string))
		}

		// Generate line numbers and rule IDs
		return gen.SliceOfN(count, genLineIgnorePair()).Map(func(pairs []lineIgnorePair) map[int][]string {
			result := make(map[int][]string)
			for _, pair := range pairs {
				result[pair.line] = append(result[pair.line], pair.ruleIDs...)
			}
			return result
		})
	}, reflect.TypeOf(map[int][]string{}))
}

// lineIgnorePair represents a line number and the rule IDs to ignore on that line.
type lineIgnorePair struct {
	line    int
	ruleIDs []string
}

// genLineIgnorePair generates a line number and associated rule IDs to ignore.
func genLineIgnorePair() gopter.Gen {
	allRules := []string{
		rules.RuleMissingTag,
		rules.RuleLatestTag,
		rules.RuleLargeBaseImage,
		rules.RuleCacheNotCleaned,
		rules.RuleConsecutiveRun,
		rules.RuleSuboptimalOrdering,
		rules.RuleUpdateWithoutInstall,
		rules.RuleMultipleCMD,
		rules.RuleMultipleEntrypoint,
		rules.RuleRelativeWorkdir,
		rules.RuleSecretInEnv,
		rules.RuleSecretInArg,
		rules.RuleNoUser,
		rules.RuleAddWithURL,
		rules.RuleAddOverCopy,
		rules.RuleMissingHealthcheck,
		rules.RuleWildcardCopy,
	}

	return gen.IntRange(1, 10).FlatMap(func(lineVal interface{}) gopter.Gen {
		line := lineVal.(int)

		return gen.IntRange(1, 3).FlatMap(func(numVal interface{}) gopter.Gen {
			numRules := numVal.(int)

			return gen.SliceOfN(numRules, gen.OneConstOf(
				allRules[0], allRules[1], allRules[2], allRules[3],
				allRules[4], allRules[5], allRules[6], allRules[7],
				allRules[8], allRules[9], allRules[10], allRules[11],
				allRules[12], allRules[13], allRules[14], allRules[15],
				allRules[16],
			)).Map(func(ruleIDs []string) lineIgnorePair {
				// Deduplicate rule IDs
				seen := make(map[string]bool)
				uniqueRules := make([]string, 0, len(ruleIDs))
				for _, r := range ruleIDs {
					if !seen[r] {
						seen[r] = true
						uniqueRules = append(uniqueRules, r)
					}
				}
				return lineIgnorePair{
					line:    line,
					ruleIDs: uniqueRules,
				}
			})
		}, reflect.TypeOf(lineIgnorePair{}))
	}, reflect.TypeOf(lineIgnorePair{}))
}

// **Feature: docker-lint, Property 8: Multi-stage Stage Isolation**
// **Validates: Requirements 1.3**
//
// Property: For any multi-stage Dockerfile, rules that apply to stage-specific context
// (like missing USER) SHALL be evaluated per-stage, not globally.
// Specifically: if one stage has a USER instruction and another doesn't, we should get
// a finding only for the stage without USER, not for the entire Dockerfile.
func TestMultiStageStageIsolation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("stage-specific rules evaluate per-stage, not globally", prop.ForAll(
		func(df *ast.Dockerfile) bool {
			// Only test multi-stage Dockerfiles (2+ stages)
			if len(df.Stages) < 2 {
				return true // Skip single-stage Dockerfiles
			}

			// Create analyzer with only the NoUser rule to isolate the test
			config := Config{
				IgnoreRules: []string{},
			}
			analyzer := NewWithDefaults(config)

			// Run analysis with only the NoUser rule
			findings := analyzer.AnalyzeWithRules(df, []string{rules.RuleNoUser})

			// Count stages with and without USER instructions
			stagesWithUser := make(map[int]bool)
			stagesWithoutUser := make(map[int]bool)

			for i, stage := range df.Stages {
				hasUser := false
				for _, instr := range stage.Instructions {
					if _, ok := instr.(*ast.UserInstruction); ok {
						hasUser = true
						break
					}
				}
				if hasUser {
					stagesWithUser[i] = true
				} else {
					stagesWithoutUser[i] = true
				}
			}

			// Count NoUser findings
			noUserFindings := 0
			for _, finding := range findings {
				if finding.RuleID == rules.RuleNoUser {
					noUserFindings++
				}
			}

			// The number of NoUser findings should equal the number of stages without USER
			expectedFindings := len(stagesWithoutUser)

			if noUserFindings != expectedFindings {
				t.Logf("Expected %d NoUser findings (stages without USER), got %d", expectedFindings, noUserFindings)
				t.Logf("Stages with USER: %v, Stages without USER: %v", stagesWithUser, stagesWithoutUser)
				return false
			}

			return true
		},
		genMultiStageDockerfile(),
	))

	properties.TestingRun(t)
}

// genMultiStageDockerfile generates multi-stage Dockerfiles with varying USER instruction presence.
func genMultiStageDockerfile() gopter.Gen {
	return gen.IntRange(2, 4).FlatMap(func(numStagesVal interface{}) gopter.Gen {
		numStages := numStagesVal.(int)

		// Generate a slice of booleans indicating whether each stage has a USER instruction
		return gen.SliceOfN(numStages, gen.Bool()).FlatMap(func(hasUserSliceVal interface{}) gopter.Gen {
			hasUserSlice := hasUserSliceVal.([]bool)

			return gen.Const(buildMultiStageDockerfile(numStages, hasUserSlice))
		}, reflect.TypeOf(&ast.Dockerfile{}))
	}, reflect.TypeOf(&ast.Dockerfile{}))
}

// buildMultiStageDockerfile constructs a multi-stage Dockerfile AST with the specified
// number of stages and USER instruction presence per stage.
func buildMultiStageDockerfile(numStages int, hasUserPerStage []bool) *ast.Dockerfile {
	stages := make([]ast.Stage, numStages)
	allInstructions := make([]ast.Instruction, 0)
	lineNum := 1

	stageNames := []string{"builder", "runtime", "final", "test"}
	images := []string{"alpine", "golang", "node", "python"}

	for i := 0; i < numStages; i++ {
		stageName := ""
		if i < len(stageNames) {
			stageName = stageNames[i]
		}

		image := "alpine"
		if i < len(images) {
			image = images[i]
		}

		// Create FROM instruction for this stage
		fromInstr := &ast.FromInstruction{
			LineNum: lineNum,
			Image:   image,
			Tag:     "3.18",
			Alias:   stageName,
		}
		lineNum++

		stageInstructions := []ast.Instruction{fromInstr}
		allInstructions = append(allInstructions, fromInstr)

		// Add a RUN instruction
		runInstr := &ast.RunInstruction{
			LineNum: lineNum,
			Command: "echo 'stage " + stageName + "'",
			Shell:   true,
		}
		lineNum++
		stageInstructions = append(stageInstructions, runInstr)
		allInstructions = append(allInstructions, runInstr)

		// Add USER instruction if specified for this stage
		if i < len(hasUserPerStage) && hasUserPerStage[i] {
			userInstr := &ast.UserInstruction{
				LineNum: lineNum,
				User:    "appuser",
				Group:   "",
			}
			lineNum++
			stageInstructions = append(stageInstructions, userInstr)
			allInstructions = append(allInstructions, userInstr)
		}

		// Add a WORKDIR instruction
		workdirInstr := &ast.WorkdirInstruction{
			LineNum: lineNum,
			Path:    "/app",
		}
		lineNum++
		stageInstructions = append(stageInstructions, workdirInstr)
		allInstructions = append(allInstructions, workdirInstr)

		stages[i] = ast.Stage{
			Name:         stageName,
			FromInstr:    fromInstr,
			Instructions: stageInstructions,
			Index:        i,
		}
	}

	return &ast.Dockerfile{
		Stages:        stages,
		Instructions:  allInstructions,
		Comments:      []ast.Comment{},
		InlineIgnores: make(map[int][]string),
	}
}
