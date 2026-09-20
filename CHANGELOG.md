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
- Support for BuildKit Dockerfile syntax: heredocs (`RUN <<EOT`), parser directives
  (`# syntax`, `# escape`), custom escape characters and `RUN --mount` flags
- Heredoc bodies are analyzed, so rules see commands written inside them

### Changed
- CLI logic moved into a testable `run()` function with explicit exit code constants
- Error messages on stderr are prefixed with `Error:`
- Parsing is delegated to BuildKit's Dockerfile parser (`github.com/moby/buildkit`),
  the same implementation `docker build` uses; the hand-written lexer was removed
- Parse error messages now come from BuildKit and are more precise
- `EXPOSE` ports are normalized (sorted) the way Docker normalizes them

### Deprecated
- N/A

### Removed
- Hand-written Dockerfile lexer (`internal/parser/lexer.go`), superseded by BuildKit

### Fixed
- CLI tests imported a stale module path (`github.com/docker-lint/docker-lint`), breaking `go test ./...`
- `ENV`/`ARG` instructions declaring several variables only exposed the first one,
  so rules such as DL4000 missed every key after it
- A registry port was mistaken for an image tag, so `FROM localhost:5000/app` was
  not reported as missing a tag
- Heredocs and `# escape` directives aborted analysis with a fatal parse error

### Security
- N/A
