FROM gcr.io/distroless/static-debian12:c125d4f134136b698c384a600ef1a05804f9a933

COPY bin/integration/p2pcp /p2pcp
COPY bin/integration/test /test

WORKDIR /data

ENTRYPOINT ["/test"]
