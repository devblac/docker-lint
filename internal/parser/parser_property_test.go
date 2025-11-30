// Package parser provides lexer, parser, and formatter for Dockerfile content.
package parser

import (
	"reflect"
	"testing"

	"github.com/docker-lint/docker-lint/internal/ast"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: docker-lint, Property 1: Parse-Format Round Trip**
// **Validates: Requirements 1.6**
//
// Property: For any valid Dockerfile AST, formatting the AST back to text
// and re-parsing it SHALL produce an equivalent AST structure (instruction
// types, order, and semantic content preserved).
func TestParseFormatRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("parse-format round trip preserves AST structure", prop.ForAll(
		func(df *ast.Dockerfile) bool {
			// Format the AST to text
			formatted := Format(df)

			// Parse the formatted text back to AST
			reparsed, err := ParseString(formatted)
			if err != nil {
				// If we can't parse the formatted output, the property fails
				return false
			}

			// Compare instruction counts
			if len(df.Instructions) != len(reparsed.Instructions) {
				return false
			}

			// Compare each instruction's type and semantic content
			for i := range df.Instructions {
				if !instructionsEquivalent(df.Instructions[i], reparsed.Instructions[i]) {
					return false
				}
			}

			return true
		},
		genDockerfile(),
	))

	properties.TestingRun(t)
}

// genDockerfile generates random valid Dockerfile ASTs for property testing.
func genDockerfile() gopter.Gen {
	return gen.IntRange(1, 5).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		return genInstructionSlice(count).Map(func(instrs []ast.Instruction) *ast.Dockerfile {
			// Ensure first instruction is FROM
			if len(instrs) > 0 {
				if _, ok := instrs[0].(*ast.FromInstruction); !ok {
					instrs[0] = &ast.FromInstruction{
						LineNum: 1,
						Image:   "alpine",
						Tag:     "3.18",
					}
				}
			}
			return &ast.Dockerfile{
				Instructions:  instrs,
				Stages:        []ast.Stage{},
				Comments:      []ast.Comment{},
				InlineIgnores: make(map[int][]string),
			}
		})
	}, reflect.TypeOf(&ast.Dockerfile{}))
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
		genEnvInstruction(),
		genArgInstruction(),
		genWorkdirInstruction(),
		genUserInstruction(),
		genExposeInstruction(),
		genCmdInstruction(),
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
		genEnvKey(),
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
	return genAbsolutePath().Map(func(path string) ast.Instruction {
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

// genExposeInstruction generates a random EXPOSE instruction.
func genExposeInstruction() gopter.Gen {
	return gen.IntRange(1, 3).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		return genPortSlice(count).Map(func(ports []string) ast.Instruction {
			return &ast.ExposeInstruction{
				LineNum: 1,
				Ports:   ports,
			}
		})
	}, reflect.TypeOf((*ast.Instruction)(nil)).Elem())
}

// genPortSlice generates a slice of ports of the given size.
func genPortSlice(n int) gopter.Gen {
	gens := make([]gopter.Gen, n)
	for i := 0; i < n; i++ {
		gens[i] = genPort()
	}
	return gopter.CombineGens(gens...).Map(func(vals []interface{}) []string {
		result := make([]string, len(vals))
		for i, v := range vals {
			result[i] = v.(string)
		}
		return result
	})
}

// genCmdInstruction generates a random CMD instruction.
func genCmdInstruction() gopter.Gen {
	return gen.Bool().FlatMap(func(shell interface{}) gopter.Gen {
		isShell := shell.(bool)
		if isShell {
			return genCommand().Map(func(cmd string) ast.Instruction {
				return &ast.CmdInstruction{
					LineNum: 1,
					Command: []string{cmd},
					Shell:   true,
				}
			})
		}
		return gen.IntRange(1, 3).FlatMap(func(n interface{}) gopter.Gen {
			count := n.(int)
			return genSimpleArgSlice(count).Map(func(args []string) ast.Instruction {
				return &ast.CmdInstruction{
					LineNum: 1,
					Command: args,
					Shell:   false,
				}
			})
		}, reflect.TypeOf((*ast.Instruction)(nil)).Elem())
	}, reflect.TypeOf((*ast.Instruction)(nil)).Elem())
}

// genSimpleArgSlice generates a slice of simple args of the given size.
func genSimpleArgSlice(n int) gopter.Gen {
	gens := make([]gopter.Gen, n)
	for i := 0; i < n; i++ {
		gens[i] = genSimpleArg()
	}
	return gopter.CombineGens(gens...).Map(func(vals []interface{}) []string {
		result := make([]string, len(vals))
		for i, v := range vals {
			result[i] = v.(string)
		}
		return result
	})
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
		"apk add curl",
		"npm install",
		"go build",
		"pip install -r requirements.txt",
	)
}

