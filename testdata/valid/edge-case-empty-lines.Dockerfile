# Edge case: Multiple empty lines between instructions
FROM alpine:3.18


WORKDIR /app



RUN echo "hello"


USER nobody


HEALTHCHECK CMD echo "ok"


CMD ["./start.sh"]
