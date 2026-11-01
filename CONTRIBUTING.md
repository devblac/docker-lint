# Contributing to docker-lint

Thank you for your interest in contributing to docker-lint!

## Development Setup

### Prerequisites

- Go 1.22.x or later (matches the `go 1.22.0` directive in `go.mod`)
- Git

### Getting Started

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/[username]/docker-lint.git
   cd docker-lint
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the project:
   ```bash
   go build -o docker-lint ./cmd/docker-lint
   ```

4. Run tests:
   ```bash
   go test ./...
   ```

## Project Structure

```
docker-lint/
├── cmd/docker-lint/     # CLI entry point
├── internal/
│   ├── ast/             # AST data structures
│   ├── parser/          # BuildKit parse result -> AST conversion
│   ├── rules/           # Lint rule implementations
│   ├── analyzer/        # Rule orchestration
│   └── formatter/       # Output formatters
└── testdata/            # Test fixtures
```

## Testing Guidelines

- Write unit tests for new functionality
- Write property-based tests for universal properties
- Use table-driven tests where appropriate
- Include test fixtures in `testdata/`
- After changing a rule or the parser, regenerate the real-world corpus snapshots
  with `go test ./internal/analyzer -run TestCorpus -update-corpus` and review the
  diff: every changed line is a finding that appeared, disappeared or moved on a real
  Dockerfile (see `testdata/corpus/SOURCES.md`)

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/parser/
```

## Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Run `go vet` to catch common issues
- Keep functions small and focused

## Pull Request Process

1. Create a feature branch from `main`
2. Make your changes with clear commit messages
3. Ensure all tests pass
4. Update documentation if needed
5. Submit a pull request with a clear description

## Adding New Rules

1. Create or update a file in `internal/rules/`
2. Implement the `Rule` interface
3. Register the rule in `registry.go`
4. Add unit tests for the rule
5. Update README.md with rule documentation

## Releasing

Releases are built by GoReleaser from a pushed tag and land as a **draft** GitHub
release, so nothing is public until a maintainer publishes it.

1. Make sure `main` is green and `goreleaser release --snapshot --clean` builds locally
   (CI also runs it on every pull request).
2. In `CHANGELOG.md`, rename `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD`, add a new
   empty `## [Unreleased]` above it, update the comparison links at the bottom, point
   the `docker-lint-docker` hook in `.pre-commit-hooks.yaml` and the version examples
   in `README.md` at the new version, and merge that change to `main`. The Release
   workflow fails if the hook names another version's image.
3. Tag the merge commit and push the tag:

   ```bash
   git tag -a vX.Y.Z -m "docker-lint vX.Y.Z"
   git push origin vX.Y.Z
   ```

4. The Release workflow runs the tests, takes the release notes from the version's
   `CHANGELOG.md` section (it fails if there is none) and uploads the binaries and
   `checksums.txt` to a draft release.
5. Review the draft on the Releases page: check its notes, download an archive and
   verify it against `checksums.txt`, then publish it. Avoid `go install ...@vX.Y.Z`
   before publishing: the Go module proxy caches a tag's contents permanently, so the
   tag can no longer be moved if the release needs fixing.
6. Publishing the release runs the Image workflow, which pushes
   `ghcr.io/devblac/docker-lint` for linux/amd64 and linux/arm64. The first time,
   make the package public in its settings on GitHub; packages start out private.

## Questions?

Open an issue for questions or discussions.
