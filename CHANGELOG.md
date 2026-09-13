# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure
- Dockerfile parser with multi-stage build support
- Lint rules for base images, layer optimization, security, and best practices
- CLI with file and stdin input support
- Text and JSON output formats
- Rule ignore configuration (--ignore flag and inline comments)
- Strict mode for CI integration

- `--version` now reports commit and build date (set via `-ldflags "-X main.commit=... -X main.date=..."`)
- `--help` output documents exit codes

### Changed
- CLI logic moved into a testable `run()` function with explicit exit code constants
- Error messages on stderr are prefixed with `Error:`

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- CLI tests imported a stale module path (`github.com/docker-lint/docker-lint`), breaking `go test ./...`

### Security
- N/A
