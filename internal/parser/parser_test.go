// Package parser provides lexer, parser, and formatter for Dockerfile content.
package parser

import (
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

// TestParseAllInstructionTypes tests parsing of valid Dockerfiles with all instruction types.
// _Requirements: 1.1_
func TestParseAllInstructionTypes(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType ast.InstructionType
		validate     func(t *testing.T, instr ast.Instruction)
	}{
		{
			name:         "FROM simple",
			input:        "FROM alpine",
			expectedType: ast.InstrFROM,
			validate: func(t *testing.T, instr ast.Instruction) {
				f := instr.(*ast.FromInstruction)
				if f.Image != "alpine" {
					t.Errorf("Image = %q, want %q", f.Image, "alpine")
				}
			},
		},
		{
			name:         "FROM with tag",
			input:        "FROM alpine:3.18",
			expectedType: ast.InstrFROM,
			validate: func(t *testing.T, instr ast.Instruction) {
				f := instr.(*ast.FromInstruction)
				if f.Image != "alpine" || f.Tag != "3.18" {
					t.Errorf("Image:Tag = %q:%q, want alpine:3.18", f.Image, f.Tag)
				}
			},
		},
		{
			name:         "FROM with digest",
			input:        "FROM alpine@sha256:abc123",
			expectedType: ast.InstrFROM,
			validate: func(t *testing.T, instr ast.Instruction) {
				f := instr.(*ast.FromInstruction)
				if f.Image != "alpine" || f.Digest != "sha256:abc123" {
					t.Errorf("Image@Digest = %q@%q, want alpine@sha256:abc123", f.Image, f.Digest)
				}
			},
		},
		{
			name:         "FROM with alias",
			input:        "FROM alpine:3.18 AS builder",
			expectedType: ast.InstrFROM,
			validate: func(t *testing.T, instr ast.Instruction) {
				f := instr.(*ast.FromInstruction)
				if f.Alias != "builder" {
					t.Errorf("Alias = %q, want %q", f.Alias, "builder")
				}
			},
		},
		{
			name:         "FROM with platform",
			input:        "FROM --platform=linux/amd64 alpine:3.18",
			expectedType: ast.InstrFROM,
			validate: func(t *testing.T, instr ast.Instruction) {
				f := instr.(*ast.FromInstruction)
				if f.Platform != "linux/amd64" {
					t.Errorf("Platform = %q, want %q", f.Platform, "linux/amd64")
				}
			},
		},
		{
			name:         "RUN shell form",
			input:        "FROM alpine\nRUN apt-get update && apt-get install -y curl",
			expectedType: ast.InstrRUN,
			validate: func(t *testing.T, instr ast.Instruction) {
				r := instr.(*ast.RunInstruction)
				if !r.Shell {
					t.Error("Shell = false, want true")
				}
				if !strings.Contains(r.Command, "apt-get") {
					t.Errorf("Command = %q, want to contain apt-get", r.Command)
				}
			},
		},
		{
			name:         "COPY simple",
			input:        "FROM alpine\nCOPY . /app",
			expectedType: ast.InstrCOPY,
			validate: func(t *testing.T, instr ast.Instruction) {
				c := instr.(*ast.CopyInstruction)
				if len(c.Sources) != 1 || c.Sources[0] != "." {
					t.Errorf("Sources = %v, want [.]", c.Sources)
				}
				if c.Dest != "/app" {
					t.Errorf("Dest = %q, want /app", c.Dest)
				}
			},
		},
		{
			name:         "COPY with --from",
			input:        "FROM alpine\nCOPY --from=builder /build/app /app",
			expectedType: ast.InstrCOPY,
			validate: func(t *testing.T, instr ast.Instruction) {
				c := instr.(*ast.CopyInstruction)
				if c.From != "builder" {
					t.Errorf("From = %q, want builder", c.From)
				}
			},
		},
		{
			name:         "COPY with --chown",
			input:        "FROM alpine\nCOPY --chown=user:group . /app",
			expectedType: ast.InstrCOPY,
			validate: func(t *testing.T, instr ast.Instruction) {
				c := instr.(*ast.CopyInstruction)
				if c.Chown != "user:group" {
					t.Errorf("Chown = %q, want user:group", c.Chown)
				}
			},
		},
		{
			name:         "ADD simple",
			input:        "FROM alpine\nADD src /app",
			expectedType: ast.InstrADD,
			validate: func(t *testing.T, instr ast.Instruction) {
				a := instr.(*ast.AddInstruction)
				if len(a.Sources) != 1 || a.Sources[0] != "src" {
					t.Errorf("Sources = %v, want [src]", a.Sources)
				}
			},
		},
		{
			name:         "ENV key=value",
			input:        "FROM alpine\nENV NODE_ENV=production",
			expectedType: ast.InstrENV,
			validate: func(t *testing.T, instr ast.Instruction) {
				e := instr.(*ast.EnvInstruction)
				if e.Key != "NODE_ENV" || e.Value != "production" {
					t.Errorf("Key=Value = %q=%q, want NODE_ENV=production", e.Key, e.Value)
				}
			},
		},
		{
			name:         "ENV key value (old format)",
			input:        "FROM alpine\nENV NODE_ENV production",
			expectedType: ast.InstrENV,
			validate: func(t *testing.T, instr ast.Instruction) {
				e := instr.(*ast.EnvInstruction)
				if e.Key != "NODE_ENV" || e.Value != "production" {
					t.Errorf("Key=Value = %q=%q, want NODE_ENV=production", e.Key, e.Value)
				}
			},
		},
		{
			name:         "ARG without default",
			input:        "FROM alpine\nARG VERSION",
			expectedType: ast.InstrARG,
			validate: func(t *testing.T, instr ast.Instruction) {
				a := instr.(*ast.ArgInstruction)
				if a.Name != "VERSION" {
					t.Errorf("Name = %q, want VERSION", a.Name)
				}
				if a.Default != "" {
					t.Errorf("Default = %q, want empty", a.Default)
				}
			},
		},
		{
			name:         "ARG with default",
			input:        "FROM alpine\nARG VERSION=1.0",
			expectedType: ast.InstrARG,
			validate: func(t *testing.T, instr ast.Instruction) {
				a := instr.(*ast.ArgInstruction)
				if a.Name != "VERSION" || a.Default != "1.0" {
					t.Errorf("Name=Default = %q=%q, want VERSION=1.0", a.Name, a.Default)
				}
			},
		},
		{
			name:         "EXPOSE single port",
			input:        "FROM alpine\nEXPOSE 8080",
			expectedType: ast.InstrEXPOSE,
			validate: func(t *testing.T, instr ast.Instruction) {
				e := instr.(*ast.ExposeInstruction)
				if len(e.Ports) != 1 || e.Ports[0] != "8080" {
					t.Errorf("Ports = %v, want [8080]", e.Ports)
				}
			},
		},
		{
			name:         "EXPOSE multiple ports",
			input:        "FROM alpine\nEXPOSE 8080 443",
			expectedType: ast.InstrEXPOSE,
			validate: func(t *testing.T, instr ast.Instruction) {
				e := instr.(*ast.ExposeInstruction)
				if len(e.Ports) != 2 {
					t.Errorf("Ports = %v, want 2 ports", e.Ports)
				}
			},
		},
		{
			name:         "WORKDIR",
			input:        "FROM alpine\nWORKDIR /app",
			expectedType: ast.InstrWORKDIR,
			validate: func(t *testing.T, instr ast.Instruction) {
				w := instr.(*ast.WorkdirInstruction)
				if w.Path != "/app" {
					t.Errorf("Path = %q, want /app", w.Path)
				}
			},
		},
		{
			name:         "USER simple",
			input:        "FROM alpine\nUSER nobody",
			expectedType: ast.InstrUSER,
			validate: func(t *testing.T, instr ast.Instruction) {
				u := instr.(*ast.UserInstruction)
				if u.User != "nobody" {
					t.Errorf("User = %q, want nobody", u.User)
				}
			},
		},
		{
			name:         "USER with group",
			input:        "FROM alpine\nUSER nobody:nogroup",
			expectedType: ast.InstrUSER,
			validate: func(t *testing.T, instr ast.Instruction) {
				u := instr.(*ast.UserInstruction)
				if u.User != "nobody" || u.Group != "nogroup" {
					t.Errorf("User:Group = %q:%q, want nobody:nogroup", u.User, u.Group)
				}
			},
		},
		{
			name:         "LABEL",
			input:        "FROM alpine\nLABEL version=1.0",
			expectedType: ast.InstrLABEL,
			validate: func(t *testing.T, instr ast.Instruction) {
				l := instr.(*ast.LabelInstruction)
				if l.Labels["version"] != "1.0" {
					t.Errorf("Labels[version] = %q, want 1.0", l.Labels["version"])
				}
			},
		},
		{
			name:         "VOLUME shell form",
			input:        "FROM alpine\nVOLUME /data",
			expectedType: ast.InstrVOLUME,
			validate: func(t *testing.T, instr ast.Instruction) {
				v := instr.(*ast.VolumeInstruction)
				if len(v.Paths) != 1 || v.Paths[0] != "/data" {
					t.Errorf("Paths = %v, want [/data]", v.Paths)
				}
			},
		},
		{
			name:         "VOLUME JSON form",
			input:        `FROM alpine` + "\n" + `VOLUME ["/data", "/logs"]`,
			expectedType: ast.InstrVOLUME,
			validate: func(t *testing.T, instr ast.Instruction) {
				v := instr.(*ast.VolumeInstruction)
				if len(v.Paths) != 2 {
					t.Errorf("Paths = %v, want 2 paths", v.Paths)
				}
			},
		},
		{
			name:         "CMD shell form",
			input:        "FROM alpine\nCMD echo hello",
			expectedType: ast.InstrCMD,
			validate: func(t *testing.T, instr ast.Instruction) {
				c := instr.(*ast.CmdInstruction)
				if !c.Shell {
					t.Error("Shell = false, want true")
				}
			},
		},
		{
			name:         "CMD exec form",
			input:        `FROM alpine` + "\n" + `CMD ["echo", "hello"]`,
			expectedType: ast.InstrCMD,
			validate: func(t *testing.T, instr ast.Instruction) {
				c := instr.(*ast.CmdInstruction)
				if c.Shell {
					t.Error("Shell = true, want false")
				}
				if len(c.Command) != 2 {
					t.Errorf("Command = %v, want 2 elements", c.Command)
				}
			},
		},
		{
			name:         "ENTRYPOINT shell form",
			input:        "FROM alpine\nENTRYPOINT /app/start.sh",
			expectedType: ast.InstrENTRYPOINT,
			validate: func(t *testing.T, instr ast.Instruction) {
				e := instr.(*ast.EntrypointInstruction)
				if !e.Shell {
					t.Error("Shell = false, want true")
				}
			},
		},
		{
			name:         "ENTRYPOINT exec form",
			input:        `FROM alpine` + "\n" + `ENTRYPOINT ["./app"]`,
			expectedType: ast.InstrENTRYPOINT,
			validate: func(t *testing.T, instr ast.Instruction) {
				e := instr.(*ast.EntrypointInstruction)
				if e.Shell {
					t.Error("Shell = true, want false")
				}
			},
		},
		{
			name:         "HEALTHCHECK NONE",
			input:        "FROM alpine\nHEALTHCHECK NONE",
			expectedType: ast.InstrHEALTHCHECK,
			validate: func(t *testing.T, instr ast.Instruction) {
				h := instr.(*ast.HealthcheckInstruction)
				if !h.None {
					t.Error("None = false, want true")
				}
			},
		},
		{
			name:         "HEALTHCHECK with options",
			input:        "FROM alpine\nHEALTHCHECK --interval=30s --timeout=10s CMD curl -f http://localhost/",
			expectedType: ast.InstrHEALTHCHECK,
			validate: func(t *testing.T, instr ast.Instruction) {
				h := instr.(*ast.HealthcheckInstruction)
				if h.Interval != "30s" {
					t.Errorf("Interval = %q, want 30s", h.Interval)
				}
				if h.Timeout != "10s" {
					t.Errorf("Timeout = %q, want 10s", h.Timeout)
				}
			},
		},
		{
			name:         "SHELL",
			input:        `FROM alpine` + "\n" + `SHELL ["/bin/bash", "-c"]`,
			expectedType: ast.InstrSHELL,
			validate: func(t *testing.T, instr ast.Instruction) {
				s := instr.(*ast.ShellInstruction)
				if len(s.Shell) != 2 {
					t.Errorf("Shell = %v, want 2 elements", s.Shell)
				}
			},
		},
		{
			name:         "STOPSIGNAL",
			input:        "FROM alpine\nSTOPSIGNAL SIGTERM",
			expectedType: ast.InstrSTOPSIGNAL,
			validate: func(t *testing.T, instr ast.Instruction) {
				s := instr.(*ast.StopsignalInstruction)
				if s.Signal != "SIGTERM" {
					t.Errorf("Signal = %q, want SIGTERM", s.Signal)
				}
			},
		},
		{
			name:         "ONBUILD",
			input:        "FROM alpine\nONBUILD RUN echo building",
			expectedType: ast.InstrONBUILD,
			validate: func(t *testing.T, instr ast.Instruction) {
				o := instr.(*ast.OnbuildInstruction)
				if o.Instruction == nil {
					t.Error("Instruction = nil, want non-nil")
					return
				}
				if o.Instruction.Type() != ast.InstrRUN {
					t.Errorf("Instruction.Type() = %v, want RUN", o.Instruction.Type())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df, err := ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString() error = %v", err)
			}

			// Find the instruction of expected type (skip FROM if testing other types)
			var found ast.Instruction
			for _, instr := range df.Instructions {
				if instr.Type() == tt.expectedType {
					if tt.expectedType == ast.InstrFROM {
						found = instr
						break
					}
					// For non-FROM, take the last one (skip the required FROM)
					found = instr
				}
			}

			if found == nil {
				t.Fatalf("instruction of type %v not found", tt.expectedType)
			}

			tt.validate(t, found)
		})
	}
}


