# Invalid: COPY with no arguments
FROM alpine:3.18
COPY
CMD ["./start.sh"]
