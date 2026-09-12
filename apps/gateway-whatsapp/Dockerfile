# syntax=docker/dockerfile:1
# Build from the monorepo root: docker build -f apps/gateway-whatsapp/Dockerfile .
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY modules ./modules
COPY apps/gateway-whatsapp ./apps/gateway-whatsapp
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -C apps/gateway-whatsapp -o /out/gateway-whatsapp ./cmd/gateway-whatsapp

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/gateway-whatsapp /usr/local/bin/gateway-whatsapp
WORKDIR /app
EXPOSE 8090
ENTRYPOINT ["gateway-whatsapp"]
