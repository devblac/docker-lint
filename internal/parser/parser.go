// Package parser provides lexer and parser for Dockerfile content.
package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/devblac/docker-lint/internal/ast"
)

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
	lexer         *Lexer
	currentToken  Token
	inlineIgnores map[int][]string
	errors        []ParseError
}

// NewParser creates a new Parser from an io.Reader.
func NewParser(r io.Reader) *Parser {
	return &Parser{
		lexer:         NewLexer(r),
		inlineIgnores: make(map[int][]string),
	}
}

// Parse parses the Dockerfile and returns the AST.
func (p *Parser) Parse(r io.Reader) (*ast.Dockerfile, error) {
	p.lexer = NewLexer(r)
	p.inlineIgnores = make(map[int][]string)
	p.errors = nil

	dockerfile := &ast.Dockerfile{
		Stages:        []ast.Stage{},
		Instructions:  []ast.Instruction{},
		Comments:      []ast.Comment{},
		InlineIgnores: make(map[int][]string),
	}

	var currentStage *ast.Stage
	stageIndex := 0

	for {
		p.currentToken = p.lexer.NextToken()

		switch p.currentToken.Type {
		case TokenEOF:
			// Finalize last stage if exists
			if currentStage != nil {
				dockerfile.Stages = append(dockerfile.Stages, *currentStage)
			}
			dockerfile.InlineIgnores = p.inlineIgnores
			if len(p.errors) > 0 {
				return dockerfile, &p.errors[0]
			}
			return dockerfile, nil

		case TokenError:
			return nil, &ParseError{
				Line:    p.currentToken.Line,
				Column:  p.currentToken.Column,
				Message: p.currentToken.Value,
			}

		case TokenComment:
			comment := ast.Comment{
				LineNum: p.currentToken.Line,
				Text:    p.currentToken.Value,
			}
			dockerfile.Comments = append(dockerfile.Comments, comment)
			p.parseInlineIgnore(p.currentToken.Value, p.currentToken.Line)

		case TokenNewline:
			// Skip empty lines
			continue

		case TokenInstruction:
			instr, err := p.parseInstruction()
			if err != nil {
				p.errors = append(p.errors, ParseError{
					Line:    p.currentToken.Line,
					Message: err.Error(),
				})
				p.skipToNextLine()
				continue
			}

			if instr == nil {
				continue
			}

			dockerfile.Instructions = append(dockerfile.Instructions, instr)

			// Handle stages for multi-stage builds
			if fromInstr, ok := instr.(*ast.FromInstruction); ok {
				// Save previous stage if exists
				if currentStage != nil {
					dockerfile.Stages = append(dockerfile.Stages, *currentStage)
				}
				// Start new stage
				currentStage = &ast.Stage{
					Name:         fromInstr.Alias,
					FromInstr:    fromInstr,
					Instructions: []ast.Instruction{fromInstr},
					Index:        stageIndex,
				}
				stageIndex++
			} else if currentStage != nil {
				currentStage.Instructions = append(currentStage.Instructions, instr)
			}

		case TokenArgument:
			// Unexpected argument without instruction
			p.errors = append(p.errors, ParseError{
				Line:    p.currentToken.Line,
				Column:  p.currentToken.Column,
				Message: fmt.Sprintf("unexpected argument without instruction: %s", p.currentToken.Value),
			})
			p.skipToNextLine()
		}
	}
}

// parseInlineIgnore extracts rule IDs from inline ignore comments.
// Format: # docker-lint ignore: RULE_ID[, RULE_ID...]
func (p *Parser) parseInlineIgnore(comment string, line int) {
	// Pattern: # docker-lint ignore: DL3006, DL3007
	re := regexp.MustCompile(`(?i)#\s*docker-lint\s+ignore:\s*(.+)`)
	matches := re.FindStringSubmatch(comment)
	if len(matches) < 2 {
		return
	}

	// Parse comma-separated rule IDs
	rulesPart := matches[1]
	rules := strings.Split(rulesPart, ",")
	var ruleIDs []string
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule != "" {
			ruleIDs = append(ruleIDs, rule)
		}
	}

	if len(ruleIDs) > 0 {
		// The ignore applies to the next line
		p.inlineIgnores[line+1] = append(p.inlineIgnores[line+1], ruleIDs...)
	}
}

