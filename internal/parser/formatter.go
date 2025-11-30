// Package parser provides lexer, parser, and formatter for Dockerfile content.
package parser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/docker-lint/docker-lint/internal/ast"
)

// Format converts a Dockerfile AST back to text representation.
// The output preserves instruction semantics for round-trip testing.
func Format(df *ast.Dockerfile) string {
	if df == nil {
		return ""
	}

	var sb strings.Builder

	for i, instr := range df.Instructions {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(formatInstruction(instr))
	}

	return sb.String()
}

// formatInstruction formats a single instruction to its text representation.
func formatInstruction(instr ast.Instruction) string {
	switch i := instr.(type) {
	case *ast.FromInstruction:
		return formatFrom(i)
	case *ast.RunInstruction:
		return formatRun(i)
	case *ast.CopyInstruction:
		return formatCopy(i)
	case *ast.AddInstruction:
		return formatAdd(i)
	case *ast.EnvInstruction:
		return formatEnv(i)
	case *ast.ArgInstruction:
		return formatArg(i)
	case *ast.ExposeInstruction:
		return formatExpose(i)
	case *ast.WorkdirInstruction:
		return formatWorkdir(i)
	case *ast.UserInstruction:
		return formatUser(i)
	case *ast.LabelInstruction:
		return formatLabel(i)
	case *ast.VolumeInstruction:
		return formatVolume(i)
	case *ast.CmdInstruction:
		return formatCmd(i)
	case *ast.EntrypointInstruction:
		return formatEntrypoint(i)
	case *ast.HealthcheckInstruction:
		return formatHealthcheck(i)
	case *ast.ShellInstruction:
		return formatShell(i)
	case *ast.StopsignalInstruction:
		return formatStopsignal(i)
	case *ast.OnbuildInstruction:
		return formatOnbuild(i)
	default:
		return ""
	}
}


// formatFrom formats a FROM instruction.
func formatFrom(f *ast.FromInstruction) string {
	var parts []string
	parts = append(parts, "FROM")

	if f.Platform != "" {
		parts = append(parts, fmt.Sprintf("--platform=%s", f.Platform))
	}

	// Build image reference
	imageRef := f.Image
	if f.Tag != "" {
		imageRef += ":" + f.Tag
	}
	if f.Digest != "" {
		imageRef += "@" + f.Digest
	}
	parts = append(parts, imageRef)

	if f.Alias != "" {
		parts = append(parts, "AS", f.Alias)
	}

	return strings.Join(parts, " ")
}

// formatRun formats a RUN instruction.
func formatRun(r *ast.RunInstruction) string {
	if r.Command == "" {
		return "RUN"
	}
	return "RUN " + r.Command
}

// formatCopy formats a COPY instruction.
func formatCopy(c *ast.CopyInstruction) string {
	var parts []string
	parts = append(parts, "COPY")

	if c.From != "" {
		parts = append(parts, fmt.Sprintf("--from=%s", c.From))
	}
	if c.Chown != "" {
		parts = append(parts, fmt.Sprintf("--chown=%s", c.Chown))
	}

	parts = append(parts, c.Sources...)
	parts = append(parts, c.Dest)

	return strings.Join(parts, " ")
}

// formatAdd formats an ADD instruction.
func formatAdd(a *ast.AddInstruction) string {
	var parts []string
	parts = append(parts, "ADD")

	if a.Chown != "" {
		parts = append(parts, fmt.Sprintf("--chown=%s", a.Chown))
	}

	parts = append(parts, a.Sources...)
	parts = append(parts, a.Dest)

	return strings.Join(parts, " ")
}

// formatEnv formats an ENV instruction.
func formatEnv(e *ast.EnvInstruction) string {
	if e.Key == "" {
		return "ENV"
	}
	return fmt.Sprintf("ENV %s=%s", e.Key, e.Value)
}

// formatArg formats an ARG instruction.
func formatArg(a *ast.ArgInstruction) string {
	if a.Name == "" {
		return "ARG"
	}
	if a.Default != "" {
		return fmt.Sprintf("ARG %s=%s", a.Name, a.Default)
	}
	return "ARG " + a.Name
}

// formatExpose formats an EXPOSE instruction.
func formatExpose(e *ast.ExposeInstruction) string {
	if len(e.Ports) == 0 {
		return "EXPOSE"
	}
	return "EXPOSE " + strings.Join(e.Ports, " ")
}

// formatWorkdir formats a WORKDIR instruction.
func formatWorkdir(w *ast.WorkdirInstruction) string {
	if w.Path == "" {
		return "WORKDIR"
	}
	return "WORKDIR " + w.Path
}

