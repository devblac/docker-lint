# Large Dockerfile with many instructions
FROM ubuntu:22.04 AS base

ARG DEBIAN_FRONTEND=noninteractive
ARG BUILD_DATE
ARG VERSION=1.0.0
ARG VCS_REF

LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${VCS_REF}"
LABEL org.opencontainers.image.title="Large Application"
LABEL org.opencontainers.image.description="A large application with many dependencies"
LABEL org.opencontainers.image.vendor="Example Corp"

ENV APP_HOME=/opt/app
ENV APP_USER=appuser
ENV APP_GROUP=appgroup
ENV LOG_LEVEL=info
ENV CONFIG_PATH=/etc/app/config.yaml

WORKDIR ${APP_HOME}

RUN groupadd -r ${APP_GROUP} && \
    useradd -r -g ${APP_GROUP} -d ${APP_HOME} -s /sbin/nologin ${APP_USER} && \
    mkdir -p /etc/app /var/log/app /var/lib/app && \
    chown -R ${APP_USER}:${APP_GROUP} ${APP_HOME} /etc/app /var/log/app /var/lib/app

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    && rm -rf /var/lib/apt/lists/*

COPY --chown=${APP_USER}:${APP_GROUP} config/ /etc/app/
COPY --chown=${APP_USER}:${APP_GROUP} scripts/ ${APP_HOME}/scripts/
COPY --chown=${APP_USER}:${APP_GROUP} bin/ ${APP_HOME}/bin/

EXPOSE 8080
EXPOSE 8443
EXPOSE 9090

VOLUME ["/var/log/app", "/var/lib/app"]

USER ${APP_USER}

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

STOPSIGNAL SIGTERM

ENTRYPOINT ["./bin/entrypoint.sh"]
CMD ["--config", "/etc/app/config.yaml", "--log-level", "info"]