// skipToNextLine advances the lexer to the next line.
func (p *Parser) skipToNextLine() {
	for {
		tok := p.lexer.NextToken()
		if tok.Type == TokenNewline || tok.Type == TokenEOF {
			break
		}
	}
}

// parseInstruction parses a single instruction based on its type.
func (p *Parser) parseInstruction() (ast.Instruction, error) {
	instrType := p.currentToken.Value
	line := p.currentToken.Line
	col := p.currentToken.Column

	// Get the argument token
	argToken := p.lexer.NextToken()
	var args string
	if argToken.Type == TokenArgument {
		args = argToken.Value
	} else if argToken.Type != TokenNewline && argToken.Type != TokenEOF {
		return nil, fmt.Errorf("expected argument after %s", instrType)
	}

	rawText := instrType
	if args != "" {
		rawText = instrType + " " + args
	}

	switch instrType {
	case "FROM":
		return p.parseFrom(line, rawText, args)
	case "RUN":
		return p.parseRun(line, rawText, args)
	case "COPY":
		return p.parseCopy(line, rawText, args)
	case "ADD":
		return p.parseAdd(line, rawText, args)
	case "ENV":
		return p.parseEnv(line, rawText, args)
	case "ARG":
		return p.parseArg(line, rawText, args)
	case "EXPOSE":
		return p.parseExpose(line, rawText, args)
	case "WORKDIR":
		return p.parseWorkdir(line, rawText, args)
	case "USER":
		return p.parseUser(line, rawText, args)
	case "LABEL":
		return p.parseLabel(line, rawText, args)
	case "VOLUME":
		return p.parseVolume(line, rawText, args)
	case "CMD":
		return p.parseCmd(line, rawText, args)
	case "ENTRYPOINT":
		return p.parseEntrypoint(line, rawText, args)
	case "HEALTHCHECK":
		return p.parseHealthcheck(line, rawText, args)
	case "SHELL":
		return p.parseShell(line, rawText, args)
	case "STOPSIGNAL":
		return p.parseStopsignal(line, rawText, args)
	case "ONBUILD":
		return p.parseOnbuild(line, rawText, args)
	case "MAINTAINER":
		// Deprecated but still valid - treat as a generic instruction
		return &ast.LabelInstruction{
			LineNum: line,
			RawText: rawText,
			Labels:  map[string]string{"maintainer": args},
		}, nil
	default:
		return nil, fmt.Errorf("unknown instruction: %s at line %d, column %d", instrType, line, col)
	}
}

// parseFrom parses a FROM instruction.
// Format: FROM [--platform=<platform>] <image>[:<tag>|@<digest>] [AS <name>]
func (p *Parser) parseFrom(line int, rawText, args string) (*ast.FromInstruction, error) {
	if args == "" {
		return nil, fmt.Errorf("FROM requires an image argument")
	}

	instr := &ast.FromInstruction{
		LineNum: line,
		RawText: rawText,
	}

	parts := splitArgs(args)
	idx := 0

	// Check for --platform flag
	for idx < len(parts) && strings.HasPrefix(parts[idx], "--platform") {
		if strings.Contains(parts[idx], "=") {
			instr.Platform = strings.SplitN(parts[idx], "=", 2)[1]
		} else if idx+1 < len(parts) {
			idx++
			instr.Platform = parts[idx]
		}
		idx++
	}

	if idx >= len(parts) {
		return nil, fmt.Errorf("FROM requires an image argument")
	}

	// Parse image reference
	imageRef := parts[idx]
	idx++

	// Check for digest (@sha256:...)
	if strings.Contains(imageRef, "@") {
		digestParts := strings.SplitN(imageRef, "@", 2)
		imageRef = digestParts[0]
		instr.Digest = digestParts[1]
	}

	// Check for tag (:tag)
	if strings.Contains(imageRef, ":") {
		tagParts := strings.SplitN(imageRef, ":", 2)
		instr.Image = tagParts[0]
		instr.Tag = tagParts[1]
	} else {
		instr.Image = imageRef
	}

	// Check for AS alias
	for idx < len(parts) {
		if strings.ToUpper(parts[idx]) == "AS" && idx+1 < len(parts) {
			instr.Alias = parts[idx+1]
			break
		}
		idx++
	}

	return instr, nil
}

