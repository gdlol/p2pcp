FROM gcr.io/distroless/static-debian12:8e1d6a6a6eca67afcb1413023f20c8bf368f205e

COPY bin/integration/p2pcp /p2pcp
COPY bin/integration/test /test

WORKDIR /data

ENTRYPOINT ["/test"]
