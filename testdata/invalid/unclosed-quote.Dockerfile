# Invalid: Unclosed quote in ENV
FROM alpine:3.18
ENV MESSAGE="Hello World
WORKDIR /app
CMD ["./start.sh"]
