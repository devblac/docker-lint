# Edge case: Special characters in values
FROM alpine:3.18

ENV SPECIAL_CHARS="hello 'world' \"quoted\" \$var"
ENV PATH_WITH_SPACES="/path/to/some dir/file"
ARG BUILD_ARG="value=with=equals"

LABEL description="A label with special chars: @#$%^&*()"

WORKDIR /app

RUN echo "Testing special chars: !@#$%"

USER nobody

HEALTHCHECK CMD echo "ok"

CMD ["echo", "done"]
