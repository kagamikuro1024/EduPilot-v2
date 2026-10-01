#!/usr/bin/env bash
# QC US-PG-01 — khung dịch vụ: config, log, trace, redis, tắt êm, giới hạn, deadline, pool, slow query, worker, Makefile, phụ thuộc chết.
# Nguồn: docs/specs/FEAT-pg-foundation/US.md v1.1 (US-PG-01 AC1…AC15) + SRS.md 3.4, 4.1, 5.6, 6.1, 6.3, 8.1, 8.4, 9.4.
# Chạy ở gốc worktree sau `pnpm dev`:  bash docs/sprints/2/qc/scripts/pg01.sh [MM …|--list]
# Hộp đen: chỉ dùng binary/ảnh đã dựng, lệnh curl/psql/redis-cli/docker/make. Không đọc mã nguồn.
source "$(dirname "$0")/lib.sh"

# ---------- tiện ích riêng của story 01 ----------
WKBIN=$QC_TMP/wk-default                      # binary worker mặc định (không tag)
RC=0; MS=0                                     # kết quả của run_gw / run_wk

bins() {  # dựng binary trần (cho phép: chạy sản phẩm, không đọc mã)
  [ -x "$GWBIN" ] || (cd backend-go && CGO_ENABLED=0 go build -o "$GWBIN" ./cmd/gateway) || return 1
  [ -x "$WKBIN" ] || (cd backend-go && CGO_ENABLED=0 go build -o "$WKBIN" ./cmd/worker) || return 1
  return 0; }

full7() {  # 7 biến bắt buộc, giá trị như lệnh `full()` của AC1 (URL trỏ cổng đóng → không chạm phụ thuộc thật)
  printf '%s\n' DATABASE_URL=postgres://u:p@127.0.0.1:1/db REDIS_URL=redis://127.0.0.1:1/0 \
    JWT_SECRET_KEY=0123456789abcdef0123456789abcdef BLOB_ENDPOINT=127.0.0.1:1 BLOB_BUCKET=b \
    BLOB_ACCESS_KEY=ak BLOB_SECRET_KEY=sk; }

# run_gw <tệp log> [VAR=VAL …] — chạy `gateway serve` với môi trường TRẮNG (env -i) ; đặt RC, MS
run_gw() { local out=$1; shift; local s e
  s=$(now_ms); env -i PATH="$PATH" HTTP_ADDR=127.0.0.1:18080 STARTUP_TIMEOUT=2s "$@" "$GWBIN" serve >"$out" 2>&1
  RC=$?; e=$(now_ms); MS=$((e-s)); }
# run_wk <tệp log> [VAR=VAL …] — chạy `worker`
run_wk() { local out=$1; shift; local s e
  s=$(now_ms); env -i PATH="$PATH" WORKER_HEALTH_ADDR=127.0.0.1:18081 STARTUP_TIMEOUT=2s "$@" "$WKBIN" >"$out" 2>&1
  RC=$?; e=$(now_ms); MS=$((e-s)); }

errline() { jq -r 'select((.level|ascii_downcase)=="error")' "$1" 2>/dev/null; }       # mọi dòng mức error
nerr_var() { errline "$1" | grep -c "$2"; }                                             # số dòng error nêu tên biến

# cfg_bad <tệp log> <tên biến> <regex giá trị không được in | rỗng> <VAR=VAL ghi đè …>
cfg_bad() { local out=$1 v=$2 val=$3; shift 3
  run_gw "$out" $(full7 | grep -v "^$v=") "$@"
  chk "rc khi $v sai" "$RC" 1
  chk_le "ms khi $v sai" "$MS" 1000
  chk_ge "dòng lỗi mức error nêu $v" "$(nerr_var "$out" "$v")" 1
  [ -n "$val" ] && chk "log không in giá trị của $v" "$(grep -cE -- "$val" "$out")" 0
  return 0; }

up_t() { $CT up -d --scale gateway="${1:-2}" --wait gateway >/dev/null 2>&1; wait_ready 90; }   # dựng lại gateway chế độ test
wait_worker() { local i=0; while [ $i -lt 90 ]; do
    [ "$($C ps --format '{{.Service}} {{.Health}}' 2>/dev/null | grep -c '^worker healthy')" -ge 1 ] && return 0
    sleep 1; i=$((i+1)); done; return 1; }
worker_up() { $C start worker >/dev/null 2>&1 || $C up -d --wait worker >/dev/null 2>&1; wait_worker; }
newtrace() { printf '%016x%016x' "$(now_ms)" "$(( RANDOM * 32768 + RANDOM + 1 ))"; }   # 32 hex, khác toàn 0
pg_sleep_cnt() { $PSQL -c "select count(*) from pg_stat_activity where query ilike '%pg_sleep%' and state='active' and pid<>pg_backend_pid()" 2>/dev/null | tr -d ' \r'; }

# ============================================================ AC1 — thiếu biến bắt buộc
tc_pg01_01() {  # AC1 — thiếu TỪNG biến trong 7 biến: rc=1, ≤ 1 s, `missing` nêu đúng tên, mức error, không in giá trị biến khác
  bins || { fail_tc "không dựng được binary gateway/worker"; return; }
  local v out named
  for v in DATABASE_URL REDIS_URL JWT_SECRET_KEY BLOB_ENDPOINT BLOB_BUCKET BLOB_ACCESS_KEY BLOB_SECRET_KEY; do
    out=$QC_OUT/tc-pg01-01-$v.log
    run_gw "$out" $(full7 | grep -v "^$v=")
    chk "rc khi thiếu $v" "$RC" 1
    chk_le "ms khi thiếu $v" "$MS" 1000
    named=$(jq -r 'select(.missing)|.missing|index("'"$v"'")!=null' "$out" 2>/dev/null | head -1)
    chk "missing nêu $v" "$named" true
    chk "mức log khi thiếu $v" "$(jq -r 'select(.missing)|.level' "$out" 2>/dev/null | head -1 | tr 'A-Z' 'a-z')" error
    [ "$v" = DATABASE_URL ] || chk "không in giá trị DATABASE_URL khi thiếu $v" "$(grep -cF 'postgres://u:p' "$out")" 0
  done; }

