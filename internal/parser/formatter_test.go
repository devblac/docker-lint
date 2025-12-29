package parser

import (
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

func TestFormatNilDockerfile(t *testing.T) {
	result := Format(nil)
	if result != "" {
		t.Errorf("Format(nil) = %q, want empty string", result)
	}
}

func TestFormatEmptyDockerfile(t *testing.T) {
	df := &ast.Dockerfile{}
	result := Format(df)
	if result != "" {
		t.Errorf("Format(empty) = %q, want empty string", result)
	}
}

func TestFormatFromInstruction(t *testing.T) {
	tests := []struct {
		name     string
		instr    *ast.FromInstruction
		expected string
	}{
		{
			name:     "simple image",
			instr:    &ast.FromInstruction{Image: "alpine"},
			expected: "FROM alpine",
		},
		{
			name:     "image with tag",
			instr:    &ast.FromInstruction{Image: "alpine", Tag: "3.18"},
			expected: "FROM alpine:3.18",
		},
		{
			name:     "image with digest",
			instr:    &ast.FromInstruction{Image: "alpine", Digest: "sha256:abc123"},
			expected: "FROM alpine@sha256:abc123",
		},
		{
			name:     "image with alias",
			instr:    &ast.FromInstruction{Image: "alpine", Tag: "3.18", Alias: "builder"},
			expected: "FROM alpine:3.18 AS builder",
		},
		{
			name:     "image with platform",
			instr:    &ast.FromInstruction{Image: "alpine", Tag: "3.18", Platform: "linux/amd64"},
			expected: "FROM --platform=linux/amd64 alpine:3.18",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFrom(tt.instr)
			if result != tt.expected {
				t.Errorf("formatFrom() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatRunInstruction(t *testing.T) {
	tests := []struct {
		name     string
		instr    *ast.RunInstruction
		expected string
	}{
		{
			name:     "simple command",
			instr:    &ast.RunInstruction{Command: "echo hello"},
			expected: "RUN echo hello",
		},
		{
			name:     "empty command",
			instr:    &ast.RunInstruction{Command: ""},
			expected: "RUN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRun(tt.instr)
			if result != tt.expected {
				t.Errorf("formatRun() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatCopyInstruction(t *testing.T) {
	tests := []struct {
		name     string
		instr    *ast.CopyInstruction
		expected string
	}{
		{
			name:     "simple copy",
			instr:    &ast.CopyInstruction{Sources: []string{"."}, Dest: "/app"},
			expected: "COPY . /app",
		},
		{
			name:     "copy with from",
			instr:    &ast.CopyInstruction{Sources: []string{"/build/app"}, Dest: "/app", From: "builder"},
			expected: "COPY --from=builder /build/app /app",
		},
		{
			name:     "copy with chown",
			instr:    &ast.CopyInstruction{Sources: []string{"."}, Dest: "/app", Chown: "user:group"},
			expected: "COPY --chown=user:group . /app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCopy(tt.instr)
			if result != tt.expected {
				t.Errorf("formatCopy() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatEnvInstruction(t *testing.T) {
	tests := []struct {
		name     string
		instr    *ast.EnvInstruction
		expected string
	}{
		{
			name:     "key value",
			instr:    &ast.EnvInstruction{Key: "PATH", Value: "/usr/local/bin"},
			expected: "ENV PATH=/usr/local/bin",
		},
		{
			name:     "empty key",
			instr:    &ast.EnvInstruction{Key: "", Value: ""},
			expected: "ENV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatEnv(tt.instr)
			if result != tt.expected {
				t.Errorf("formatEnv() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatCmdInstruction(t *testing.T) {
	tests := []struct {
		name     string
		instr    *ast.CmdInstruction
		expected string
	}{
		{
			name:     "shell form",
			instr:    &ast.CmdInstruction{Command: []string{"echo hello"}, Shell: true},
			expected: "CMD echo hello",
		},
		{
			name:     "exec form",
			instr:    &ast.CmdInstruction{Command: []string{"echo", "hello"}, Shell: false},
			expected: `CMD ["echo","hello"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCmd(tt.instr)
			if result != tt.expected {
				t.Errorf("formatCmd() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatHealthcheckInstruction(t *testing.T) {
	tests := []struct {
		name     string
		instr    *ast.HealthcheckInstruction
		expected string
	}{
		{
			name:     "none",
			instr:    &ast.HealthcheckInstruction{None: true},
			expected: "HEALTHCHECK NONE",
		},
		{
			name:     "with options",
			instr:    &ast.HealthcheckInstruction{Interval: "30s", Timeout: "10s", Command: []string{"curl -f http://localhost/"}},
			expected: "HEALTHCHECK --interval=30s --timeout=10s CMD curl -f http://localhost/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHealthcheck(tt.instr)
			if result != tt.expected {
				t.Errorf("formatHealthcheck() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatFullDockerfile(t *testing.T) {
	df := &ast.Dockerfile{
		Instructions: []ast.Instruction{
			&ast.FromInstruction{Image: "alpine", Tag: "3.18"},
			&ast.RunInstruction{Command: "apk add --no-cache curl"},
			&ast.CopyInstruction{Sources: []string{"."}, Dest: "/app"},
			&ast.WorkdirInstruction{Path: "/app"},
			&ast.CmdInstruction{Command: []string{"./app"}, Shell: false},
		},
	}

	result := Format(df)
	expected := `FROM alpine:3.18
RUN apk add --no-cache curl
COPY . /app
WORKDIR /app
CMD ["./app"]`

	if result != expected {
		t.Errorf("Format() =\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestFormatRoundTrip(t *testing.T) {
	// Test that parsing and formatting produces semantically equivalent output
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple dockerfile",
			input: "FROM alpine:3.18",
		},
		{
			name:  "multi-instruction",
			input: "FROM alpine:3.18\nRUN echo hello\nCOPY . /app",
		},
		{
			name:  "with workdir and user",
			input: "FROM alpine:3.18\nWORKDIR /app\nUSER nobody",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse original
			df1, err := ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString() error = %v", err)
			}

			// Format back to string
			formatted := Format(df1)

			// Parse formatted output
			df2, err := ParseString(formatted)
			if err != nil {
				t.Fatalf("ParseString(formatted) error = %v", err)
			}

			// Compare instruction counts
			if len(df1.Instructions) != len(df2.Instructions) {
				t.Errorf("Instruction count mismatch: %d vs %d", len(df1.Instructions), len(df2.Instructions))
			}

			// Compare instruction types
			for i := range df1.Instructions {
				if df1.Instructions[i].Type() != df2.Instructions[i].Type() {
					t.Errorf("Instruction %d type mismatch: %s vs %s",
						i, df1.Instructions[i].Type(), df2.Instructions[i].Type())
				}
			}
		})
	}
}

func TestFormatAllInstructionTypes(t *testing.T) {
	// Test formatting of all instruction types
	tests := []struct {
		name     string
		instr    ast.Instruction
		contains string
	}{
		{"FROM", &ast.FromInstruction{Image: "alpine"}, "FROM alpine"},
		{"RUN", &ast.RunInstruction{Command: "echo test"}, "RUN echo test"},
		{"COPY", &ast.CopyInstruction{Sources: []string{"src"}, Dest: "dst"}, "COPY src dst"},
		{"ADD", &ast.AddInstruction{Sources: []string{"src"}, Dest: "dst"}, "ADD src dst"},
		{"ENV", &ast.EnvInstruction{Key: "FOO", Value: "bar"}, "ENV FOO=bar"},
		{"ARG", &ast.ArgInstruction{Name: "VERSION"}, "ARG VERSION"},
		{"ARG with default", &ast.ArgInstruction{Name: "VERSION", Default: "1.0"}, "ARG VERSION=1.0"},
		{"EXPOSE", &ast.ExposeInstruction{Ports: []string{"8080"}}, "EXPOSE 8080"},
		{"WORKDIR", &ast.WorkdirInstruction{Path: "/app"}, "WORKDIR /app"},
		{"USER", &ast.UserInstruction{User: "nobody"}, "USER nobody"},
		{"USER with group", &ast.UserInstruction{User: "nobody", Group: "nogroup"}, "USER nobody:nogroup"},
		{"LABEL", &ast.LabelInstruction{Labels: map[string]string{"version": "1.0"}}, "LABEL version=1.0"},
		{"VOLUME", &ast.VolumeInstruction{Paths: []string{"/data"}}, "VOLUME"},
		{"CMD shell", &ast.CmdInstruction{Command: []string{"echo hi"}, Shell: true}, "CMD echo hi"},
		{"ENTRYPOINT", &ast.EntrypointInstruction{Command: []string{"app"}, Shell: false}, "ENTRYPOINT"},
		{"HEALTHCHECK NONE", &ast.HealthcheckInstruction{None: true}, "HEALTHCHECK NONE"},
		{"SHELL", &ast.ShellInstruction{Shell: []string{"/bin/bash", "-c"}}, "SHELL"},
		{"STOPSIGNAL", &ast.StopsignalInstruction{Signal: "SIGTERM"}, "STOPSIGNAL SIGTERM"},
		{"ONBUILD", &ast.OnbuildInstruction{Instruction: &ast.RunInstruction{Command: "echo build"}}, "ONBUILD RUN echo build"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatInstruction(tt.instr)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatInstruction() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}