// parseRun parses a RUN instruction.
func (p *Parser) parseRun(line int, rawText, args string) (*ast.RunInstruction, error) {
	instr := &ast.RunInstruction{
		LineNum: line,
		RawText: rawText,
		Command: args,
		Shell:   !isExecForm(args),
	}
	return instr, nil
}

// parseCopy parses a COPY instruction.
// Format: COPY [--from=<name>] [--chown=<user>:<group>] <src>... <dest>
func (p *Parser) parseCopy(line int, rawText, args string) (*ast.CopyInstruction, error) {
	if args == "" {
		return nil, fmt.Errorf("COPY requires source and destination arguments")
	}

	instr := &ast.CopyInstruction{
		LineNum: line,
		RawText: rawText,
	}

	parts := splitArgs(args)
	var sources []string
	idx := 0

	// Parse flags
	for idx < len(parts) {
		if strings.HasPrefix(parts[idx], "--from=") {
			instr.From = strings.TrimPrefix(parts[idx], "--from=")
			idx++
		} else if strings.HasPrefix(parts[idx], "--chown=") {
			instr.Chown = strings.TrimPrefix(parts[idx], "--chown=")
			idx++
		} else if strings.HasPrefix(parts[idx], "--") {
			// Skip other flags
			idx++
		} else {
			break
		}
	}

	// Remaining parts are sources and destination
	for idx < len(parts) {
		sources = append(sources, parts[idx])
		idx++
	}

	if len(sources) < 2 {
		return nil, fmt.Errorf("COPY requires at least source and destination")
	}

	instr.Sources = sources[:len(sources)-1]
	instr.Dest = sources[len(sources)-1]

	return instr, nil
}

// parseAdd parses an ADD instruction.
// Format: ADD [--chown=<user>:<group>] <src>... <dest>
func (p *Parser) parseAdd(line int, rawText, args string) (*ast.AddInstruction, error) {
	if args == "" {
		return nil, fmt.Errorf("ADD requires source and destination arguments")
	}

	instr := &ast.AddInstruction{
		LineNum: line,
		RawText: rawText,
	}

	parts := splitArgs(args)
	var sources []string
	idx := 0

	// Parse flags
	for idx < len(parts) {
		if strings.HasPrefix(parts[idx], "--chown=") {
			instr.Chown = strings.TrimPrefix(parts[idx], "--chown=")
			idx++
		} else if strings.HasPrefix(parts[idx], "--") {
			// Skip other flags
			idx++
		} else {
			break
		}
	}

	// Remaining parts are sources and destination
	for idx < len(parts) {
		sources = append(sources, parts[idx])
		idx++
	}

	if len(sources) < 2 {
		return nil, fmt.Errorf("ADD requires at least source and destination")
	}

	instr.Sources = sources[:len(sources)-1]
	instr.Dest = sources[len(sources)-1]

	return instr, nil
}

// parseEnv parses an ENV instruction.
// Format: ENV <key>=<value> ... or ENV <key> <value>
func (p *Parser) parseEnv(line int, rawText, args string) (*ast.EnvInstruction, error) {
	if args == "" {
		return nil, fmt.Errorf("ENV requires key and value")
	}

	instr := &ast.EnvInstruction{
		LineNum: line,
		RawText: rawText,
	}

	// Check for key=value format
	if strings.Contains(args, "=") {
		eqIdx := strings.Index(args, "=")
		instr.Key = strings.TrimSpace(args[:eqIdx])
		instr.Value = strings.TrimSpace(args[eqIdx+1:])
	} else {
		// Old format: ENV key value
		parts := splitArgs(args)
		if len(parts) >= 1 {
			instr.Key = parts[0]
		}
		if len(parts) >= 2 {
			instr.Value = strings.Join(parts[1:], " ")
		}
	}

	return instr, nil
}

