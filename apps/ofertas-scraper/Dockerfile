# syntax=docker/dockerfile:1
# Build from the monorepo root: docker build -f apps/ofertas-scraper/Dockerfile .
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY modules/ofertas-store ./modules/ofertas-store
COPY apps/ofertas-scraper ./apps/ofertas-scraper
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -C apps/ofertas-scraper -o /out/ofertas-scraper ./cmd/ofertas-scraper

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata poppler-utils \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/ofertas-scraper /usr/local/bin/ofertas-scraper
COPY apps/ofertas-scraper/prompts /scraper/prompts
COPY apps/ofertas-scraper/schemas /scraper/schemas
COPY apps/ofertas-scraper/seed /scraper/seed
WORKDIR /scraper
ENTRYPOINT ["ofertas-scraper"]