// formatUser formats a USER instruction.
func formatUser(u *ast.UserInstruction) string {
	if u.User == "" {
		return "USER"
	}
	if u.Group != "" {
		return fmt.Sprintf("USER %s:%s", u.User, u.Group)
	}
	return "USER " + u.User
}


// formatLabel formats a LABEL instruction.
func formatLabel(l *ast.LabelInstruction) string {
	if len(l.Labels) == 0 {
		return "LABEL"
	}

	var parts []string
	parts = append(parts, "LABEL")

	// Sort keys for deterministic output
	keys := make([]string, 0, len(l.Labels))
	for k := range l.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := l.Labels[k]
		// Quote value if it contains spaces
		if strings.Contains(v, " ") {
			parts = append(parts, fmt.Sprintf("%s=\"%s\"", k, v))
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return strings.Join(parts, " ")
}

// formatVolume formats a VOLUME instruction.
func formatVolume(v *ast.VolumeInstruction) string {
	if len(v.Paths) == 0 {
		return "VOLUME"
	}
	// Use JSON format for consistency
	jsonBytes, err := json.Marshal(v.Paths)
	if err != nil {
		return "VOLUME " + strings.Join(v.Paths, " ")
	}
	return "VOLUME " + string(jsonBytes)
}

// formatCmd formats a CMD instruction.
func formatCmd(c *ast.CmdInstruction) string {
	if len(c.Command) == 0 {
		return "CMD"
	}
	if c.Shell {
		return "CMD " + strings.Join(c.Command, " ")
	}
	// Exec form
	jsonBytes, err := json.Marshal(c.Command)
	if err != nil {
		return "CMD " + strings.Join(c.Command, " ")
	}
	return "CMD " + string(jsonBytes)
}

// formatEntrypoint formats an ENTRYPOINT instruction.
func formatEntrypoint(e *ast.EntrypointInstruction) string {
	if len(e.Command) == 0 {
		return "ENTRYPOINT"
	}
	if e.Shell {
		return "ENTRYPOINT " + strings.Join(e.Command, " ")
	}
	// Exec form
	jsonBytes, err := json.Marshal(e.Command)
	if err != nil {
		return "ENTRYPOINT " + strings.Join(e.Command, " ")
	}
	return "ENTRYPOINT " + string(jsonBytes)
}

// formatHealthcheck formats a HEALTHCHECK instruction.
func formatHealthcheck(h *ast.HealthcheckInstruction) string {
	if h.None {
		return "HEALTHCHECK NONE"
	}

	var parts []string
	parts = append(parts, "HEALTHCHECK")

	if h.Interval != "" {
		parts = append(parts, fmt.Sprintf("--interval=%s", h.Interval))
	}
	if h.Timeout != "" {
		parts = append(parts, fmt.Sprintf("--timeout=%s", h.Timeout))
	}
	if h.Retries != "" {
		parts = append(parts, fmt.Sprintf("--retries=%s", h.Retries))
	}
	if h.Start != "" {
		parts = append(parts, fmt.Sprintf("--start-period=%s", h.Start))
	}

	if len(h.Command) > 0 {
		parts = append(parts, "CMD")
		// Use exec form if multiple command parts
		if len(h.Command) > 1 {
			jsonBytes, err := json.Marshal(h.Command)
			if err == nil {
				parts = append(parts, string(jsonBytes))
			} else {
				parts = append(parts, strings.Join(h.Command, " "))
			}
		} else {
			parts = append(parts, h.Command[0])
		}
	}

	return strings.Join(parts, " ")
}

// formatShell formats a SHELL instruction.
func formatShell(s *ast.ShellInstruction) string {
	if len(s.Shell) == 0 {
		return "SHELL"
	}
	jsonBytes, err := json.Marshal(s.Shell)
	if err != nil {
		return "SHELL " + strings.Join(s.Shell, " ")
	}
	return "SHELL " + string(jsonBytes)
}

// formatStopsignal formats a STOPSIGNAL instruction.
func formatStopsignal(s *ast.StopsignalInstruction) string {
	if s.Signal == "" {
		return "STOPSIGNAL"
	}
	return "STOPSIGNAL " + s.Signal
}

// formatOnbuild formats an ONBUILD instruction.
func formatOnbuild(o *ast.OnbuildInstruction) string {
	if o.Instruction == nil {
		return "ONBUILD"
	}
	return "ONBUILD " + formatInstruction(o.Instruction)
}
