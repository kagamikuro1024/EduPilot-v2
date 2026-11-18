#!/bin/bash
# QC2-I r4apia: dựng stack auth RIÊNG (không đụng ep-r): DB qc2i_auth trong ep-r-postgres-1, redis/mailpit/worker/2 gateway riêng tên qc2i-*.
# Dùng: qc2i-r4apia-stack.sh up | down | status | psql <sql> | rds <args…> | gw <1|2> restart …
# Gateway: gw1 :18221 và gw2 :18222 (giới hạn auth MẶC ĐỊNH), gw3 :18224 (giới hạn auth nới 1000 — cho chuỗi khoá/đo nhiều lần); Mailpit API :18223; đều 127.0.0.1.
# Bootstrap: Admin qc2ia-admin@edupilot.local qua `gateway admin create`. DB trống riêng cho TC-P206-30: qc2i_empty (up0).
set -u
REPO=$HOME/Documents/TA_Agent_qc2; NET=ep-r_default; IMG_GW=edupilot-gateway-test; IMG_WK=edupilot-worker-test
DB=qc2i_auth; PGC=ep-r-postgres-1
HEAVY=${HEAVY-/tmp/qc2-i/heavy.sh}  # HEAVY= (rỗng) bỏ qua bộ hạn chế: chỉ dùng cho container nhỏ chạy nền
DBURL="postgres://edupilot:edupilot-dev@postgres:5432/$DB?sslmode=disable"
envs() { # biến chung gateway/worker (tối giản; để trống = mặc định)
  echo -e "APP_ENV=test\nJWT_SECRET_KEY=edupilot-dev-jwt-secret-key-change-me-0001\nAPP_ENCRYPTION_KEY=ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=\nATTENDANCE_TOKEN_KEY=ZWR1cGlsb3QtZGV2LWF0dGVuZGFuY2UtdG9rZW4ta2U=\nDATABASE_URL=$DBURL\nPGBOUNCER_URL=$DBURL\nREDIS_URL=redis://qc2i-redis:6379/0\nBLOB_ENDPOINT=minio:9000\nBLOB_BUCKET=edupilot\nBLOB_ACCESS_KEY=edupilot-dev\nBLOB_SECRET_KEY=edupilot-dev-secret\nBLOB_USE_SSL=false\nBLOB_PUBLIC_ENDPOINT=localhost:9700\nCORS_ORIGINS=http://localhost:3000,https://localhost:1143\nAPP_PUBLIC_URL=https://localhost\nLLM_PROVIDER=fake\nLLM_DEMO_FILE=/seed/demo/fake_llm.json\nTEAMS_ENABLED=false\nSMTP_HOST=qc2i-mail\nSMTP_PORT=1025\nSMTP_TLS=none\nMAIL_FROM=EduPilot <no-reply@edupilot.local>\nGRAPH_AUTH_URL=http://mock-graph:8090\nGRAPH_BASE_URL=http://mock-graph:8090/v1.0\nJUDGE_URL=http://judge:5050\nDOCLING_URL=http://docling:5001"
}
# $1 = tên, $2 = cổng host, thêm tuỳ chọn env qua EXTRA (mảng -e)
gw() { name=$1; port=$2; shift 2; docker rm -f $name >/dev/null 2>&1
  envs > /tmp/qc2i-r4apia.env
  $HEAVY docker run -d --name $name --network $NET --env-file /tmp/qc2i-r4apia.env "$@" -v $REPO/seed/demo:/seed/demo:ro -p 127.0.0.1:$port:8080 $IMG_GW; }
case ${1:-} in
up)
  docker rm -f qc2i-redis qc2i-mail qc2i-worker qc2i-gw1 qc2i-gw2 qc2i-gw3 >/dev/null 2>&1
  $HEAVY docker run -d --name qc2i-redis --network $NET redis:8 >/dev/null
  $HEAVY docker run -d --name qc2i-mail --network $NET -p 127.0.0.1:18223:8025 axllent/mailpit:v1.27 >/dev/null
  docker exec $PGC psql -U edupilot -d postgres -c "drop database if exists $DB with (force)" -c "create database $DB" || exit 1
  envs > /tmp/qc2i-r4apia.env
  ${MIGHEAVY-/tmp/qc2-i/heavy.sh} docker run --rm --network $NET --env-file /tmp/qc2i-r4apia.env edupilot-gateway:latest migrate up | tail -3
  gw qc2i-gw1 18221 >/dev/null
  gw qc2i-gw2 18222 >/dev/null
  gw qc2i-gw3 18224 -e AUTH_LOGIN_IP_PER_MIN=100000 -e AUTH_LOGIN_IP_FAIL_PER_15M=1000000 -e AUTH_REGISTER_IP_PER_HOUR=100000 -e AUTH_FORGOT_IP_PER_HOUR=100000 -e AUTH_TOKEN_IP_PER_MIN=100000 -e AUTH_FORGOT_EMAIL_PER_HOUR=1000 >/dev/null
  $HEAVY docker run -d --name qc2i-worker --network $NET --env-file /tmp/qc2i-r4apia.env -v $REPO/seed/demo:/seed/demo:ro $IMG_WK >/dev/null
  for i in $(seq 60); do curl -sf http://127.0.0.1:18221/api/v1/healthz >/dev/null && curl -sf http://127.0.0.1:18222/api/v1/healthz >/dev/null && curl -sf http://127.0.0.1:18224/api/v1/healthz >/dev/null && { echo ready; break; }; sleep 3; done
  ADMIN_PASSWORD='Qc2ia!Admin-Pw-2026x' $HEAVY docker run --rm --network $NET --env-file /tmp/qc2i-r4apia.env -e ADMIN_PASSWORD edupilot-gateway:latest admin create --email qc2ia-admin@edupilot.local --name 'QC2IA Admin'
  ADMIN_PASSWORD='Qc2ia!Admin-Pw-2026x' $HEAVY docker run --rm --network $NET --env-file /tmp/qc2i-r4apia.env -e ADMIN_PASSWORD edupilot-gateway:latest admin create --email qc2ia-admin2@edupilot.local --name 'QC2IA Admin 2' ;;
up0) # DB trống riêng (TC-P206-30)
  docker exec $PGC psql -U edupilot -d postgres -c "drop database if exists qc2i_empty with (force)" -c "create database qc2i_empty" || exit 1
  $HEAVY docker run --rm --network $NET -e DATABASE_URL="postgres://edupilot:edupilot-dev@postgres:5432/qc2i_empty?sslmode=disable" edupilot-gateway:latest migrate up | tail -2 ;;
down) docker rm -f qc2i-redis qc2i-mail qc2i-worker qc2i-gw1 qc2i-gw2 qc2i-gw3 >/dev/null 2>&1; docker exec $PGC psql -U edupilot -d postgres -c "drop database if exists $DB with (force)" -c "drop database if exists qc2i_empty with (force)" ;;
status) docker ps -a --format '{{.Names}} {{.Status}}' | grep qc2i- ;;
psql) shift; docker exec -i $PGC psql -U edupilot -d $DB -At -F '|' -c "$1" ;;
rds) shift; docker exec qc2i-redis redis-cli "$@" ;;
*) echo "up|down|status|psql|rds"; exit 2 ;;
esac
