# Tech Stack

## Language & Runtime
- Go 1.21+
- Single binary compilation, no runtime dependencies

## Build System
Go modules (standard Go toolchain)

## Common Commands

```bash
# Install dependencies
go mod download

# Build binary
go build -o docker-lint ./cmd/docker-lint

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/parser/

# Format code
go fmt ./...

# Static analysis
go vet ./...
```

## Code Style
- Standard Go conventions
- Run `go fmt` before committing
- Run `go vet` to catch common issues
- Keep functions small and focused

## Testing Approach
- Unit tests for all functionality
- Property-based tests for universal properties
- Table-driven tests where appropriate
- Test fixtures in `testdata/`
