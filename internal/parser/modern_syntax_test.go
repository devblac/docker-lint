package parser

import (
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/ast"
)

// findRun returns the nth RUN instruction in the Dockerfile.
func findRun(t *testing.T, df *ast.Dockerfile, n int) *ast.RunInstruction {
	t.Helper()
	count := 0
	for _, instr := range df.Instructions {
		if run, ok := instr.(*ast.RunInstruction); ok {
			if count == n {
				return run
			}
			count++
		}
	}
	t.Fatalf("RUN instruction #%d not found", n)
	return nil
}

// TestParseHeredoc verifies that BuildKit heredoc syntax parses and that the
// heredoc body is exposed to rules through the command text.
func TestParseHeredoc(t *testing.T) {
	input := `FROM python:3.12-slim
RUN <<EOT
apt-get update
apt-get install -y curl
EOT
USER app
`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Instructions) != 3 {
		t.Fatalf("len(Instructions) = %d, want 3", len(df.Instructions))
	}

	run := findRun(t, df, 0)
	if run.Line() != 2 {
		t.Errorf("RUN line = %d, want 2", run.Line())
	}
	// Rules match against the command text, so the heredoc body must be part of it.
	for _, want := range []string{"apt-get update", "apt-get install -y curl"} {
		if !strings.Contains(run.Command, want) {
			t.Errorf("RUN command missing %q: %q", want, run.Command)
		}
	}

	// The instruction after the heredoc keeps its real line number.
	user, ok := df.Instructions[2].(*ast.UserInstruction)
	if !ok {
		t.Fatalf("Instructions[2] = %T, want *ast.UserInstruction", df.Instructions[2])
	}
	if user.Line() != 6 {
		t.Errorf("USER line = %d, want 6", user.Line())
	}
}

// TestParseHeredocBodyIsNotComment verifies that a '#' line inside a heredoc is
// body content, not a comment, and cannot smuggle in an ignore directive.
func TestParseHeredocBodyIsNotComment(t *testing.T) {
	input := `FROM alpine:3.20
RUN <<EOT
# docker-lint ignore: DKL102
echo hello
EOT
`

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Comments) != 0 {
		t.Errorf("len(Comments) = %d, want 0: %+v", len(df.Comments), df.Comments)
	}
	if len(df.InlineIgnores) != 0 {
		t.Errorf("InlineIgnores = %v, want empty", df.InlineIgnores)
	}
}

// TestParseEscapeDirective verifies support for the `# escape=` parser directive.
func TestParseEscapeDirective(t *testing.T) {
	input := "# escape=`\n" +
		"FROM mcr.microsoft.com/windows/servercore:ltsc2022\n" +
		"RUN powershell -Command `\n" +
		"    Write-Host hi\n" +
		"USER ContainerUser\n"

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Instructions) != 3 {
		t.Fatalf("len(Instructions) = %d, want 3", len(df.Instructions))
	}

	run := findRun(t, df, 0)
	if !strings.Contains(run.Command, "Write-Host hi") {
		t.Errorf("RUN command missing continuation content: %q", run.Command)
	}

	user, ok := df.Instructions[2].(*ast.UserInstruction)
	if !ok {
		t.Fatalf("Instructions[2] = %T, want *ast.UserInstruction", df.Instructions[2])
	}
	if user.User != "ContainerUser" {
		t.Errorf("USER = %q, want ContainerUser", user.User)
	}
}

// TestParseSyntaxDirective verifies that a `# syntax=` directive does not shift
// instruction line numbers.
func TestParseSyntaxDirective(t *testing.T) {
	input := "# syntax=docker/dockerfile:1\nFROM alpine:3.20\n"

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Instructions) != 1 {
		t.Fatalf("len(Instructions) = %d, want 1", len(df.Instructions))
	}
	if df.Instructions[0].Line() != 2 {
		t.Errorf("FROM line = %d, want 2", df.Instructions[0].Line())
	}
}

// TestParseEnvMultiplePairs verifies that every key of a multi-pair ENV is
// exposed, so rules cannot miss the second and later keys.
func TestParseEnvMultiplePairs(t *testing.T) {
	input := "FROM alpine:3.20\nENV APP=1 DB_PASSWORD=hunter2 PORT=8080\n"

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	want := map[string]string{"APP": "1", "DB_PASSWORD": "hunter2", "PORT": "8080"}
	got := make(map[string]string)
	for _, instr := range df.Instructions {
		env, ok := instr.(*ast.EnvInstruction)
		if !ok {
			continue
		}
		got[env.Key] = env.Value
		if env.Line() != 2 {
			t.Errorf("ENV %q line = %d, want 2", env.Key, env.Line())
		}
	}

	if len(got) != len(want) {
		t.Fatalf("parsed %d ENV keys, want %d: %v", len(got), len(want), got)
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("ENV[%q] = %q, want %q", key, got[key], value)
		}
	}
}

// TestParseArgMultipleNames verifies that a multi-name ARG yields one
// instruction per declared variable.
func TestParseArgMultipleNames(t *testing.T) {
	input := "FROM alpine:3.20\nARG VERSION=1.0 BUILD_TOKEN\n"

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	got := make(map[string]string)
	for _, instr := range df.Instructions {
		if arg, ok := instr.(*ast.ArgInstruction); ok {
			got[arg.Name] = arg.Default
		}
	}

	if len(got) != 2 {
		t.Fatalf("parsed %d ARG names, want 2: %v", len(got), got)
	}
	if got["VERSION"] != "1.0" {
		t.Errorf("ARG VERSION default = %q, want 1.0", got["VERSION"])
	}
	if value, ok := got["BUILD_TOKEN"]; !ok || value != "" {
		t.Errorf("ARG BUILD_TOKEN = (%q, present=%v), want empty default", value, ok)
	}
}

