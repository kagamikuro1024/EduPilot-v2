#!/bin/bash
# SmkT4N: TC-P211-02/28/30/40 trên gateway PHỤ + DB bản sao (qc2i_s4t = pg_dump của ep-r) + redis riêng; xoá sau khi đo. KHÔNG đụng ep-r.
set -u; NET=ep-r_default; PGC=ep-r-postgres-1; DB=qc2i_s4t; PORT=18293; B=http://127.0.0.1:$PORT/api/v1
L=~/Documents/TA_Agent_qc2/docs/regression/evidence-4/logs; OUT=$L/s4n-today${NOK6:+-nok6}${ONLY30:+-30}.txt; PWD_SEED='Edupilot#Seed-2026'
ENVF=$(mktemp); cleanup() { docker rm -f s4n-gwt s4n-redis >/dev/null 2>&1; docker exec $PGC psql -U edupilot -d postgres -qc "drop database if exists $DB with (force)" >/dev/null 2>&1; }; trap 'cleanup; rm -f $ENVF' EXIT
pq() { docker exec -i $PGC psql -U edupilot -d $DB -At "$@"; }
docker inspect ep-r-gateway-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -v -E '^(APP_ENV|DATABASE_URL|PGBOUNCER_URL|REDIS_URL|REQUEST_TIMEOUT|AUTH_[A-Z_]*|PATH)=' | grep . > $ENVF
cleanup; docker exec $PGC psql -U edupilot -d postgres -qc "create database $DB"
docker exec $PGC pg_dump -U edupilot -d edupilot | pq -q >/dev/null 2>&1
pq -c "create role s4n_app login password 's4n'" -c "grant all on all tables in schema public to s4n_app" -c "grant all on all sequences in schema public to s4n_app" -c "grant usage on schema public to s4n_app" -c "grant execute on all functions in schema public to s4n_app" 2>&1 | tail -1
pq -c "create function s4n_slow(n float) returns bool language plpgsql as 'begin perform pg_sleep(n); return true; end'" -c "grant execute on function s4n_slow(float) to s4n_app" >/dev/null
docker run -d --name s4n-redis --network $NET redis:8 >/dev/null
URL="postgres://s4n_app:s4n@postgres:5432/$DB?sslmode=disable"
startgw() { docker rm -f s4n-gwt >/dev/null 2>&1; docker run -d --name s4n-gwt --network $NET --env-file $ENVF -e APP_ENV=test -e DATABASE_URL=$URL -e PGBOUNCER_URL=$URL -e REDIS_URL=redis://s4n-redis:6379/0 -e REQUEST_TIMEOUT=${1:-30s} -e AUTH_LOGIN_IP_PER_MIN=100000 -e AUTH_LOGIN_IP_FAIL_PER_15M=1000000 -e RATE_LIMIT_IP_PER_MIN=100000 -e RATE_LIMIT_USER_PER_MIN=100000 -e APP_PUBLIC_URL=https://localhost -e TEAMS_ENABLED=false -e LLM_PROVIDER=fake -e SMTP_HOST=localhost -e SMTP_PORT=1025 -e SMTP_TLS=none -p 127.0.0.1:$PORT:8080 edupilot-gateway:latest >/dev/null
 for i in $(seq 40); do [ "$(curl -s -o /dev/null -w %{http_code} $B/healthz)" = 200 ] && return; sleep 2; done; echo "gateway không lên"; }
