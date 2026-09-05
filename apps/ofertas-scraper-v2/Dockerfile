# syntax=docker/dockerfile:1
# Build from the monorepo root: docker build -f apps/ofertas-scraper-v2/Dockerfile .
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY apps/ofertas-scraper-v2 ./apps/ofertas-scraper-v2
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -C apps/ofertas-scraper-v2 -o /out/ofertas-scraper-v2 ./cmd/ofertas-scraper-v2

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/ofertas-scraper-v2 /usr/local/bin/ofertas-scraper-v2
COPY apps/ofertas-scraper-v2/seed /scraper/seed
WORKDIR /scraper
ENTRYPOINT ["ofertas-scraper-v2"]
