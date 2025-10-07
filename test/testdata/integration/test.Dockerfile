# FROM gcr.io/distroless/static-debian12
FROM mcr.microsoft.com/devcontainers/base:bookworm

COPY bin/integration/p2pcp /p2pcp
COPY bin/integration/test /test

WORKDIR /data

ENTRYPOINT ["/test"]