// TestParseMultiStageDockerfile tests parsing of multi-stage Dockerfiles.
// _Requirements: 1.3_
func TestParseMultiStageDockerfile(t *testing.T) {
	input := `# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

# Production stage
FROM alpine:3.18 AS runtime
WORKDIR /app
COPY --from=builder /app/main .
USER nobody
EXPOSE 8080
CMD ["./main"]`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	// Verify stages
	if len(df.Stages) != 2 {
		t.Errorf("len(Stages) = %d, want 2", len(df.Stages))
	}

	// Verify first stage
	if df.Stages[0].Name != "builder" {
		t.Errorf("Stages[0].Name = %q, want builder", df.Stages[0].Name)
	}
	if df.Stages[0].Index != 0 {
		t.Errorf("Stages[0].Index = %d, want 0", df.Stages[0].Index)
	}
	if df.Stages[0].FromInstr.Image != "golang" {
		t.Errorf("Stages[0].FromInstr.Image = %q, want golang", df.Stages[0].FromInstr.Image)
	}

	// Verify second stage
	if df.Stages[1].Name != "runtime" {
		t.Errorf("Stages[1].Name = %q, want runtime", df.Stages[1].Name)
	}
	if df.Stages[1].Index != 1 {
		t.Errorf("Stages[1].Index = %d, want 1", df.Stages[1].Index)
	}
	if df.Stages[1].FromInstr.Image != "alpine" {
		t.Errorf("Stages[1].FromInstr.Image = %q, want alpine", df.Stages[1].FromInstr.Image)
	}

	// Verify COPY --from references builder stage
	var copyInstr *ast.CopyInstruction
	for _, instr := range df.Stages[1].Instructions {
		if c, ok := instr.(*ast.CopyInstruction); ok && c.From != "" {
			copyInstr = c
			break
		}
	}
	if copyInstr == nil {
		t.Fatal("COPY --from instruction not found in second stage")
	}
	if copyInstr.From != "builder" {
		t.Errorf("COPY.From = %q, want builder", copyInstr.From)
	}
}

