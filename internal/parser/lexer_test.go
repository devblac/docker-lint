// Package parser provides lexer and parser for Dockerfile content.
package parser

import (
	"strings"
	"testing"
)

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		tokenType TokenType
		expected  string
	}{
		{TokenInstruction, "INSTRUCTION"},
		{TokenArgument, "ARGUMENT"},
		{TokenComment, "COMMENT"},
		{TokenNewline, "NEWLINE"},
		{TokenEOF, "EOF"},
		{TokenError, "ERROR"},
		{TokenType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		got := tt.tokenType.String()
		if got != tt.expected {
			t.Errorf("TokenType(%d).String() = %q, want %q", tt.tokenType, got, tt.expected)
		}
	}
}

func TestIsValidInstruction(t *testing.T) {
	validInstructions := []string{
		"FROM", "RUN", "COPY", "ADD", "ENV", "ARG", "EXPOSE", "WORKDIR",
		"USER", "LABEL", "VOLUME", "CMD", "ENTRYPOINT", "HEALTHCHECK",
		"SHELL", "STOPSIGNAL", "ONBUILD", "MAINTAINER",
		// Test case insensitivity
		"from", "From", "FROM",
	}

	for _, instr := range validInstructions {
		if !IsValidInstruction(instr) {
			t.Errorf("IsValidInstruction(%q) = false, want true", instr)
		}
	}

	invalidInstructions := []string{
		"INVALID", "DOCKER", "BUILD", "PUSH", "", "123",
	}

	for _, instr := range invalidInstructions {
		if IsValidInstruction(instr) {
			t.Errorf("IsValidInstruction(%q) = true, want false", instr)
		}
	}
}

