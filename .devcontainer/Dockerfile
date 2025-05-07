FROM ghcr.io/astral-sh/uv:0.7 AS uv

FROM mcr.microsoft.com/devcontainers/javascript-node:22

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
    clang

RUN npm install --global pnpm@latest-10

COPY --from=uv /uv /uvx /usr/local/bin/
