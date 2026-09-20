package parser

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/devblac/docker-lint/internal/ast"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/parser"
)

// convertNode parses a single BuildKit node and converts it to AST instructions.
//
// Most instructions map one-to-one, but ENV and ARG may declare several
// variables in a single instruction; each one becomes its own AST node so rules
// see every key, all sharing the source line of the instruction they came from.
func convertNode(node *dockerfile.Node) ([]ast.Instruction, error) {
	cmd, err := instructions.ParseInstruction(node)
	if err != nil {
		return nil, err
	}
	return convertCommand(cmd, node.StartLine, rawText(node))
}

// rawText returns the original source text of an instruction.
func rawText(node *dockerfile.Node) string {
	return strings.TrimSpace(node.Original)
}

// convertCommand converts a parsed BuildKit command into AST instructions,
// reported at the given line with the given raw source text.
func convertCommand(cmd interface{}, line int, raw string) ([]ast.Instruction, error) {
	switch c := cmd.(type) {
	case *instructions.Stage:
		image, tag, digest := splitImageRef(c.BaseName)
		return one(&ast.FromInstruction{
			LineNum:  line,
			RawText:  raw,
			Image:    image,
			Tag:      tag,
			Digest:   digest,
			Alias:    c.Name,
			Platform: c.Platform,
		}), nil

	case *instructions.RunCommand:
		return one(&ast.RunInstruction{
			LineNum: line,
			RawText: raw,
			Command: shellCommandText(c.ShellDependantCmdLine),
			Shell:   c.PrependShell,
		}), nil

	case *instructions.CopyCommand:
		return one(&ast.CopyInstruction{
			LineNum: line,
			RawText: raw,
			Sources: c.SourcePaths,
			Dest:    c.DestPath,
			From:    c.From,
			Chown:   c.Chown,
		}), nil

	case *instructions.AddCommand:
		return one(&ast.AddInstruction{
			LineNum: line,
			RawText: raw,
			Sources: c.SourcePaths,
			Dest:    c.DestPath,
			Chown:   c.Chown,
		}), nil

	case *instructions.EnvCommand:
		instrs := make([]ast.Instruction, 0, len(c.Env))
		for _, kvp := range c.Env {
			instrs = append(instrs, &ast.EnvInstruction{
				LineNum: line,
				RawText: raw,
				Key:     kvp.Key,
				Value:   kvp.Value,
			})
		}
		return instrs, nil

	case *instructions.ArgCommand:
		instrs := make([]ast.Instruction, 0, len(c.Args))
		for _, arg := range c.Args {
			instrs = append(instrs, &ast.ArgInstruction{
				LineNum: line,
				RawText: raw,
				Name:    arg.Key,
				Default: arg.ValueString(),
			})
		}
		return instrs, nil

	case *instructions.LabelCommand:
		labels := make(map[string]string, len(c.Labels))
		for _, kvp := range c.Labels {
			labels[kvp.Key] = kvp.Value
		}
		return one(&ast.LabelInstruction{LineNum: line, RawText: raw, Labels: labels}), nil

	case *instructions.MaintainerCommand:
		// MAINTAINER is deprecated in favour of a label.
		return one(&ast.LabelInstruction{
			LineNum: line,
			RawText: raw,
			Labels:  map[string]string{"maintainer": c.Maintainer},
		}), nil

	case *instructions.ExposeCommand:
		return one(&ast.ExposeInstruction{LineNum: line, RawText: raw, Ports: c.Ports}), nil

	case *instructions.WorkdirCommand:
		return one(&ast.WorkdirInstruction{LineNum: line, RawText: raw, Path: c.Path}), nil

	case *instructions.UserCommand:
		user, group := splitUser(c.User)
		return one(&ast.UserInstruction{LineNum: line, RawText: raw, User: user, Group: group}), nil

	case *instructions.VolumeCommand:
		return one(&ast.VolumeInstruction{LineNum: line, RawText: raw, Paths: c.Volumes}), nil

	case *instructions.CmdCommand:
		return one(&ast.CmdInstruction{
			LineNum: line,
			RawText: raw,
			Command: c.CmdLine,
			Shell:   c.PrependShell,
		}), nil

	case *instructions.EntrypointCommand:
		return one(&ast.EntrypointInstruction{
			LineNum: line,
			RawText: raw,
			Command: c.CmdLine,
			Shell:   c.PrependShell,
		}), nil

	case *instructions.HealthCheckCommand:
		return one(convertHealthcheck(c, line, raw)), nil

	case *instructions.ShellCommand:
		return one(&ast.ShellInstruction{LineNum: line, RawText: raw, Shell: c.Shell}), nil

	case *instructions.StopSignalCommand:
		return one(&ast.StopsignalInstruction{LineNum: line, RawText: raw, Signal: c.Signal}), nil

	case *instructions.OnbuildCommand:
		inner, err := parseOnbuildExpression(c.Expression, line)
		if err != nil {
			return nil, err
		}
		return one(&ast.OnbuildInstruction{LineNum: line, RawText: raw, Instruction: inner}), nil
	}

	// Unknown instructions are rejected by BuildKit before reaching here, so an
	// unhandled type means a command this linter has no rules for yet.
	return nil, nil
}