tc_pg01_02() {  # AC1 — thiếu biến → tiến trình KHÔNG mở cổng HTTP (kiểm 15 lần trong lúc chạy)
  bins || { fail_tc "không dựng được binary"; return; }
  local out=$QC_OUT/tc-pg01-02.log hits=0 i=0 c pid rc
  ( env -i PATH="$PATH" HTTP_ADDR=127.0.0.1:18080 STARTUP_TIMEOUT=2s $(full7 | grep -v '^JWT_SECRET_KEY=') "$GWBIN" serve >"$out" 2>&1 ) &
  pid=$!
  while [ $i -lt 15 ]; do
    c=$(curl -s -o /dev/null -m 1 -w '%{http_code}' http://127.0.0.1:18080/healthz 2>/dev/null)
    [ "$c" = 000 ] || hits=$((hits+1)); i=$((i+1))
  done
  wait $pid; rc=$?
  chk "số lần cổng 127.0.0.1:18080 trả lời" "$hits" 0
  chk "rc" "$rc" 1; }

tc_pg01_03() {  # AC1 — thiếu CẢ 7: `missing` liệt kê đủ 7 tên trong ĐÚNG MỘT dòng JSON
  bins || { fail_tc "không dựng được binary"; return; }
  local out=$QC_OUT/tc-pg01-03.log
  run_gw "$out"
  chk "rc" "$RC" 1
  chk_le "ms" "$MS" 1000
  chk "missing (đã sắp xếp)" "$(jq -c 'select(.missing)|.missing|sort' "$out" 2>/dev/null | head -1)" \
    '["BLOB_ACCESS_KEY","BLOB_BUCKET","BLOB_ENDPOINT","BLOB_SECRET_KEY","DATABASE_URL","JWT_SECRET_KEY","REDIS_URL"]'
  chk "số dòng stdout" "$(grep -c . "$out")" 1
  chk "số dòng có .missing" "$(jq -c 'select(.missing)' "$out" 2>/dev/null | wc -l | tr -d ' ')" 1; }

tc_pg01_04() {  # AC1 — worker thiếu DATABASE_URL
  bins || { fail_tc "không dựng được binary"; return; }
  local out=$QC_OUT/tc-pg01-04.log
  run_wk "$out" REDIS_URL=redis://127.0.0.1:1/0
  chk "rc" "$RC" 1; chk_le "ms" "$MS" 1000
  chk "missing của worker (nguyên văn AC1)" "$(jq -c 'select(.missing)|.missing' "$out" 2>/dev/null | head -1)" '["DATABASE_URL"]'; }

tc_pg01_05() {  # AC1 — worker thiếu REDIS_URL
  bins || { fail_tc "không dựng được binary"; return; }
  local out=$QC_OUT/tc-pg01-05.log
  run_wk "$out" DATABASE_URL=postgres://u:p@127.0.0.1:1/db
  chk "rc" "$RC" 1; chk_le "ms" "$MS" 1000
  chk "missing của worker" "$(jq -c 'select(.missing)|.missing' "$out" 2>/dev/null | head -1)" '["REDIS_URL"]'; }

tc_pg01_06() {  # AC1 — worker thiếu cả hai
  bins || { fail_tc "không dựng được binary"; return; }
  local out=$QC_OUT/tc-pg01-06.log
  run_wk "$out"
  chk "rc" "$RC" 1; chk_le "ms" "$MS" 1000
  chk "missing của worker (đã sắp xếp)" "$(jq -c 'select(.missing)|.missing|sort' "$out" 2>/dev/null | head -1)" '["DATABASE_URL","REDIS_URL"]'
  chk "số dòng stdout" "$(grep -c . "$out")" 1; }

tc_pg01_07() {  # AC1 / SRS 8.1 — biến toàn khoảng trắng = thiếu (rỗng sau TrimSpace)
  bins || { fail_tc "không dựng được binary"; return; }
  local out=$QC_OUT/tc-pg01-07a.log out2=$QC_OUT/tc-pg01-07b.log
  run_gw "$out" $(full7 | grep -v '^JWT_SECRET_KEY=') "JWT_SECRET_KEY=   "
  chk "rc (JWT_SECRET_KEY ba dấu cách)" "$RC" 1
  chk "missing nêu JWT_SECRET_KEY" "$(jq -r 'select(.missing)|.missing|index("JWT_SECRET_KEY")!=null' "$out" 2>/dev/null | head -1)" true
  run_gw "$out2" $(full7 | grep -v '^BLOB_BUCKET=') "BLOB_BUCKET=$(printf '\t ')"
  chk "rc (BLOB_BUCKET tab+cách)" "$RC" 1
  chk "missing nêu BLOB_BUCKET" "$(jq -r 'select(.missing)|.missing|index("BLOB_BUCKET")!=null' "$out2" 2>/dev/null | head -1)" true; }

tc_pg01_08() {  # AC1 — test Go
  gt ./internal/platform/config 'TestLoad_MissingEnv|TestLoad_Worker'; }

# ============================================================ AC2 — giá trị biến sai
tc_pg01_09() {  # AC2 — JWT_SECRET_KEY 31 byte
  bins || { fail_tc "không dựng được binary"; return; }
  cfg_bad "$QC_OUT/tc-pg01-09.log" JWT_SECRET_KEY 'short-secret' JWT_SECRET_KEY=short-secret-31-bytes-xxxxxxxxxx; }

tc_pg01_10() {  # AC2 — DB_MAX_CONNS=0
  bins || { fail_tc "không dựng được binary"; return; }
  cfg_bad "$QC_OUT/tc-pg01-10.log" DB_MAX_CONNS '' DB_MAX_CONNS=0; }

tc_pg01_11() {  # AC2 — JWT_EXPIRATION=abc
  bins || { fail_tc "không dựng được binary"; return; }
  cfg_bad "$QC_OUT/tc-pg01-11.log" JWT_EXPIRATION '(^|[^0-9a-f])abc([^0-9a-f]|$)' JWT_EXPIRATION=abc; }

tc_pg01_12() {  # AC2 — APP_CORS_ALLOWED_ORIGINS=*  (SRS 6.7: giá trị `*` bị từ chối lúc khởi động)
  bins || { fail_tc "không dựng được binary"; return; }
  cfg_bad "$QC_OUT/tc-pg01-12.log" APP_CORS_ALLOWED_ORIGINS '' 'APP_CORS_ALLOWED_ORIGINS=*'; }

tc_pg01_13() {  # AC2 — APP_ENV=staging
  bins || { fail_tc "không dựng được binary"; return; }
  cfg_bad "$QC_OUT/tc-pg01-13.log" APP_ENV 'staging' APP_ENV=staging; }

tc_pg01_14() {  # AC2 (biên) — JWT_SECRET_KEY đúng 32 byte hợp lệ / 31 byte lỗi
  bins || { fail_tc "không dựng được binary"; return; }
  local ok=0123456789abcdef0123456789abcdef bad=0123456789abcdef0123456789abcde
  local o1=$QC_OUT/tc-pg01-14a.log o2=$QC_OUT/tc-pg01-14b.log
  chk "độ dài khoá hợp lệ" "${#ok}" 32
  chk "độ dài khoá lỗi" "${#bad}" 31
  run_gw "$o1" $(full7 | grep -v '^JWT_SECRET_KEY=') "JWT_SECRET_KEY=$ok"
  chk "32 byte: số dòng error nêu JWT_SECRET_KEY" "$(nerr_var "$o1" JWT_SECRET_KEY)" 0
  chk_ge "32 byte: đi tới bước chờ phụ thuộc (ms ≈ STARTUP_TIMEOUT 2s)" "$MS" 1500
  run_gw "$o2" $(full7 | grep -v '^JWT_SECRET_KEY=') "JWT_SECRET_KEY=$bad"
  chk "31 byte: rc" "$RC" 1
  chk_le "31 byte: ms" "$MS" 1000
  chk_ge "31 byte: dòng error nêu JWT_SECRET_KEY" "$(nerr_var "$o2" JWT_SECRET_KEY)" 1; }

tc_pg01_15() {  # AC2 (biên) — DB_MAX_CONNS 1 và 100 hợp lệ, 101 lỗi (SRS 8.1: 1–100)
  bins || { fail_tc "không dựng được binary"; return; }
  local n out
  for n in 1 100; do
    out=$QC_OUT/tc-pg01-15-$n.log
    run_gw "$out" $(full7) DB_MAX_CONNS=$n
    chk "DB_MAX_CONNS=$n: số dòng error nêu biến" "$(nerr_var "$out" DB_MAX_CONNS)" 0
    chk_ge "DB_MAX_CONNS=$n: đi tới bước chờ phụ thuộc (ms)" "$MS" 1500
  done
  out=$QC_OUT/tc-pg01-15-101.log
  run_gw "$out" $(full7) DB_MAX_CONNS=101
  chk "DB_MAX_CONNS=101: rc" "$RC" 1
  chk_le "DB_MAX_CONNS=101: ms" "$MS" 1000
  chk_ge "DB_MAX_CONNS=101: dòng error nêu biến" "$(nerr_var "$out" DB_MAX_CONNS)" 1; }

tc_pg01_16() {  # AC2 (biên) — APP_ENV dev|test|production đều hợp lệ (SRS 8.1)
  bins || { fail_tc "không dựng được binary"; return; }
  local e out
  for e in dev test production; do
    out=$QC_OUT/tc-pg01-16-$e.log
    run_gw "$out" $(full7) APP_ENV=$e
    chk "APP_ENV=$e: số dòng error nêu APP_ENV" "$(nerr_var "$out" APP_ENV)" 0
    chk_ge "APP_ENV=$e: đi tới bước chờ phụ thuộc (ms)" "$MS" 1500
  done; }

tc_pg01_17() {  # AC2 — test Go
  gt ./internal/platform/config TestLoad_Invalid; }

# ============================================================ AC3 — mặc định dev + `config loaded` + không bí mật
tc_pg01_18() {  # AC3 — mỗi bản gateway ghi ĐÚNG MỘT dòng `config loaded` có "secrets":"[redacted]"
  ensure_mode default || { fail_tc "không về được chế độ default"; return; }
  local id n rc
  for id in $(gw_ids); do
    rc=$(docker inspect -f '{{.RestartCount}}' "$id" 2>/dev/null)
    chk "RestartCount của ${id:0:12} (tiền điều kiện: không restart)" "$rc" 0
    n=$(docker logs "$id" 2>&1 | jq -c 'select(.msg=="config loaded")' 2>/dev/null | wc -l | tr -d ' ')
    chk "số dòng 'config loaded' của ${id:0:12}" "$n" 1
    chk_ge "dòng có \"secrets\":\"[redacted]\"" "$(docker logs "$id" 2>&1 | grep -c '"secrets":"\[redacted\]"')" 1
  done; }

tc_pg01_19() {  # AC3 — binary trần, CHỈ 7 biến bắt buộc → mặc định dev hiệu lực (DB_MAX_CONNS 10, REQUEST_TIMEOUT 30s, SSE_HEARTBEAT 25s)
  bins || { fail_tc "không dựng được binary"; return; }
  local out=$QC_OUT/tc-pg01-19.log usr pw db bb ba bs pid i=0 line
  usr=$(envv POSTGRES_USER); [ -n "$usr" ] || usr=edupilot
  db=$(envv POSTGRES_DB);   [ -n "$db" ]  || db=edupilot
  pw=$(envv POSTGRES_PASSWORD); bb=$(envv BLOB_BUCKET); ba=$(envv BLOB_ACCESS_KEY); bs=$(envv BLOB_SECRET_KEY)
  [ -n "$pw" ] && [ -n "$bb" ] && [ -n "$ba" ] && [ -n "$bs" ] || { fail_tc "thiếu POSTGRES_PASSWORD/BLOB_* trong .env.local"; return; }
  ( env -i PATH="$PATH" \
      DATABASE_URL="postgres://$usr:$pw@127.0.0.1:5433/$db" REDIS_URL="redis://127.0.0.1:6380/0" \
      JWT_SECRET_KEY="$SECRET" BLOB_ENDPOINT=127.0.0.1:9000 BLOB_BUCKET="$bb" \
      BLOB_ACCESS_KEY="$ba" BLOB_SECRET_KEY="$bs" "$GWBIN" serve >"$out" 2>&1 ) &
  pid=$!
  while [ $i -lt 40 ] && ! grep -q 'config loaded' "$out" 2>/dev/null; do sleep 0.5; i=$((i+1)); done
  kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
  line=$(jq -c 'select(.msg=="config loaded")' "$out" 2>/dev/null | head -1)
  chk "số dòng 'config loaded'" "$(jq -c 'select(.msg=="config loaded")' "$out" 2>/dev/null | wc -l | tr -d ' ')" 1
  chk_re "DB_MAX_CONNS mặc định 10" "$line" '(^|[^0-9])10([^0-9]|$)'
  chk_re "REQUEST_TIMEOUT mặc định 30s" "$line" '30s'
  chk_re "SSE_HEARTBEAT mặc định 25s" "$line" '25s'
  chk_re 'secrets đã che' "$line" '"secrets":"\[redacted\]"'
  chk "dòng không chứa khoá JWT thật" "$(printf '%s' "$line" | grep -cF "$SECRET")" 0; }

tc_pg01_20() {  # AC3 — không dòng log nào chứa bí mật (lệnh nguyên văn của AC3)
  [ -n "$SECRET" ] || { fail_tc "không đọc được JWT_SECRET_KEY từ .env.local"; return; }
  chk "số dòng log chứa bí mật" \
    "$($C logs --no-log-prefix gateway worker 2>/dev/null | grep -cE "$SECRET|edupilot-dev-secret|edupilot-dev@")" 0; }

tc_pg01_21() {  # AC3 — test Go
  gt ./internal/platform/config 'TestLoad_Defaults|TestConfig_NoSecretInLog'; }

# ============================================================ AC4 — log JSON, trace_id, không PII
tc_pg01_22() {  # AC4 — MỌI dòng log gateway+worker là JSON có time/level/msg/service/instance/trace_id 32 hex ≠ 0
  local all n
  all=$($C logs --no-log-prefix gateway worker 2>/dev/null | jq -s 'all(.[]; (.trace_id|type=="string") and (.trace_id|test("^[0-9a-f]{32}$")) and (.trace_id!="00000000000000000000000000000000") and (.service|type=="string") and (.instance|type=="string") and (.time!=null) and (.level!=null) and (.msg!=null))' 2>/dev/null)
  n=$($C logs --no-log-prefix gateway worker 2>/dev/null | jq -s 'length' 2>/dev/null)
  chk "mọi dòng log đạt (JSON + 6 trường + trace_id 32 hex ≠ 0)" "$all" true
  chk_ge "số dòng log đã kiểm" "${n:-0}" 10; }

tc_pg01_23() {  # AC4 — `service` đúng tên, `instance` phân biệt được hai bản gateway
  local s n
  s=$($C logs --no-log-prefix gateway 2>/dev/null | jq -r '.service' 2>/dev/null | sort -u | paste -sd, -)
  chk "service trong log gateway" "$s" gateway
  s=$($C logs --no-log-prefix worker 2>/dev/null | jq -r '.service' 2>/dev/null | sort -u | paste -sd, -)
  chk "service trong log worker" "$s" worker
  n=$($C logs --no-log-prefix gateway 2>/dev/null | jq -r '.instance' 2>/dev/null | sort -u | grep -c .)
  chk "số instance khác nhau trong log gateway" "$n" "$(gw_ids | grep -c .)"; }

tc_pg01_24() {  # AC4 — dòng log lúc DỪNG (SIGTERM worker) cũng đủ trường và trace_id ≠ 0
  local all n
  $C stop -t 15 worker >/dev/null 2>&1
  all=$($C logs --no-log-prefix --since 40s worker 2>/dev/null | jq -s 'all(.[]; (.trace_id|test("^[0-9a-f]{32}$")) and (.trace_id!="00000000000000000000000000000000") and .service and .instance)' 2>/dev/null)
  n=$($C logs --no-log-prefix --since 40s worker 2>/dev/null | jq -s 'length' 2>/dev/null)
  worker_up || fail_tc "worker không trở lại healthy sau khi khôi phục"
  chk "mọi dòng log quanh lúc dừng đạt" "$all" true
  chk_ge "số dòng log quanh lúc dừng" "${n:-0}" 1; }

tc_pg01_25() {  # AC4 — log không chứa PII (email mồi trong thân request) và không chứa token
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local canary=pii-canary-8842@x.com sig
  curl -sk --max-time 20 -X POST -H "$H" -H 'Content-Type: application/json' \
    -H "Idempotency-Key: qc-pii-$(now_ms)" -d "{\"name\":\"$canary\"}" "$GW/api/v1/_test/items" >/dev/null 2>&1
  sleep 2
  chk "số dòng log chứa email mồi" "$($C logs --no-log-prefix --since 60s gateway worker 2>/dev/null | grep -c 'pii-canary-8842')" 0
  sig=$(printf '%s' "${H#Authorization: Bearer }" | cut -d. -f3)
  [ -n "$sig" ] || { fail_tc "không lấy được chữ ký token"; return; }
  chk "số dòng log chứa chữ ký token" "$($C logs --no-log-prefix --since 60s gateway worker 2>/dev/null | grep -cF "$sig")" 0; }

tc_pg01_26() {  # AC4 — test Go
  gt './internal/platform/log ./internal/platform/otel' 'TestHandler_AlwaysTraceID|TestStartupHasTrace'; }

# ============================================================ AC5 — traceparent → thân lỗi + log
tc_pg01_27() {  # AC5 — traceparent đến → thân lỗi mang đúng trace_id, có header X-Request-Id
  local tp='00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01' body hf=$QC_TMP/h-pg01-27
  body=$(curl -sk --max-time 20 -D "$hf" -H "traceparent: $tp" "$GW/api/v1/khong-co" 2>/dev/null)
  chk "trace_id trong thân lỗi" "$(printf '%s' "$body" | jq -r '.trace_id // empty' 2>/dev/null)" 4bf92f3577b34da6a3ce929d0e0e4736
  chk "mã" "$(code -H "traceparent: $tp" "$GW/api/v1/khong-co")" 404
  chk "code lỗi" "$(printf '%s' "$body" | jq -r '.code // empty' 2>/dev/null)" NOT_FOUND
  chk_re "có header X-Request-Id" "$(tr -d '\r' < "$hf" | hval x-request-id)" '.+'; }

tc_pg01_28() {  # AC5 — mọi dòng log của request đó mang cùng trace_id
  local tp='00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01' n
  curl -sk --max-time 20 -o /dev/null -H "traceparent: $tp" "$GW/api/v1/khong-co" 2>/dev/null
  sleep 2
  n=$($C logs --no-log-prefix --since 60s gateway 2>/dev/null | jq -c 'select(.trace_id=="4bf92f3577b34da6a3ce929d0e0e4736")' 2>/dev/null | wc -l | tr -d ' ')
  chk_ge "số dòng log mang trace_id của request" "${n:-0}" 1; }

tc_pg01_29() {  # AC5 — không có traceparent → gateway tự sinh trace mới; X-Request-Id = trace_id (SRS 6.5)
  local body hf=$QC_TMP/h-pg01-29 tid rid
  body=$(curl -sk --max-time 20 -D "$hf" "$GW/api/v1/khong-co" 2>/dev/null)
  tid=$(printf '%s' "$body" | jq -r '.trace_id // empty' 2>/dev/null)
  rid=$(tr -d '\r' < "$hf" | hval x-request-id)
  chk_re "trace_id tự sinh 32 hex" "$tid" '^[0-9a-f]{32}$'
  chk_ne "trace_id khác toàn 0" "$tid" 00000000000000000000000000000000
  chk_ne "trace_id khác trace của TC-27" "$tid" 4bf92f3577b34da6a3ce929d0e0e4736
  chk "X-Request-Id = trace_id" "$rid" "$tid"; }

tc_pg01_30() {  # AC5 — test Go
  gt ./internal/httpapi TestTraceID_InErrorBodyAndLogs; }

# ============================================================ AC6 — khoá Redis
tc_pg01_31() {  # AC6 — test Go (hàm dựng khoá + hằng TTL)
  gt ./internal/platform/redis 'TestKeys|TestTTLConstants'; }

tc_pg01_32() {  # AC6 — tự tạo mỗi loại khoá rồi kiểm MỌI khoá `ep:*` đều có TTL (không khoá nào -1)
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local t p pid noexp
  t=$(tok STUDENT "$U1")
  newitem "qc-ac6-$(now_ms)" >/dev/null 2>&1                      # ep:idem:*
  code -H "$H" "$GW/api/v1/_test/whoami" >/dev/null                # ep:rl:ip:* và ep:rl:user:*
  pid=$(sse_cap "$QC_OUT/tc-pg01-32.sse" 6 "$t")                   # ep:sse:conn:*
  sleep 1
  curl -sk --max-time 10 -o /dev/null -X POST -H "Authorization: Bearer $t" -H 'Content-Type: application/json' \
    -d '{"type":"test.ping","data":{"n":1}}' "$GW/api/v1/_test/events" 2>/dev/null   # ep:sse:buf:*
  sleep 1
  for p in 'ep:idem:*' 'ep:rl:*' 'ep:sse:conn:*' 'ep:sse:buf:*'; do
    chk_ge "có khoá $p" "$($RDS --scan --pattern "$p" 2>/dev/null | grep -c .)" 1
  done
  noexp=$($RDS --scan --pattern 'ep:*' 2>/dev/null | tr -d '\r' | while read -r k; do
            [ -n "$k" ] && echo "$($RDS ttl "$k" 2>/dev/null | tr -d '\r') $k"; done | grep -c '^-1 ')
  chk "số khoá ep:* không có TTL" "$noexp" 0
  wait "$pid" 2>/dev/null; return 0; }

tc_pg01_33() {  # AC6 — TTL đúng hằng số SRS 5.6 (idem 24 h, rate limit 120 s, SSE conn 150 s, SSE buf 1 h)
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local t pid k ttl
  t=$(tok STUDENT "$U1")
  newitem "qc-ttl-$(now_ms)" >/dev/null 2>&1
  code -H "$H" "$GW/api/v1/_test/whoami" >/dev/null
  pid=$(sse_cap "$QC_OUT/tc-pg01-33.sse" 6 "$t")
  sleep 1
  curl -sk --max-time 10 -o /dev/null -X POST -H "Authorization: Bearer $t" -H 'Content-Type: application/json' \
    -d '{"type":"test.ping","data":{"n":1}}' "$GW/api/v1/_test/events" 2>/dev/null
  sleep 1
  k=$($RDS --scan --pattern 'ep:idem:*' 2>/dev/null | tr -d '\r' | grep -v ':lock$' | head -1)
  ttl=$($RDS ttl "$k" 2>/dev/null | tr -d '\r'); chk_le "TTL ep:idem ≤ 86400" "${ttl:-x}" 86400; chk_ge "TTL ep:idem ≥ 86300" "${ttl:-x}" 86300
  k=$($RDS --scan --pattern 'ep:rl:*' 2>/dev/null | tr -d '\r' | head -1)
  ttl=$($RDS ttl "$k" 2>/dev/null | tr -d '\r'); chk_le "TTL ep:rl ≤ 120" "${ttl:-x}" 120; chk_ge "TTL ep:rl ≥ 60" "${ttl:-x}" 60
  k=$($RDS --scan --pattern 'ep:sse:conn:*' 2>/dev/null | tr -d '\r' | head -1)
  ttl=$($RDS ttl "$k" 2>/dev/null | tr -d '\r'); chk_le "TTL ep:sse:conn ≤ 150" "${ttl:-x}" 150; chk_ge "TTL ep:sse:conn ≥ 100" "${ttl:-x}" 100
  k=$($RDS --scan --pattern 'ep:sse:buf:*' 2>/dev/null | tr -d '\r' | head -1)
  ttl=$($RDS ttl "$k" 2>/dev/null | tr -d '\r'); chk_le "TTL ep:sse:buf ≤ 3600" "${ttl:-x}" 3600; chk_ge "TTL ep:sse:buf ≥ 3500" "${ttl:-x}" 3500
  wait "$pid" 2>/dev/null; return 0; }

tc_pg01_34() {  # AC6 — không khoá nào ngoài `ep:` / `outbox.dispatch` / `jobs.`
  local n
  n=$($RDS --scan 2>/dev/null | tr -d '\r' | grep -c -vE '^(ep:|outbox\.dispatch|jobs\.)')
  chk "số khoá Redis ngoài quy ước" "$n" 0
  [ "$n" = 0 ] || $RDS --scan 2>/dev/null | grep -vE '^(ep:|outbox\.dispatch|jobs\.)' > "$QC_OUT/tc-pg01-34-la.txt"
  return 0; }

# ============================================================ AC7 — tắt máy êm
tc_pg01_35() {  # AC7 (2) — request chậm 3 s đang chạy HOÀN TẤT 200 khi nhận SIGTERM
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local f=$QC_OUT/tc-pg01-35.code pid
  ( curl -sk -o /dev/null -m 30 -w '%{http_code}' "$GW/api/v1/_test/slow?ms=3000" > "$f" 2>/dev/null ) &
  pid=$!
  sleep 0.5
  $C stop -t 30 gateway >/dev/null 2>&1
  wait "$pid" 2>/dev/null
  up_t 2
  chk "mã của request chậm" "$(cat "$f" 2>/dev/null)" 200; }

tc_pg01_36() {  # AC7 (5) — mọi bản gateway thoát mã 0, trước SHUTDOWN_TIMEOUT 25 s
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local pid s e codes
  ( curl -sk -o /dev/null -m 30 "$GW/api/v1/_test/slow?ms=3000" >/dev/null 2>&1 ) &
  pid=$!
  sleep 0.5; s=$(now_ms); $C stop -t 30 gateway >/dev/null 2>&1; e=$(now_ms); wait "$pid" 2>/dev/null
  codes=$($C ps -a --format '{{.Service}} {{.ExitCode}}' 2>/dev/null | grep '^gateway' | awk '{print $2}' | sort -u | paste -sd, -)
  up_t 2
  chk "mã thoát của mọi bản gateway" "$codes" 0
  chk_le "thời gian tắt (ms)" "$((e-s))" 25000; }

tc_pg01_37() {  # AC7 (1) — SIGTERM → /api/v1/readyz trả 503 NOT_READY ngay
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  up_t 1 || fail_tc "không hạ được về một bản gateway"
  local id pid out body c
  id=$($C ps -q gateway 2>/dev/null | head -1)
  ( curl -sk -o /dev/null -m 20 "$GW/api/v1/_test/slow?ms=5000" >/dev/null 2>&1 ) &
  pid=$!
  sleep 0.4
  docker kill -s TERM "$id" >/dev/null 2>&1
  sleep 0.3
  out=$(curl -sk -m 5 -w '\n%{http_code}' "$GW/api/v1/readyz" 2>/dev/null)
  wait "$pid" 2>/dev/null
  up_t 2
  c=$(printf '%s' "$out" | tail -1); body=$(printf '%s' "$out" | sed '$d')
  chk "mã readyz khi đang tắt" "$c" 503
  chk "code" "$(printf '%s' "$body" | jq -r '.code // empty' 2>/dev/null)" NOT_READY
  chk "details.draining (SRS 6.1)" "$(printf '%s' "$body" | jq -r '.details.draining // empty' 2>/dev/null)" true; }

tc_pg01_38() {  # AC7 (3) — client SSE nhận `event: shutdown` rồi kết nối đóng trong ≤ 1 s
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  up_t 1 || fail_tc "không hạ được về một bản gateway"
  local f=$QC_OUT/tc-pg01-38.sse id pid s e i=0
  pid=$(sse_cap "$f" 30 "$(tok STUDENT "$U1")")
  sleep 1.5
  id=$($C ps -q gateway 2>/dev/null | head -1)
  s=$(now_ms); docker kill -s TERM "$id" >/dev/null 2>&1
  while kill -0 "$pid" 2>/dev/null && [ $i -lt 100 ]; do sleep 0.1; i=$((i+1)); done
  e=$(now_ms); wait "$pid" 2>/dev/null
  up_t 2
  chk "số dòng 'event: shutdown' trong stream" "$(grep -c '^event: shutdown' "$f" 2>/dev/null)" 1
  chk_le "thời gian đóng kết nối sau SIGTERM (ms; AC 1000 + 200 sai số đo)" "$((e-s))" 1200; }

tc_pg01_39() {  # AC7 (4) — kết nối mới sau SIGTERM bị từ chối hoặc nhận 503
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  up_t 1 || fail_tc "không hạ được về một bản gateway"
  local id pid c
  id=$($C ps -q gateway 2>/dev/null | head -1)
  ( curl -sk -o /dev/null -m 20 "$GW/api/v1/_test/slow?ms=5000" >/dev/null 2>&1 ) &
  pid=$!
  sleep 0.4
  docker kill -s TERM "$id" >/dev/null 2>&1
  sleep 0.3
  c=$(code "$GW/api/v1/healthz")
  wait "$pid" 2>/dev/null
  up_t 2
  chk_nre "kết nối mới KHÔNG được 2xx" "$c" '^2'
  chk_re "mã của kết nối mới (503 / từ chối)" "$c" '^(000|502|503)$'; }

tc_pg01_40() {  # AC7 (nhánh lỗi) — request vượt SHUTDOWN_TIMEOUT bị cắt → log warn số request bị cắt, thoát mã 1
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local pid s e ec nmax
  env SHUTDOWN_TIMEOUT=2s $CT up -d --force-recreate --scale gateway=1 --wait gateway >/dev/null 2>&1
  wait_ready 90 || fail_tc "gateway không sẵn sàng sau khi đặt SHUTDOWN_TIMEOUT=2s"
  ( curl -sk -o /dev/null -m 40 "$GW/api/v1/_test/slow?ms=10000" >/dev/null 2>&1 ) &
  pid=$!
  sleep 0.5; s=$(now_ms); $C stop -t 30 gateway >/dev/null 2>&1; e=$(now_ms); wait "$pid" 2>/dev/null
  ec=$($C ps -a --format '{{.Service}} {{.ExitCode}}' 2>/dev/null | grep '^gateway' | awk '{print $2}' | head -1)
  $C logs --no-log-prefix --since 90s gateway > "$QC_OUT/tc-pg01-40.log" 2>&1
  nmax=$(jq -c 'select((.level|ascii_downcase)=="warn")' "$QC_OUT/tc-pg01-40.log" 2>/dev/null \
         | jq -r '[to_entries[]|select(.value|type=="number")|.value]|max // 0' 2>/dev/null | sort -n | tail -1)
  gw_restore test
  chk "mã thoát khi cắt request quá hạn" "$ec" 1
  chk_le "thời gian tắt (ms) ≈ SHUTDOWN_TIMEOUT 2 s" "$((e-s))" 8000
  chk_ge "dòng warn có số request bị cắt ≥ 1" "${nmax:-0}" 1; }

tc_pg01_41() {  # AC7 — test Go
  gt './cmd/gateway ./internal/httpapi' 'TestGracefulShutdown|TestShutdown_ForcedAfterTimeout'; }

# ============================================================ AC8 — giới hạn tầng HTTP
tc_pg01_42() {  # AC8 — thân 2 MB > MAX_BODY_BYTES → 413 PAYLOAD_TOO_LARGE
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local out
  out=$(head -c 2000000 /dev/zero | tr '\0' a | curl -sk --max-time 60 -X POST -H 'Content-Type: application/json' \
        -H "$H" --data-binary @- -w '\n%{http_code}' "$GW/api/v1/_test/items" 2>/dev/null)
  chk "mã" "$(printf '%s' "$out" | tail -1)" 413
  chk "code" "$(printf '%s' "$out" | sed '$d' | jq -r '.code // empty' 2>/dev/null)" PAYLOAD_TOO_LARGE
  chk "details.max_bytes (SRS 6.1)" "$(printf '%s' "$out" | sed '$d' | jq -r '.details.max_bytes // empty' 2>/dev/null)" 1048576; }

tc_pg01_43() {  # AC8 (biên dưới) — thân đúng 1 048 576 byte KHÔNG bị 413
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local f=$QC_TMP/body-1mib.json c
  { printf '{"name":"'; head -c 1048565 /dev/zero | tr '\0' a; printf '"}'; } > "$f"
  chk "kích thước thân" "$(wc -c < "$f" | tr -d ' ')" 1048576
  c=$(CURL_MAX=60 code -X POST -H 'Content-Type: application/json' -H "$H" -H "Idempotency-Key: qc-1mib-$(now_ms)" \
      --data-binary @"$f" "$GW/api/v1/_test/items")
  chk_nre "mã KHÔNG phải 413 (thân hợp lệ JSON nên có thể 2xx hoặc 422)" "$c" '^413$'
  chk_re "mã nằm trong dải hợp lệ" "$c" '^(200|201|202|400|415|422)$'; }

tc_pg01_44() {  # AC8 (biên trên) — thân 1 048 577 byte → 413
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local f=$QC_TMP/body-1mib1.json c
  { printf '{"name":"'; head -c 1048566 /dev/zero | tr '\0' a; printf '"}'; } > "$f"
  chk "kích thước thân" "$(wc -c < "$f" | tr -d ' ')" 1048577
  c=$(CURL_MAX=60 code -X POST -H 'Content-Type: application/json' -H "$H" -H "Idempotency-Key: qc-1mib1-$(now_ms)" \
      --data-binary @"$f" "$GW/api/v1/_test/items")
  chk "mã" "$c" 413; }

tc_pg01_45() {  # AC8 — gửi header chậm hơn 5 s → server đóng kết nối (ReadHeaderTimeout 5 s), thẳng gateway:8080
  local s e out
  s=$(now_ms)
  out=$(rawhttp gateway 8080 'GET /healthz HTTP/1.1\r\nHost: x\r\n' 12)
  e=$(now_ms)
  chk_nre "không nhận được trả lời 2xx" "$out" '^HTTP/1\.1 2'
  chk_ge "thời gian tới lúc đóng (ms)" "$((e-s))" 3000
  chk_le "thời gian tới lúc đóng (ms)" "$((e-s))" 11000; }

tc_pg01_46() {  # AC8 — header lớn hơn 64 KiB → 431
  local big out
  big=$(head -c 70000 /dev/zero | tr '\0' a)
  out=$(rawhttp gateway 8080 "GET /healthz HTTP/1.1\r\nHost: x\r\nX-Qc-Big: $big\r\n\r\n" 8)
  printf '%s' "$out" | head -3 > "$QC_OUT/tc-pg01-46.txt" 2>/dev/null
  chk_re "dòng trạng thái" "$(printf '%s' "$out" | head -1 | tr -d '\r')" '^HTTP/1\.1 431'; }

tc_pg01_47() {  # AC8 — thân gửi chậm quá ReadTimeout 15 s → đóng
  local s e out
  s=$(now_ms)
  out=$(rawhttp gateway 8080 'POST /api/v1/_test/items HTTP/1.1\r\nHost: x\r\nContent-Type: application/json\r\nContent-Length: 50\r\n\r\n{"name":"a' 25)
  e=$(now_ms)
  chk_nre "không nhận được trả lời 2xx" "$out" '^HTTP/1\.1 2'
  chk_ge "thời gian tới lúc đóng (ms)" "$((e-s))" 12000
  chk_le "thời gian tới lúc đóng (ms)" "$((e-s))" 24000; }

tc_pg01_48() {  # AC8 — trả lời lỗi theo định dạng thống nhất (SRS 6.1): đúng tập trường, trace_id 32 hex
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local body keys
  body=$(head -c 2000000 /dev/zero | tr '\0' a | curl -sk --max-time 60 -X POST -H 'Content-Type: application/json' \
         -H "$H" --data-binary @- "$GW/api/v1/_test/items" 2>/dev/null)
  keys=$(printf '%s' "$body" | jq -r 'keys_unsorted|map(select(.=="code" or .=="message" or .=="details" or .=="retry_after" or .=="trace_id"))|length' 2>/dev/null)
  chk "số trường hợp lệ = tổng số trường" "$keys" "$(printf '%s' "$body" | jq -r 'keys|length' 2>/dev/null)"
  chk_re "trace_id 32 hex" "$(printf '%s' "$body" | jq -r '.trace_id // empty' 2>/dev/null)" '^[0-9a-f]{32}$'
  chk_re "message tiếng Việt, không lộ nội bộ" "$(printf '%s' "$body" | jq -r '.message // empty' 2>/dev/null)" '.+'
  chk_nre "message không chứa đường dẫn/SQL" "$(printf '%s' "$body" | jq -r '.message // empty' 2>/dev/null)" '(/[a-z]+/[a-z]+\.go|select |insert )'; }

tc_pg01_49() {  # AC8 — test Go
  gt ./internal/httpapi 'TestServerLimits|TestServerLimits_SlowHeader|TestServerLimits_HugeHeader'; }

# ============================================================ AC9 — deadline xuống tận DB/Redis
tc_pg01_50() {  # AC9 — REQUEST_TIMEOUT=1s, db-sleep?seconds=5 → 504 DEADLINE_EXCEEDED trong ≤ 1,6 s
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  gw_env test REQUEST_TIMEOUT=1s || fail_tc "không dựng lại gateway với REQUEST_TIMEOUT=1s"
  local s e out
  s=$(now_ms); out=$(curl -sk -m 10 -w '\n%{http_code}' "$GW/api/v1/_test/db-sleep?seconds=5" 2>/dev/null); e=$(now_ms)
  gw_restore test
  chk "mã" "$(printf '%s' "$out" | tail -1)" 504
  chk "code" "$(printf '%s' "$out" | sed '$d' | jq -r '.code // empty' 2>/dev/null)" DEADLINE_EXCEEDED
  chk_le "thời gian trả lời (ms)" "$((e-s))" 1600; }

tc_pg01_51() {  # AC9 — truy vấn pg_sleep bị HUỶ ở Postgres (pg_stat_activity trống sau 1 s)
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  gw_env test REQUEST_TIMEOUT=1s || fail_tc "không dựng lại gateway với REQUEST_TIMEOUT=1s"
  local n
  curl -sk -m 10 -o /dev/null "$GW/api/v1/_test/db-sleep?seconds=5" 2>/dev/null
  sleep 1
  n=$(pg_sleep_cnt)
  gw_restore test
  chk "số truy vấn pg_sleep còn chạy" "$n" 0; }

tc_pg01_52() {  # AC9 — redis-block (BLPOP) cũng 504 và không còn client blocked
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  gw_env test REQUEST_TIMEOUT=1s || fail_tc "không dựng lại gateway với REQUEST_TIMEOUT=1s"
  local out n
  out=$(curl -sk -m 10 -w '\n%{http_code}' "$GW/api/v1/_test/redis-block?seconds=5" 2>/dev/null)
  sleep 1
  n=$($RDS client list 2>/dev/null | grep -c 'cmd=blpop')
  gw_restore test
  chk "mã" "$(printf '%s' "$out" | tail -1)" 504
  chk "code" "$(printf '%s' "$out" | sed '$d' | jq -r '.code // empty' 2>/dev/null)" DEADLINE_EXCEEDED
  chk "số client Redis còn blocked (cmd=blpop)" "$n" 0; }

tc_pg01_53() {  # AC9 — client ngắt giữa chừng (curl -m 0.3) cũng huỷ truy vấn (REQUEST_TIMEOUT mặc định 30 s)
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local n0 n1
  n0=$(pg_sleep_cnt)
  curl -sk -m 0.3 -o /dev/null "$GW/api/v1/_test/db-sleep?seconds=5" 2>/dev/null
  sleep 1.5
  n1=$(pg_sleep_cnt)
  chk "số truy vấn pg_sleep trước khi gọi" "$n0" 0
  chk "số truy vấn pg_sleep 1,5 s sau khi client ngắt" "$n1" 0; }

tc_pg01_54() {  # AC9 — test Go
  gt ./internal/httpapi 'TestDeadline_DB|TestDeadline_Redis|TestDeadline_ClientCancel'; }

# ============================================================ AC10 — pool pgx
tc_pg01_55() {  # AC10 — test Go
  gt ./internal/platform/db 'TestPool_MaxConns|TestPool_AcquireHonorsDeadline'; }

tc_pg01_56() {  # AC10 (hộp đen) — DB_MAX_CONNS=3, 20 request đồng thời giữ truy vấn 1 s → không bao giờ > 3 kết nối
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  env DB_MAX_CONNS=3 $CT up -d --force-recreate --scale gateway=1 --wait gateway >/dev/null 2>&1
  wait_ready 90 || fail_tc "gateway không sẵn sàng với DB_MAX_CONNS=3"
  local i n max=0 f=$QC_OUT/tc-pg01-56-samples.txt
  : > "$f"
  i=0; while [ $i -lt 20 ]; do
    ( curl -sk -o /dev/null -m 30 "$GW/api/v1/_test/db-sleep?seconds=1" >/dev/null 2>&1 & ) ; i=$((i+1)); done
  i=0; while [ $i -lt 25 ]; do
    n=$($PSQL -c "select count(*) from pg_stat_activity where application_name='edupilot-gateway'" 2>/dev/null | tr -d ' \r')
    case $n in ''|*[!0-9]*) n=0;; esac
    echo "$n" >> "$f"; [ "$n" -gt "$max" ] && max=$n
    i=$((i+1))
  done
  $C exec -T postgres psql "$PGB" -At -c 'SHOW POOLS' > "$QC_OUT/tc-pg01-56-pgbouncer.txt" 2>&1
  gw_restore test
  chk_le "số kết nối tối đa của pool (application_name='edupilot-gateway')" "$max" 3
  chk_ge "đã quan sát được ít nhất 1 kết nối của pool (nếu 0 → xem Q-QC-01-3)" "$max" 1; }

