#!/bin/bash
# COS startup: only gateway-whatsapp. Postgres via Cloud SQL private IP. Secrets via metadata SA.
set -eu
echo "gateway-whatsapp startup begin" >/dev/console
exec >/var/log/gateway-whatsapp-startup.log 2>&1
set +x

PROJECT=ofertas-de-supermercado
DATA=/var/lib/gateway-whatsapp
PGHOST=10.100.0.3
IMAGE_DEFAULT=southamerica-east1-docker.pkg.dev/ofertas-de-supermercado/apps/gateway-whatsapp:ac5268b

until command -v docker >/dev/null && docker info >/dev/null 2>&1; do sleep 2; done
echo "docker ready" >/dev/console

until curl -sf -o /dev/null -H "Metadata-Flavor: Google" \
  http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token; do
  sleep 2
done

IMAGE=$(curl -sf -H "Metadata-Flavor: Google" \
  http://metadata.google.internal/computeMetadata/v1/instance/attributes/gateway-image || true)
IMAGE=${IMAGE:-$IMAGE_DEFAULT}

mkdir -p "$DATA/data/midia"
chmod 700 "$DATA"

TOKEN_JSON=$(curl -sf -H "Metadata-Flavor: Google" \
  http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token)
ACCESS=$(printf '%s' "$TOKEN_JSON" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
if [ -z "$ACCESS" ]; then
  echo "no access token" >/dev/console
  exit 1
fi

fetch_secret() {
  local n=$1 i=0 raw
  while [ "$i" -lt 15 ]; do
    raw=$(curl -sf -H "Authorization: Bearer ${ACCESS}" \
      "https://secretmanager.googleapis.com/v1/projects/${PROJECT}/secrets/${n}/versions/latest:access" || true)
    if [ -n "$raw" ]; then
      printf '%s' "$raw" | sed -n 's/.*"data": *"\([^"]*\)".*/\1/p' | base64 -d
      return 0
    fi
    i=$((i + 1))
    sleep 2
  done
  echo "secret $n failed" >/dev/console
  return 1
}

to_private_pg() {
  # unix-socket URL from Secret Manager → Cloud SQL private IP on the default VPC
  printf '%s' "$1" | sed "s|@localhost/ofertas?host=/cloudsql/[^&]*\\&|@${PGHOST}:5432/ofertas?|"
}

umask 077
{
  printf 'DATABASE_URL=%s\n' "$(to_private_pg "$(fetch_secret database-url)")"
  printf 'GATEWAY_TOKEN=%s\n' "$(fetch_secret gateway-token)"
  printf 'TZ=America/Sao_Paulo\n'
  printf 'HTTP_ADDR=:8090\n'
  printf 'CORS_ORIGIN=https://ofertas-backoffice-3qrcgfe65q-rj.a.run.app\n'
  printf 'WHATSAPP_STUB=0\n'
  printf 'WHATSAPP_SESSION_PATH=/data/whatsmeow.db\n'
  printf 'MIDIA_ROOT=/data/midia\n'
  printf 'WHATSAPP_ALLOWLIST=5551999784248,120363430294212306@g.us\n'
  printf 'WHATSAPP_ACK_TEXTO=Recebi. O ajudante de ofertas ainda não está ligado neste canal.\n'
  printf 'AGENTE_URL=https://agente-ofertas-3qrcgfe65q-rj.a.run.app\n'
} >"$DATA/env"

export HOME="$DATA"
export DOCKER_CONFIG="$DATA/.docker"
mkdir -p "$DOCKER_CONFIG"
printf '%s' "$ACCESS" | docker login -u oauth2accesstoken --password-stdin \
  https://southamerica-east1-docker.pkg.dev >/dev/null

docker pull "$IMAGE"

docker rm -f cloud-sql-proxy gateway-whatsapp >/dev/null 2>&1 || true

docker run -d --name gateway-whatsapp --restart always \
  -p 8090:8090 \
  --env-file "$DATA/env" \
  -v "$DATA/data:/data" \
  "$IMAGE"

echo "gateway-whatsapp container started" >/dev/console
