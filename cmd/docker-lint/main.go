// Package main provides the docker-lint CLI entry point.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/devblac/docker-lint/internal/analyzer"
	"github.com/devblac/docker-lint/internal/ast"
	"github.com/devblac/docker-lint/internal/config"
	"github.com/devblac/docker-lint/internal/files"
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
		noConfig   bool
		ignoreCSV  string
		formatFlag string
		configPath string
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
	fs.BoolVar(&noConfig, "no-config", false, "")
	fs.StringVar(&ignoreCSV, "ignore", "", "")
	fs.StringVar(&formatFlag, "format", "", "")
	fs.StringVar(&formatFlag, "f", "", "")
	fs.StringVar(&configPath, "config", "", "")
	fs.StringVar(&configPath, "c", "", "")

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

	outputFormat, err := resolveFormat(formatFlag, jsonOutput)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitFatal
	}

	if noConfig && configPath != "" {
		fmt.Fprintln(stderr, "Error: --config conflicts with --no-config")
		return ExitFatal
	}
	cfg, cfgPath, err := loadConfig(configPath, noConfig)
	if err != nil {
		fmt.Fprintf(stderr, "Error: invalid configuration: %v\n", err)
		return ExitFatal
	}

	// Flags given on the command line win over the configuration file.
	explicit := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
	if !explicit["strict"] && !explicit["s"] && cfg.Strict != nil {
		strict = *cfg.Strict
	}
	if !explicit["quiet"] && !explicit["q"] && cfg.Quiet != nil {
		quiet = *cfg.Quiet
	}

	cliIgnores := parseIgnoreList(ignoreCSV)
	warnUnknownRules(cliIgnores, "--ignore", stderr)
	if cfgPath != "" {
		warnUnknownRules(cfg.RuleIDs(), cfgPath, stderr)
	}

	var ignoreRules []string
	ignoreRules = append(ignoreRules, cfg.Ignore...)
	ignoreRules = append(ignoreRules, cfg.DisabledRules()...)
	ignoreRules = append(ignoreRules, cliIgnores...)

	anlzr := analyzer.NewWithDefaults(analyzer.Config{
		IgnoreRules:       ignoreRules,
		SeverityOverrides: cfg.SeverityOverrides(),
	})

	// Exclude patterns are relative to the configuration file's directory.
	exclude := files.Exclusions{Patterns: cfg.Exclude}
	if cfgPath != "" {
		exclude.Base = filepath.Dir(cfgPath)
	}

	reports, single, fatal := lintInputs(fs.Args(), exclude, stdin, anlzr, stderr)

	out := output{format: outputFormat, quiet: quiet, tool: toolInfo(cfg)}
	if formatErr := out.write(reports, single, stdout); formatErr != nil {
		fmt.Fprintf(stderr, "Error: failed to format output: %v\n", formatErr)
		return ExitFatal
	}

	if fatal {
		return ExitFatal
	}
	var all []ast.Finding
	for _, r := range reports {
		all = append(all, r.Findings...)
	}
	return determineExitCode(all, strict)
}

// loadConfig returns the configuration to use: the file named by --config,
// none with --no-config, or else the nearest .docker-lint.yaml. It also
// returns the path the configuration came from, "" when there is none.
func loadConfig(path string, disabled bool) (*config.Config, string, error) {
	if disabled {
		return &config.Config{}, "", nil
	}
	if path == "" {
		found, err := config.Find(".")
		if err != nil || found == "" {
			return &config.Config{}, "", err
		}
		path = found
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, "", err
	}
	return cfg, path, nil
}

// lintInputs lints stdin when no path is given, or else every Dockerfile the
// paths expand to. A file that cannot be read or parsed is reported on stderr
// and skipped, and fatal is set; the other files are still linted. single is
// true when the input is stdin or one file named directly.
func lintInputs(paths []string, exclude files.Exclusions, stdin io.Reader, anlzr *analyzer.Analyzer, stderr io.Writer) (reports []formatter.Report, single, fatal bool) {
	if len(paths) == 0 {
		if stdin == nil {
			stdin = os.Stdin
		}
		dockerfile, err := parser.ParseReader(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "Error: failed to parse Dockerfile: %v\n", err)
			return nil, true, true
		}
		return []formatter.Report{{Filename: "stdin", Findings: anlzr.Analyze(dockerfile)}}, true, false
	}

	expanded, err := files.Expand(paths, exclude)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return nil, false, true
	}
	single = len(paths) == 1 && len(expanded) == 1 && expanded[0] == paths[0]

	reports = make([]formatter.Report, 0, len(expanded))
	for _, path := range expanded {
		dockerfile, err := parseFile(path)
		if err != nil {
			if single {
				fmt.Fprintf(stderr, "Error: %v\n", err)
			} else {
				fmt.Fprintf(stderr, "Error: %s: %v\n", path, err)
			}
			fatal = true
			continue
		}
		reports = append(reports, formatter.Report{Filename: path, Findings: anlzr.Analyze(dockerfile)})
	}
	return reports, single, fatal
}

func parseFile(path string) (*ast.Dockerfile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	dockerfile, err := parser.ParseReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Dockerfile: %w", err)
	}
	return dockerfile, nil
}

// Output formats accepted by --format.
const (
	formatText   = "text"
	formatJSON   = "json"
	formatSARIF  = "sarif"
	formatGitHub = "github"
)

// projectURL is reported as the tool's home page in SARIF output.
const projectURL = "https://github.com/devblac/docker-lint"

