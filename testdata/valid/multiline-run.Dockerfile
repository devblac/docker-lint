# Dockerfile with multi-line RUN instructions
FROM debian:bullseye-slim

RUN apt-get update && \
    apt-get install -y \
        curl \
        wget \
        git \
        vim \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY . .

USER nobody

HEALTHCHECK CMD curl -f http://localhost/ || exit 1

CMD ["./run.sh"]
