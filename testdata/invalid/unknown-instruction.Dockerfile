# Invalid: Contains unknown instruction
FROM alpine:3.18
WORKDIR /app
FOOBAR some arguments
CMD ["./start.sh"]
