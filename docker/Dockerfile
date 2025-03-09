FROM scratch AS darwin-amd64-base
FROM scratch AS darwin-arm64-base
FROM scratch AS freebsd-amd64-base
FROM --platform=linux/amd64 gcr.io/distroless/static-debian12 AS linux-amd64-base
FROM --platform=linux/arm64 gcr.io/distroless/static-debian12 AS linux-arm64-base
FROM --platform=linux/arm/v7 gcr.io/distroless/static-debian12 AS linux-arm-v7base
FROM scratch AS linux-riscv64-base
FROM scratch AS windows-amd64-base
FROM scratch AS windows-arm64-base

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
FROM ${TARGETOS}-${TARGETARCH}-${TARGETVARIANT}base

ARG TARGETPLATFORM

COPY bin/docker/${TARGETPLATFORM} /

WORKDIR /data

ENTRYPOINT ["/p2pcp"]