tc_pg01_57() {  # AC10 — request chờ lấy kết nối vẫn tuân deadline: hết deadline → 504, không treo
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  env DB_MAX_CONNS=1 REQUEST_TIMEOUT=2s $CT up -d --force-recreate --scale gateway=1 --wait gateway >/dev/null 2>&1
  wait_ready 90 || fail_tc "gateway không sẵn sàng với DB_MAX_CONNS=1"
  local i s e n504
  : > "$QC_OUT/tc-pg01-57.codes"
  s=$(now_ms)
  i=0; while [ $i -lt 5 ]; do
    ( code "$GW/api/v1/_test/db-sleep?seconds=5" >> "$QC_OUT/tc-pg01-57.codes" ) & i=$((i+1)); done
  wait
  e=$(now_ms)
  n504=$(tr -d '\n' < "$QC_OUT/tc-pg01-57.codes" | grep -o 504 | grep -c .)
  gw_restore test
  chk "số request trả 504" "$n504" 5
  chk_le "tổng thời gian 5 request song song (ms; không treo)" "$((e-s))" 6000; }

# ============================================================ AC11 — log truy vấn chậm
tc_pg01_58() {  # AC11 — truy vấn 250 ms → ĐÚNG MỘT dòng `slow query` mức warn, duration_ms ≥ 200, đúng trace_id, có tên truy vấn
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local tid n line
  tid=$(newtrace)
  curl -sk -m 20 -o /dev/null -H "traceparent: 00-$tid-00f067aa0ba902b7-01" "$GW/api/v1/_test/db-sleep?seconds=0.25" 2>/dev/null
  sleep 2
  $C logs --no-log-prefix --since 60s gateway 2>/dev/null | jq -c 'select(.msg=="slow query" and .trace_id=="'"$tid"'")' 2>/dev/null > "$QC_OUT/tc-pg01-58.log"
  n=$(grep -c . "$QC_OUT/tc-pg01-58.log"); line=$(head -1 "$QC_OUT/tc-pg01-58.log")
  chk "số dòng 'slow query' của request" "$n" 1
  chk "mức log" "$(printf '%s' "$line" | jq -r '.level // empty' 2>/dev/null | tr 'A-Z' 'a-z')" warn
  chk_ge "duration_ms" "$(printf '%s' "$line" | jq -r '.duration_ms // 0' 2>/dev/null | cut -d. -f1)" 200
  chk_ge "có trường tên truy vấn (khoá khớp /quer|name/, giá trị khác rỗng)" \
    "$(printf '%s' "$line" | jq -r '[to_entries[]|select((.key|test("quer|name";"i")) and (.key!="duration_ms") and ((.value|tostring|length)>0))]|length' 2>/dev/null)" 1; }

