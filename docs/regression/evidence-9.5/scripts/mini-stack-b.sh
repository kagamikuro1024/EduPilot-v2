#!/bin/bash
# QC2 r95b: stack T0 sạch riêng (ep-r95b-*): DB edupilot_t0_r95b trong ep-r95-pg, redis/gateway/worker/mock-imap/mock-graph riêng; dùng chung minio/docling/mail/judge. Không đụng ep-r-*.
set -e
R=/Users/kuro/Documents/TA_Agent_qc2; NET=ep-r_default; N=ep-r95b; DB=edupilot_t0_r95b; PORT=18098
U="postgres://edupilot:edupilot-dev@ep-r95-pg:5432/$DB?sslmode=disable"
docker exec ep-r95-pg psql -U edupilot -d postgres -qc "drop database if exists $DB" -c "create database $DB"
docker exec ep-r95-pg psql -U edupilot -d $DB -qc "create extension if not exists vector" -c "create extension if not exists pg_trgm" 2>&1 | head -2 || true
mk(){ grep -vE '^(DATABASE_URL|PGBOUNCER_URL|REDIS_URL|HTTP_ADDR|CORS_ORIGINS|GRAPH_BASE_URL|GRAPH_AUTH_URL|SMTP_HOST)=' "$1" > "$2"; printf "SMTP_HOST=ep-r95-mail\nGRAPH_BASE_URL=http://$N-mock-graph:8090/v1.0\nGRAPH_AUTH_URL=http://$N-mock-graph:8090\nDATABASE_URL=%s\nPGBOUNCER_URL=%s\nREDIS_URL=redis://$N-redis:6379/0\nCORS_ORIGINS=http://127.0.0.1:$PORT,http://localhost:$PORT\n" "$U" "$U" >> "$2"; }
mk /tmp/qc2-r95-gateway.env /tmp/qc2-$N-gw.env; mk /tmp/qc2-r95-worker.env /tmp/qc2-$N-wk.env
docker rm -f $N-redis $N-gateway $N-worker $N-mock-imap $N-mock-graph >/dev/null 2>&1 || true
docker run -d --name $N-mock-imap --network $NET -p 127.0.0.1:18191:8091 edupilot-mock-imap >/dev/null
docker run -d --name $N-mock-graph --network $NET -p 127.0.0.1:18190:8090 -e MOCK_GRAPH_DATA=/data/mock-graph edupilot-mock-graph >/dev/null
docker run -d --name $N-redis --network $NET redis:8 redis-server --appendonly yes --appendfsync everysec --maxmemory-policy noeviction >/dev/null
docker run --rm --network $NET --env-file /tmp/qc2-$N-gw.env --read-only --tmpfs /tmp edupilot-gateway-test:latest /gateway migrate up 2>&1 | tail -3
docker run -d --name $N-gateway --network $NET --env-file /tmp/qc2-$N-gw.env -p 127.0.0.1:$PORT:8080 --read-only --tmpfs /tmp -v "$R/seed/demo:/seed/demo:ro" edupilot-gateway-test:latest >/dev/null
docker run -d --name $N-worker --network $NET --env-file /tmp/qc2-$N-wk.env --read-only --tmpfs /tmp -v "$R/seed/demo:/seed/demo:ro" edupilot-worker-test:latest >/dev/null
docker network connect ep-r_judge_net $N-worker
echo started $N
