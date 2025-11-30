# docker-lint

A minimal, production-grade CLI tool for statically analyzing Dockerfiles to detect common inefficiencies, anti-patterns, and security issues.

## Features

- **Static Analysis**: Purely static analysis - never executes Docker builds
- **Single Binary**: Compiles to a single Go binary with no runtime dependencies
- **CI/CD Ready**: Clear exit codes and JSON output for pipeline integration
- **Configurable**: Ignore specific rules via flags or inline comments
- **Security Focused**: Detects secrets in ENV/ARG without exposing values

## Installation

### From Source

```bash
go install github.com/[username]/docker-lint/cmd/docker-lint@latest
```

### Binary Download

Download the latest release from the [Releases](https://github.com/[username]/docker-lint/releases) page.

## Usage

```bash
# Analyze a Dockerfile
docker-lint Dockerfile

# Analyze from stdin
cat Dockerfile | docker-lint

# JSON output for CI integration
docker-lint --json Dockerfile

# Strict mode (treat warnings as errors)
docker-lint --strict Dockerfile

# Ignore specific rules
docker-lint --ignore DL3006,DL3008 Dockerfile

# List all available rules
docker-lint --rules
```

### Inline Ignores

Disable specific rules for the next line using comments:

```dockerfile
# docker-lint ignore: DL3007
FROM ubuntu:latest
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (no errors, or warnings only in non-strict mode) |
| 1 | Lint errors found (or warnings in strict mode) |
| 2 | Fatal error (file not found, parse error) |

## Rules

### Base Image Rules (DL3xxx)

| ID | Severity | Description |
|----|----------|-------------|
| DL3006 | Warning | Missing explicit image tag |
| DL3007 | Warning | Using 'latest' tag |
| DL3008 | Warning | Large base image without slim variant |

### Layer Optimization Rules (DL3xxx)

| ID | Severity | Description |
|----|----------|-------------|
| DL3009 | Warning | Package manager cache not cleaned |
| DL3010 | Warning | Consecutive RUN instructions |
| DL3011 | Warning | Suboptimal layer ordering |
| DL3012 | Warning | Package update without install |

### Security Rules (DL4xxx)

| ID | Severity | Description |
|----|----------|-------------|
| DL4000 | Warning | Potential secret in ENV |
| DL4001 | Warning | Potential secret in ARG |
| DL4002 | Warning | No USER instruction (running as root) |
| DL4003 | Warning | ADD with URL (prefer curl/wget) |
| DL4004 | Warning | ADD where COPY would suffice |

### Best Practice Rules (DL5xxx)

| ID | Severity | Description |
|----|----------|-------------|
| DL3001 | Warning | Multiple CMD instructions |
| DL3002 | Warning | Multiple ENTRYPOINT instructions |
| DL3003 | Warning | WORKDIR with relative path |
| DL5000 | Warning | Missing HEALTHCHECK |
| DL5001 | Info | Wildcard in COPY/ADD source |

## Output Formats

### Text (default)

```
Dockerfile:1:1: [warning] DL3007: Using 'latest' tag; pin to a specific version for reproducibility
Dockerfile:3:1: [warning] DL3010: Multiple consecutive RUN instructions; consider combining
```

### JSON (--json)

```json
{
  "file": "Dockerfile",
  "findings": [
    {
      "rule_id": "DL3007",
      "severity": "warning",
      "line": 1,
      "message": "Using 'latest' tag; pin to a specific version for reproducibility"
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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

[MIT](LICENSE)