// parseArg parses an ARG instruction.
// Format: ARG <name>[=<default value>]
func (p *Parser) parseArg(line int, rawText, args string) (*ast.ArgInstruction, error) {
	if args == "" {
		return nil, fmt.Errorf("ARG requires a name")
	}

	instr := &ast.ArgInstruction{
		LineNum: line,
		RawText: rawText,
	}

	if strings.Contains(args, "=") {
		eqIdx := strings.Index(args, "=")
		instr.Name = strings.TrimSpace(args[:eqIdx])
		instr.Default = strings.TrimSpace(args[eqIdx+1:])
	} else {
		instr.Name = strings.TrimSpace(args)
	}

	return instr, nil
}

// parseExpose parses an EXPOSE instruction.
// Format: EXPOSE <port> [<port>/<protocol>...]
func (p *Parser) parseExpose(line int, rawText, args string) (*ast.ExposeInstruction, error) {
	instr := &ast.ExposeInstruction{
		LineNum: line,
		RawText: rawText,
		Ports:   splitArgs(args),
	}
	return instr, nil
}

// parseWorkdir parses a WORKDIR instruction.
// Format: WORKDIR /path/to/workdir
func (p *Parser) parseWorkdir(line int, rawText, args string) (*ast.WorkdirInstruction, error) {
	if args == "" {
		return nil, fmt.Errorf("WORKDIR requires a path")
	}

	instr := &ast.WorkdirInstruction{
		LineNum: line,
		RawText: rawText,
		Path:    args,
	}
	return instr, nil
}

// parseUser parses a USER instruction.
// Format: USER <user>[:<group>]
func (p *Parser) parseUser(line int, rawText, args string) (*ast.UserInstruction, error) {
	if args == "" {
		return nil, fmt.Errorf("USER requires a user")
	}

	instr := &ast.UserInstruction{
		LineNum: line,
		RawText: rawText,
	}

	if strings.Contains(args, ":") {
		parts := strings.SplitN(args, ":", 2)
		instr.User = parts[0]
		instr.Group = parts[1]
	} else {
		instr.User = args
	}

	return instr, nil
}

// parseLabel parses a LABEL instruction.
// Format: LABEL <key>=<value> <key>=<value> ...
func (p *Parser) parseLabel(line int, rawText, args string) (*ast.LabelInstruction, error) {
	instr := &ast.LabelInstruction{
		LineNum: line,
		RawText: rawText,
		Labels:  make(map[string]string),
	}

	// Parse key=value pairs
	pairs := parseKeyValuePairs(args)
	for k, v := range pairs {
		instr.Labels[k] = v
	}

	return instr, nil
}

// parseVolume parses a VOLUME instruction.
// Format: VOLUME ["/data"] or VOLUME /data /data2
func (p *Parser) parseVolume(line int, rawText, args string) (*ast.VolumeInstruction, error) {
	instr := &ast.VolumeInstruction{
		LineNum: line,
		RawText: rawText,
	}

	// Check for JSON array format
	if strings.HasPrefix(strings.TrimSpace(args), "[") {
		var paths []string
		if err := json.Unmarshal([]byte(args), &paths); err == nil {
			instr.Paths = paths
			return instr, nil
		}
	}

	// Shell format
	instr.Paths = splitArgs(args)
	return instr, nil
}

// parseCmd parses a CMD instruction.
// Format: CMD ["executable","param1","param2"] or CMD command param1 param2
func (p *Parser) parseCmd(line int, rawText, args string) (*ast.CmdInstruction, error) {
	instr := &ast.CmdInstruction{
		LineNum: line,
		RawText: rawText,
	}

	if isExecForm(args) {
		instr.Shell = false
		instr.Command = parseExecForm(args)
	} else {
		instr.Shell = true
		instr.Command = []string{args}
	}

	return instr, nil
}

