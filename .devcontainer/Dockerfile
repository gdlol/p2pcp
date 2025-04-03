FROM ghcr.io/astral-sh/uv:0.6 AS uv

FROM mcr.microsoft.com/devcontainers/javascript-node:22

RUN npm install --global pnpm@latest-10

COPY --from=uv /uv /uvx /usr/local/bin/
