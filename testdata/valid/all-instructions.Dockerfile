# Dockerfile demonstrating all standard instructions
FROM ubuntu:22.04 AS base

# Build arguments
ARG BUILD_VERSION=1.0.0
ARG DEBIAN_FRONTEND=noninteractive

# Environment variables
ENV APP_HOME=/app
ENV APP_VERSION=${BUILD_VERSION}

# Labels
LABEL maintainer="test@example.com"
LABEL version="${BUILD_VERSION}"

# Set working directory
WORKDIR ${APP_HOME}

# Install packages
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Copy files
COPY package.json ./
COPY src/ ./src/

# Add files (archive extraction)
ADD archive.tar.gz /tmp/

# Expose ports
EXPOSE 80
EXPOSE 443/tcp
EXPOSE 8080/udp

# Volume mount points
VOLUME ["/data", "/logs"]

# Set user
USER appuser:appgroup

# Shell configuration
SHELL ["/bin/bash", "-c"]

# Stop signal
STOPSIGNAL SIGTERM

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# On build trigger
ONBUILD RUN echo "Building child image"

# Entry point and command
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["--config", "/etc/app/config.yaml"]
