#!/bin/bash
# Dựng lại DB trống qc3seed (stack seed phụ của qc3 trên ep-r; KHÔNG đụng DB edupilot): dừng gateway/worker phụ → drop/create → migrate → flush Redis db 13 → chạy lại.
set -e
IMG=$(docker inspect ep-r-gateway-s7seed --format '{{.Config.Image}}')
docker stop ep-r-gateway-s7seed ep-r-worker-s7seed >/dev/null
docker exec ep-r-postgres-1 psql -U edupilot -d postgres -qc "drop database if exists qc3seed with (force)" -c "create database qc3seed"
docker exec ep-r-redis-1 redis-cli -n 13 flushdb >/dev/null
docker inspect ep-r-gateway-s7seed --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -v '^$' > /tmp/qc3-s7seed.env
docker run --rm --network ep-r_default --env-file /tmp/qc3-s7seed.env $IMG migrate up 2>&1 | tail -2; rm -f /tmp/qc3-s7seed.env
docker start ep-r-gateway-s7seed ep-r-worker-s7seed >/dev/null
for i in $(seq 1 30); do curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:18095/healthz | grep -q 200 && break; sleep 2; done
curl -s -w ' health=%{http_code}\n' http://127.0.0.1:18095/healthz
curl -s -X DELETE http://127.0.0.1:18026/api/v1/messages >/dev/null || true
