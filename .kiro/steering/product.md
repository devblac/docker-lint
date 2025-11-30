# Product Overview

docker-lint is a minimal, production-grade CLI tool for statically analyzing Dockerfiles.

## Purpose
Detect common inefficiencies, anti-patterns, and security issues in Dockerfiles without executing Docker builds.

## Key Characteristics
- Single Go binary with no runtime dependencies
- CI/CD ready with clear exit codes (0=success, 1=lint errors, 2=fatal)
- JSON output for pipeline integration
- Configurable rule ignoring via flags or inline comments
- Security-focused: detects secrets without exposing values

## Target Users
- Developers writing Dockerfiles
- CI/CD pipelines for automated quality gates
- Security teams auditing container configurations
