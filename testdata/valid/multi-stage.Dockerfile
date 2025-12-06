# Multi-stage build example
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/main ./cmd/main.go

# Runtime stage
FROM alpine:3.18 AS runtime
WORKDIR /app
COPY --from=builder /app/main .
USER nobody
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["/app/main"]
