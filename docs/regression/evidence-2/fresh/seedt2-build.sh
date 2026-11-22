#!/bin/bash
# Dựng image qc2seedt2 từng service (KHÔNG start). Env giống dev.mjs cho offset 4200.
cd /Users/kuro/checkouts/qc2seedt2
L=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-2/fresh/seedt2
O=https://localhost:4643
E=(env -i HOME="$HOME" PATH="$PATH" DOCKER_HOST=unix://$HOME/.colima/default/docker.sock COMPOSE_PROJECT_NAME=qc2seedt2 COMPOSE_PROFILES=ingest EP_PORT_OFFSET=4200
  EP_PORT_POSTGRES=9633 EP_PORT_REDIS=10580 EP_PORT_MINIO=13200 EP_PORT_MINIO_CONSOLE=13201 EP_PORT_SMTP=5225 EP_PORT_MAILPIT=12225 EP_PORT_HTTP=4280 EP_PORT_HTTPS=4643 EP_PORT_FRONTEND=7200 EP_PORT_MOCK_GRAPH=12290 EP_PORT_MOCK_IMAP=12291
  RATE_LIMIT_IP_PER_MIN=5000 AUTH_LOGIN_IP_PER_MIN=1000 AUTH_REGISTER_IP_PER_HOUR=1000 AUTH_TOKEN_IP_PER_MIN=1000
  BLOB_PUBLIC_ENDPOINT=localhost:4643 BLOB_PUBLIC_USE_SSL=true NEXT_PUBLIC_API_URL=$O APP_PUBLIC_URL=$O CORS_ORIGINS=$O,http://localhost:7200)
C=(docker compose --env-file .env.local -f docker-compose.local.yml -p qc2seedt2)
svcs=$("${E[@]}" "${C[@]}" config --format json | python3 -c "import sys,json; print(' '.join(k for k,v in json.load(sys.stdin)['services'].items() if 'build' in v))")
echo "services có build: $svcs" > $L/build.log
for s in $svcs; do
  free=$(colima ssh -- free -m | awk 'NR==2{print $7}'); echo "$(date +%H:%M:%S) build $s (available ${free} MB)" >> $L/build.log
  if [ "$free" -lt 2500 ]; then echo "THIẾU RAM, dừng" >> $L/build.log; exit 1; fi
  "${E[@]}" "${C[@]}" build $s >> $L/build.log 2>&1; echo "$(date +%H:%M:%S) $s rc=$?" >> $L/build.log
done
echo BUILD_DONE >> $L/build.log
