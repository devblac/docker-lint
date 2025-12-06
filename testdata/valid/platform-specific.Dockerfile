# Dockerfile with platform specification
FROM --platform=linux/amd64 golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

FROM --platform=linux/amd64 alpine:3.18
WORKDIR /app
COPY --from=builder /app/main .
USER nobody
HEALTHCHECK CMD ./main health
CMD ["./main", "serve"]
