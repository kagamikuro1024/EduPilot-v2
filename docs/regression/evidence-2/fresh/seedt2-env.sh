#!/bin/bash
# Recreate gateway + worker của qc2seedt2 với env bổ sung: seedt2-env.sh KEY=VAL ... (không KEY = về mặc định). Không đụng stack khác.
cd /Users/kuro/checkouts/qc2seedt2
O=https://localhost:4643
E=(env -i HOME="$HOME" PATH="$PATH" DOCKER_HOST=unix://$HOME/.colima/default/docker.sock COMPOSE_PROJECT_NAME=qc2seedt2 COMPOSE_PROFILES=ingest EP_PORT_OFFSET=4200
  EP_PORT_POSTGRES=9633 EP_PORT_REDIS=10580 EP_PORT_MINIO=13200 EP_PORT_MINIO_CONSOLE=13201 EP_PORT_SMTP=5225 EP_PORT_MAILPIT=12225 EP_PORT_HTTP=4280 EP_PORT_HTTPS=4643 EP_PORT_FRONTEND=7200 EP_PORT_MOCK_GRAPH=12290 EP_PORT_MOCK_IMAP=12291
  RATE_LIMIT_IP_PER_MIN=5000 AUTH_LOGIN_IP_PER_MIN=1000 AUTH_REGISTER_IP_PER_HOUR=1000 AUTH_TOKEN_IP_PER_MIN=1000
  BLOB_PUBLIC_ENDPOINT=localhost:4643 BLOB_PUBLIC_USE_SSL=true NEXT_PUBLIC_API_URL=$O APP_PUBLIC_URL=$O CORS_ORIGINS=$O,http://localhost:7200 "$@")
C=(docker compose --env-file .env.local -f docker-compose.local.yml -p qc2seedt2)
"${E[@]}" "${C[@]}" up -d --no-deps --force-recreate --wait gateway worker 2>&1 | tail -4
