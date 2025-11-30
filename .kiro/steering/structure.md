# Project Structure

```
docker-lint/
├── cmd/docker-lint/     # CLI entry point (main package)
├── internal/
│   ├── ast/             # AST data structures for Dockerfile representation
│   ├── parser/          # Lexer and parser for Dockerfiles
│   ├── rules/           # Lint rule implementations
│   ├── analyzer/        # Rule orchestration and execution
│   └── formatter/       # Output formatters (text, JSON)
├── testdata/            # Test fixtures (sample Dockerfiles)
├── .kiro/
│   ├── specs/           # Feature specifications
│   └── steering/        # AI assistant guidance
└── [root files]         # README, LICENSE, CONTRIBUTING, CHANGELOG
```

## Key Conventions

### Adding New Rules
1. Create/update file in `internal/rules/`
2. Implement the `Rule` interface
3. Register in `registry.go`
4. Add unit tests
5. Document in README.md

### Rule ID Naming
- DL3xxx: Base image and layer optimization rules
- DL4xxx: Security rules
- DL5xxx: Best practice rules

### Internal Package
All core logic lives in `internal/` to prevent external imports and maintain API stability.
