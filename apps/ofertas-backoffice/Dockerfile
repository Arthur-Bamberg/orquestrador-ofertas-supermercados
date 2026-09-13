# syntax=docker/dockerfile:1
FROM node:22-bookworm AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_BASE=
ARG VITE_GATEWAY_BASE=/gateway
ENV VITE_API_BASE=$VITE_API_BASE
ENV VITE_GATEWAY_BASE=$VITE_GATEWAY_BASE
RUN npm run build

FROM nginx:1.27-alpine
ENV API_UPSTREAM=http://ofertas-api:8080
ENV API_HOST=ofertas-api
ENV GATEWAY_UPSTREAM=http://gateway-whatsapp:8090/
ENV GATEWAY_HOST=gateway-whatsapp
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
