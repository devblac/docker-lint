# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.1] - 2026-10-24

First published release. The Changed, Removed and Fixed sections describe
differences from the code on `main` before it, for anyone who built it from
source. Version 0.1.0 was tagged but never published.

### Added
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
- Regression corpus of 52 real-world Dockerfiles (Docker Official Images and
  `docker/awesome-compose`) with golden-file snapshots checked by `TestCorpus`
- DKL001 (error): instructions before the first FROM, or no FROM at all, which
  `docker build` rejects but BuildKit's parser accepts
- `--ignore` warns about rule IDs that match no rule, and names the replacement
  of a pre-rename `DLxxxx` ID
- Release pipeline: prebuilt binaries for Linux, macOS and Windows (amd64 and
  arm64) with checksums, built by GoReleaser from a tag into a draft release;
  CI builds a snapshot release on every pull request
- `--format` flag selecting `text`, `json`, `sarif` or `github`; `--json` stays as
  a shorthand for `--format json`
- SARIF 2.1.0 output for GitHub code scanning and other security dashboards,
  validated against the official schema
- GitHub annotations output, which shows findings on pull request diffs in any
  repository, without code scanning
- GitHub Action (`uses: devblac/docker-lint@<tag>`) that installs a
  checksum-verified release binary, annotates pull requests and can upload
  SARIF to code scanning; tested on Linux, macOS and Windows runners
- Lint several Dockerfiles in one run: pass several paths, a directory (searched
  recursively for Dockerfiles) or a glob pattern with `**`; JSON output becomes
  an array and SARIF a single run covering every file
- `.docker-lint.yaml` configuration file, found in the current directory or a
  parent up to the repository root: ignored rules, per-rule severity overrides
  (including `off`), `strict`, `quiet` and `exclude` patterns; `--config` and
  `--no-config` flags
- The GitHub Action lints several paths or patterns (one per line) and takes a
  `config` input
- `--version` falls back to the version Go records in the binary, so builds from
  `go install github.com/devblac/docker-lint/cmd/docker-lint@v0.1.1` report it
- Flags may follow the paths (`docker-lint Dockerfile --json`), `--` ends the
  flags, and `-` reads stdin
- The GitHub Action installs the docker-lint release matching the tag it is used
  at (`@v0.1.1` installs v0.1.1), so pinning the action pins the linter
- Tests that every rule reports a real-world example of its problem, alongside
  the corpus that guards against false positives

### Changed
- **Breaking:** rule IDs move from `DLxxxx`, which collide with hadolint's IDs for
  different rules, to a `DKL` prefix grouped by category. The old IDs are not
  accepted as aliases. Update `--ignore` flags and `# docker-lint ignore:` comments:

  - Base image: DL3006 → DKL101, DL3007 → DKL102, DL3008 → DKL103
  - Layers: DL3009 → DKL201, DL3010 → DKL202, DL3011 → DKL203, DL3012 → DKL204
  - Security: DL4000 → DKL301, DL4001 → DKL302, DL4002 → DKL303, DL4003 → DKL304,
    DL4004 → DKL305
  - Best practices: DL3001 → DKL401, DL3002 → DKL402, DL3003 → DKL403,
    DL5000 → DKL404, DL5001 → DKL405

- CLI logic moved into a testable `run()` function with explicit exit code constants
- Error messages on stderr are prefixed with `Error:`
- Parsing is delegated to BuildKit's Dockerfile parser (`github.com/moby/buildkit`),
  the same implementation `docker build` uses; the hand-written lexer was removed
- Parse error messages now come from BuildKit and are more precise
- `EXPOSE` ports are normalized (sorted) the way Docker normalizes them
- DKL303 now judges only the final stage, follows `FROM <stage>` to inherit an
  earlier stage's USER, and is renamed "Final image runs as root"
- DKL301/DKL302 match variable names word by word instead of by substring
- DKL404 (missing HEALTHCHECK) and DKL202 (consecutive RUN) are now `info`: on the
  corpus they fired on 51 and 39 of 52 files, and Kubernetes ignores HEALTHCHECK
- DKL103 judges only the base of the final image
- DKL203 reports the whole build context copied before installing dependencies
  (`COPY . .` then `npm ci`, `pip install -r` or `go mod download`), and
  recognises many more installers (npm ci, pnpm, bun, yarn, uv, poetry, pipenv,
  bundle, composer, cargo fetch, dotnet restore, mvn dependency:go-offline)
- Directory searches skip Dockerfile templates, backups and docs
  (`Dockerfile.template`, `.j2`, `.bak`, `.md`); name such a file to lint it

### Removed
- Hand-written Dockerfile lexer (`internal/parser/lexer.go`), superseded by BuildKit

### Fixed
- CLI tests imported a stale module path (`github.com/docker-lint/docker-lint`), breaking `go test ./...`
- `ENV`/`ARG` instructions declaring several variables only exposed the first one,
  so rules such as DKL301 missed every key after it
- A registry port was mistaken for an image tag, so `FROM localhost:5000/app` was
  not reported as missing a tag
- Heredocs and `# escape` directives aborted analysis with a fatal parse error
- DKL103 flagged large images used only to compile in a build stage
- DKL201 did not recognise `apt-get dist-clean` (Debian trixie images) or
  `--mount=type=cache`, which keeps the package cache out of the layer
- DKL203 flagged a COPY whose files the following install step uses
- DKL303 flagged every build stage without USER, although only the final image
  ships, and missed a final `USER root`
- DKL303 flagged base images that run as non-root by default (distroless `:nonroot`)
- DKL301/DKL302 flagged names that merely contain a secret word (`TOKENIZER_MODEL`),
  that describe a secret (`PASSWORD_MIN_LENGTH`, `TOKEN_URL`), the Docker secrets
  `_FILE` convention (`POSTGRES_PASSWORD_FILE`), and empty `ENV` placeholders
- DKL101/DKL102/DKL103 flagged `FROM <earlier stage>` as an untagged image
- DKL101 flagged `FROM ${BASE}`, whose tag is only known at build time; DKL103
  likewise skips variable tags
- DKL203 flagged the recommended pattern of copying dependency manifests first
  (`COPY go.mod go.sum ./`, `COPY package*.json ./`) because it inspected the
  destination instead of the sources
- DKL304 flagged `ADD --checksum=...`, which Docker recommends for remote files,
  and `ADD` of a Git repository; DKL305 said COPY could replace `ADD git@...`
- DKL201 ignored `PIP_NO_CACHE_DIR` set with ENV or for the command
- `exclude` patterns are relative to the configuration file, so they also apply
  when docker-lint runs from a subdirectory or is given absolute paths

## [0.1.0] - 2026-10-24

Tagged but never published. A review of the draft release found the problems
fixed in 0.1.1, and the tag could not be moved because the Go module proxy had
already cached it. Use 0.1.1 instead.

[Unreleased]: https://github.com/devblac/docker-lint/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/devblac/docker-lint/releases/tag/v0.1.1
[0.1.0]: https://github.com/devblac/docker-lint/tree/v0.1.0
