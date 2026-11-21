#!/bin/bash
# qc2 SEED: stack nhỏ riêng cho các ca SEED (sprint 8, P212…). DB trống qc2i_seed trong ep-r-postgres-1 (KHÔNG đụng DB edupilot),
# redis/mailpit/gateway/worker riêng tên qc2s-*, dùng chung minio + docling của ep-r (docling ep-r đang chạy; bucket qc2s riêng).
# Dùng: qc2s-stack.sh up | down | status | seed [args…] | psql <sql> | rds <args…>
# gateway http://127.0.0.1:18231/api/v1 ; Mailpit API http://127.0.0.1:18233 ; giới hạn tốc độ MẶC ĐỊNH (không ghi đè, theo TC-P212-19).
set -u
NET=ep-r_default; DB=qc2i_seed; PGC=ep-r-postgres-1; REPO=$HOME/Documents/TA_Agent_qc2
GWENV=/tmp/qc2s-gw.env; WKENV=/tmp/qc2s-wk.env
mkenv() {
  docker inspect ep-r-gateway-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -vE '^(AUTH_|RATE_LIMIT_|PGBOUNCER_URL)' | grep -v '^$' \
    | sed -E "s#^DATABASE_URL=(postgres://[^@]+@postgres:5432/)[^?]*#DATABASE_URL=\1$DB#; s#^REDIS_URL=.*#REDIS_URL=redis://qc2s-redis:6379/0#; s#^SMTP_HOST=.*#SMTP_HOST=qc2s-mail#; s#^APP_PUBLIC_URL=.*#APP_PUBLIC_URL=https://localhost#" > $GWENV
  docker inspect ep-r-worker-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -vE '^(AUTH_|RATE_LIMIT_|PGBOUNCER_URL)' | grep -v '^$' \
    | sed -E "s#^DATABASE_URL=(postgres://[^@]+@postgres:5432/)[^?]*#DATABASE_URL=\1$DB#; s#^REDIS_URL=.*#REDIS_URL=redis://qc2s-redis:6379/0#; s#^SMTP_HOST=.*#SMTP_HOST=qc2s-mail#" > $WKENV
  grep -q '^SMTP_HOST=' $GWENV || echo SMTP_HOST=qc2s-mail >> $GWENV
  # LIMITS=dev (mặc định): giới hạn nới như scripts/dev.mjs devLimits; LIMITS=default: giới hạn mặc định (TC-P212-19)
  if [ "${LIMITS:-dev}" = dev ]; then printf 'RATE_LIMIT_IP_PER_MIN=5000\nAUTH_LOGIN_IP_PER_MIN=1000\nAUTH_REGISTER_IP_PER_HOUR=1000\nAUTH_TOKEN_IP_PER_MIN=1000\n' >> $GWENV; fi
}
case ${1:-} in
up)
  free=$(colima ssh -- free -m 2>/dev/null | awk '/^Mem/{print $7}'); echo "available MB: $free"; [ "${free:-0}" -ge 2500 ] || { echo "RAM < 2500 MB: dừng"; exit 3; }
  docker rm -f qc2s-redis qc2s-mail qc2s-worker qc2s-gw >/dev/null 2>&1
  docker exec $PGC psql -U edupilot -d postgres -c "drop database if exists $DB with (force)" -c "create database $DB" || exit 1
  mkenv
  docker run -d --name qc2s-redis --network $NET --memory 128m redis:8 >/dev/null
  docker run -d --name qc2s-mail --network $NET --memory 128m -p 127.0.0.1:18233:8025 axllent/mailpit:v1.27 >/dev/null
  docker run --rm --network $NET --env-file $GWENV edupilot-gateway:latest migrate up | tail -2
  docker run -d --name qc2s-gw --network $NET --env-file $GWENV --memory 512m -v $REPO/seed/demo:/seed/demo:ro -p 127.0.0.1:18231:8080 edupilot-gateway-test >/dev/null
  docker run -d --name qc2s-worker --network $NET --env-file $WKENV --memory 512m -v $REPO/seed/demo:/seed/demo:ro edupilot-worker-test >/dev/null
  docker network connect ep-r_judge_net qc2s-worker 2>/dev/null; docker network connect ep-r_judge_net qc2s-gw 2>/dev/null
  for i in $(seq 60); do curl -sf http://127.0.0.1:18231/api/v1/readyz >/dev/null && break; sleep 2; done; curl -s http://127.0.0.1:18231/api/v1/readyz; echo ;;
seed) shift
  cd $REPO && MOCK_IMAP_URL=http://127.0.0.1:8791 MOCK_GRAPH_URL=http://127.0.0.1:8790 NODE_TLS_REJECT_UNAUTHORIZED=0 API_URL=http://127.0.0.1:18231/api/v1 MAILPIT_URL=http://127.0.0.1:18233 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' SEED_ADMIN_CMD="docker exec -i qc2s-gw /gateway admin create" node scripts/seed.mjs "$@" ;;
down)
  docker rm -f qc2s-redis qc2s-mail qc2s-worker qc2s-gw >/dev/null 2>&1
  docker exec $PGC psql -U edupilot -d postgres -c "drop database if exists $DB with (force)"; rm -f $GWENV $WKENV; echo "qc2s: đã gỡ + DROP DATABASE $DB" ;;
status) docker ps -a --format '{{.Names}} {{.Status}}' | grep qc2s- ; docker exec $PGC psql -U edupilot -d postgres -tAc "select datname from pg_database where datname='$DB'" ;;
psql) shift; docker exec -i $PGC psql -U edupilot -d $DB -At -F '|' -c "$1" ;;
rds) shift; docker exec qc2s-redis redis-cli "$@" ;;
*) echo "up|seed|down|status|psql|rds"; exit 2 ;;
esac