tc_pg01_59() {  # AC11 — truy vấn 50 ms → KHÔNG sinh dòng nào
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local tid n
  tid=$(newtrace)
  curl -sk -m 20 -o /dev/null -H "traceparent: 00-$tid-00f067aa0ba902b7-01" "$GW/api/v1/_test/db-sleep?seconds=0.05" 2>/dev/null
  sleep 2
  n=$($C logs --no-log-prefix --since 60s gateway 2>/dev/null | jq -c 'select(.msg=="slow query" and .trace_id=="'"$tid"'")' 2>/dev/null | wc -l | tr -d ' ')
  chk "số dòng 'slow query' của truy vấn 50 ms" "$n" 0; }

tc_pg01_60() {  # AC11 — dòng `slow query` KHÔNG chứa giá trị tham số
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local tid line
  tid=$(newtrace)
  curl -sk -m 20 -o /dev/null -H "traceparent: 00-$tid-00f067aa0ba902b7-01" "$GW/api/v1/_test/db-sleep?seconds=0.37" 2>/dev/null
  sleep 2
  line=$($C logs --no-log-prefix --since 60s gateway 2>/dev/null | jq -c 'select(.msg=="slow query" and .trace_id=="'"$tid"'")' 2>/dev/null | head -1)
  chk_re "có dòng slow query để kiểm" "$line" 'slow query'
  chk "số lần xuất hiện giá trị tham số 0.37" "$(printf '%s' "$line" | grep -cF '0.37')" 0; }