// convertHealthcheck converts a HEALTHCHECK command, whose Test slice is
// prefixed with the check kind ("NONE", "CMD" or "CMD-SHELL").
func convertHealthcheck(c *instructions.HealthCheckCommand, line int, raw string) ast.Instruction {
	instr := &ast.HealthcheckInstruction{LineNum: line, RawText: raw}
	if c.Health == nil {
		return instr
	}

	instr.Interval = durationString(c.Health.Interval)
	instr.Timeout = durationString(c.Health.Timeout)
	instr.Start = durationString(c.Health.StartPeriod)
	if c.Health.Retries > 0 {
		instr.Retries = strconv.Itoa(c.Health.Retries)
	}

	if len(c.Health.Test) > 0 {
		if strings.EqualFold(c.Health.Test[0], "NONE") {
			instr.None = true
		} else {
			instr.Command = c.Health.Test[1:]
		}
	}

	return instr
}

// parseOnbuildExpression parses the instruction wrapped by ONBUILD, reporting
// it at the line of the ONBUILD itself.
func parseOnbuildExpression(expression string, line int) (ast.Instruction, error) {
	result, err := dockerfile.Parse(strings.NewReader(expression))
	if err != nil {
		return nil, err
	}
	if result.AST == nil || len(result.AST.Children) == 0 {
		return nil, errors.New("ONBUILD requires an instruction")
	}

	node := result.AST.Children[0]
	cmd, err := instructions.ParseInstruction(node)
	if err != nil {
		return nil, err
	}

	instrs, err := convertCommand(cmd, line, rawText(node))
	if err != nil {
		return nil, err
	}
	if len(instrs) == 0 {
		return nil, errors.New("ONBUILD requires a supported instruction")
	}

	return instrs[0], nil
}

// shellCommandText renders a shell-dependant command line as a single string,
// appending any heredoc bodies so rules can inspect their contents.
func shellCommandText(cmdLine instructions.ShellDependantCmdLine) string {
	text := strings.Join(cmdLine.CmdLine, " ")
	for _, file := range cmdLine.Files {
		if file.Data == "" {
			continue
		}
		text += "\n" + file.Data
	}
	return text
}

// splitImageRef splits an image reference into name, tag and digest.
// A colon separates the tag only when it appears after the last slash;
// otherwise it is a registry port, as in "localhost:5000/app".
func splitImageRef(ref string) (image, tag, digest string) {
	image = ref

	if at := strings.LastIndex(image, "@"); at != -1 {
		digest = image[at+1:]
		image = image[:at]
	}

	if colon := strings.LastIndex(image, ":"); colon > strings.LastIndex(image, "/") {
		tag = image[colon+1:]
		image = image[:colon]
	}

	return image, tag, digest
}

// splitUser splits a USER argument into its user and group parts.
func splitUser(value string) (user, group string) {
	if colon := strings.Index(value, ":"); colon != -1 {
		return value[:colon], value[colon+1:]
	}
	return value, ""
}

// durationString renders a healthcheck duration, empty when unset.
func durationString(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}

// one wraps a single instruction in a slice.
func one(instr ast.Instruction) []ast.Instruction {
	return []ast.Instruction{instr}
}