// TestParseMultiStageWithoutAlias tests multi-stage builds without explicit aliases.
func TestParseMultiStageWithoutAlias(t *testing.T) {
	input := `FROM golang:1.21
RUN go build

FROM alpine:3.18
COPY --from=0 /app /app`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Stages) != 2 {
		t.Errorf("len(Stages) = %d, want 2", len(df.Stages))
	}

	// First stage should have empty name
	if df.Stages[0].Name != "" {
		t.Errorf("Stages[0].Name = %q, want empty", df.Stages[0].Name)
	}

	// COPY --from=0 should reference stage index
	var copyInstr *ast.CopyInstruction
	for _, instr := range df.Instructions {
		if c, ok := instr.(*ast.CopyInstruction); ok && c.From != "" {
			copyInstr = c
			break
		}
	}
	if copyInstr == nil {
		t.Fatal("COPY --from instruction not found")
	}
	if copyInstr.From != "0" {
		t.Errorf("COPY.From = %q, want 0", copyInstr.From)
	}
}


// TestParseErrorReporting tests error reporting for malformed Dockerfiles.
// _Requirements: 1.4_
func TestParseErrorReporting(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorLine   int
	}{
		{
			name:        "FROM without image",
			input:       "FROM",
			expectError: true,
		},
		{
			name:        "COPY without destination",
			input:       "FROM alpine\nCOPY .",
			expectError: true,
		},
		{
			name:        "ADD without destination",
			input:       "FROM alpine\nADD src",
			expectError: true,
		},
		{
			name:        "WORKDIR without path",
			input:       "FROM alpine\nWORKDIR",
			expectError: true,
		},
		{
			name:        "USER without user",
			input:       "FROM alpine\nUSER",
			expectError: true,
		},
		{
			name:        "ARG without name",
			input:       "FROM alpine\nARG",
			expectError: true,
		},
		{
			name:        "STOPSIGNAL without signal",
			input:       "FROM alpine\nSTOPSIGNAL",
			expectError: true,
		},
		{
			name:        "ONBUILD without instruction",
			input:       "FROM alpine\nONBUILD",
			expectError: true,
		},
		{
			name:        "Unknown instruction",
			input:       "FROM alpine\nINVALID command",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseString(tt.input)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestParseErrorWithLineNumber tests that parse errors include line numbers.
func TestParseErrorWithLineNumber(t *testing.T) {
	input := `FROM alpine
RUN echo hello
INVALID instruction
RUN echo world`

	_, err := ParseString(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Error should be a ParseError with line information
	if pe, ok := err.(*ParseError); ok {
		if pe.Line != 3 {
			t.Errorf("ParseError.Line = %d, want 3", pe.Line)
		}
	}
}

// TestParseEmptyDockerfile tests parsing an empty Dockerfile.
func TestParseEmptyDockerfile(t *testing.T) {
	df, err := ParseString("")
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Instructions) != 0 {
		t.Errorf("len(Instructions) = %d, want 0", len(df.Instructions))
	}
	if len(df.Stages) != 0 {
		t.Errorf("len(Stages) = %d, want 0", len(df.Stages))
	}
}

// TestParseCommentsOnly tests parsing a Dockerfile with only comments.
func TestParseCommentsOnly(t *testing.T) {
	input := `# This is a comment
# Another comment`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Instructions) != 0 {
		t.Errorf("len(Instructions) = %d, want 0", len(df.Instructions))
	}
	if len(df.Comments) != 2 {
		t.Errorf("len(Comments) = %d, want 2", len(df.Comments))
	}
}


// TestParseInlineIgnoreComments tests extraction of inline ignore comments.
// _Requirements: 8.3_
func TestParseInlineIgnoreComments(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedLine   int
		expectedRuleID string
	}{
		{
			name: "single rule ignore",
			input: `FROM alpine
# docker-lint ignore: DL3006
FROM ubuntu`,
			expectedLine:   3,
			expectedRuleID: "DL3006",
		},
		{
			name: "multiple rules ignore",
			input: `FROM alpine
# docker-lint ignore: DL3006, DL3007
FROM ubuntu:latest`,
			expectedLine:   3,
			expectedRuleID: "DL3006",
		},
		{
			name: "case insensitive",
			input: `FROM alpine
# Docker-Lint Ignore: DL4000
ENV PASSWORD=secret`,
			expectedLine:   3,
			expectedRuleID: "DL4000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df, err := ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString() error = %v", err)
			}

			rules, ok := df.InlineIgnores[tt.expectedLine]
			if !ok {
				t.Errorf("no inline ignores for line %d", tt.expectedLine)
				return
			}

			found := false
			for _, rule := range rules {
				if rule == tt.expectedRuleID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("rule %q not found in ignores for line %d: %v", tt.expectedRuleID, tt.expectedLine, rules)
			}
		})
	}
}

