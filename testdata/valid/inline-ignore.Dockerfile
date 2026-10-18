# Dockerfile with inline ignore comments
FROM ubuntu
# docker-lint ignore: DKL101
FROM debian

WORKDIR /app

# docker-lint ignore: DKL303
RUN apt-get update

# docker-lint ignore: DKL201,DKL202
RUN apt-get install -y curl

CMD ["./start.sh"]