tc_pg01_61() {  # AC11 — test Go
  gt ./internal/platform/db 'TestSlowQueryLog|TestSlowQueryLog_NoArgs'; }

# ============================================================ AC12 — worker
tc_pg01_62() {  # AC12 — /healthz nội bộ của worker ở :8081 trả 200 (gọi thẳng từ container postgres)
  local out
  out=$(rawhttp worker 8081 'GET /healthz HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n' 5)
  printf '%s' "$out" | head -5 > "$QC_OUT/tc-pg01-62.txt" 2>/dev/null
  chk_re "dòng trạng thái" "$(printf '%s' "$out" | head -1 | tr -d '\r')" '^HTTP/1\.1 200'; }

tc_pg01_63() {  # AC12 — `worker -healthcheck` thoát 0 (chứng minh qua Health của container, distroless không có shell)
  local wid hc
  wid=$($C ps -q worker 2>/dev/null | head -1)
  [ -n "$wid" ] || { fail_tc "không tìm thấy container worker"; return; }
  hc=$(docker inspect -f '{{json .Config.Healthcheck.Test}}' "$wid" 2>/dev/null)
  chk_re "healthcheck gọi -healthcheck" "$hc" 'healthcheck'
  chk "Health của worker" "$(docker inspect -f '{{.State.Health.Status}}' "$wid" 2>/dev/null)" healthy
  chk "FailingStreak" "$(docker inspect -f '{{.State.Health.FailingStreak}}' "$wid" 2>/dev/null)" 0; }

