// Package parser turns Dockerfile source into the docker-lint AST.
//
// Lexing and syntax analysis are delegated to BuildKit's Dockerfile parser -
// the same implementation `docker build` uses - so heredocs, parser directives
// (`# syntax`, `# escape`), custom escape characters, line continuations and
// comments inside continuations all behave exactly as they do in a real build.
// This package only converts BuildKit's generic node tree into the typed AST
// that lint rules consume.
package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/devblac/docker-lint/internal/ast"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/parser"
)

// ignoreDirective matches inline ignore comments.
// Format: # docker-lint ignore: RULE_ID[, RULE_ID...]
var ignoreDirective = regexp.MustCompile(`(?i)#\s*docker-lint\s+ignore:\s*(.+)`)

// ParseError represents a parsing error with location information.
type ParseError struct {
	Line    int
	Column  int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// Parser parses Dockerfile content into an AST.
type Parser struct {
	src io.Reader
}

// NewParser creates a new Parser from an io.Reader.
func NewParser(r io.Reader) *Parser {
	return &Parser{src: r}
}

// Parse parses the Dockerfile and returns the AST. When r is nil the reader
// passed to NewParser is used.
//
// Instructions that cannot be parsed are collected rather than aborting the
// whole file: the returned AST holds everything that did parse, alongside the
// first error encountered.
func (p *Parser) Parse(r io.Reader) (*ast.Dockerfile, error) {
	if r == nil {
		r = p.src
	}
	if r == nil {
		r = strings.NewReader("")
	}

	src, err := io.ReadAll(r)
	if err != nil {
		return nil, &ParseError{Message: err.Error()}
	}

	// BuildKit rejects a file that declares no instructions, but an empty or
	// comment-only Dockerfile is something rules should still be able to
	// inspect, so handle it here instead of surfacing a parse error.
	if !hasInstructions(src) {
		return &ast.Dockerfile{
			Stages:        []ast.Stage{},
			Instructions:  []ast.Instruction{},
			Comments:      scanComments(src, nil),
			InlineIgnores: make(map[int][]string),
		}, nil
	}

	result, err := dockerfile.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, toParseError(err)
	}

	nodes := result.AST.Children

	df := &ast.Dockerfile{
		Stages:        []ast.Stage{},
		Instructions:  []ast.Instruction{},
		Comments:      scanComments(src, nodes),
		InlineIgnores: make(map[int][]string),
	}

	var parseErrors []ParseError
	var currentStage *ast.Stage
	stageIndex := 0

	for _, node := range nodes {
		instrs, err := convertNode(node)
		if err != nil {
			parseErrors = append(parseErrors, ParseError{
				Line:    node.StartLine,
				Message: err.Error(),
			})
			continue
		}

		for _, instr := range instrs {
			df.Instructions = append(df.Instructions, instr)

			// A FROM instruction opens a new build stage.
			if from, ok := instr.(*ast.FromInstruction); ok {
				if currentStage != nil {
					df.Stages = append(df.Stages, *currentStage)
				}
				currentStage = &ast.Stage{
					Name:         from.Alias,
					FromInstr:    from,
					Instructions: []ast.Instruction{from},
					Index:        stageIndex,
				}
				stageIndex++
				continue
			}

			if currentStage != nil {
				currentStage.Instructions = append(currentStage.Instructions, instr)
			}
		}
	}

	if currentStage != nil {
		df.Stages = append(df.Stages, *currentStage)
	}

	df.InlineIgnores = buildInlineIgnores(df.Comments, nodes)

	if len(parseErrors) > 0 {
		return df, &parseErrors[0]
	}
	return df, nil
}

// ParseString is a convenience function to parse a Dockerfile from a string.
func ParseString(s string) (*ast.Dockerfile, error) {
	return NewParser(nil).Parse(strings.NewReader(s))
}

// ParseReader is a convenience function to parse a Dockerfile from an io.Reader.
func ParseReader(r io.Reader) (*ast.Dockerfile, error) {
	return NewParser(nil).Parse(r)
}

// toParseError converts a BuildKit parse error into a ParseError, keeping the
// source location when BuildKit reports one.
func toParseError(err error) *ParseError {
	pe := &ParseError{Message: err.Error()}

	var loc *dockerfile.ErrorLocation
	if errors.As(err, &loc) && len(loc.Locations) > 0 && len(loc.Locations[0]) > 0 {
		start := loc.Locations[0][0].Start
		pe.Line = start.Line
		pe.Column = start.Character
	}

	return pe
}

// hasInstructions reports whether the source holds at least one line that is
// neither blank nor a comment.
func hasInstructions(src []byte) bool {
	for _, raw := range strings.Split(string(src), "\n") {
		line := strings.TrimSpace(raw)
		if line != "" && !strings.HasPrefix(line, "#") {
			return true
		}
	}
	return false
}

// scanComments collects every comment line in the source, skipping lines that
// belong to a heredoc body (where '#' is content, not a comment).
func scanComments(src []byte, nodes []*dockerfile.Node) []ast.Comment {
	skip := heredocLines(nodes)

	var comments []ast.Comment
	for i, raw := range strings.Split(string(src), "\n") {
		lineNum := i + 1
		if skip[lineNum] {
			continue
		}

		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if strings.HasPrefix(line, "#") {
			comments = append(comments, ast.Comment{LineNum: lineNum, Text: line})
		}
	}

	return comments
}

// heredocLines returns the set of line numbers occupied by heredoc bodies.
// A heredoc body is always the tail of its instruction's line range.
func heredocLines(nodes []*dockerfile.Node) map[int]bool {
	lines := make(map[int]bool)
	for _, node := range nodes {
		if len(node.Heredocs) == 0 {
			continue
		}
		for line := node.StartLine + 1; line <= node.EndLine; line++ {
			lines[line] = true
		}
	}
	return lines
}

// buildInlineIgnores maps line numbers to the rule IDs suppressed on them.
// An ignore comment applies to the next instruction that follows it, so blank
// lines and stacked comments between the directive and the instruction do not
// break the association.
func buildInlineIgnores(comments []ast.Comment, nodes []*dockerfile.Node) map[int][]string {
	ignores := make(map[int][]string)

	startLines := make([]int, 0, len(nodes))
	for _, node := range nodes {
		startLines = append(startLines, node.StartLine)
	}
	sort.Ints(startLines)

	for _, comment := range comments {
		ruleIDs := parseIgnoreDirective(comment.Text)
		if len(ruleIDs) == 0 {
			continue
		}

		// Attach to the next instruction; fall back to the following line when
		// the directive trails the last instruction.
		target := comment.LineNum + 1
		if idx := sort.SearchInts(startLines, comment.LineNum+1); idx < len(startLines) {
			target = startLines[idx]
		}

		ignores[target] = append(ignores[target], ruleIDs...)
	}

	return ignores
}

// parseIgnoreDirective extracts rule IDs from an inline ignore comment.
func parseIgnoreDirective(comment string) []string {
	matches := ignoreDirective.FindStringSubmatch(comment)
	if len(matches) < 2 {
		return nil
	}

	var ruleIDs []string
	for _, rule := range strings.Split(matches[1], ",") {
		if trimmed := strings.TrimSpace(rule); trimmed != "" {
			ruleIDs = append(ruleIDs, trimmed)
		}
	}
	return ruleIDs
}