// TestParseImageReferenceForms verifies that a colon is read as a tag separator
// only after the last slash, so registry ports are not mistaken for tags.
func TestParseImageReferenceForms(t *testing.T) {
	tests := []struct {
		name   string
		ref    string
		image  string
		tag    string
		digest string
	}{
		{name: "bare image", ref: "alpine", image: "alpine"},
		{name: "tagged image", ref: "alpine:3.20", image: "alpine", tag: "3.20"},
		{name: "registry port without tag", ref: "localhost:5000/app", image: "localhost:5000/app"},
		{name: "registry port with tag", ref: "localhost:5000/app:v1", image: "localhost:5000/app", tag: "v1"},
		{
			name:   "digest",
			ref:    "alpine@sha256:0000000000000000000000000000000000000000000000000000000000000000",
			image:  "alpine",
			digest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:   "registry port with digest",
			ref:    "localhost:5000/app@sha256:0000000000000000000000000000000000000000000000000000000000000000",
			image:  "localhost:5000/app",
			digest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df, err := ParseString("FROM " + tt.ref + "\n")
			if err != nil {
				t.Fatalf("ParseString() error = %v", err)
			}

			from, ok := df.Instructions[0].(*ast.FromInstruction)
			if !ok {
				t.Fatalf("Instructions[0] = %T, want *ast.FromInstruction", df.Instructions[0])
			}
			if from.Image != tt.image {
				t.Errorf("Image = %q, want %q", from.Image, tt.image)
			}
			if from.Tag != tt.tag {
				t.Errorf("Tag = %q, want %q", from.Tag, tt.tag)
			}
			if from.Digest != tt.digest {
				t.Errorf("Digest = %q, want %q", from.Digest, tt.digest)
			}
		})
	}
}

// TestParseRunMountFlag verifies that BuildKit RUN flags are consumed instead of
// leaking into the command text.
func TestParseRunMountFlag(t *testing.T) {
	input := "FROM golang:1.22-alpine\n" +
		"RUN --mount=type=cache,target=/go/pkg/mod go mod download\n" +
		"RUN --mount=type=secret,id=token go build -o /app\n"

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	first := findRun(t, df, 0)
	if first.Command != "go mod download" {
		t.Errorf("RUN command = %q, want %q", first.Command, "go mod download")
	}

	second := findRun(t, df, 1)
	if second.Command != "go build -o /app" {
		t.Errorf("RUN command = %q, want %q", second.Command, "go build -o /app")
	}

	// Only cache mounts are reported; the secret mount is not a cache.
	if len(first.CacheMounts) != 1 || first.CacheMounts[0] != "/go/pkg/mod" {
		t.Errorf("CacheMounts = %v, want [/go/pkg/mod]", first.CacheMounts)
	}
	if len(second.CacheMounts) != 0 {
		t.Errorf("CacheMounts = %v, want none for a secret mount", second.CacheMounts)
	}
}

// TestParseRunCacheMountVariants verifies cache mount targets across option
// spellings, multiple mounts and unexpanded variables.
func TestParseRunCacheMountVariants(t *testing.T) {
	input := "FROM debian:bookworm\n" +
		"ARG CACHE_DIR=/root/.cache\n" +
		"RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \\\n" +
		"    --mount=type=cache,dst=/var/lib/apt \\\n" +
		"    --mount=type=cache,target=${CACHE_DIR} \\\n" +
		"    --mount=type=bind,target=/src \\\n" +
		"    apt-get update\n"

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	got := findRun(t, df, 0).CacheMounts
	want := []string{"/var/cache/apt", "/var/lib/apt", "${CACHE_DIR}"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("CacheMounts = %v, want %v", got, want)
	}
}

// TestParseCommentInsideContinuation verifies that a comment inside a line
// continuation is skipped rather than breaking the instruction.
func TestParseCommentInsideContinuation(t *testing.T) {
	input := "FROM alpine:3.20\n" +
		"RUN echo first && \\\n" +
		"    # explain the next step\n" +
		"    echo second\n"

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(df.Instructions) != 2 {
		t.Fatalf("len(Instructions) = %d, want 2", len(df.Instructions))
	}

	run := findRun(t, df, 0)
	for _, want := range []string{"echo first", "echo second"} {
		if !strings.Contains(run.Command, want) {
			t.Errorf("RUN command missing %q: %q", want, run.Command)
		}
	}
}

// TestParseInlineIgnoreAcrossBlankLines verifies that an ignore directive
// attaches to the next instruction even when blank lines separate them.
func TestParseInlineIgnoreAcrossBlankLines(t *testing.T) {
	input := "FROM alpine:3.20\n" +
		"\n" +
		"# docker-lint ignore: DKL102\n" +
		"\n" +
		"FROM ubuntu:latest\n"

	df, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	rules, ok := df.InlineIgnores[5]
	if !ok {
		t.Fatalf("no inline ignores for line 5: %v", df.InlineIgnores)
	}
	if len(rules) != 1 || rules[0] != "DKL102" {
		t.Errorf("InlineIgnores[5] = %v, want [DKL102]", rules)
	}
}
