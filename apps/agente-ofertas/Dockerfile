# syntax=docker/dockerfile:1
# Build from the monorepo root: docker build -f apps/agente-ofertas/Dockerfile .
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY modules/ofertas-store ./modules/ofertas-store
COPY apps/agente-ofertas ./apps/agente-ofertas
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -C apps/agente-ofertas -o /out/agente-ofertas ./cmd/agente-ofertas

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/agente-ofertas /usr/local/bin/agente-ofertas
WORKDIR /app
EXPOSE 8091
ENTRYPOINT ["agente-ofertas"]
