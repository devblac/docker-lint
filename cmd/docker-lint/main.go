package main

import (
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

// version is set at build time using -ldflags. Defaults to "dev" when not set.
var version = "dev"

func main() {
	var (
		jsonOutput bool
		quiet      bool
		strict     bool
		versionFlg bool
		rulesFlag  bool
		ignoreCSV  string
	)

	flag.BoolVar(&jsonOutput, "json", false, "Output findings as JSON")
	flag.BoolVar(&jsonOutput, "j", false, "Output findings as JSON")

	flag.BoolVar(&quiet, "quiet", false, "Suppress informational messages (show only warnings and errors)")
	flag.BoolVar(&quiet, "q", false, "Suppress informational messages (show only warnings and errors)")

	flag.BoolVar(&strict, "strict", false, "Treat warnings as errors")
	flag.BoolVar(&strict, "s", false, "Treat warnings as errors")

	flag.BoolVar(&versionFlg, "version", false, "Show version information")
	flag.BoolVar(&versionFlg, "v", false, "Show version information")

	flag.BoolVar(&rulesFlag, "rules", false, "List all available rules with descriptions")

	flag.StringVar(&ignoreCSV, "ignore", "", "Comma-separated list of rule IDs to ignore")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] [file]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if versionFlg {
		fmt.Println(version)
		return
	}

	if rulesFlag {
		listRules()
		return
	}

	args := flag.Args()
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "too many arguments: only one Dockerfile path is supported")
		os.Exit(2)
	}

	filename := "stdin"
	var reader io.Reader = os.Stdin

	if len(args) == 1 {
		filename = args[0]
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open file: %v\n", err)
			os.Exit(2)
		}
		defer file.Close()
		reader = file
	}

	dockerfile, err := parser.ParseReader(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse Dockerfile: %v\n", err)
		os.Exit(2)
	}

	ignoreRules := parseIgnoreList(ignoreCSV)

	anlzr := analyzer.NewWithDefaults(analyzer.Config{IgnoreRules: ignoreRules})
	findings := anlzr.Analyze(dockerfile)

	var errorsCount, warningsCount int
	for _, finding := range findings {
		switch finding.Severity {
		case ast.SeverityError:
			errorsCount++
		case ast.SeverityWarning:
			warningsCount++
		}
	}

	if jsonOutput {
		jsonFormatter := formatter.NewJSONFormatter(filename, quiet)
		if err := jsonFormatter.Format(findings, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "failed to format JSON output: %v\n", err)
			os.Exit(2)
		}
	} else {
		textFormatter := formatter.NewTextFormatter(filename, quiet)
		if err := textFormatter.Format(findings, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "failed to format text output: %v\n", err)
			os.Exit(2)
		}
	}

	if errorsCount > 0 || (strict && warningsCount > 0) {
		os.Exit(1)
	}
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

func listRules() {
	for _, rule := range rules.DefaultRegistry.All() {
		fmt.Printf("%s\t[%s]\t%s - %s\n", rule.ID(), rule.Severity().String(), rule.Name(), rule.Description())
	}
}