// parseEntrypoint parses an ENTRYPOINT instruction.
// Format: ENTRYPOINT ["executable", "param1", "param2"] or ENTRYPOINT command param1 param2
func (p *Parser) parseEntrypoint(line int, rawText, args string) (*ast.EntrypointInstruction, error) {
	instr := &ast.EntrypointInstruction{
		LineNum: line,
		RawText: rawText,
	}

	if isExecForm(args) {
		instr.Shell = false
		instr.Command = parseExecForm(args)
	} else {
		instr.Shell = true
		instr.Command = []string{args}
	}

	return instr, nil
}

// parseHealthcheck parses a HEALTHCHECK instruction.
// Format: HEALTHCHECK [OPTIONS] CMD command or HEALTHCHECK NONE
func (p *Parser) parseHealthcheck(line int, rawText, args string) (*ast.HealthcheckInstruction, error) {
	instr := &ast.HealthcheckInstruction{
		LineNum: line,
		RawText: rawText,
	}

	args = strings.TrimSpace(args)

	if strings.ToUpper(args) == "NONE" {
		instr.None = true
		return instr, nil
	}

	// Parse options and CMD
	parts := splitArgs(args)
	idx := 0

	for idx < len(parts) {
		part := parts[idx]
		if strings.HasPrefix(part, "--interval=") {
			instr.Interval = strings.TrimPrefix(part, "--interval=")
		} else if strings.HasPrefix(part, "--timeout=") {
			instr.Timeout = strings.TrimPrefix(part, "--timeout=")
		} else if strings.HasPrefix(part, "--retries=") {
			instr.Retries = strings.TrimPrefix(part, "--retries=")
		} else if strings.HasPrefix(part, "--start-period=") {
			instr.Start = strings.TrimPrefix(part, "--start-period=")
		} else if strings.ToUpper(part) == "CMD" {
			// Rest is the command
			remaining := strings.Join(parts[idx+1:], " ")
			if isExecForm(remaining) {
				instr.Command = parseExecForm(remaining)
			} else {
				instr.Command = []string{remaining}
			}
			break
		}
		idx++
	}

	return instr, nil
}

// parseShell parses a SHELL instruction.
// Format: SHELL ["executable", "parameters"]
func (p *Parser) parseShell(line int, rawText, args string) (*ast.ShellInstruction, error) {
	instr := &ast.ShellInstruction{
		LineNum: line,
		RawText: rawText,
	}

	if isExecForm(args) {
		instr.Shell = parseExecForm(args)
	} else {
		instr.Shell = splitArgs(args)
	}

	return instr, nil
}

// parseStopsignal parses a STOPSIGNAL instruction.
// Format: STOPSIGNAL signal
func (p *Parser) parseStopsignal(line int, rawText, args string) (*ast.StopsignalInstruction, error) {
	if args == "" {
		return nil, fmt.Errorf("STOPSIGNAL requires a signal")
	}

	instr := &ast.StopsignalInstruction{
		LineNum: line,
		RawText: rawText,
		Signal:  args,
	}
	return instr, nil
}

