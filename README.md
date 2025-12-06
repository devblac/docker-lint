# docker-lint

A minimal, production-grade CLI tool for statically analyzing Dockerfiles to detect common inefficiencies, anti-patterns, and security issues.

## Features

- **Static Analysis**: Purely static analysis - never executes Docker builds
- **Single Binary**: Compiles to a single Go binary with no runtime dependencies
- **CI/CD Ready**: Clear exit codes (0, 1, 2) and JSON output for pipeline integration
- **Configurable**: Ignore specific rules via CLI flags or inline comments
- **Security Focused**: Detects secrets in ENV/ARG without exposing actual values
- **Multi-stage Support**: Correctly analyzes multi-stage Dockerfiles with per-stage rule evaluation
- **Comprehensive Rules**: 17 built-in rules covering base images, layer optimization, security, and best practices

## Installation

### From Source

```bash
go install github.com/docker-lint/docker-lint/cmd/docker-lint@latest
```

### Build from Repository

```bash
git clone https://github.com/docker-lint/docker-lint.git
cd docker-lint
go build -o docker-lint ./cmd/docker-lint
```

### Binary Download

Download the latest release from the [Releases](https://github.com/docker-lint/docker-lint/releases) page.

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
docker-lint --ignore DL3006,DL3008 Dockerfile

# Suppress informational messages
docker-lint --quiet Dockerfile

# List all available rules
docker-lint --rules
```

### Inline Ignores

Disable specific rules for the next line using comments:

```dockerfile
# docker-lint ignore: DL3007
FROM ubuntu:latest

# docker-lint ignore: DL4002
# No USER instruction needed for this build stage
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - no errors found (warnings allowed in non-strict mode) |
| 1 | Lint errors found, or warnings found in strict mode |
| 2 | Fatal error - file not found, permission denied, or parse error |

## Rules

docker-lint includes 17 built-in rules organized into four categories.

### Base Image Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DL3006 | Warning | Missing explicit image tag | Always tag the version of an image explicitly to ensure reproducible builds |
| DL3007 | Warning | Using 'latest' tag | Using 'latest' tag can lead to unpredictable builds as the image may change |
| DL3008 | Warning | Large base image | Consider using a smaller base image variant (slim, alpine) to reduce image size |

### Layer Optimization Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DL3009 | Warning | Package manager cache not cleaned | Clean package manager cache in the same RUN instruction to reduce image size |
| DL3010 | Warning | Consecutive RUN instructions | Combine consecutive RUN instructions to reduce the number of layers |
| DL3011 | Warning | Suboptimal layer ordering | Place instructions that change less frequently earlier to optimize layer caching |
| DL3012 | Warning | Package update without install | Combine package update with install in the same RUN instruction to avoid cache issues |

### Security Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DL4000 | Warning | Potential secret in ENV | Avoid storing secrets in ENV instructions as they persist in the image layers |
| DL4001 | Warning | Potential secret in ARG | Avoid storing secrets in ARG instructions as they are visible in image history |
| DL4002 | Warning | No USER instruction | Containers should not run as root; specify a USER instruction |
| DL4003 | Warning | ADD with URL | Using ADD with URLs is discouraged; use curl or wget in RUN for better control |
| DL4004 | Warning | ADD where COPY would suffice | Use COPY instead of ADD when not extracting archives or fetching URLs |

### Best Practice Rules

| ID | Severity | Name | Description |
|----|----------|------|-------------|
| DL3001 | Warning | Multiple CMD instructions | Only the last CMD instruction takes effect; multiple CMD instructions are likely a mistake |
| DL3002 | Warning | Multiple ENTRYPOINT instructions | Only the last ENTRYPOINT instruction takes effect; multiple ENTRYPOINT instructions are likely a mistake |
| DL3003 | Warning | WORKDIR with relative path | Use absolute paths in WORKDIR to avoid confusion about the current directory |
| DL5000 | Warning | Missing HEALTHCHECK | Add a HEALTHCHECK instruction to enable container health monitoring |
| DL5001 | Info | Wildcard in COPY/ADD source | Wildcard patterns in COPY/ADD may include unnecessary files, increasing build context size |

## Output Formats

### Text (default)

Human-readable output with file path, line number, severity, rule ID, and message:

```
Dockerfile:1:1: [warning] DL3007: Using 'latest' tag for image 'ubuntu' is not recommended
  Suggestion: Pin to a specific version like 'ubuntu:<version>' for reproducible builds
Dockerfile:3:1: [warning] DL3010: Found 2 consecutive RUN instructions that could be combined
  Suggestion: Combine RUN instructions using '&&' to reduce layers
```

### JSON (`--json`)

Machine-readable JSON output for CI/CD integration:

```json
{
  "file": "Dockerfile",
  "findings": [
    {
      "rule_id": "DL3007",
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
git clone https://github.com/docker-lint/docker-lint.git
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
