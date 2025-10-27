FROM gcr.io/distroless/static-debian12

COPY .local/bin/integration/p2pcp /p2pcp
COPY .local/bin/integration/test /test

WORKDIR /data

ENTRYPOINT ["/test"]