tc_pg01_64() {  # AC12 — vòng lặp consumer chạy thật: một dòng outbox được relay nhặt trong ≤ 5 s
  local i=0 n=0
  $PSQL -c "insert into outbox (topic, payload) values ('test.qc.ping', '{\"qc\":1}')" >/dev/null 2>&1 \
    || { fail_tc "không chèn được dòng outbox (cần US-PG-02)"; return; }
  while [ $i -lt 10 ]; do
    n=$($PSQL -c "select count(*) from outbox where topic='test.qc.ping' and (enqueued_at is not null or attempts > 0 or dead_at is not null)" 2>/dev/null | tr -d ' \r')
    case $n in ''|*[!0-9]*) n=0;; esac
    [ "$n" -ge 1 ] && break
    sleep 0.5; i=$((i+1))
  done
  $PSQL -c "delete from outbox where topic='test.qc.ping'" >/dev/null 2>&1
  chk_ge "số dòng outbox đã được worker nhặt trong ≤ 5 s" "$n" 1; }

tc_pg01_65() {  # AC12 — SIGTERM worker → thoát mã 0 trong ≤ 10 s
  local s e ec
  s=$(now_ms); $C stop -t 15 worker >/dev/null 2>&1; e=$(now_ms)
  ec=$($C ps -a --format '{{.Service}} {{.ExitCode}}' 2>/dev/null | grep '^worker' | awk '{print $2}' | head -1)
  worker_up || fail_tc "worker không trở lại healthy sau khi khôi phục"
  chk "mã thoát của worker" "$ec" 0
  chk_le "thời gian tắt worker (ms)" "$((e-s))" 10000; }

