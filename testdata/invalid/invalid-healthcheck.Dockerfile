# Invalid: HEALTHCHECK with no CMD
FROM alpine:3.18
WORKDIR /app
HEALTHCHECK
CMD ["./start.sh"]
