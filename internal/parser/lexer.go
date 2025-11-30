// Package parser provides lexer and parser for Dockerfile content.
package parser

import (
	"bufio"
	"io"
	"strings"
	"unicode"
)

// TokenType represents the type of a lexer token.
type TokenType int

const (
	TokenInstruction TokenType = iota // Dockerfile instruction keyword (FROM, RUN, etc.)
	TokenArgument                     // Instruction argument
	TokenComment                      // Comment line starting with #
	TokenNewline                      // End of line
	TokenEOF                          // End of file
	TokenError                        // Lexer error
)

// String returns the string representation of a TokenType.
func (t TokenType) String() string {
	switch t {
	case TokenInstruction:
		return "INSTRUCTION"
	case TokenArgument:
		return "ARGUMENT"
	case TokenComment:
		return "COMMENT"
	case TokenNewline:
		return "NEWLINE"
	case TokenEOF:
		return "EOF"
	case TokenError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Token represents a lexer token.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// validInstructions contains all valid Dockerfile instruction keywords.
var validInstructions = map[string]bool{
	"FROM":        true,
	"RUN":         true,
	"COPY":        true,
	"ADD":         true,
	"ENV":         true,
	"ARG":         true,
	"EXPOSE":      true,
	"WORKDIR":     true,
	"USER":        true,
	"LABEL":       true,
	"VOLUME":      true,
	"CMD":         true,
	"ENTRYPOINT":  true,
	"HEALTHCHECK": true,
	"SHELL":       true,
	"STOPSIGNAL":  true,
	"ONBUILD":     true,
	"MAINTAINER":  true, // Deprecated but still valid
}

// IsValidInstruction checks if a string is a valid Dockerfile instruction.
func IsValidInstruction(s string) bool {
	return validInstructions[strings.ToUpper(s)]
}

// Lexer tokenizes Dockerfile content.
type Lexer struct {
	reader      *bufio.Reader
	line        int
	column      int
	currentLine string
	linePos     int
	atEOF       bool
	peekedToken *Token
}

// NewLexer creates a new Lexer from an io.Reader.
func NewLexer(r io.Reader) *Lexer {
	return &Lexer{
		reader: bufio.NewReader(r),
		line:   0,
		column: 0,
	}
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	// Return peeked token if available
	if l.peekedToken != nil {
		tok := *l.peekedToken
		l.peekedToken = nil
		return tok
	}

	return l.scanToken()
}

// PeekToken returns the next token without consuming it.
func (l *Lexer) PeekToken() Token {
	if l.peekedToken != nil {
		return *l.peekedToken
	}
	tok := l.scanToken()
	l.peekedToken = &tok
	return tok
}

// scanToken performs the actual token scanning.
func (l *Lexer) scanToken() Token {
	// Read next line if needed
	if l.currentLine == "" || l.linePos >= len(l.currentLine) {
		if l.atEOF {
			return Token{Type: TokenEOF, Line: l.line, Column: l.column}
		}
		if !l.readNextLine() {
			return Token{Type: TokenEOF, Line: l.line, Column: l.column}
		}
	}

	// Skip leading whitespace
	l.skipWhitespace()

	if l.linePos >= len(l.currentLine) {
		// End of line reached
		tok := Token{Type: TokenNewline, Value: "\n", Line: l.line, Column: l.linePos + 1}
		l.currentLine = ""
		return tok
	}

	startCol := l.linePos + 1

	// Check for comment
	if l.currentLine[l.linePos] == '#' {
		comment := l.currentLine[l.linePos:]
		l.linePos = len(l.currentLine)
		return Token{Type: TokenComment, Value: comment, Line: l.line, Column: startCol}
	}

	// Check if this is the start of a line (after whitespace) - could be an instruction
	if l.isAtLineStart() {
		word := l.scanWord()
		if IsValidInstruction(word) {
			return Token{Type: TokenInstruction, Value: strings.ToUpper(word), Line: l.line, Column: startCol}
		}
		// Not an instruction, treat as argument
		return Token{Type: TokenArgument, Value: word, Line: l.line, Column: startCol}
	}

	// Scan argument (rest of the line, handling continuations and quotes)
	arg := l.scanArgument()
	return Token{Type: TokenArgument, Value: arg, Line: l.line, Column: startCol}
}


// readNextLine reads the next line from the input, handling line continuations.
func (l *Lexer) readNextLine() bool {
	var fullLine strings.Builder
	firstLine := true

	for {
		line, err := l.reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return false
		}

		if firstLine {
			l.line++
			firstLine = false
		}

		// Remove trailing newline
		line = strings.TrimRight(line, "\r\n")

		// Check for line continuation (backslash at end)
		if strings.HasSuffix(line, "\\") {
			// Remove the backslash and continue reading
			fullLine.WriteString(strings.TrimSuffix(line, "\\"))
			fullLine.WriteString(" ") // Replace continuation with space
			if err == io.EOF {
				l.atEOF = true
				break
			}
			l.line++ // Increment line for continuation
			continue
		}

		fullLine.WriteString(line)

		if err == io.EOF {
			l.atEOF = true
		}
		break
	}

	l.currentLine = fullLine.String()
	l.linePos = 0
	l.column = 0

	return len(l.currentLine) > 0 || !l.atEOF
}

// skipWhitespace advances past any whitespace characters.
func (l *Lexer) skipWhitespace() {
	for l.linePos < len(l.currentLine) {
		ch := l.currentLine[l.linePos]
		if ch != ' ' && ch != '\t' {
			break
		}
		l.linePos++
	}
}

// isAtLineStart checks if we're at the logical start of a line (for instruction detection).
func (l *Lexer) isAtLineStart() bool {
	// Check if everything before current position is whitespace
	for i := 0; i < l.linePos; i++ {
		ch := l.currentLine[i]
		if ch != ' ' && ch != '\t' {
			return false
		}
	}
	return true
}

// scanWord scans a single word (alphanumeric characters).
func (l *Lexer) scanWord() string {
	start := l.linePos
	for l.linePos < len(l.currentLine) {
		ch := rune(l.currentLine[l.linePos])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			break
		}
		l.linePos++
	}
	return l.currentLine[start:l.linePos]
}

