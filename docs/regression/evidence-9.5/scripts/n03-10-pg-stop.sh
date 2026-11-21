#!/bin/bash
# TC-N03-10 (SmkS4, 2026-11-21): seed T1 chạy, dừng Postgres giữa chừng → seed phải thoát ≠ 0, in tên bước/bảng, không treo, không đụng DB `edupilot`.
# Dùng postgres của stack smk91 (qc2 sở hữu, dừng hẳn sau khi chạy): DB edupilot_t1 tạo từ template edupilot (có admin).  Chạy từ gốc repo.
set -u
PG=smk91-postgres-1; STOP_AFTER=${STOP_AFTER:-25}
P() { docker exec -i $PG psql -U edupilot -At "$@"; }
P -d postgres -c "drop database if exists edupilot_t1" -c "create database edupilot_t1 template edupilot" >/dev/null
U0=$(P -d edupilot -c "select count(*) from users"); echo "users(edupilot) trước: $U0"
L=docs/regression/evidence-9.5/rerun/n03-10-seed-$(date +%H%M).log
T1_PSQL="docker exec -i $PG psql -U edupilot -d edupilot_t1 -v ON_ERROR_STOP=1 -qAt" SEED_DEFAULT_PASSWORD=x timeout 600 node scripts/seed-t1.mjs --reset > $L 2>&1 &
SP=$!; sleep $STOP_AFTER; echo "$(date +%T) docker stop $PG (seed đang chạy: $(kill -0 $SP 2>/dev/null && echo có || echo đã xong))"; docker stop $PG >/dev/null
T0=$(date +%s); wait $SP; RC=$?; echo "seed rc=$RC sau $(( $(date +%s)-T0 )) s kể từ lúc dừng Postgres"
docker start $PG >/dev/null; sleep 6
U1=$(P -d edupilot -c "select count(*) from users"); echo "users(edupilot) sau: $U1"
echo "--- log seed:"; cat $L
