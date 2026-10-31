# docker-lint container image.
#
#   docker run --rm -i ghcr.io/devblac/docker-lint < Dockerfile
#   docker run --rm -v "$PWD:/work" ghcr.io/devblac/docker-lint .
#
# Built for linux/amd64 and linux/arm64 by .github/workflows/image.yml when a
# release is published. The build stage runs on the build machine's platform
# and cross-compiles, so no emulation is needed.

FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /src

# Download modules in their own layer, so source changes reuse it.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /out/docker-lint ./cmd/docker-lint

# A static binary needs no libc or shell; the nonroot variant runs as UID 65532.
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="docker-lint" \
      org.opencontainers.image.description="Static analyzer for Dockerfiles" \
      org.opencontainers.image.source="https://github.com/devblac/docker-lint" \
      org.opencontainers.image.licenses="MIT"

COPY --from=build /out/docker-lint /usr/local/bin/docker-lint

WORKDIR /work

# docker-lint ignore: DKL404
ENTRYPOINT ["docker-lint"]