// resolveFormat combines --format and its --json shorthand into one format.
func resolveFormat(format string, jsonOutput bool) (string, error) {
	if jsonOutput {
		if format != "" && format != formatJSON {
			return "", fmt.Errorf("--json conflicts with --format %s", format)
		}
		return formatJSON, nil
	}

	switch format {
	case "":
		return formatText, nil
	case formatText, formatJSON, formatSARIF, formatGitHub:
		return format, nil
	default:
		return "", fmt.Errorf("unknown output format %q (want text, json, sarif or github)", format)
	}
}

// reportFormatter writes findings in one output format, for a single file
// (Format) or for several (FormatReports).
type reportFormatter interface {
	Format(findings []ast.Finding, w io.Writer) error
	FormatReports(reports []formatter.Report, w io.Writer) error
}

// output writes the lint reports in the chosen format.
type output struct {
	format string
	quiet  bool
	tool   formatter.ToolInfo
}

// write writes the reports. A single input keeps the single-file output shape
// (a JSON object rather than an array); nothing is written when no file could
// be linted, so a fatal error leaves stdout empty.
func (o output) write(reports []formatter.Report, single bool, w io.Writer) error {
	switch {
	case len(reports) == 0:
		return nil
	case single:
		return o.formatter(reports[0].Filename).Format(reports[0].Findings, w)
	default:
		return o.formatter("").FormatReports(reports, w)
	}
}

func (o output) formatter(filename string) reportFormatter {
	switch o.format {
	case formatJSON:
		return formatter.NewJSONFormatter(filename, o.quiet)
	case formatSARIF:
		return formatter.NewSARIFFormatter(filename, o.quiet, o.tool)
	case formatGitHub:
		return formatter.NewGitHubFormatter(filename, o.quiet)
	default:
		return formatter.NewTextFormatter(filename, o.quiet)
	}
}

// toolInfo describes docker-lint and every built-in rule for SARIF output,
// with default severities as changed by the configuration.
func toolInfo(cfg *config.Config) formatter.ToolInfo {
	overrides := cfg.SeverityOverrides()
	all := rules.DefaultRegistry.All()
	infos := make([]formatter.RuleInfo, 0, len(all))
	for _, rule := range all {
		severity := rule.Severity()
		if s, ok := overrides[rule.ID()]; ok {
			severity = s
		}
		infos = append(infos, formatter.RuleInfo{
			ID:          rule.ID(),
			Name:        rule.Name(),
			Description: rule.Description(),
			Severity:    severity,
		})
	}
	return formatter.ToolInfo{
		Name:           "docker-lint",
		Version:        version,
		InformationURI: projectURL,
		Rules:          infos,
	}
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

// warnUnknownRules reports rule IDs from --ignore or the configuration file
// (source) that match no rule, which would otherwise silently do nothing. IDs
// from before the DKL rename get their replacement named.
func warnUnknownRules(ids []string, source string, stderr io.Writer) {
	for _, id := range ids {
		if rules.DefaultRegistry.Get(id) != nil {
			continue
		}
		if replacement, ok := rules.LegacyRuleIDs[id]; ok {
			fmt.Fprintf(stderr, "Warning: rule %s was renamed to %s; %s in %s has no effect\n", id, replacement, id, source)
			continue
		}
		fmt.Fprintf(stderr, "Warning: unknown rule %s in %s; run --rules to list available rules\n", id, source)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `docker-lint - static analyzer for Dockerfiles

Usage:
  docker-lint [flags] [path ...]

Each path is a Dockerfile, a directory searched recursively for Dockerfiles,
or a glob pattern such as "services/**/Dockerfile". Reads from stdin when no
path is given.

Flags:
  -h, --help            Show this help message
  -v, --version         Show version information
  -f, --format <fmt>    Output format: text (default), json, sarif, github
  -j, --json            Shorthand for --format json
  -q, --quiet           Suppress informational messages (show only warnings and errors)
  -s, --strict          Treat warnings as errors
      --ignore <rules>  Comma-separated list of rule IDs to ignore
  -c, --config <file>   Configuration file (default: nearest .docker-lint.yaml)
      --no-config       Do not read a configuration file
      --rules           List all available rules with descriptions

Exit Codes:
  0  No errors found (warnings allowed in non-strict mode)
  1  Lint errors found, or warnings found in strict mode
  2  Fatal error (invalid arguments or configuration, file not found, parse error)
`)
}

func printVersion(w io.Writer) {
	info, _ := debug.ReadBuildInfo()
	v, c, d := buildVersion(info)
	fmt.Fprintf(w, "docker-lint version %s (commit %s, built %s)\n", v, c, d)
}

// buildVersion returns the version, commit and build date to report. Release
// builds set them with -ldflags. Other builds fall back to what Go records in
// the binary: the module version for "go install ...@v0.1.0", and the VCS
// revision and commit time for a build inside a Git checkout.
func buildVersion(info *debug.BuildInfo) (v, c, d string) {
	v, c, d = version, commit, date
	if info == nil {
		return v, c, d
	}
	if v == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		v = strings.TrimPrefix(info.Main.Version, "v")
	}
	for _, setting := range info.Settings {
		switch {
		case setting.Key == "vcs.revision" && c == "none":
			c = setting.Value
		case setting.Key == "vcs.time" && d == "unknown":
			d = setting.Value
		}
	}
	return v, c, d
}

func listRules(w io.Writer) {
	fmt.Fprintln(w, "Available rules:")
	for _, rule := range rules.DefaultRegistry.All() {
		fmt.Fprintf(w, "  %s\t[%s]\t%s - %s\n", rule.ID(), rule.Severity().String(), rule.Name(), rule.Description())
	}
}
