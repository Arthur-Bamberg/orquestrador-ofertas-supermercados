# syntax=docker/dockerfile:1
# Build from the monorepo root: docker build -f apps/ofertas-web-scraper/Dockerfile .
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY modules/ofertas-store ./modules/ofertas-store
COPY apps/ofertas-web-scraper ./apps/ofertas-web-scraper
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -C apps/ofertas-web-scraper -o /out/ofertas-web-scraper ./cmd/ofertas-web-scraper

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/ofertas-web-scraper /usr/local/bin/ofertas-web-scraper
COPY apps/ofertas-web-scraper/seed /scraper/seed
WORKDIR /scraper
ENTRYPOINT ["ofertas-web-scraper"]
