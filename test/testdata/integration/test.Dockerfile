FROM gcr.io/distroless/static-debian12:debug

COPY bin/integration/p2pcp /p2pcp
COPY bin/integration/test /test

WORKDIR /data

ENTRYPOINT ["/test"]