// TestParseInlineIgnoreMultipleRules tests extraction of multiple rules from a single ignore comment.
func TestParseInlineIgnoreMultipleRules(t *testing.T) {
	input := `FROM alpine
# docker-lint ignore: DL3006, DL3007, DL3008
FROM ubuntu`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	rules, ok := df.InlineIgnores[3]
	if !ok {
		t.Fatal("no inline ignores for line 3")
	}

	expectedRules := []string{"DL3006", "DL3007", "DL3008"}
	if len(rules) != len(expectedRules) {
		t.Errorf("len(rules) = %d, want %d", len(rules), len(expectedRules))
	}

	for _, expected := range expectedRules {
		found := false
		for _, rule := range rules {
			if rule == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("rule %q not found in ignores", expected)
		}
	}
}

// TestParseCommentPreservation tests that comments are preserved in the AST.
func TestParseCommentPreservation(t *testing.T) {
	input := `# Build stage comment
FROM alpine:3.18
# Install dependencies
RUN apk add curl`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Comments) != 2 {
		t.Errorf("len(Comments) = %d, want 2", len(df.Comments))
	}

	// Verify comment content
	if df.Comments[0].Text != "# Build stage comment" {
		t.Errorf("Comments[0].Text = %q, want '# Build stage comment'", df.Comments[0].Text)
	}
	if df.Comments[0].LineNum != 1 {
		t.Errorf("Comments[0].LineNum = %d, want 1", df.Comments[0].LineNum)
	}
}


