# syntax=docker/dockerfile:1
FROM node:22-bookworm AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_BASE=http://localhost:8080
ARG VITE_GATEWAY_BASE=http://localhost:8090
ARG VITE_GATEWAY_TOKEN=
ENV VITE_API_BASE=$VITE_API_BASE
ENV VITE_GATEWAY_BASE=$VITE_GATEWAY_BASE
ENV VITE_GATEWAY_TOKEN=$VITE_GATEWAY_TOKEN
RUN npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