// parseOnbuild parses an ONBUILD instruction.
// Format: ONBUILD <INSTRUCTION>
func (p *Parser) parseOnbuild(line int, rawText, args string) (*ast.OnbuildInstruction, error) {
	if args == "" {
		return nil, fmt.Errorf("ONBUILD requires an instruction")
	}

	instr := &ast.OnbuildInstruction{
		LineNum: line,
		RawText: rawText,
	}

	// Parse the wrapped instruction
	parts := splitArgs(args)
	if len(parts) == 0 {
		return nil, fmt.Errorf("ONBUILD requires an instruction")
	}

	instrType := strings.ToUpper(parts[0])
	instrArgs := ""
	if len(parts) > 1 {
		instrArgs = strings.Join(parts[1:], " ")
	}
	innerRaw := instrType
	if instrArgs != "" {
		innerRaw = instrType + " " + instrArgs
	}

	// Create a temporary parser state to parse the inner instruction
	savedToken := p.currentToken
	p.currentToken = Token{Type: TokenInstruction, Value: instrType, Line: line}

	var innerInstr ast.Instruction
	var err error

	switch instrType {
	case "RUN":
		innerInstr, err = p.parseRun(line, innerRaw, instrArgs)
	case "COPY":
		innerInstr, err = p.parseCopy(line, innerRaw, instrArgs)
	case "ADD":
		innerInstr, err = p.parseAdd(line, innerRaw, instrArgs)
	case "ENV":
		innerInstr, err = p.parseEnv(line, innerRaw, instrArgs)
	case "ARG":
		innerInstr, err = p.parseArg(line, innerRaw, instrArgs)
	case "EXPOSE":
		innerInstr, err = p.parseExpose(line, innerRaw, instrArgs)
	case "WORKDIR":
		innerInstr, err = p.parseWorkdir(line, innerRaw, instrArgs)
	case "USER":
		innerInstr, err = p.parseUser(line, innerRaw, instrArgs)
	case "LABEL":
		innerInstr, err = p.parseLabel(line, innerRaw, instrArgs)
	case "VOLUME":
		innerInstr, err = p.parseVolume(line, innerRaw, instrArgs)
	case "CMD":
		innerInstr, err = p.parseCmd(line, innerRaw, instrArgs)
	case "ENTRYPOINT":
		innerInstr, err = p.parseEntrypoint(line, innerRaw, instrArgs)
	case "HEALTHCHECK":
		innerInstr, err = p.parseHealthcheck(line, innerRaw, instrArgs)
	case "SHELL":
		innerInstr, err = p.parseShell(line, innerRaw, instrArgs)
	case "STOPSIGNAL":
		innerInstr, err = p.parseStopsignal(line, innerRaw, instrArgs)
	default:
		err = fmt.Errorf("invalid ONBUILD instruction: %s", instrType)
	}

	p.currentToken = savedToken

	if err != nil {
		return nil, err
	}

	instr.Instruction = innerInstr
	return instr, nil
}

// Helper functions

// splitArgs splits arguments respecting quotes.
func splitArgs(s string) []string {
	var result []string
	var current strings.Builder
	inDoubleQuote := false
	inSingleQuote := false

	s = strings.TrimSpace(s)

	for i := 0; i < len(s); i++ {
		ch := s[i]

		// Handle escape sequences
		if ch == '\\' && i+1 < len(s) && !inSingleQuote {
			current.WriteByte(ch)
			i++
			if i < len(s) {
				current.WriteByte(s[i])
			}
			continue
		}

		// Handle quotes
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			current.WriteByte(ch)
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			current.WriteByte(ch)
			continue
		}

		// Handle whitespace
		if (ch == ' ' || ch == '\t') && !inDoubleQuote && !inSingleQuote {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// isExecForm checks if the argument is in JSON exec form ["cmd", "arg1", ...].
func isExecForm(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
}

// parseExecForm parses a JSON exec form into a string slice.
func parseExecForm(s string) []string {
	s = strings.TrimSpace(s)
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		// Fallback: try to parse manually
		s = strings.TrimPrefix(s, "[")
		s = strings.TrimSuffix(s, "]")
		parts := strings.Split(s, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			part = strings.Trim(part, "\"'")
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}

// parseKeyValuePairs parses key=value pairs from a string.
func parseKeyValuePairs(s string) map[string]string {
	result := make(map[string]string)
	parts := splitArgs(s)

	for _, part := range parts {
		if strings.Contains(part, "=") {
			eqIdx := strings.Index(part, "=")
			key := strings.TrimSpace(part[:eqIdx])
			value := strings.TrimSpace(part[eqIdx+1:])
			// Remove surrounding quotes from value
			value = strings.Trim(value, "\"'")
			result[key] = value
		}
	}

	return result
}

// ParseString is a convenience function to parse a Dockerfile from a string.
func ParseString(s string) (*ast.Dockerfile, error) {
	p := NewParser(strings.NewReader(s))
	return p.Parse(strings.NewReader(s))
}

// ParseReader is a convenience function to parse a Dockerfile from an io.Reader.
func ParseReader(r io.Reader) (*ast.Dockerfile, error) {
	p := &Parser{
		inlineIgnores: make(map[int][]string),
	}
	return p.Parse(r)
}