// scanArgument scans the rest of the line as an argument, handling quotes and escapes.
func (l *Lexer) scanArgument() string {
	l.skipWhitespace()

	if l.linePos >= len(l.currentLine) {
		return ""
	}

	var result strings.Builder
	inDoubleQuote := false
	inSingleQuote := false

	for l.linePos < len(l.currentLine) {
		ch := l.currentLine[l.linePos]

		// Handle escape sequences
		if ch == '\\' && l.linePos+1 < len(l.currentLine) && !inSingleQuote {
			nextCh := l.currentLine[l.linePos+1]
			// Handle common escape sequences
			switch nextCh {
			case 'n':
				result.WriteByte('\n')
				l.linePos += 2
				continue
			case 't':
				result.WriteByte('\t')
				l.linePos += 2
				continue
			case '"', '\'', '\\', ' ':
				result.WriteByte(nextCh)
				l.linePos += 2
				continue
			default:
				// Keep the backslash and next char as-is
				result.WriteByte(ch)
				l.linePos++
				continue
			}
		}

		// Handle quotes
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			result.WriteByte(ch)
			l.linePos++
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			result.WriteByte(ch)
			l.linePos++
			continue
		}

		// Handle comment start (only if not in quotes)
		if ch == '#' && !inDoubleQuote && !inSingleQuote {
			break
		}

		result.WriteByte(ch)
		l.linePos++
	}

	return strings.TrimSpace(result.String())
}


// Reset resets the lexer to read from a new reader.
func (l *Lexer) Reset(r io.Reader) {
	l.reader = bufio.NewReader(r)
	l.line = 0
	l.column = 0
	l.currentLine = ""
	l.linePos = 0
	l.atEOF = false
	l.peekedToken = nil
}

// CurrentLine returns the current line number being processed.
func (l *Lexer) CurrentLine() int {
	return l.line
}

// Tokenize reads all tokens from the input and returns them as a slice.
// This is useful for testing and debugging.
func (l *Lexer) Tokenize() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF || tok.Type == TokenError {
			break
		}
	}
	return tokens
}

// TokenizeString is a convenience function to tokenize a string.
func TokenizeString(s string) []Token {
	lexer := NewLexer(strings.NewReader(s))
	return lexer.Tokenize()
}