tc_pg01_66() {  # AC12 — test Go
  gt ./cmd/worker 'TestWorker_Health|TestWorker_Shutdown'; }

# ============================================================ AC13 — Makefile
tc_pg01_67() {  # AC13 — có đủ 5 mục tiêu run, test, lint, sqlc, migrate
  local t missing=""
  for t in run test lint sqlc migrate; do
    make -C backend-go -n "$t" >/dev/null 2>&1 || missing="$missing $t"
  done
  chk "mục tiêu Makefile thiếu" "$missing" ""; }

tc_pg01_68() {  # AC13 — `make test` = go test -race ./... với TESTCONTAINERS_RYUK_DISABLED=true
  local out=$QC_OUT/tc-pg01-68.txt
  make -C backend-go -n test > "$out" 2>&1
  chk "số lần RYUK_DISABLED=true" "$(grep -c 'RYUK_DISABLED=true' "$out")" 1
  chk_ge "có cờ -race" "$(grep -c -- '-race' "$out")" 1
  chk_ge "chạy toàn bộ gói (./...)" "$(grep -c '\./\.\.\.' "$out")" 1; }

tc_pg01_69() {  # AC13 — tự lấy DOCKER_HOST của colima khi chưa đặt
  local out=$QC_OUT/tc-pg01-69.txt
  make -C backend-go -n test > "$out" 2>&1
  chk_ge "có DOCKER_HOST" "$(grep -c 'DOCKER_HOST' "$out")" 1
  chk_ge "trỏ tới socket colima" "$(grep -ci 'colima' "$out")" 1; }

# ============================================================ AC14 — phụ thuộc chết lúc chạy / chưa lên lúc khởi động
tc_pg01_70() {  # AC14 — tắt Redis: healthz vẫn 200, readyz 503 NOT_READY + details.redis="down" trong ≤ 10 s
  local i=0 out c body hz sec=99
  $C stop redis >/dev/null 2>&1
  while [ $i -lt 10 ]; do
    out=$(curl -sk -m 5 -w '\n%{http_code}' "$GW/api/v1/readyz" 2>/dev/null)
    c=$(printf '%s' "$out" | tail -1); body=$(printf '%s' "$out" | sed '$d')
    if [ "$c" = 503 ] && [ "$(printf '%s' "$body" | jq -r '.details.redis // empty' 2>/dev/null)" = down ]; then sec=$i; break; fi
    sleep 1; i=$((i+1))
  done
  hz=$(code "$GW/api/v1/healthz")
  $C start redis >/dev/null 2>&1; wait_ready 60
  chk "mã readyz" "$c" 503
  chk "code" "$(printf '%s' "$body" | jq -r '.code // empty' 2>/dev/null)" NOT_READY
  chk "details.redis" "$(printf '%s' "$body" | jq -r '.details.redis // empty' 2>/dev/null)" down
  chk_le "số giây tới khi readyz báo down" "$sec" 10
  chk "mã healthz khi Redis chết" "$hz" 200; }

tc_pg01_71() {  # AC14 — bật Redis lại → readyz 200 trong ≤ 10 s, không cần khởi động lại gateway
  local i=0 c sec=99
  $C stop redis >/dev/null 2>&1; sleep 3
  $C start redis >/dev/null 2>&1
  while [ $i -lt 10 ]; do
    c=$(code "$GW/api/v1/readyz"); [ "$c" = 200 ] && { sec=$i; break; }
    sleep 1; i=$((i+1))
  done
  wait_ready 60
  chk "mã readyz sau khi Redis trở lại" "$c" 200
  chk_le "số giây tới khi readyz 200" "$sec" 10; }

