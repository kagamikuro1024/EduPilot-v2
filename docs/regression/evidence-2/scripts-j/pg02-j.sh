#!/usr/bin/env bash
# QC2 (SmkJ) — TC-PG02-39/40/41 chạy trên Postgres TẠM qc2-j-pg (docker run, cổng 6199, mạng qc2-j-net), KHÔNG chạm ep-r.
# Logic chấm giữ nguyên từ docs/sprints/2/qc/scripts/pg02.sh; chỉ đổi cách gọi psql/pg_dump/migrate sang container tạm.
set -u
IMG=${IMG:-edupilot-gateway:latest}; PG=qc2-j-pg; NET=qc2-j-net
OUT=${QC_OUT:-/tmp/qc2-j-out}; mkdir -p "$OUT"
PSQL() { docker exec -i $PG psql -U edupilot -d edupilot -v ON_ERROR_STOP=1 -At "$@"; }
mig() { local o rc; o=$(docker run --rm --network $NET -e APP_ENV=dev -e DATABASE_URL=postgres://edupilot:qcj@$PG:5432/edupilot?sslmode=disable --entrypoint /gateway $IMG migrate "$1" 2>&1); rc=$?; printf '%s\n' "$o" > "$OUT/mig-$1.log"; return $rc; }
dump() { docker exec $PG pg_dump -U edupilot -d edupilot -s 2>/dev/null; }
ok=1; chk() { if [ "$2" = "$3" ]; then echo "    ok   $1 = [$2]"; else echo "    FAIL $1: thực tế [$2] ≠ mong đợi [$3]"; ok=0; fi; }
verdict() { if [ $ok = 1 ]; then echo "-- $1 PASS"; else echo "-- $1 FAIL"; fi; ok=1; }
T5="select count(*) from pg_tables where schemaname='public' and tablename in ('users','audit_log','outbox','jobs','idempotency_keys')"
echo "== TC-PG02-39"
mig up; echo "    (chuẩn bị) migrate up rc=$?; version=$(PSQL -c 'select max(version_id) from goose_db_version where is_applied')"
dump > $OUT/a.sql; mig down; rcd=$?; mig up; rcu=$?; dump > $OUT/b.sql
chk "migrate down rc" $rcd 0; chk "migrate up rc" $rcu 0
if diff <(grep -vE '^\\(un)?restrict ' $OUT/a.sql) <(grep -vE '^\\(un)?restrict ' $OUT/b.sql) > $OUT/39.diff 2>&1; then echo "    ok   lược đồ sau down+up giống hệt (diff rỗng; $(wc -l < $OUT/a.sql) dòng)"; else echo "    FAIL lược đồ đổi"; head -20 $OUT/39.diff; ok=0; fi
verdict TC-PG02-39
echo "== TC-PG02-40"
mig down; rcd=$?; n=$(PSQL -c "$T5"); v=$(PSQL -c "select extname from pg_extension where extname='vector'"); mig up; rcu=$?
chk "migrate down rc" $rcd 0; chk "số bảng của 00001 còn lại sau down" "$n" 0; chk "extension vector còn sau down" "$v" vector; chk "migrate up rc" $rcu 0; chk "5 bảng trở lại sau up" "$(PSQL -c "$T5")" 5
verdict TC-PG02-40
echo "== TC-PG02-41"
n0=$(PSQL -c "select count(*) from goose_db_version"); mig up; rc=$?; n1=$(PSQL -c "select count(*) from goose_db_version")
chk "migrate up lần hai rc" $rc 0; chk "goose_db_version không thêm dòng" "$n1" "$n0"
chk "vẫn đúng 5 bảng + goose_db_version" "$(PSQL -c "select count(*) from pg_tables where schemaname='public' and tablename not like '\_test\_%'")" 6
verdict TC-PG02-41
