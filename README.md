# docker-lint

[![CI](https://img.shields.io/github/actions/workflow/status/devblac/docker-lint/ci.yml?branch=main&logo=github)](https://github.com/devblac/docker-lint/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/devblac/docker-lint)](https://goreportcard.com/report/github.com/devblac/docker-lint) [![Go Reference](https://pkg.go.dev/badge/github.com/devblac/docker-lint.svg)](https://pkg.go.dev/github.com/devblac/docker-lint) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE) [![Release](https://img.shields.io/github/v/release/devblac/docker-lint?display_name=release&sort=semver)](https://github.com/devblac/docker-lint/releases)

A minimal, production-grade CLI tool for statically analyzing Dockerfiles to detect common inefficiencies, anti-patterns, and security issues.

## Features

- **Static Analysis**: Purely static analysis - never executes Docker builds
- **Real Dockerfile Syntax**: Parsing is delegated to BuildKit, the parser `docker build` itself uses, so heredocs (`RUN <<EOT`), parser directives (`# syntax`, `# escape`), `RUN --mount` flags and line continuations are understood exactly as Docker understands them
- **Single Binary**: Compiles to a single Go binary with no runtime dependencies
- **CI/CD Ready**: Clear exit codes (0, 1, 2); text, JSON, SARIF and GitHub annotation output; a GitHub Action
- **Whole Repositories**: Lint every Dockerfile under a directory or matching a glob in one run
- **Configurable**: A `.docker-lint.yaml` file, CLI flags or inline comments to ignore rules or change their severity
- **Security Focused**: Detects secrets in ENV/ARG without exposing actual values
- **Multi-stage Support**: Correctly analyzes multi-stage Dockerfiles with per-stage rule evaluation
- **Comprehensive Rules**: 18 built-in rules covering Dockerfile structure, base images, layer optimization, security, and best practices

## Installation

### Binary Download

Prebuilt binaries for Linux, macOS and Windows (amd64 and arm64) are attached to each
[release](https://github.com/devblac/docker-lint/releases), along with a `checksums.txt`.

```bash
# Linux amd64; replace the version and platform as needed
VERSION=0.1.1
curl -sSLO "https://github.com/devblac/docker-lint/releases/download/v${VERSION}/docker-lint_${VERSION}_linux_amd64.tar.gz"
curl -sSLO "https://github.com/devblac/docker-lint/releases/download/v${VERSION}/checksums.txt"
sha256sum --check --ignore-missing checksums.txt
tar -xzf "docker-lint_${VERSION}_linux_amd64.tar.gz" docker-lint
sudo mv docker-lint /usr/local/bin/
```

Archives are named `docker-lint_<version>_<os>_<arch>`, with `.zip` for Windows and
`.tar.gz` elsewhere.

### From Source

Requires Go 1.22 or later (as specified in `go.mod`).

```bash
go install github.com/devblac/docker-lint/cmd/docker-lint@latest
```

### Build from Repository

```bash
git clone https://github.com/devblac/docker-lint.git
cd docker-lint
go build -o docker-lint ./cmd/docker-lint
```

## Usage

```bash
docker-lint [flags] [path ...]
```

Each path is a Dockerfile, a directory, or a glob pattern. A directory is searched recursively for files named
`Dockerfile`, `Containerfile`, `Dockerfile.*` or `*.Dockerfile` (skipping `.git` and `node_modules`, and templates,
backups and docs such as `Dockerfile.template`, `Dockerfile.j2` or `Dockerfile.bak`, which name a file explicitly
to lint). Patterns
support `**` for any number of directories and are expanded by docker-lint itself, so quote them to make them
work the same in every shell, including on Windows. A directory or pattern that matches no Dockerfile is a fatal
error, so a CI job cannot pass without linting anything. With no path, or with `-`, docker-lint reads from stdin.
Flags may come before or after the paths; use `--` before a path that starts with `-`.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--help` | `-h` | Show help message |
| `--version` | `-v` | Show version information |
| `--format <fmt>` | `-f` | Output format: `text` (default), `json`, `sarif`, `github` |
| `--json` | `-j` | Shorthand for `--format json` |
| `--quiet` | `-q` | Suppress informational messages (show only warnings and errors) |
| `--strict` | `-s` | Treat warnings as errors (exit code 1 if any warnings) |
| `--ignore <rules>` | | Comma-separated list of rule IDs to ignore |
| `--config <file>` | `-c` | Configuration file (default: the nearest `.docker-lint.yaml`) |
| `--no-config` | | Do not read a configuration file |
| `--rules` | | List all available rules with descriptions |

### Examples

```bash
# Analyze a Dockerfile
docker-lint Dockerfile

# Analyze every Dockerfile in a repository
docker-lint .

# Analyze Dockerfiles matching a pattern
docker-lint 'services/**/Dockerfile'

# Analyze from stdin
cat Dockerfile | docker-lint

# JSON output for CI integration
docker-lint --json Dockerfile > results.json

# Strict mode for CI (fail on warnings)
docker-lint --strict Dockerfile

# Ignore specific rules
docker-lint --ignore DKL101,DKL103 Dockerfile

# Suppress informational messages
docker-lint --quiet Dockerfile

# List all available rules
docker-lint --rules
```

### Inline Ignores

Disable specific rules for the next line using comments:

```dockerfile
# docker-lint ignore: DKL102
FROM ubuntu:latest

# docker-lint ignore: DKL303
# No USER instruction needed for this build stage
```

### Configuration File

docker-lint reads `.docker-lint.yaml` (or `.docker-lint.yml`) from the current directory, or from the nearest
parent directory up to the repository root. Use `--config` to pick another file and `--no-config` to read none.

```yaml
# Rules that are never reported.
ignore:
  - DKL103

# Change a rule's severity: error, warning, info, or off to disable it.
severity:
  DKL303: error   # fail the build when the final image runs as root
  DKL404: off     # disable the rule

# Fail on warnings too (same as --strict).
strict: true

# Hide informational findings (same as --quiet).
quiet: false

# Skip matching files when a directory or pattern is expanded. Patterns are
# relative to this file's directory; one without a slash matches a file or
# directory name anywhere, as in .gitignore.
exclude:
  - vendor
  - "examples/**"
```

Command-line flags win over the file: `--strict=false` turns off `strict: true`, and `--ignore` adds to the
ignored rules. Unknown keys and invalid severities are fatal errors, so a typo does not silently disable the
configuration. A file named explicitly on the command line is always linted, even if it matches `exclude`.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - no errors found (warnings allowed in non-strict mode) |
| 1 | Lint errors found, or warnings found in strict mode |
| 2 | Fatal error - invalid arguments or configuration, file not found, or parse error |

When several files are linted, one that cannot be read or parsed is reported and the others are still linted;
the exit code is then 2.

## Rules

docker-lint includes 18 built-in rules. Rule IDs use the `DKL` prefix and are grouped by category in hundreds:
`DKL0xx` structure, `DKL1xx` base image, `DKL2xx` layers, `DKL3xx` security, `DKL4xx` best practices.

> **Renamed rule IDs.** Before v0.1.1, the first release, rules used `DLxxxx` IDs, which collide with
> [hadolint](https://github.com/hadolint/hadolint)'s IDs for different rules. The old IDs are not accepted;
> passing one to `--ignore` prints its replacement. See the [CHANGELOG](CHANGELOG.md) for the full mapping.

### Structure Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DKL001 | Error | Instruction before FROM | Every build must start with FROM; only ARG may appear before the first FROM. Docker fails to build such a file |

### Base Image Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DKL101 | Warning | Missing explicit image tag | Always tag the version of an image explicitly to ensure reproducible builds |
| DKL102 | Warning | Using 'latest' tag | Using 'latest' tag can lead to unpredictable builds as the image may change |
| DKL103 | Warning | Large base image | Consider a smaller variant (slim, alpine) for the base of the final image. Build stages are not checked; a final stage built on one is |

### Layer Optimization Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DKL201 | Warning | Package manager cache not cleaned | Clean package manager cache in the same RUN instruction to reduce image size. `apt-get dist-clean` and `--mount=type=cache` count as clean |
| DKL202 | Info | Consecutive RUN instructions | Combine consecutive RUN instructions to reduce the number of layers |
| DKL203 | Warning | Suboptimal layer ordering | Place instructions that change less frequently earlier to optimize layer caching, e.g. copy only the dependency manifests before `npm ci`, `pip install -r` or `go mod download`, not the whole build context |
| DKL204 | Warning | Package update without install | Combine package update with install in the same RUN instruction to avoid cache issues |

### Security Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DKL301 | Warning | Potential secret in ENV | Avoid storing secrets in ENV instructions as they persist in the image layers |
| DKL302 | Warning | Potential secret in ARG | Avoid storing secrets in ARG instructions as they are visible in image history |
| DKL303 | Warning | Final image runs as root | The final image should not run as root; set a non-root USER in the final stage. Build stages are not checked, and a USER inherited from an earlier stage counts |
| DKL304 | Warning | ADD with URL | An unverified download: pin it with `ADD --checksum=sha256:...` or fetch it in RUN. Git repositories (`ADD https://.../repo.git#ref`) are not reported |
| DKL305 | Warning | ADD where COPY would suffice | Use COPY instead of ADD when not extracting archives or fetching URLs |

### Best Practice Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DKL401 | Warning | Multiple CMD instructions | Only the last CMD instruction takes effect; multiple CMD instructions are likely a mistake |
| DKL402 | Warning | Multiple ENTRYPOINT instructions | Only the last ENTRYPOINT instruction takes effect; multiple ENTRYPOINT instructions are likely a mistake |
| DKL403 | Warning | WORKDIR with relative path | Use absolute paths in WORKDIR to avoid confusion about the current directory |
| DKL404 | Info | Missing HEALTHCHECK | Add a HEALTHCHECK instruction to enable container health monitoring (Kubernetes ignores it and uses its own probes) |
| DKL405 | Info | Wildcard in COPY/ADD source | Wildcard patterns in COPY/ADD may include unnecessary files, increasing build context size |

## Output Formats

### Text (default)

Human-readable output with file path, line number, severity, rule ID, and message:

```
Dockerfile:1:1: [warning] DKL102: Using 'latest' tag for image 'ubuntu' is not recommended
  Suggestion: Pin to a specific version like 'ubuntu:<version>' for reproducible builds
Dockerfile:3:1: [info] DKL202: Found 2 consecutive RUN instructions that could be combined
  Suggestion: Combine RUN instructions using '&&' to reduce layers
```

### JSON (`--format json`)

Machine-readable JSON output for CI/CD integration. A single file (or stdin) produces one object; several files
produce an array with one such object per file:

```json
{
  "file": "Dockerfile",
  "findings": [
    {
      "rule_id": "DKL102",
      "severity": "warning",
      "line": 1,
      "column": 1,
      "message": "Using 'latest' tag for image 'ubuntu' is not recommended",
      "suggestion": "Pin to a specific version like 'ubuntu:<version>' for reproducible builds"
    }
  ],
  "summary": {
    "total": 1,
    "errors": 0,
    "warnings": 1,
    "info": 0
  }
}
```

### SARIF (`--format sarif`)

[SARIF 2.1.0](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html), the format GitHub code
scanning and other security dashboards ingest. The report carries every rule's metadata, so findings show
up with their name, description and severity. The exit code is unchanged, so CI still fails on errors while
the report is uploaded.

### GitHub annotations (`--format github`)

GitHub Actions workflow commands, which GitHub renders as annotations on the changed lines of a pull
request. They need no code scanning, so they work in private repositories without GitHub Advanced Security:

```
::warning file=Dockerfile,line=1,col=1,title=DKL102::Using 'latest' tag for image 'ubuntu' is not recommended
```

## CI/CD Integration

The repository's CI workflow runs `go test ./... -cover`. Coverage uploads to Codecov are attempted only when a `CODECOV_TOKEN` secret is configured; otherwise the upload step is skipped while tests still gate the build.

### GitHub Actions

The docker-lint action installs a release binary (checksum-verified), lints your Dockerfiles and annotates
the pull request diff with the findings:

```yaml
- uses: actions/checkout@v7
- uses: devblac/docker-lint@v0.1.1
  with:
    dockerfile: Dockerfile
```

To also see findings in the repository's **Security** tab, upload SARIF to code scanning. This needs the
`security-events: write` permission, and GitHub Advanced Security on private repositories:

```yaml
permissions:
  contents: read
  security-events: write

steps:
  - uses: actions/checkout@v7
  - uses: devblac/docker-lint@v0.1.1
    with:
      dockerfile: |
        Dockerfile
        services/**/Dockerfile
      strict: true
      ignore: DKL404
      upload-sarif: true
```

| Input | Default | Description |
|-------|---------|-------------|
| `dockerfile` | `Dockerfile` | Files, directories or glob patterns to lint, one per line |
| `config` | | Configuration file (default: `.docker-lint.yaml` at the repository root) |
| `version` | action's tag | docker-lint release to install: by default the one matching the action's tag (`@v0.1.1` installs v0.1.1), or the latest release when the action is used from a branch. Also `latest` or a tag such as `v0.1.1` |
| `format` | `github` | Log output: `github` (annotations), `text`, `json` or `sarif` |
| `strict` | `false` | Fail on warnings, not only on errors |
| `quiet` | `false` | Hide informational findings |
| `ignore` | | Comma-separated rule IDs to ignore |
| `sarif-file` | | Also write a SARIF report here (`docker-lint.sarif` when uploading) |
| `upload-sarif` | `false` | Upload the SARIF report to GitHub code scanning |
| `github-token` | `github.token` | Token used to download the docker-lint release |

| Output | Description |
|--------|-------------|
| `exit-code` | docker-lint's exit code: `0` clean, `1` lint errors (or warnings with `strict`), `2` fatal error |
| `sarif-file` | Path of the SARIF report, when one was written |

Pinning the action pins the linter: `@v0.1.1` installs docker-lint v0.1.1. When you pin the action to a commit
SHA, also set `version` to the matching tag; otherwise the latest release is installed.

The action fails the step with docker-lint's exit code and exposes it as the `exit-code` output. A SARIF
report is still written and uploaded when the lint fails, and covers every Dockerfile that could be parsed;
when none could, no report is written rather than an empty one.

### GitLab CI

```yaml
lint-dockerfile:
  script:
    - docker-lint --strict Dockerfile
  allow_failure: false
```

### Jenkins Pipeline

```groovy
stage('Lint Dockerfile') {
    steps {
        sh 'docker-lint --strict Dockerfile'
    }
}
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing guidelines, and contribution process.

### Quick Start for Contributors

```bash
# Clone the repository
git clone https://github.com/devblac/docker-lint.git
cd docker-lint

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o docker-lint ./cmd/docker-lint

# Run linter on itself
./docker-lint testdata/valid/minimal.Dockerfile
```

## License

[MIT](LICENSE)
