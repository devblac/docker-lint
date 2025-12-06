# Dockerfile with inline ignore comments
FROM ubuntu
# docker-lint ignore: DL3006
FROM debian

WORKDIR /app

# docker-lint ignore: DL4002
RUN apt-get update

# docker-lint ignore: DL3009,DL3010
RUN apt-get install -y curl

CMD ["./start.sh"]