tc_pg01_72() {  # AC14 — tiến trình KHÔNG thoát, KHÔNG restart qua chu kỳ Redis chết/sống
  local id before after
  id=$($C ps -q gateway 2>/dev/null | head -1)
  [ -n "$id" ] || { fail_tc "không tìm thấy container gateway"; return; }
  before=$(docker inspect -f '{{.RestartCount}}|{{.State.StartedAt}}|{{.State.Pid}}|{{.State.Running}}' "$id" 2>/dev/null)
  $C stop redis >/dev/null 2>&1; sleep 8; $C start redis >/dev/null 2>&1; wait_ready 60
  after=$(docker inspect -f '{{.RestartCount}}|{{.State.StartedAt}}|{{.State.Pid}}|{{.State.Running}}' "$id" 2>/dev/null)
  chk "RestartCount|StartedAt|Pid|Running trước = sau" "$after" "$before"
  chk_re "tiến trình vẫn chạy" "$after" 'true$'; }

tc_pg01_73() {  # AC14 — tắt Postgres: readyz 503 details.db="down", healthz vẫn 200
  local i=0 out c body hz sec=99
  $C stop postgres >/dev/null 2>&1
  while [ $i -lt 10 ]; do
    out=$(curl -sk -m 5 -w '\n%{http_code}' "$GW/api/v1/readyz" 2>/dev/null)
    c=$(printf '%s' "$out" | tail -1); body=$(printf '%s' "$out" | sed '$d')
    if [ "$c" = 503 ] && [ "$(printf '%s' "$body" | jq -r '.details.db // empty' 2>/dev/null)" = down ]; then sec=$i; break; fi
    sleep 1; i=$((i+1))
  done
  hz=$(code "$GW/api/v1/healthz")
  heal
  chk "mã readyz" "$c" 503
  chk "code" "$(printf '%s' "$body" | jq -r '.code // empty' 2>/dev/null)" NOT_READY
  chk "details.db" "$(printf '%s' "$body" | jq -r '.details.db // empty' 2>/dev/null)" down
  chk_le "số giây tới khi readyz báo down" "$sec" 10
  chk "mã healthz khi Postgres chết" "$hz" 200; }

tc_pg01_74() {  # AC14 — tắt CẢ HAI: details.db="down" và details.redis="down", tiến trình không thoát
  local i=0 out c body id before after
  id=$($C ps -q gateway 2>/dev/null | head -1)
  before=$(docker inspect -f '{{.RestartCount}}|{{.State.Running}}' "$id" 2>/dev/null)
  $C stop postgres redis >/dev/null 2>&1
  while [ $i -lt 12 ]; do
    out=$(curl -sk -m 5 -w '\n%{http_code}' "$GW/api/v1/readyz" 2>/dev/null)
    c=$(printf '%s' "$out" | tail -1); body=$(printf '%s' "$out" | sed '$d')
    [ "$c" = 503 ] && break
    sleep 1; i=$((i+1))
  done
  after=$(docker inspect -f '{{.RestartCount}}|{{.State.Running}}' "$id" 2>/dev/null)
  heal
  chk "mã readyz" "$c" 503
  chk "details.db" "$(printf '%s' "$body" | jq -r '.details.db // empty' 2>/dev/null)" down
  chk "details.redis" "$(printf '%s' "$body" | jq -r '.details.redis // empty' 2>/dev/null)" down
  chk "RestartCount|Running trước = sau" "$after" "$before"; }

tc_pg01_75() {  # AC14 / SRS 3.4 — khởi động khi Postgres chưa lên: thử lại mỗi 1 s, log warn, quá STARTUP_TIMEOUT thoát mã 1 nêu tên phụ thuộc
  bins || { fail_tc "không dựng được binary"; return; }
  local out=$QC_OUT/tc-pg01-75.log nwarn
  run_gw "$out" $(full7 | grep -v '^DATABASE_URL=' | grep -v '^REDIS_URL=') \
    DATABASE_URL=postgres://u:p@127.0.0.1:1/db REDIS_URL="redis://127.0.0.1:6380/0" STARTUP_TIMEOUT=3s
  nwarn=$(jq -c 'select((.level|ascii_downcase)=="warn")' "$out" 2>/dev/null | wc -l | tr -d ' ')
  chk "rc" "$RC" 1
  chk_ge "ms ≈ STARTUP_TIMEOUT 3 s" "$MS" 2500
  chk_le "ms" "$MS" 9000
  chk_ge "số dòng warn (mỗi giây một dòng)" "${nwarn:-0}" 2
  chk_le "số dòng warn" "${nwarn:-0}" 6
  chk_ge "dòng lỗi nêu tên phụ thuộc Postgres" "$(errline "$out" | grep -ciE 'postgres|database|\"db\"|db ')" 1; }

tc_pg01_76() {  # AC14 / SRS 3.4 — khởi động khi Redis chưa lên: thoát mã 1 nêu redis
  bins || { fail_tc "không dựng được binary"; return; }
  local out=$QC_OUT/tc-pg01-76.log nwarn usr pw db
  usr=$(envv POSTGRES_USER); [ -n "$usr" ] || usr=edupilot
  db=$(envv POSTGRES_DB); [ -n "$db" ] || db=edupilot
  pw=$(envv POSTGRES_PASSWORD)
  [ -n "$pw" ] || { fail_tc "thiếu POSTGRES_PASSWORD trong .env.local"; return; }
  run_gw "$out" $(full7 | grep -v '^DATABASE_URL=' | grep -v '^REDIS_URL=') \
    DATABASE_URL="postgres://$usr:$pw@127.0.0.1:5433/$db" REDIS_URL=redis://127.0.0.1:1/0 STARTUP_TIMEOUT=3s
  nwarn=$(jq -c 'select((.level|ascii_downcase)=="warn")' "$out" 2>/dev/null | wc -l | tr -d ' ')
  chk "rc" "$RC" 1
  chk_ge "ms ≈ STARTUP_TIMEOUT 3 s" "$MS" 2500
  chk_le "ms" "$MS" 9000
  chk_ge "số dòng warn" "${nwarn:-0}" 2
  chk_ge "dòng lỗi nêu tên phụ thuộc Redis" "$(errline "$out" | grep -ci 'redis')" 1; }

tc_pg01_77() {  # AC14 — test Go
  gt ./internal/httpapi 'TestReadyz_DependencyDown|TestStartup_WaitsForDeps'; }

# ============================================================ AC15 — không áp dụng (ràng buộc an toàn thay thế)
tc_pg01_78() {  # AC15 — không có bí mật trong repo (lệnh nguyên văn của AC15)
  local out
  out=$(git grep -nE 'JWT_SECRET_KEY=[^ ]{20,}' -- ':!.env.example' ':!docs' 2>/dev/null)
  printf '%s' "$out" > "$QC_OUT/tc-pg01-78.txt"
  chk "số dòng khớp bí mật trong repo" "$(printf '%s' "$out" | grep -c .)" 0; }

tc_pg01_79() {  # AC15 — .env.example chỉ có giá trị dev giả, khác bí mật thật ở .env.local
  local v
  chk_ok ".env.example tồn tại" test -f .env.example
  for v in DATABASE_URL REDIS_URL JWT_SECRET_KEY BLOB_ENDPOINT BLOB_BUCKET BLOB_ACCESS_KEY BLOB_SECRET_KEY; do
    chk_ge "có khai báo $v" "$(grep -c "^$v=" .env.example)" 1
  done
  chk_ne "JWT_SECRET_KEY ở .env.example khác .env.local" "$(grep '^JWT_SECRET_KEY=' .env.example | cut -d= -f2-)" "$SECRET"
  chk_ne "BLOB_SECRET_KEY ở .env.example khác .env.local" "$(grep '^BLOB_SECRET_KEY=' .env.example | cut -d= -f2-)" "$(envv BLOB_SECRET_KEY)"
  chk_ne "POSTGRES_PASSWORD ở .env.example khác .env.local" "$(grep '^POSTGRES_PASSWORD=' .env.example | cut -d= -f2-)" "$(envv POSTGRES_PASSWORD)"; }

tc_pg01_80() {  # AC15 — log không lộ bí mật THẬT của .env.local (khoá JWT, khoá blob, mật khẩu DB)
  local p n=0 v
  for v in JWT_SECRET_KEY BLOB_SECRET_KEY POSTGRES_PASSWORD PGBOUNCER_STATS_PASSWORD; do
    p=$(envv "$v"); [ -n "$p" ] || continue
    n=$($C logs --no-log-prefix gateway worker 2>/dev/null | grep -cF "$p")
    chk "số dòng log chứa giá trị $v" "$n" 0
  done; }

main 01 "$@"
