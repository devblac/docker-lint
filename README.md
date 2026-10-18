# docker-lint

[![CI](https://img.shields.io/github/actions/workflow/status/devblac/docker-lint/ci.yml?branch=main&logo=github)](https://github.com/devblac/docker-lint/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/devblac/docker-lint)](https://goreportcard.com/report/github.com/devblac/docker-lint) [![Go Reference](https://pkg.go.dev/badge/github.com/devblac/docker-lint.svg)](https://pkg.go.dev/github.com/devblac/docker-lint) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE) [![Release](https://img.shields.io/github/v/release/devblac/docker-lint?display_name=release&sort=semver)](https://github.com/devblac/docker-lint/releases)

A minimal, production-grade CLI tool for statically analyzing Dockerfiles to detect common inefficiencies, anti-patterns, and security issues.

## Features

- **Static Analysis**: Purely static analysis - never executes Docker builds
- **Real Dockerfile Syntax**: Parsing is delegated to BuildKit, the parser `docker build` itself uses, so heredocs (`RUN <<EOT`), parser directives (`# syntax`, `# escape`), `RUN --mount` flags and line continuations are understood exactly as Docker understands them
- **Single Binary**: Compiles to a single Go binary with no runtime dependencies
- **CI/CD Ready**: Clear exit codes (0, 1, 2) and JSON output for pipeline integration
- **Configurable**: Ignore specific rules via CLI flags or inline comments
- **Security Focused**: Detects secrets in ENV/ARG without exposing actual values
- **Multi-stage Support**: Correctly analyzes multi-stage Dockerfiles with per-stage rule evaluation
- **Comprehensive Rules**: 18 built-in rules covering Dockerfile structure, base images, layer optimization, security, and best practices

## Installation

### Binary Download

Prebuilt binaries for Linux, macOS and Windows (amd64 and arm64) are attached to each
[release](https://github.com/devblac/docker-lint/releases), along with a `checksums.txt`.

```bash
# Linux amd64; replace the version and platform as needed
VERSION=0.1.0
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
docker-lint [flags] [file]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--help` | `-h` | Show help message |
| `--version` | `-v` | Show version information |
| `--json` | `-j` | Output findings as JSON |
| `--quiet` | `-q` | Suppress informational messages (show only warnings and errors) |
| `--strict` | `-s` | Treat warnings as errors (exit code 1 if any warnings) |
| `--ignore <rules>` | | Comma-separated list of rule IDs to ignore |
| `--rules` | | List all available rules with descriptions |

### Examples

```bash
# Analyze a Dockerfile
docker-lint Dockerfile

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

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - no errors found (warnings allowed in non-strict mode) |
| 1 | Lint errors found, or warnings found in strict mode |
| 2 | Fatal error - file not found, permission denied, or parse error |

## Rules

docker-lint includes 18 built-in rules. Rule IDs use the `DKL` prefix and are grouped by category in hundreds:
`DKL0xx` structure, `DKL1xx` base image, `DKL2xx` layers, `DKL3xx` security, `DKL4xx` best practices.

> **Renamed rule IDs.** Before v0.1.0, rules used `DLxxxx` IDs, which collide with
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
| DKL203 | Warning | Suboptimal layer ordering | Place instructions that change less frequently earlier to optimize layer caching |
| DKL204 | Warning | Package update without install | Combine package update with install in the same RUN instruction to avoid cache issues |

### Security Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DKL301 | Warning | Potential secret in ENV | Avoid storing secrets in ENV instructions as they persist in the image layers |
| DKL302 | Warning | Potential secret in ARG | Avoid storing secrets in ARG instructions as they are visible in image history |
| DKL303 | Warning | Final image runs as root | The final image should not run as root; set a non-root USER in the final stage. Build stages are not checked, and a USER inherited from an earlier stage counts |
| DKL304 | Warning | ADD with URL | Using ADD with URLs is discouraged; use curl or wget in RUN for better control |
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
Dockerfile:3:1: [warning] DKL202: Found 2 consecutive RUN instructions that could be combined
  Suggestion: Combine RUN instructions using '&&' to reduce layers
```

### JSON (`--json`)

Machine-readable JSON output for CI/CD integration:

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

## CI/CD Integration

The repository's CI workflow runs `go test ./... -cover`. Coverage uploads to Codecov are attempted only when a `CODECOV_TOKEN` secret is configured; otherwise the upload step is skipped while tests still gate the build.

### GitHub Actions

```yaml
- name: Lint Dockerfile
  run: |
    docker-lint --strict --json Dockerfile > lint-results.json
    if [ $? -ne 0 ]; then
      cat lint-results.json
      exit 1
    fi
```

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
