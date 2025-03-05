FROM gcr.io/distroless/static-debian12

COPY /bin/p2pcp /p2pcp
COPY /bin/test /test

WORKDIR /data

ENTRYPOINT ["/test"]