tok() { curl -s -X POST $B/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"$1@edupilot.local\",\"password\":\"$PWD_SEED\"}" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("access_token",""))'; }
get() { curl -s -o /tmp/s4n-body.json -w '%{http_code} %{time_total}s' -H "authorization: Bearer $1" $B${2:-/me/today}; }
{ echo "== cấu hình: gateway edupilot-gateway:latest APP_ENV=test, DB bản sao $DB ($(pq -c 'select count(*) from users') user, $(pq -c 'select count(*) from courses') lớp), redis riêng, Postgres $(docker exec $PGC postgres -V | cut -c1-40)"
startgw 30s; TG=$(tok teacher); TS=$(tok sv.gioi); TA=$(tok admin)
echo "== nền: GV $(get $TG) · SV $(get $TS) · Admin $(get $TA)"
if [ -z "${ONLY30:-}" ]; then
# --- P211-02: Provider lỗi (revoke select trên assignments / submissions → AssignmentProvider lỗi)
docker exec $PGC true; before=$(docker logs s4n-gwt 2>&1 | grep -c 'bỏ Provider lỗi')
docker exec s4n-redis redis-cli flushall >/dev/null
pq -c "revoke select on assignments from s4n_app" >/dev/null; sleep 1
echo "== P211-02 (revoke select on assignments): GV $(get $TG) items=$(python3 -c "import json;j=json.load(open('/tmp/s4n-body.json'));print(len(j.get('items') or j.get('actions') or []), list(j.keys())[:6])") · SV $(get $TS) · Admin $(get $TA)"
docker logs s4n-gwt 2>&1 | grep 'bỏ Provider lỗi' | tail -2 | cut -c1-330; echo "log warn mới: $(( $(docker logs s4n-gwt 2>&1 | grep -c 'bỏ Provider lỗi') - before ))"
pq -c "grant select on assignments to s4n_app" >/dev/null
# --- P211-40: DB chậm quá REQUEST_TIMEOUT (3 s) trên enrollments, và provider chậm (assignments 1 s) còn lại trả
startgw 3s; TG=$(tok teacher); docker exec s4n-redis redis-cli flushall >/dev/null
pq -c "alter table assignments enable row level security" -c "create policy s4n_slow on assignments using (s4n_slow(1))" >/dev/null
echo "== P211-40 (a) assignments chậm 1 s/dòng (> 150 ms của Provider): GV $(get $TG) (phần còn lại): $(head -c 200 /tmp/s4n-body.json)"
pq -c "drop policy s4n_slow on assignments" -c "alter table assignments disable row level security" -c "alter table enrollments enable row level security" -c "create policy s4n_slow on enrollments using (s4n_slow(5))" >/dev/null; docker exec s4n-redis redis-cli flushall >/dev/null
echo "== P211-40 (b) enrollments chậm 5 s > REQUEST_TIMEOUT=3s: GV $(get $TG) → $(head -c 260 /tmp/s4n-body.json)"
pq -c "drop policy s4n_slow on enrollments" -c "alter table enrollments disable row level security" >/dev/null; startgw 30s; TG=$(tok teacher); TS=$(tok sv.gioi)
echo "== sau khôi phục: GV $(get $TG) · SV $(get $TS)"
# --- P211-28: Redis chết → tính trực tiếp, log error ≤ 1 lần/30 s
docker stop s4n-redis >/dev/null; sleep 1; t0=$(date -u +%Y-%m-%dT%H:%M:%S)
for i in $(seq 12); do get $TG >/dev/null; done; echo "== P211-28 (redis dừng): 12 yêu cầu GV liên tiếp mã: $(for i in 1 2 3; do get $TG; echo -n ' | '; done) SV $(get $TS)"
echo "log error fail-open từ $t0 (12+ yêu cầu trong $(( $(date +%s) - $(date -j -u -f %Y-%m-%dT%H:%M:%S $t0 +%s) )) s): $(docker logs s4n-gwt --since $t0 2>&1 | grep -c 'Redis lỗi, tính trực tiếp')"
docker start s4n-redis >/dev/null; sleep 3
fi
# --- P211-30: số truy vấn SQL/yêu cầu (tắt cache = redis dừng) và k6
pq -c "alter database $DB set log_statement='all'" >/dev/null; startgw 30s; TG=$(tok teacher); TS=$(tok sv.gioi); docker stop s4n-redis >/dev/null; sleep 1
count() { n0=$(docker logs $PGC 2>&1 | wc -l); curl -s -o /dev/null -H "authorization: Bearer $1" $B${3:-/me/today}; sleep 1; L2=$(docker logs $PGC 2>&1 | tail -n +$((n0+1)) | grep -E "(execute|statement)[^:]*: "); echo "$L2" | cut -c1-210 > $L/s4n-today-queries-$2.txt; echo "${3:-/me/today}: truy vấn có tên (sqlc)=$(echo "$L2" | grep -c -- '-- name:'), ping pool=$(echo "$L2" | grep -c -- '-- ping'), khác (đăng ký kiểu pgx regtype/pg_type)=$(echo "$L2" | grep -vc -e '-- name:' -e '-- ping')"; }
CID=$(curl -s -H "authorization: Bearer $TS" $B/me/courses | python3 -c 'import sys,json; print(json.load(sys.stdin)["items"][0]["course"]["id"])'); count $TG warm >/dev/null; count $TS warm2 /courses/$CID/today >/dev/null
echo "== P211-30 (cache tắt = redis dừng; log_statement=all, đếm trong log Postgres của DB bản sao; lần 2 sau khởi động pool): GV $(count $TG gv) · GV (lần 3) $(count $TG gv2) · SV 1 lớp $(count $TS sv /courses/$CID/today) · SV tất cả lớp $(count $TS svall)"
pq -c "alter database $DB reset log_statement" >/dev/null; docker start s4n-redis >/dev/null; sleep 3
TG=$(tok teacher); TS=$(tok sv.gioi); curl -s -D- -o /dev/null -H "authorization: Bearer $TG" $B/me/today | grep -i '^etag' ; E1=$(curl -s -D- -o /dev/null -H "authorization: Bearer $TG" $B/me/today | grep -i '^etag' | awk '{print $2}' | tr -d '\r')
echo "== ETag $E1; If-None-Match → $(curl -s -o /dev/null -w '%{http_code}' -H "authorization: Bearer $TG" -H "If-None-Match: $E1" $B/me/today)"
if [ -z "${NOK6:-}" ]; then
echo "== burst 40 yêu cầu/token (1 vòng nhanh): GV $(for i in $(seq 40); do get $TG | cut -d' ' -f1; done | sort | uniq -c | tr '\n' ' ') · SV $(for i in $(seq 40); do get $TS | cut -d' ' -f1; done | sort | uniq -c | tr '\n' ' ')"
echo "== trước k6: 6 yêu cầu GV: $(for i in 1 2 3 4 5 6; do get $TG | cut -d' ' -f1; done | tr '\n' ' ') · SV: $(for i in 1 2 3 4 5 6; do get $TS | cut -d' ' -f1; done | tr '\n' ' ')" 
echo "== k6 20 req/s 60 s, token GV + SV luân phiên, qua gateway phụ (không Caddy): máy $(sysctl -n machdep.cpu.brand_string), $(sysctl -n hw.ncpu) lõi, colima $(colima ssh -- nproc) CPU / $(colima ssh -- free -m | awk 'NR==2{print $2}') MB"
k6 run --no-color -e BASE=http://127.0.0.1:$PORT -e TOKENS=$TG,$TS ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts/s4n-today-k6.js > $L/s4n-today-k6.txt 2>&1; grep -E "http_req_duration|http_req_failed|http_reqs|iterations|✓|✗|status:" $L/s4n-today-k6.txt | head -40
fi
} > $OUT 2>&1
cat $OUT