func genPath() gopter.Gen {
	return gen.OneConstOf(".", "src", "app", "/app", "/usr/src/app", "package.json")
}

func genAbsolutePath() gopter.Gen {
	return gen.OneConstOf("/app", "/usr/src/app", "/home/user", "/var/www", "/opt/app")
}

func genEnvKey() gopter.Gen {
	return gen.OneConstOf("PATH", "HOME", "NODE_ENV", "GO_VERSION", "APP_PORT")
}

func genEnvValue() gopter.Gen {
	return gen.OneConstOf("/usr/local/bin", "/home/user", "production", "1.21", "8080")
}

func genUsername() gopter.Gen {
	return gen.OneConstOf("nobody", "root", "app", "node", "www-data")
}

func genPort() gopter.Gen {
	return gen.OneConstOf("80", "443", "8080", "3000", "5432", "6379")
}

func genSimpleArg() gopter.Gen {
	return gen.OneConstOf("echo", "hello", "world", "app", "start", "run")
}

// instructionsEquivalent compares two instructions for semantic equivalence.
func instructionsEquivalent(a, b ast.Instruction) bool {
	if a.Type() != b.Type() {
		return false
	}

	switch ai := a.(type) {
	case *ast.FromInstruction:
		bi := b.(*ast.FromInstruction)
		return ai.Image == bi.Image && ai.Tag == bi.Tag && ai.Alias == bi.Alias && ai.Digest == bi.Digest && ai.Platform == bi.Platform

	case *ast.RunInstruction:
		bi := b.(*ast.RunInstruction)
		return ai.Command == bi.Command && ai.Shell == bi.Shell

	case *ast.CopyInstruction:
		bi := b.(*ast.CopyInstruction)
		return reflect.DeepEqual(ai.Sources, bi.Sources) && ai.Dest == bi.Dest && ai.From == bi.From && ai.Chown == bi.Chown

	case *ast.AddInstruction:
		bi := b.(*ast.AddInstruction)
		return reflect.DeepEqual(ai.Sources, bi.Sources) && ai.Dest == bi.Dest && ai.Chown == bi.Chown

	case *ast.EnvInstruction:
		bi := b.(*ast.EnvInstruction)
		return ai.Key == bi.Key && ai.Value == bi.Value

	case *ast.ArgInstruction:
		bi := b.(*ast.ArgInstruction)
		return ai.Name == bi.Name && ai.Default == bi.Default

	case *ast.WorkdirInstruction:
		bi := b.(*ast.WorkdirInstruction)
		return ai.Path == bi.Path

	case *ast.UserInstruction:
		bi := b.(*ast.UserInstruction)
		return ai.User == bi.User && ai.Group == bi.Group

	case *ast.ExposeInstruction:
		bi := b.(*ast.ExposeInstruction)
		return reflect.DeepEqual(ai.Ports, bi.Ports)

	case *ast.CmdInstruction:
		bi := b.(*ast.CmdInstruction)
		return reflect.DeepEqual(ai.Command, bi.Command) && ai.Shell == bi.Shell

	case *ast.EntrypointInstruction:
		bi := b.(*ast.EntrypointInstruction)
		return reflect.DeepEqual(ai.Command, bi.Command) && ai.Shell == bi.Shell

	case *ast.LabelInstruction:
		bi := b.(*ast.LabelInstruction)
		return reflect.DeepEqual(ai.Labels, bi.Labels)

	case *ast.VolumeInstruction:
		bi := b.(*ast.VolumeInstruction)
		return reflect.DeepEqual(ai.Paths, bi.Paths)

	case *ast.HealthcheckInstruction:
		bi := b.(*ast.HealthcheckInstruction)
		return ai.None == bi.None && ai.Interval == bi.Interval && ai.Timeout == bi.Timeout &&
			ai.Retries == bi.Retries && ai.Start == bi.Start && reflect.DeepEqual(ai.Command, bi.Command)

	case *ast.ShellInstruction:
		bi := b.(*ast.ShellInstruction)
		return reflect.DeepEqual(ai.Shell, bi.Shell)

	case *ast.StopsignalInstruction:
		bi := b.(*ast.StopsignalInstruction)
		return ai.Signal == bi.Signal

	case *ast.OnbuildInstruction:
		bi := b.(*ast.OnbuildInstruction)
		if ai.Instruction == nil && bi.Instruction == nil {
			return true
		}
		if ai.Instruction == nil || bi.Instruction == nil {
			return false
		}
		return instructionsEquivalent(ai.Instruction, bi.Instruction)

	default:
		return false
	}
}
