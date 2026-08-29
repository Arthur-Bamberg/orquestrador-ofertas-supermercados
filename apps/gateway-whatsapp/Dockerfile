# syntax=docker/dockerfile:1
FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gateway-whatsapp ./cmd/gateway-whatsapp

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /gateway-whatsapp /usr/local/bin/gateway-whatsapp
WORKDIR /app
EXPOSE 8090
ENTRYPOINT ["gateway-whatsapp"]
