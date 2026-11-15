#!/bin/bash
# QC2-I r4mail: mini-stack riêng (pg + redis + mailpit + SMTP giả + gateway + worker), không chạm ep-r.
# dùng: qc2i-r4mail-stack.sh up | worker <tên> <smtp_host> [ENV=VAL…] | killw <tên> | down
# Ảnh gateway/worker lấy từ ep-r (edupilot-gateway-test / edupilot-worker-test).
set -u
N=qc2i-net; CTL=/tmp/qc2i-ctl
ENVF=/tmp/qc2i-mail.env
base_env() { cat <<E
APP_ENV=test
DATABASE_URL=postgres://edupilot:edupilot-dev@qc2i-pg:5432/edupilot?sslmode=disable
REDIS_URL=redis://qc2i-redis:6379/0
SMTP_PORT=1025
SMTP_TLS=none
MAIL_FROM=EduPilot <no-reply@edupilot.local>
MAIL_SEND_TIMEOUT=10s
APP_PUBLIC_URL=https://localhost
APP_ENCRYPTION_KEY=ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=
ATTENDANCE_TOKEN_KEY=ZWR1cGlsb3QtZGV2LWF0dGVuZGFuY2UtdG9rZW4ta2U=
JWT_SECRET_KEY=edupilot-dev-jwt-secret-key-change-me-0001
BLOB_ENDPOINT=qc2i-minio:9000
BLOB_ACCESS_KEY=edupilot-dev
BLOB_SECRET_KEY=edupilot-dev-secret
BLOB_BUCKET=edupilot
BLOB_USE_SSL=false
LLM_PROVIDER=fake
AUTH_LOGIN_IP_PER_MIN=1000
AUTH_REGISTER_IP_PER_HOUR=1000
AUTH_TOKEN_IP_PER_MIN=1000
RATE_LIMIT_IP_PER_MIN=5000
CORS_ORIGINS=http://localhost:3000
E
}
case "$1" in
up)
  mkdir -p $CTL; echo ok > $CTL/mode; : > $CTL/log; base_env > $ENVF
  docker network create $N >/dev/null 2>&1
  docker run -d --name qc2i-pg --network $N -e POSTGRES_USER=edupilot -e POSTGRES_PASSWORD=edupilot-dev -e POSTGRES_DB=edupilot pgvector/pgvector:pg18 >/dev/null
  docker run -d --name qc2i-redis --network $N redis:8 >/dev/null
  docker run -d --name qc2i-mailpit --network $N -p 127.0.0.1:18725:8025 -e MP_IGNORE_DUPLICATE_IDS=1 axllent/mailpit:v1.27 >/dev/null
  docker run -d --name qc2i-fakesmtp --network $N -v /tmp/qc2i-fakesmtp:/fakesmtp:ro -v $CTL:/ctl -e CTL=/ctl alpine:latest /fakesmtp >/dev/null
  docker run -d --name qc2i-minio --network $N -e MINIO_ROOT_USER=edupilot-dev -e MINIO_ROOT_PASSWORD=edupilot-dev-secret pgsty/minio:RELEASE.2026-08-04T00-00-00Z server /data >/dev/null 2>&1
  for i in $(seq 60); do docker exec qc2i-pg pg_isready -U edupilot >/dev/null 2>&1 && break; sleep 2; done
  docker run --rm --network $N --env-file $ENVF edupilot-gateway-test migrate up
  docker run -d --name qc2i-gateway --network $N --env-file $ENVF -e SMTP_HOST=qc2i-mailpit -e INSTANCE_ID=qc2i-gateway -p 127.0.0.1:18081:8080 -v $HOME/Documents/TA_Agent_qc2/seed/demo:/seed/demo:ro edupilot-gateway-test >/dev/null
  ;;
worker) # worker <tên> <smtp_host> [ENV=VAL…]
  n=$2; h=$3; shift 3; extra=(); for e in "$@"; do extra+=(-e "$e"); done
  docker rm -f $n >/dev/null 2>&1
  docker run -d --name $n --network $N --env-file $ENVF -e SMTP_HOST=$h -e INSTANCE_ID=$n "${extra[@]}" -v $HOME/Documents/TA_Agent_qc2/seed/demo:/seed/demo:ro edupilot-worker-test
  ;;
killw) docker kill -s KILL $2 ;;
down)
  docker rm -f $(docker ps -aq --filter name='^qc2i-' | tr '\n' ' ') 2>/dev/null
  docker network rm $N 2>/dev/null; rm -rf $CTL $ENVF /tmp/qc2i-fakesmtp
  ;;
esac