// TestParseLineNumbers tests that line numbers are correctly tracked.
func TestParseLineNumbers(t *testing.T) {
	input := `FROM alpine:3.18
WORKDIR /app
COPY . .
RUN echo hello
CMD ["./app"]`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	expectedLines := []int{1, 2, 3, 4, 5}
	if len(df.Instructions) != len(expectedLines) {
		t.Fatalf("len(Instructions) = %d, want %d", len(df.Instructions), len(expectedLines))
	}

	for i, instr := range df.Instructions {
		if instr.Line() != expectedLines[i] {
			t.Errorf("Instructions[%d].Line() = %d, want %d", i, instr.Line(), expectedLines[i])
		}
	}
}

// TestParseMultiLineContinuation tests parsing of multi-line instructions.
func TestParseMultiLineContinuation(t *testing.T) {
	input := `FROM alpine
RUN apt-get update && \
    apt-get install -y \
    curl \
    wget`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	// Should have 2 instructions: FROM and RUN
	if len(df.Instructions) != 2 {
		t.Errorf("len(Instructions) = %d, want 2", len(df.Instructions))
	}

	// RUN instruction should contain the full command
	runInstr := df.Instructions[1].(*ast.RunInstruction)
	if !strings.Contains(runInstr.Command, "apt-get update") {
		t.Errorf("RUN command missing 'apt-get update': %q", runInstr.Command)
	}
	if !strings.Contains(runInstr.Command, "curl") {
		t.Errorf("RUN command missing 'curl': %q", runInstr.Command)
	}
}

