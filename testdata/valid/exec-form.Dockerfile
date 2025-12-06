# Dockerfile using exec form for commands
FROM alpine:3.18

WORKDIR /app

COPY ["src/", "dest/"]

RUN ["apk", "add", "--no-cache", "curl"]

USER nobody

HEALTHCHECK CMD ["wget", "-q", "--spider", "http://localhost/"]

ENTRYPOINT ["./entrypoint.sh"]
CMD ["--help"]
