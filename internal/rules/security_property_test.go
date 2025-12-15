// Package rules provides lint rule implementations for docker-lint.
package rules

import (
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: docker-lint, Property 5: Secret Value Non-Exposure**
// **Validates: Requirements 4.1, 4.2**
//
// Property: For any Dockerfile containing ENV or ARG instructions with secret-pattern keys,
// the findings and all output SHALL NOT contain the actual values of those instructions.
func TestSecretValueNonExposure(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	envRule := &SecretInEnvRule{}
	argRule := &SecretInArgRule{}

	// Property: ENV with secret key produces finding that does NOT contain the secret value
	properties.Property("ENV with secret key finding does not expose secret value", prop.ForAll(
		func(secretKey string, secretValue string) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.EnvInstruction{
						LineNum: 1,
						Key:     secretKey,
						Value:   secretValue,
					},
				},
			}

			findings := envRule.Check(dockerfile)

			// Should have exactly one finding
			if len(findings) != 1 {
				return false
			}

			// The finding message and suggestion must NOT contain the secret value
			if strings.Contains(findings[0].Message, secretValue) {
				return false
			}
			if strings.Contains(findings[0].Suggestion, secretValue) {
				return false
			}

			// The finding should contain the key name (for context) but not the value
			if !strings.Contains(findings[0].Message, secretKey) {
				return false
			}

			return true
		},
		genSecretKeyName(),
		genSecretValue(),
	))

	// Property: ARG with secret key produces finding that does NOT contain the secret value
	properties.Property("ARG with secret key finding does not expose secret value", prop.ForAll(
		func(secretKey string, secretValue string) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.ArgInstruction{
						LineNum: 1,
						Name:    secretKey,
						Default: secretValue,
					},
				},
			}

			findings := argRule.Check(dockerfile)

			// Should have exactly one finding
			if len(findings) != 1 {
				return false
			}

			// The finding message and suggestion must NOT contain the secret value
			if strings.Contains(findings[0].Message, secretValue) {
				return false
			}
			if strings.Contains(findings[0].Suggestion, secretValue) {
				return false
			}

			// The finding should contain the key name (for context) but not the value
			if !strings.Contains(findings[0].Message, secretKey) {
				return false
			}

			return true
		},
		genSecretKeyName(),
		genSecretValue(),
	))

	// Property: Multiple ENV/ARG with secrets - none of the values are exposed
	properties.Property("multiple secret instructions do not expose any values", prop.ForAll(
		func(secrets []secretPair) bool {
			instructions := make([]ast.Instruction, 0, len(secrets)*2)
			for i, s := range secrets {
				instructions = append(instructions, &ast.EnvInstruction{
					LineNum: i*2 + 1,
					Key:     s.key,
					Value:   s.value,
				})
				instructions = append(instructions, &ast.ArgInstruction{
					LineNum: i*2 + 2,
					Name:    s.key,
					Default: s.value,
				})
			}

			dockerfile := &ast.Dockerfile{
				Instructions: instructions,
			}

			envFindings := envRule.Check(dockerfile)
			argFindings := argRule.Check(dockerfile)

			// Check all ENV findings
			for _, f := range envFindings {
				for _, s := range secrets {
					if strings.Contains(f.Message, s.value) || strings.Contains(f.Suggestion, s.value) {
						return false
					}
				}
			}

			// Check all ARG findings
			for _, f := range argFindings {
				for _, s := range secrets {
					if strings.Contains(f.Message, s.value) || strings.Contains(f.Suggestion, s.value) {
						return false
					}
				}
			}

			return true
		},
		genSecretPairs(1, 5),
	))

	// Property: Non-secret keys do not trigger findings
	properties.Property("non-secret keys do not trigger findings", prop.ForAll(
		func(nonSecretKey string, value string) bool {
			dockerfile := &ast.Dockerfile{
				Instructions: []ast.Instruction{
					&ast.EnvInstruction{
						LineNum: 1,
						Key:     nonSecretKey,
						Value:   value,
					},
					&ast.ArgInstruction{
						LineNum: 2,
						Name:    nonSecretKey,
						Default: value,
					},
				},
			}

			envFindings := envRule.Check(dockerfile)
			argFindings := argRule.Check(dockerfile)

			// Should have no findings for non-secret keys
			return len(envFindings) == 0 && len(argFindings) == 0
		},
		genNonSecretKeyName(),
		genSecretValue(),
	))

	properties.TestingRun(t)
}

// secretPair holds a key-value pair for testing
type secretPair struct {
	key   string
	value string
}

// Generator helpers

// genSecretKeyName generates key names that match secret patterns
func genSecretKeyName() gopter.Gen {
	return gen.OneConstOf(
		"PASSWORD",
		"DB_PASSWORD",
		"SECRET",
		"APP_SECRET",
		"TOKEN",
		"AUTH_TOKEN",
		"API_KEY",
		"APIKEY",
		"PRIVATE_KEY",
		"PRIVATEKEY",
		"ACCESS_KEY",
		"ACCESSKEY",
		"CREDENTIALS",
		"SSH_KEY",
		"ENCRYPTION_KEY",
		"MY_PASSWORD",
		"SECRET_KEY",
		"api_key",
		"private_key",
	)
}

// genSecretValue generates realistic secret values that should never be exposed
func genSecretValue() gopter.Gen {
	return gen.OneConstOf(
		"super_secret_password_123",
		"sk-1234567890abcdef",
		"ghp_xxxxxxxxxxxxxxxxxxxx",
		"AKIAIOSFODNN7EXAMPLE",
		"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"-----BEGIN RSA PRIVATE KEY-----",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"xoxb-1234-5678-abcdef",
		"SG.xxxxxxxxxxxxxxxxxxxxxx",
		"p@ssw0rd!",
		"my-super-secret-value",
		"db_password_prod_2024",
	)
}

// genNonSecretKeyName generates key names that do NOT match secret patterns
func genNonSecretKeyName() gopter.Gen {
	return gen.OneConstOf(
		"APP_NAME",
		"PORT",
		"HOST",
		"DEBUG",
		"LOG_LEVEL",
		"NODE_ENV",
		"DATABASE_URL",
		"REDIS_HOST",
		"CACHE_TTL",
		"MAX_CONNECTIONS",
		"TIMEOUT",
		"VERSION",
	)
}

// genSecretPairs generates a slice of secret key-value pairs
func genSecretPairs(minSize, maxSize int) gopter.Gen {
	return gen.IntRange(minSize, maxSize).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		return gen.SliceOfN(count, genSecretPair())
	}, nil)
}

// genSecretPair generates a single secret key-value pair
func genSecretPair() gopter.Gen {
	return gopter.CombineGens(
		genSecretKeyName(),
		genSecretValue(),
	).Map(func(vals []interface{}) secretPair {
		return secretPair{
			key:   vals[0].(string),
			value: vals[1].(string),
		}
	})
}