func TestLexerBasicInstructions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "FROM instruction",
			input: "FROM alpine:3.18",
			expected: []Token{
				{Type: TokenInstruction, Value: "FROM", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "alpine:3.18", Line: 1, Column: 6},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "RUN instruction",
			input: "RUN apt-get update",
			expected: []Token{
				{Type: TokenInstruction, Value: "RUN", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "apt-get update", Line: 1, Column: 5},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "COPY instruction",
			input: "COPY . /app",
			expected: []Token{
				{Type: TokenInstruction, Value: "COPY", Line: 1, Column: 1},
				{Type: TokenArgument, Value: ". /app", Line: 1, Column: 6},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "ADD instruction",
			input: "ADD https://example.com/file.tar.gz /app/",
			expected: []Token{
				{Type: TokenInstruction, Value: "ADD", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "https://example.com/file.tar.gz /app/", Line: 1, Column: 5},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "ENV instruction",
			input: "ENV NODE_ENV=production",
			expected: []Token{
				{Type: TokenInstruction, Value: "ENV", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "NODE_ENV=production", Line: 1, Column: 5},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "ARG instruction",
			input: "ARG VERSION=1.0",
			expected: []Token{
				{Type: TokenInstruction, Value: "ARG", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "VERSION=1.0", Line: 1, Column: 5},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "EXPOSE instruction",
			input: "EXPOSE 8080",
			expected: []Token{
				{Type: TokenInstruction, Value: "EXPOSE", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "8080", Line: 1, Column: 8},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "WORKDIR instruction",
			input: "WORKDIR /app",
			expected: []Token{
				{Type: TokenInstruction, Value: "WORKDIR", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "/app", Line: 1, Column: 9},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "USER instruction",
			input: "USER appuser",
			expected: []Token{
				{Type: TokenInstruction, Value: "USER", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "appuser", Line: 1, Column: 6},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "LABEL instruction",
			input: `LABEL maintainer="test@example.com"`,
			expected: []Token{
				{Type: TokenInstruction, Value: "LABEL", Line: 1, Column: 1},
				{Type: TokenArgument, Value: `maintainer="test@example.com"`, Line: 1, Column: 7},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "VOLUME instruction",
			input: "VOLUME /data",
			expected: []Token{
				{Type: TokenInstruction, Value: "VOLUME", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "/data", Line: 1, Column: 8},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "CMD instruction exec form",
			input: `CMD ["node", "server.js"]`,
			expected: []Token{
				{Type: TokenInstruction, Value: "CMD", Line: 1, Column: 1},
				{Type: TokenArgument, Value: `["node", "server.js"]`, Line: 1, Column: 5},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "ENTRYPOINT instruction",
			input: `ENTRYPOINT ["docker-entrypoint.sh"]`,
			expected: []Token{
				{Type: TokenInstruction, Value: "ENTRYPOINT", Line: 1, Column: 1},
				{Type: TokenArgument, Value: `["docker-entrypoint.sh"]`, Line: 1, Column: 12},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "HEALTHCHECK instruction",
			input: "HEALTHCHECK CMD curl -f http://localhost/",
			expected: []Token{
				{Type: TokenInstruction, Value: "HEALTHCHECK", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "CMD curl -f http://localhost/", Line: 1, Column: 13},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "SHELL instruction",
			input: `SHELL ["/bin/bash", "-c"]`,
			expected: []Token{
				{Type: TokenInstruction, Value: "SHELL", Line: 1, Column: 1},
				{Type: TokenArgument, Value: `["/bin/bash", "-c"]`, Line: 1, Column: 7},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "STOPSIGNAL instruction",
			input: "STOPSIGNAL SIGTERM",
			expected: []Token{
				{Type: TokenInstruction, Value: "STOPSIGNAL", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "SIGTERM", Line: 1, Column: 12},
				{Type: TokenEOF, Line: 1},
			},
		},
		{
			name:  "ONBUILD instruction",
			input: "ONBUILD RUN npm install",
			expected: []Token{
				{Type: TokenInstruction, Value: "ONBUILD", Line: 1, Column: 1},
				{Type: TokenArgument, Value: "RUN npm install", Line: 1, Column: 9},
				{Type: TokenEOF, Line: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := TokenizeString(tt.input)
			if len(tokens) != len(tt.expected) {
				t.Errorf("got %d tokens, want %d", len(tokens), len(tt.expected))
				for i, tok := range tokens {
					t.Logf("  token[%d]: %+v", i, tok)
				}
				return
			}
			for i, tok := range tokens {
				if tok.Type != tt.expected[i].Type {
					t.Errorf("token[%d].Type = %v, want %v", i, tok.Type, tt.expected[i].Type)
				}
				if tok.Value != tt.expected[i].Value {
					t.Errorf("token[%d].Value = %q, want %q", i, tok.Value, tt.expected[i].Value)
				}
				if tok.Line != tt.expected[i].Line {
					t.Errorf("token[%d].Line = %d, want %d", i, tok.Line, tt.expected[i].Line)
				}
			}
		})
	}
}

func TestLexerMultiLineHandling(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedTokens    int
		expectedInstrType TokenType
		expectedInstrVal  string
	}{
		{
			name: "RUN with backslash continuation",
			input: `RUN apt-get update && \
    apt-get install -y curl`,
			expectedTokens:    3, // INSTRUCTION, ARGUMENT, EOF
			expectedInstrType: TokenInstruction,
			expectedInstrVal:  "RUN",
		},
		{
			name: "Multiple line continuation",
			input: `RUN apt-get update && \
    apt-get install -y \
    curl \
    wget`,
			expectedTokens:    3, // INSTRUCTION, ARGUMENT, EOF
			expectedInstrType: TokenInstruction,
			expectedInstrVal:  "RUN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := TokenizeString(tt.input)
			if len(tokens) != tt.expectedTokens {
				t.Errorf("got %d tokens, want %d", len(tokens), tt.expectedTokens)
				for i, tok := range tokens {
					t.Logf("  token[%d]: %+v", i, tok)
				}
				return
			}
			// Verify first token is the expected instruction
			if tokens[0].Type != tt.expectedInstrType {
				t.Errorf("token[0].Type = %v, want %v", tokens[0].Type, tt.expectedInstrType)
			}
			if tokens[0].Value != tt.expectedInstrVal {
				t.Errorf("token[0].Value = %q, want %q", tokens[0].Value, tt.expectedInstrVal)
			}
			// Verify second token is an argument (continuation was handled)
			if tokens[1].Type != TokenArgument {
				t.Errorf("token[1].Type = %v, want ARGUMENT", tokens[1].Type)
			}
			// Verify last token is EOF
			if tokens[len(tokens)-1].Type != TokenEOF {
				t.Errorf("last token.Type = %v, want EOF", tokens[len(tokens)-1].Type)
			}
		})
	}
}

func TestLexerCommentPreservation(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedComment string
		hasInstruction  bool
	}{
		{
			name:            "Single comment line",
			input:           "# This is a comment",
			expectedComment: "# This is a comment",
			hasInstruction:  false,
		},
		{
			name: "Comment before instruction",
			input: `# Build stage
FROM alpine:3.18`,
			expectedComment: "# Build stage",
			hasInstruction:  true,
		},
		{
			name: "Inline ignore comment",
			input: `# docker-lint ignore: DL3006
FROM alpine`,
			expectedComment: "# docker-lint ignore: DL3006",
			hasInstruction:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := TokenizeString(tt.input)

			// Find comment token
			var foundComment bool
			var foundInstruction bool
			for _, tok := range tokens {
				if tok.Type == TokenComment {
					foundComment = true
					if tok.Value != tt.expectedComment {
						t.Errorf("comment value = %q, want %q", tok.Value, tt.expectedComment)
					}
				}
				if tok.Type == TokenInstruction {
					foundInstruction = true
				}
			}

			if !foundComment {
				t.Error("expected to find a comment token")
			}
			if tt.hasInstruction && !foundInstruction {
				t.Error("expected to find an instruction token")
			}
		})
	}
}

func TestLexerMultipleComments(t *testing.T) {
	input := `# Comment 1
# Comment 2
FROM alpine`

	tokens := TokenizeString(input)

	// Count comment tokens
	commentCount := 0
	for _, tok := range tokens {
		if tok.Type == TokenComment {
			commentCount++
		}
	}

	if commentCount != 2 {
		t.Errorf("got %d comment tokens, want 2", commentCount)
	}
}

func TestLexerEdgeCases(t *testing.T) {
	t.Run("Empty input", func(t *testing.T) {
		tokens := TokenizeString("")
		if len(tokens) != 1 || tokens[0].Type != TokenEOF {
			t.Errorf("empty input should produce single EOF token, got %+v", tokens)
		}
	})

	t.Run("Only whitespace", func(t *testing.T) {
		tokens := TokenizeString("   \t  ")
		// Should end with EOF
		if tokens[len(tokens)-1].Type != TokenEOF {
			t.Errorf("last token should be EOF, got %v", tokens[len(tokens)-1].Type)
		}
	})

	t.Run("Empty lines between instructions", func(t *testing.T) {
		tokens := TokenizeString("FROM alpine\n\nRUN echo hello")
		// Should have both FROM and RUN instructions
		var foundFrom, foundRun bool
		for _, tok := range tokens {
			if tok.Type == TokenInstruction && tok.Value == "FROM" {
				foundFrom = true
			}
			if tok.Type == TokenInstruction && tok.Value == "RUN" {
				foundRun = true
			}
		}
		if !foundFrom || !foundRun {
			t.Errorf("expected both FROM and RUN instructions, foundFrom=%v, foundRun=%v", foundFrom, foundRun)
		}
	})

	t.Run("Instruction with leading whitespace", func(t *testing.T) {
		tokens := TokenizeString("  FROM alpine")
		if tokens[0].Type != TokenInstruction || tokens[0].Value != "FROM" {
			t.Errorf("first token should be FROM instruction, got %+v", tokens[0])
		}
	})

	t.Run("Quoted string with spaces", func(t *testing.T) {
		tokens := TokenizeString(`LABEL description="A multi word description"`)
		// Find the argument token
		for _, tok := range tokens {
			if tok.Type == TokenArgument {
				if tok.Value != `description="A multi word description"` {
					t.Errorf("argument value = %q, want %q", tok.Value, `description="A multi word description"`)
				}
				return
			}
		}
		t.Error("expected to find argument token")
	})

	t.Run("Single quoted string", func(t *testing.T) {
		tokens := TokenizeString(`ENV MESSAGE='Hello World'`)
		for _, tok := range tokens {
			if tok.Type == TokenArgument {
				if tok.Value != `MESSAGE='Hello World'` {
					t.Errorf("argument value = %q, want %q", tok.Value, `MESSAGE='Hello World'`)
				}
				return
			}
		}
		t.Error("expected to find argument token")
	})

	t.Run("Special characters in argument", func(t *testing.T) {
		tokens := TokenizeString(`RUN echo "Hello $USER" && ls -la`)
		for _, tok := range tokens {
			if tok.Type == TokenArgument {
				if tok.Value != `echo "Hello $USER" && ls -la` {
					t.Errorf("argument value = %q, want %q", tok.Value, `echo "Hello $USER" && ls -la`)
				}
				return
			}
		}
		t.Error("expected to find argument token")
	})

	t.Run("Escaped characters", func(t *testing.T) {
		tokens := TokenizeString(`RUN echo \"quoted\"`)
		for _, tok := range tokens {
			if tok.Type == TokenArgument {
				// Escaped quotes should be converted
				if tok.Value != `echo "quoted"` {
					t.Errorf("argument value = %q, want %q", tok.Value, `echo "quoted"`)
				}
				return
			}
		}
		t.Error("expected to find argument token")
	})

	t.Run("Tab as whitespace", func(t *testing.T) {
		tokens := TokenizeString("FROM\talpine")
		if tokens[0].Type != TokenInstruction || tokens[0].Value != "FROM" {
			t.Errorf("first token should be FROM instruction, got %+v", tokens[0])
		}
		if tokens[1].Type != TokenArgument || tokens[1].Value != "alpine" {
			t.Errorf("second token should be alpine argument, got %+v", tokens[1])
		}
	})

	t.Run("FROM with AS alias", func(t *testing.T) {
		tokens := TokenizeString("FROM alpine:3.18 AS builder")
		for _, tok := range tokens {
			if tok.Type == TokenArgument {
				if tok.Value != "alpine:3.18 AS builder" {
					t.Errorf("argument value = %q, want %q", tok.Value, "alpine:3.18 AS builder")
				}
				return
			}
		}
		t.Error("expected to find argument token")
	})

	t.Run("COPY with --from flag", func(t *testing.T) {
		tokens := TokenizeString("COPY --from=builder /app /app")
		for _, tok := range tokens {
			if tok.Type == TokenArgument {
				if tok.Value != "--from=builder /app /app" {
					t.Errorf("argument value = %q, want %q", tok.Value, "--from=builder /app /app")
				}
				return
			}
		}
		t.Error("expected to find argument token")
	})

	t.Run("ARG without default value", func(t *testing.T) {
		tokens := TokenizeString("ARG VERSION")
		for _, tok := range tokens {
			if tok.Type == TokenArgument {
				if tok.Value != "VERSION" {
					t.Errorf("argument value = %q, want %q", tok.Value, "VERSION")
				}
				return
			}
		}
		t.Error("expected to find argument token")
	})

	t.Run("Multiple EXPOSE ports", func(t *testing.T) {
		tokens := TokenizeString("EXPOSE 8080 443 3000")
		for _, tok := range tokens {
			if tok.Type == TokenArgument {
				if tok.Value != "8080 443 3000" {
					t.Errorf("argument value = %q, want %q", tok.Value, "8080 443 3000")
				}
				return
			}
		}
		t.Error("expected to find argument token")
	})
}

func TestLexerMultipleInstructions(t *testing.T) {
	input := `FROM alpine:3.18
WORKDIR /app
COPY . .
RUN go build -o app
CMD ["./app"]`

	tokens := TokenizeString(input)

	// Verify we get the expected instruction sequence
	expectedInstructions := []string{"FROM", "WORKDIR", "COPY", "RUN", "CMD"}
	instructionCount := 0

	for _, tok := range tokens {
		if tok.Type == TokenInstruction {
			if instructionCount >= len(expectedInstructions) {
				t.Errorf("got more instructions than expected")
				break
			}
			if tok.Value != expectedInstructions[instructionCount] {
				t.Errorf("instruction[%d] = %q, want %q", instructionCount, tok.Value, expectedInstructions[instructionCount])
			}
			instructionCount++
		}
	}

	if instructionCount != len(expectedInstructions) {
		t.Errorf("got %d instructions, want %d", instructionCount, len(expectedInstructions))
	}
}

func TestLexerPeekToken(t *testing.T) {
	input := "FROM alpine"
	lexer := NewLexer(strings.NewReader(input))

	// Peek should return the first token without consuming it
	peeked := lexer.PeekToken()
	if peeked.Type != TokenInstruction || peeked.Value != "FROM" {
		t.Errorf("PeekToken() = %+v, want Instruction FROM", peeked)
	}

	// Peek again should return the same token
	peeked2 := lexer.PeekToken()
	if peeked2.Type != peeked.Type || peeked2.Value != peeked.Value {
		t.Errorf("second PeekToken() = %+v, want same as first %+v", peeked2, peeked)
	}

	// NextToken should return the peeked token
	next := lexer.NextToken()
	if next.Type != peeked.Type || next.Value != peeked.Value {
		t.Errorf("NextToken() after Peek = %+v, want %+v", next, peeked)
	}

	// Next NextToken should be the argument
	next2 := lexer.NextToken()
	if next2.Type != TokenArgument || next2.Value != "alpine" {
		t.Errorf("second NextToken() = %+v, want Argument alpine", next2)
	}
}

func TestLexerReset(t *testing.T) {
	input1 := "FROM alpine"
	input2 := "RUN echo hello"

	lexer := NewLexer(strings.NewReader(input1))

	// Tokenize first input
	tok1 := lexer.NextToken()
	if tok1.Type != TokenInstruction || tok1.Value != "FROM" {
		t.Errorf("first input token = %+v, want Instruction FROM", tok1)
	}

	// Reset with new input
	lexer.Reset(strings.NewReader(input2))

	// Tokenize second input
	tok2 := lexer.NextToken()
	if tok2.Type != TokenInstruction || tok2.Value != "RUN" {
		t.Errorf("after reset token = %+v, want Instruction RUN", tok2)
	}
}

func TestLexerCurrentLine(t *testing.T) {
	input := `FROM alpine
RUN echo hello`

	lexer := NewLexer(strings.NewReader(input))

	// Read first instruction
	lexer.NextToken()
	if lexer.CurrentLine() != 1 {
		t.Errorf("CurrentLine() after first instruction = %d, want 1", lexer.CurrentLine())
	}

	// Read argument
	lexer.NextToken()
	// Read newline
	lexer.NextToken()
	// Read second instruction
	lexer.NextToken()
	if lexer.CurrentLine() != 2 {
		t.Errorf("CurrentLine() after second instruction = %d, want 2", lexer.CurrentLine())
	}
}

func TestLexerComplexDockerfile(t *testing.T) {
	input := `# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

# Production stage
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/main .
USER nobody
EXPOSE 8080
CMD ["./main"]`

	tokens := TokenizeString(input)

	// Count token types
	counts := make(map[TokenType]int)
	for _, tok := range tokens {
		counts[tok.Type]++
	}

	// Verify we have expected token types
	if counts[TokenComment] != 2 {
		t.Errorf("got %d comments, want 2", counts[TokenComment])
	}
	if counts[TokenInstruction] != 12 {
		t.Errorf("got %d instructions, want 12", counts[TokenInstruction])
	}
	if counts[TokenEOF] != 1 {
		t.Errorf("got %d EOF tokens, want 1", counts[TokenEOF])
	}
}

func TestTokenizeString(t *testing.T) {
	input := "FROM alpine"
	tokens := TokenizeString(input)

	if len(tokens) < 3 {
		t.Errorf("TokenizeString returned %d tokens, want at least 3", len(tokens))
		return
	}

	if tokens[0].Type != TokenInstruction || tokens[0].Value != "FROM" {
		t.Errorf("first token = %+v, want Instruction FROM", tokens[0])
	}
	if tokens[1].Type != TokenArgument || tokens[1].Value != "alpine" {
		t.Errorf("second token = %+v, want Argument alpine", tokens[1])
	}
	if tokens[len(tokens)-1].Type != TokenEOF {
		t.Errorf("last token = %+v, want EOF", tokens[len(tokens)-1])
	}
}