// TestParseNewParser tests the NewParser constructor.
func TestParseNewParser(t *testing.T) {
	input := "FROM alpine"
	p := NewParser(strings.NewReader(input))
	if p == nil {
		t.Fatal("NewParser() returned nil")
	}

	df, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(df.Instructions) != 1 {
		t.Errorf("len(Instructions) = %d, want 1", len(df.Instructions))
	}
}

// TestParseComplexDockerfile tests parsing a realistic, complex Dockerfile.
func TestParseComplexDockerfile(t *testing.T) {
	input := `# syntax=docker/dockerfile:1
ARG GO_VERSION=1.21

# Build stage
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /app ./cmd/app

# Production stage
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app /usr/local/bin/app
USER nobody:nogroup
EXPOSE 8080/tcp
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["app"]
CMD ["serve"]`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	// Verify we have 2 stages
	if len(df.Stages) != 2 {
		t.Errorf("len(Stages) = %d, want 2", len(df.Stages))
	}

	// Verify ARG before FROM is captured
	if len(df.Instructions) < 1 {
		t.Fatal("no instructions parsed")
	}

	// Count instruction types
	typeCounts := make(map[ast.InstructionType]int)
	for _, instr := range df.Instructions {
		typeCounts[instr.Type()]++
	}

	// Verify expected instruction counts
	if typeCounts[ast.InstrFROM] != 2 {
		t.Errorf("FROM count = %d, want 2", typeCounts[ast.InstrFROM])
	}
	if typeCounts[ast.InstrRUN] != 3 {
		t.Errorf("RUN count = %d, want 3", typeCounts[ast.InstrRUN])
	}
	if typeCounts[ast.InstrCOPY] != 3 {
		t.Errorf("COPY count = %d, want 3", typeCounts[ast.InstrCOPY])
	}
}

// TestParseRawText tests that Raw() returns the original instruction text.
func TestParseRawText(t *testing.T) {
	input := "FROM alpine:3.18 AS builder"
	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Instructions) != 1 {
		t.Fatalf("len(Instructions) = %d, want 1", len(df.Instructions))
	}

	raw := df.Instructions[0].Raw()
	if raw != input {
		t.Errorf("Raw() = %q, want %q", raw, input)
	}
}
