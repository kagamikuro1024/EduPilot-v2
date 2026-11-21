#!/bin/bash
# TC-N03-12 (SmkS4, 2026-11-21): Postgres tạm trên tmpfs 160 MB (container qc2-n03-pg, xoá sau khi chạy) → seed T1 phải thoát ≠ 0 và in bảng đang ghi.
# Lược đồ + admin lấy từ smk91-postgres-1 (qc2 sở hữu). Chạy từ gốc repo.
set -u
SRC=smk91-postgres-1; T=qc2-n03-pg; L=docs/regression/evidence-9.5/rerun/n03-12-seed-$(date +%H%M).log
docker rm -f $T >/dev/null 2>&1
docker run -d --name $T --tmpfs /var/lib/postgresql:rw,size=${TMPFS:-160m} -e POSTGRES_USER=edupilot -e POSTGRES_PASSWORD=x -e POSTGRES_DB=edupilot_t1 pgvector/pgvector:pg18 >/dev/null
for i in $(seq 1 30); do docker exec $T pg_isready -U edupilot -d edupilot_t1 >/dev/null 2>&1 && break; sleep 2; done
sleep 3
docker exec $SRC pg_dump -U edupilot -d edupilot --schema-only --no-owner | docker exec -i $T psql -U edupilot -d edupilot_t1 -q -o /dev/null 2>&1 | grep -v -E "^$|already exists" | head -5
docker exec $SRC pg_dump -U edupilot -d edupilot --data-only --no-owner -t users | docker exec -i $T psql -U edupilot -d edupilot_t1 -q -o /dev/null
docker exec $T psql -U edupilot -d edupilot_t1 -At -c "delete from users where role<>'ADMIN'; select 'admin', count(*) from users; select 'df', pg_size_pretty(pg_database_size('edupilot_t1'))"; docker exec $T df -h /var/lib/postgresql | tail -1
T1_PSQL="docker exec -i $T psql -U edupilot -d edupilot_t1 -v ON_ERROR_STOP=1 -qAt" SEED_DEFAULT_PASSWORD=x timeout 900 node scripts/seed-t1.mjs --reset > $L 2>&1; RC=$?
echo "seed rc=$RC"; docker exec $T df -h /var/lib/postgresql | tail -1; echo "--- log seed:"; tail -12 $L
docker rm -f $T >/dev/null; echo "đã xoá $T"
