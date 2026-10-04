// Package main provides the docker-lint CLI entry point.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/devblac/docker-lint/internal/analyzer"
	"github.com/devblac/docker-lint/internal/ast"
	"github.com/devblac/docker-lint/internal/formatter"
	"github.com/devblac/docker-lint/internal/parser"
	"github.com/devblac/docker-lint/internal/rules"
)

// Exit codes returned by the CLI.
const (
	ExitSuccess    = 0 // No errors (warnings allowed in non-strict mode)
	ExitLintErrors = 1 // Lint errors found, or warnings in strict mode
	ExitFatal      = 2 // File not found, invalid arguments, or parse error
)

// Build information, set at build time using -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the CLI with the given arguments and streams, returning the exit code.
// If stdin is nil, os.Stdin is used.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var (
		jsonOutput bool
		quiet      bool
		strict     bool
		versionFlg bool
		rulesFlag  bool
		ignoreCSV  string
	)

	fs := flag.NewFlagSet("docker-lint", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	fs.BoolVar(&jsonOutput, "json", false, "")
	fs.BoolVar(&jsonOutput, "j", false, "")
	fs.BoolVar(&quiet, "quiet", false, "")
	fs.BoolVar(&quiet, "q", false, "")
	fs.BoolVar(&strict, "strict", false, "")
	fs.BoolVar(&strict, "s", false, "")
	fs.BoolVar(&versionFlg, "version", false, "")
	fs.BoolVar(&versionFlg, "v", false, "")
	fs.BoolVar(&rulesFlag, "rules", false, "")
	fs.StringVar(&ignoreCSV, "ignore", "", "")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return ExitSuccess
		}
		fmt.Fprintf(stderr, "Error: %v\n\n", err)
		printUsage(stderr)
		return ExitFatal
	}

	if versionFlg {
		printVersion(stdout)
		return ExitSuccess
	}

	if rulesFlag {
		listRules(stdout)
		return ExitSuccess
	}

	positional := fs.Args()
	if len(positional) > 1 {
		fmt.Fprintln(stderr, "Error: too many arguments: only one Dockerfile path is supported")
		return ExitFatal
	}

	filename := "stdin"
	reader := stdin
	if reader == nil {
		reader = os.Stdin
	}

	if len(positional) == 1 {
		filename = positional[0]
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(stderr, "Error: failed to open file: %v\n", err)
			return ExitFatal
		}
		defer file.Close()
		reader = file
	}

	dockerfile, err := parser.ParseReader(reader)
	if err != nil {
		fmt.Fprintf(stderr, "Error: failed to parse Dockerfile: %v\n", err)
		return ExitFatal
	}

	ignoreRules := parseIgnoreList(ignoreCSV)
	warnUnknownRules(ignoreRules, stderr)

	anlzr := analyzer.NewWithDefaults(analyzer.Config{IgnoreRules: ignoreRules})
	findings := anlzr.Analyze(dockerfile)

	var formatErr error
	if jsonOutput {
		formatErr = formatter.NewJSONFormatter(filename, quiet).Format(findings, stdout)
	} else {
		formatErr = formatter.NewTextFormatter(filename, quiet).Format(findings, stdout)
	}
	if formatErr != nil {
		fmt.Fprintf(stderr, "Error: failed to format output: %v\n", formatErr)
		return ExitFatal
	}

	return determineExitCode(findings, strict)
}

// determineExitCode returns ExitLintErrors if any errors exist, or if warnings
// exist in strict mode; otherwise ExitSuccess.
func determineExitCode(findings []ast.Finding, strict bool) int {
	for _, finding := range findings {
		switch finding.Severity {
		case ast.SeverityError:
			return ExitLintErrors
		case ast.SeverityWarning:
			if strict {
				return ExitLintErrors
			}
		}
	}
	return ExitSuccess
}

func parseIgnoreList(csv string) []string {
	if csv == "" {
		return nil
	}

	parts := strings.Split(csv, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// warnUnknownRules reports --ignore entries that match no rule, which would
// otherwise silently ignore nothing. IDs from before the DKL rename get their
// replacement named.
func warnUnknownRules(ids []string, stderr io.Writer) {
	for _, id := range ids {
		if rules.DefaultRegistry.Get(id) != nil {
			continue
		}
		if replacement, ok := rules.LegacyRuleIDs[id]; ok {
			fmt.Fprintf(stderr, "Warning: rule %s was renamed to %s; --ignore %s has no effect\n", id, replacement, id)
			continue
		}
		fmt.Fprintf(stderr, "Warning: unknown rule %s in --ignore; run --rules to list available rules\n", id)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `docker-lint - static analyzer for Dockerfiles

Usage:
  docker-lint [flags] [file]

Reads from stdin when no file is given.

Flags:
  -h, --help            Show this help message
  -v, --version         Show version information
  -j, --json            Output findings as JSON
  -q, --quiet           Suppress informational messages (show only warnings and errors)
  -s, --strict          Treat warnings as errors
      --ignore <rules>  Comma-separated list of rule IDs to ignore
      --rules           List all available rules with descriptions

Exit Codes:
  0  No errors found (warnings allowed in non-strict mode)
  1  Lint errors found, or warnings found in strict mode
  2  Fatal error (invalid arguments, file not found, parse error)
`)
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "docker-lint version %s (commit %s, built %s)\n", version, commit, date)
}

func listRules(w io.Writer) {
	fmt.Fprintln(w, "Available rules:")
	for _, rule := range rules.DefaultRegistry.All() {
		fmt.Fprintf(w, "  %s\t[%s]\t%s - %s\n", rule.ID(), rule.Severity().String(), rule.Name(), rule.Description())
	}
}
