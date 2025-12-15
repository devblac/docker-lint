# Contributing to docker-lint

Thank you for your interest in contributing to docker-lint!

## Development Setup

### Prerequisites

- Go 1.22 or later (matches the `go` directive in `go.mod`)
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
│   ├── parser/          # Lexer and parser
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

## Questions?

Open an issue for questions or discussions.
