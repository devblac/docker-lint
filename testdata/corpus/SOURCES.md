# Regression corpus

Real-world Dockerfiles used to catch behaviour changes and to measure false
positives. Each `*.Dockerfile` has a `*.Dockerfile.golden` next to it holding the
findings docker-lint currently reports for it; `TestCorpus` in
`internal/analyzer` fails when they drift.

After an intended change to a rule or the parser, regenerate the golden files
and review the diff before committing:

```bash
go test ./internal/analyzer -run TestCorpus -update-corpus
```

## Measurement

Every finding on the corpus was reviewed by hand when it was introduced. The
first review found 21 false positives, all fixed in the same change:

| Rule | False positives | Cause |
|---|---|---|
| DL3008 | 8 | Large images used only to compile in a build stage |
| DL3009 | 6 | `apt-get dist-clean`, used by Debian trixie official images, not recognised |
| DL3009 | 5 | `pip install` under `--mount=type=cache` flagged for a cache that never reaches the layer |
| DL3011 | 2 | A COPY the install step depends on (it runs the copied script) |

The same review recalibrated two rules to `info`, because they fired almost
everywhere: DL5000 (missing HEALTHCHECK) on 51 of 52 files and DL3010
(consecutive RUN) on 39 of 52.

Known remaining false positive, left on purpose: DL3009 flags
`official/python/3.14_windows_windowsservercore-ltsc2022.Dockerfile` because the
text "pip install" appears inside a `Write-Host` message. Telling a command
from a string needs real shell parsing; a regex anchor would miss common forms
such as `DEBIAN_FRONTEND=noninteractive apt-get install`.

DL4002 fires on 16 of the 17 official images. That is accurate - they run as
root - and by design: they are base images whose entrypoints drop privileges at
runtime. Application Dockerfiles are the audience the rule is written for.

## Sources

The files are copied verbatim, pinned to the commits below, and remain under
their original licenses. Only permissively licensed sources are included;
`docker-library/mysql` was left out because it is GPL-2.0.

## `official/` - Docker Official Images

Base-image builds: heavy package installation, GPG verification, multi-variant
(Debian, Alpine, Windows) layouts.

| Directory | Source | Commit | License |
|---|---|---|---|
| `official/python` | [docker-library/python](https://github.com/docker-library/python) | `496ece1bae134063d73efab7ac8cf265fcb855fa` | MIT |
| `official/golang` | [docker-library/golang](https://github.com/docker-library/golang) | `b7049346ed2ad454eb4ad53e336d48ef8d9d9141` | BSD-3-Clause |
| `official/node` | [nodejs/docker-node](https://github.com/nodejs/docker-node) | `2428fc80ba1b241329831eaed2647485d98380b2` | MIT |
| `official/postgres` | [docker-library/postgres](https://github.com/docker-library/postgres) | `d588a44673ea9d123c1acb1a6924de10a27fc315` | MIT |
| `official/httpd` | [docker-library/httpd](https://github.com/docker-library/httpd) | `c5248004fec9acf8c96ecd74fb6f5bd7d53237dc` | Apache-2.0 |
| `official/ruby` | [docker-library/ruby](https://github.com/docker-library/ruby) | `3f45e4c352eff67a95e61c8aa0bec55deee5a8fc` | BSD-2-Clause |
| `official/php` | [docker-library/php](https://github.com/docker-library/php) | `2e430cd1a7e33368cb6917b689bbb5b9bf43c75c` | MIT |

File names flatten the upstream path: `official/python/3.14_slim-trixie.Dockerfile`
is `3.14/slim-trixie/Dockerfile` in the upstream repository.

## `apps/` - Application Dockerfiles

Typical application images (Flask, Django, Go, Node, Spring, Rust, ASP.NET...).

| Directory | Source | Commit | License |
|---|---|---|---|
| `apps` | [docker/awesome-compose](https://github.com/docker/awesome-compose) | `30f4b7f6a6c3b0c0ecf4d4efb0de203c48d11562` | CC0-1.0 |

File names flatten the upstream path: `apps/nginx-golang_backend.Dockerfile` is
`nginx-golang/backend/Dockerfile` in the upstream repository.
