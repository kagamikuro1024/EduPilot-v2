#!/bin/bash
# QC2 r95: gateway + worker + redis riêng trên mạng ep-r_default, DB riêng trong postgres của ep-r. Không đụng container ep-r-*.
#   mini-stack.sh <tên> <db> <cổng host>      ví dụ: mini-stack.sh ep-r95 edupilot_t1_r95 18095
# Env mẫu lấy từ `docker inspect ep-r-{gateway,worker}-1` ghi ra /tmp/qc2-r95-{gateway,worker}.env (có bí mật dev, không commit).
set -e
N=${1:?tên}; DB=${2:?db}; PORT=${3:?cổng}; NET=ep-r_default
U="postgres://edupilot:edupilot-dev@ep-r95-pg:5432/$DB?sslmode=disable"
R=/Users/kuro/Documents/TA_Agent_qc2
mk(){ grep -vE '^(DATABASE_URL|PGBOUNCER_URL|REDIS_URL|HTTP_ADDR|CORS_ORIGINS)=' "$1" > "$2"; printf "SMTP_HOST=ep-r95-mail\n" >> "$2"; printf 'DATABASE_URL=%s\nPGBOUNCER_URL=%s\nREDIS_URL=redis://%s-redis:6379/0\nCORS_ORIGINS=http://127.0.0.1:%s,http://localhost:%s\n' "$U" "$U" "$N" "$PORT" "$PORT" >> "$2"; }
mk /tmp/qc2-r95-gateway.env /tmp/qc2-$N-gw.env; mk /tmp/qc2-r95-worker.env /tmp/qc2-$N-wk.env
docker rm -f $N-redis $N-gateway $N-worker >/dev/null 2>&1 || true
docker run -d --name $N-redis --network $NET redis:8 redis-server --appendonly yes --appendfsync everysec --maxmemory-policy noeviction >/dev/null
docker run -d --name $N-gateway --network $NET --env-file /tmp/qc2-$N-gw.env -p 127.0.0.1:$PORT:8080 --read-only --tmpfs /tmp -v "$R/seed/demo:/seed/demo:ro" edupilot-gateway-test:latest >/dev/null
docker run -d --name $N-worker --network $NET --env-file /tmp/qc2-$N-wk.env --read-only --tmpfs /tmp -v "$R/seed/demo:/seed/demo:ro" edupilot-worker-test:latest >/dev/null
echo started $N
docker network connect ep-r_judge_net $N-worker
