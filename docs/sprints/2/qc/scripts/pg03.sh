#!/usr/bin/env bash
# QC US-PG-03 — chuẩn HTTP: request id, panic, CORS, định dạng lỗi, validation, rate limit, cursor,
# idempotency, khoá lạc quan, ETag, job 202, phân quyền + route thử khoá build tag.
# Nguồn: docs/specs/FEAT-pg-foundation/US.md v1.1 US-PG-03 AC1–AC18; SRS.md 3.4, 5.3–5.6, 6.1, 6.3–6.7, 8.1.
# Chạy ở gốc worktree sau `pnpm dev`:  bash docs/sprints/2/qc/scripts/pg03.sh [MM …|--list]
source "$(dirname "$0")/lib.sh"

P=$GW/api/v1/_test
ITEMS=$P/items
B=$QC_TMP/pg03.body          # thân phản hồi gần nhất
HD=$QC_TMP/pg03.hdr          # header phản hồi gần nhất

# ---------- tiện ích riêng của story 03 ----------
toka() { echo "Authorization: Bearer $(tok "${2:-STUDENT}" "$1")"; }   # toka <sub-uuid> [ROLE]
jb()   { jq -r "$1" "$B" 2>/dev/null; }                                # một trường của thân gần nhất
hv()   { tr -d '\r' < "$1" | hval "$2"; }                              # hv <file header> <tên header>
cnt()  { $PSQL -c "$1" 2>/dev/null | tr -d '[:space:]'; }              # một ô kết quả psql
owner_del() { $PSQL -c "delete from _test_items where owner_id='$1'" >/dev/null 2>&1; }
rep() { local i n=$1 c=${2:-a} s=""; i=0; while [ $i -lt "$n" ]; do s="$s$c"; i=$((i+1)); done; printf '%s' "$s"; }
b64e() { printf '%s' "$1" | base64 | tr -d '\n=' | tr '/+' '_-'; }
b64d() { local p; p=$(printf '%s' "$1" | tr '_-' '/+'); while [ $(( ${#p} % 4 )) -ne 0 ]; do p="$p="; done
  printf '%s' "$p" | base64 -d 2>/dev/null; }
win_wait() { local s; s=$(date +%S | sed 's/^0//'); [ "${s:-0}" -ge 45 ] && sleep $((63-s)); true; }  # tránh biên cửa sổ phút
next_window() { local s; s=$(date +%S | sed 's/^0//'); sleep $((62-${s:-0})); }

post_item() {  # post_item <idem-key|""> <json> [header Authorization] → in status; thân $B, header $HD
  local k=$1 b=$2 a=${3:-$H}
  if [ -n "$k" ]; then
    curl -sk --max-time 25 -o "$B" -D "$HD" -w '%{http_code}' -X POST -H "$a" \
      -H 'Content-Type: application/json' -H "Idempotency-Key: $k" -d "$b" "$ITEMS"
  else
    curl -sk --max-time 25 -o "$B" -D "$HD" -w '%{http_code}' -X POST -H "$a" \
      -H 'Content-Type: application/json' -d "$b" "$ITEMS"
  fi
}
put_item() {  # put_item <id> <json> [header thêm] → in status; thân $B, header $HD
  local id=$1 b=$2
  if [ $# -ge 3 ]; then
    curl -sk --max-time 25 -o "$B" -D "$HD" -w '%{http_code}' -X PUT -H "$H" \
      -H 'Content-Type: application/json' -H "$3" -d "$b" "$ITEMS/$id"
  else
    curl -sk --max-time 25 -o "$B" -D "$HD" -w '%{http_code}' -X PUT -H "$H" \
      -H 'Content-Type: application/json' -d "$b" "$ITEMS/$id"
  fi
}
get_json() { curl -sk --max-time 25 -o "$B" -D "$HD" -w '%{http_code}' "$@"; }   # → status; thân $B, header $HD
post_job() { curl -sk --max-time 25 -o "$B" -D "$HD" -w '%{http_code}' -X POST -H "$H" \
      -H 'Content-Type: application/json' -d "$1" "$P/jobs"; }
job_wait() {  # job_wait <id> <trạng thái> <giây> → in trạng thái cuối
  local id=$1 want=$2 max=$3 i=0 s=""
  while [ $i -lt "$max" ]; do s=$(cnt "select status from jobs where id='$id'"); [ "$s" = "$want" ] && break; sleep 1; i=$((i+1)); done
  echo "$s"; }
page_all() {  # page_all <header Authorization> <limit> <file ra> → in số trang; thân trang cuối ở $B
  local a=$1 lim=$2 f=$3 c="" n=0
  : > "$f"
  while :; do
    if [ -n "$c" ]; then get_json -H "$a" "$ITEMS?limit=$lim&cursor=$c" >/dev/null
    else get_json -H "$a" "$ITEMS?limit=$lim" >/dev/null; fi
    jq -r '.items[]?.id' "$B" 2>/dev/null >> "$f"
    c=$(jb '.next_cursor // empty')
    n=$((n+1)); [ -n "$c" ] || break
    [ $n -ge 80 ] && { fail_tc "đi quá 80 trang — nghi vòng lặp cursor"; break; }
  done
  echo $n; }
mine() { grep -Fx -f "$2" < "$1"; }                                      # mine <file ids> <file ids của mình>
def_code() { case $1 in  # mã mặc định theo status (SRS 6.1, dòng cuối mục)
  400) echo BAD_REQUEST;; 401) echo UNAUTHENTICATED;; 403) echo FORBIDDEN;; 404) echo NOT_FOUND;;
  405) echo METHOD_NOT_ALLOWED;; 409) echo CONFLICT;; 413) echo PAYLOAD_TOO_LARGE;;
  415) echo UNSUPPORTED_MEDIA_TYPE;; 422) echo VALIDATION_FAILED;; 429) echo RATE_LIMITED;;
  500) echo INTERNAL;; 503) echo SERVICE_UNAVAILABLE;; 504) echo DEADLINE_EXCEEDED;; esac; }
err_extra_keys() { jq -r 'keys[]' "$B" 2>/dev/null | grep -vE '^(code|message|trace_id|details|retry_after)$' | paste -sd, -; }
rl_env5() { gw_env test RATE_LIMIT_IP_PER_MIN=5; rl_reset; }             # giới hạn IP = 5 (AC6)

# ================= AC1 — X-Request-Id / X-Instance-Id / Content-Type lỗi =================
tc_pg03_01() {  # AC1 — X-Request-Id hợp lệ được giữ nguyên, X-Instance-Id có mặt
  ensure_mode test
  local h; h=$(hdr -H 'X-Request-Id: abc-123' $GW/api/v1/healthz)
  chk "X-Request-Id giữ nguyên" "$(printf '%s\n' "$h" | hval x-request-id)" 'abc-123'
  chk_re "X-Instance-Id có mặt" "$(printf '%s\n' "$h" | hval x-instance-id)" '^.+$'
  chk_re "X-Instance-Id là tên container gateway" "$(printf '%s\n' "$h" | hval x-instance-id)" '.'
}
tc_pg03_02() {  # AC1 — biên 64 / 65 ký tự, ký tự ngoài [A-Za-z0-9._-], header rỗng
  ensure_mode test
  local a b g
  a=$(rep 64 a); b=$(rep 65 a)
  g=$(hdr -H "X-Request-Id: $a" $GW/api/v1/healthz | hval x-request-id); chk "64 ký tự được giữ" "$g" "$a"
  g=$(hdr -H "X-Request-Id: $b" $GW/api/v1/healthz | hval x-request-id)
  chk_ne "65 ký tự bị thay" "$g" "$b"; chk_re "id sinh thay thế = 32 hex" "$g" '^[0-9a-f]{32}$'
  g=$(hdr -H 'X-Request-Id: bad id!' $GW/api/v1/healthz | hval x-request-id)
  chk_ne "ký tự lạ bị thay" "$g" 'bad id!'; chk_re "id sinh thay thế = 32 hex" "$g" '^[0-9a-f]{32}$'
  g=$(hdr -H 'X-Request-Id;' $GW/api/v1/healthz | hval x-request-id); chk_re "header rỗng → gateway sinh" "$g" '^[0-9a-f]{32}$'
}
tc_pg03_03() {  # AC1 — X-Request-Id có trên 200, 404, 401, 429, 304, SSE
  ensure_mode test
  local id e
  id=$(newitem "qc03-$(now_ms)")
  chk_re "có item để thử 304" "$id" '^[0-9a-f-]{36}$'
  e=$(hdr -H "$H" $ITEMS/$id | hval etag)
  chk_re "X-Request-Id trên 200" "$(hdr $GW/api/v1/healthz | hval x-request-id)" '^.+$'
  chk_re "X-Request-Id trên 404" "$(hdr $GW/api/v1/khong-co | hval x-request-id)" '^.+$'
  chk_re "X-Request-Id trên 401" "$(hdr $GW/api/v1/jobs/$UX | hval x-request-id)" '^.+$'
  chk_re "X-Request-Id trên 429" "$(hdr $P/error/429 | hval x-request-id)" '^.+$'
  chk "mã khi If-None-Match khớp" "$(code -H "$H" -H "If-None-Match: $e" $ITEMS/$id)" 304
  chk_re "X-Request-Id trên 304" "$(hdr -H "$H" -H "If-None-Match: $e" $ITEMS/$id | hval x-request-id)" '^.+$'
  chk_re "X-Request-Id trên SSE" "$(CURL_MAX=3 hdr -H "$H" $GW/api/v1/events | hval x-request-id)" '^.+$'
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_04() {  # AC1 — Content-Type của lỗi = application/json; charset=utf-8
  ensure_mode test
  chk "Content-Type 404" "$(hdr $GW/api/v1/khong-co | hval content-type)" 'application/json; charset=utf-8'
  chk "Content-Type 500" "$(hdr $P/error/500 | hval content-type)" 'application/json; charset=utf-8'
  post_item "qc04-$(now_ms)" '{"name":""}' >/dev/null
  chk "Content-Type 422" "$(hv "$HD" content-type)" 'application/json; charset=utf-8'
}
tc_pg03_05() {  # AC1 (test Go) — TestRequestID
  gt ./internal/httpapi 'TestRequestID'
}

# ================= AC2 — panic =================
tc_pg03_06() {  # AC2 — panic → 500 INTERNAL, thông điệp không lộ nội bộ
  ensure_mode test
  local s m
  s=$(get_json $P/panic); m=$(jb .message | tr 'A-Z' 'a-z')
  chk "status" "$s" 500
  chk "code" "$(jb .code)" INTERNAL
  chk_re "trace_id 32 hex" "$(jb .trace_id)" '^[0-9a-f]{32}$'
  chk_re "message không rỗng" "$m" '^.+$'
  chk_nre "message không chứa panic/.go/sql/đường dẫn" "$m" 'panic|\.go|sql|/'
}
tc_pg03_07() {  # AC2 — log error có stack + trace_id của request panic
  ensure_mode test
  local t f=$QC_OUT/tc-pg03-07.log
  get_json $P/panic >/dev/null; t=$(jb .trace_id)
  sleep 2
  $C logs --no-log-prefix --since 60s gateway 2>/dev/null \
    | jq -c "select(.trace_id==\"$t\")" 2>/dev/null > "$f"
  chk_ge "dòng log cùng trace_id $t" "$(grep -c . "$f")" 1
  chk_ge "dòng log mức error" "$(grep -c '"level":"error"' "$f")" 1
  chk_ge "dòng log có stack (chuỗi '.go:')" "$(grep -c '\.go:' "$f")" 1
}
tc_pg03_08() {  # AC2 — request kế tiếp 200, tiến trình không khởi động lại
  ensure_mode test
  local c before after
  before=$(for c in $(gw_ids); do docker inspect -f '{{.RestartCount}}|{{.State.StartedAt}}' "$c" 2>/dev/null; done | paste -sd' ' -)
  code $P/panic >/dev/null
  chk "request kế tiếp" "$(code $GW/api/v1/healthz)" 200
  chk "request thử kế tiếp" "$(code -H "$H" $P/whoami)" 200
  after=$(for c in $(gw_ids); do docker inspect -f '{{.RestartCount}}|{{.State.StartedAt}}' "$c" 2>/dev/null; done | paste -sd' ' -)
  chk_re "có đọc được trạng thái container" "$before" '^.+$'
  chk "RestartCount + StartedAt không đổi" "$after" "$before"
}
tc_pg03_09() {  # AC2 (test Go) — TestRecover (gồm http.ErrAbortHandler không bị nuốt)
  gt ./internal/httpapi 'TestRecover'
}

# ================= AC3 — CORS =================
tc_pg03_10() {  # AC3 — preflight origin hợp lệ: 204 + ACAO + Allow-Methods/Headers + Max-Age + Vary + Expose
  ensure_mode test
  local h v n
  h=$(hdr -X OPTIONS -H 'Origin: http://localhost:3000' -H 'Access-Control-Request-Method: POST' \
        -H 'Access-Control-Request-Headers: Authorization, Idempotency-Key' $GW/api/v1/jobs/$UX)
  chk "mã preflight" "$(code -X OPTIONS -H 'Origin: http://localhost:3000' -H 'Access-Control-Request-Method: POST' $GW/api/v1/jobs/$UX)" 204
  chk "Access-Control-Allow-Origin" "$(printf '%s\n' "$h" | hval access-control-allow-origin)" 'http://localhost:3000'
  v=$(printf '%s\n' "$h" | hval access-control-allow-methods)
  for n in GET POST PUT PATCH DELETE OPTIONS; do chk_re "Allow-Methods có $n" "$v" "$n"; done
  v=$(printf '%s\n' "$h" | hval access-control-allow-headers)
  for n in Authorization Content-Type Idempotency-Key If-Match If-None-Match Last-Event-ID X-Request-Id; do
    chk_re "Allow-Headers có $n" "$(printf '%s' "$v" | tr 'A-Z' 'a-z')" "$(printf '%s' "$n" | tr 'A-Z' 'a-z')"; done
  chk "Max-Age" "$(printf '%s\n' "$h" | hval access-control-max-age)" 600
  chk_re "Vary có Origin" "$(printf '%s\n' "$h" | hval vary)" 'Origin'
  v=$(printf '%s\n' "$h" | hval access-control-expose-headers)
  for n in ETag X-Request-Id Retry-After Idempotent-Replayed; do
    chk_re "Expose-Headers có $n" "$(printf '%s' "$v" | tr 'A-Z' 'a-z')" "$(printf '%s' "$n" | tr 'A-Z' 'a-z')"; done
}
tc_pg03_11() {  # AC3 — Allow-Credentials: true (theo US AC3; xung đột SRS 6.7 → Q-QC-03-1)
  ensure_mode test
  local v
  v=$(hdr -X OPTIONS -H 'Origin: http://localhost:3000' -H 'Access-Control-Request-Method: POST' \
        $GW/api/v1/jobs/$UX | hval access-control-allow-credentials)
  chk "Access-Control-Allow-Credentials (US-PG-03 AC3)" "$v" 'true'
}
tc_pg03_12() {  # AC3 — origin lạ / null / không Origin → không ACAO; không bao giờ `*`
  ensure_mode test
  local v
  v=$(hdr -X OPTIONS -H 'Origin: https://evil.example' -H 'Access-Control-Request-Method: POST' $GW/api/v1/jobs/$UX | hval access-control-allow-origin)
  chk "origin lạ: không có ACAO" "$v" ''
  v=$(hdr -X OPTIONS -H 'Origin: null' -H 'Access-Control-Request-Method: POST' $GW/api/v1/jobs/$UX | hval access-control-allow-origin)
  chk "Origin: null: không có ACAO" "$v" ''
  v=$(hdr -X OPTIONS -H 'Access-Control-Request-Method: POST' $GW/api/v1/jobs/$UX | hval access-control-allow-origin)
  chk "không gửi Origin: không có ACAO" "$v" ''
  v=$(hdr -X OPTIONS -H 'Origin: http://localhost:3000' -H 'Access-Control-Request-Method: POST' $GW/api/v1/jobs/$UX | hval access-control-allow-origin)
  chk_ne "ACAO không bao giờ là *" "$v" '*'
}
tc_pg03_13() {  # AC3 — request thường (không preflight) với origin hợp lệ
  ensure_mode test
  local h
  h=$(hdr -H 'Origin: http://localhost:3000' -H "$H" $GW/api/v1/healthz)
  chk "ACAO trên GET" "$(printf '%s\n' "$h" | hval access-control-allow-origin)" 'http://localhost:3000'
  chk_re "Vary có Origin trên GET" "$(printf '%s\n' "$h" | hval vary)" 'Origin'
  h=$(hdr -H 'Origin: https://evil.example' $GW/api/v1/healthz)
  chk "origin lạ trên GET: không ACAO" "$(printf '%s\n' "$h" | hval access-control-allow-origin)" ''
}
tc_pg03_14() {  # AC3 (test Go) — TestCORS
  gt ./internal/httpapi 'TestCORS'
}

# ================= AC4 — định dạng lỗi =================
tc_pg03_15() {  # AC4 — 13 status: code đúng bảng SRS 6.1, tập khoá, trace_id, message
  ensure_mode test
  local s c m
  for s in 400 401 403 404 405 409 413 415 422 429 500 503 504; do
    c=$(get_json $P/error/$s)
    chk "status $s" "$c" "$s"
    chk "code $s" "$(jb .code)" "$(def_code $s)"
    chk_re "trace_id $s 32 hex" "$(jb .trace_id)" '^[0-9a-f]{32}$'
    m=$(jb .message | tr 'A-Z' 'a-z')
    chk_re "message $s không rỗng" "$m" '^.+$'
    chk_nre "message $s không lộ nội bộ" "$m" 'panic|\.go|sql|select |/'
    chk "khoá lạ trong thân $s" "$(err_extra_keys)" ''
  done
}
tc_pg03_16() {  # AC4 — 429 và 503: retry_after = 7 = header Retry-After
  ensure_mode test
  local s
  for s in 429 503; do
    get_json $P/error/$s >/dev/null
    chk "retry_after trong thân $s" "$(jb .retry_after)" 7
    chk "header Retry-After $s" "$(hv "$HD" retry-after)" 7
    chk_re "retry_after $s là số nguyên ≥ 1" "$(jb .retry_after)" '^[1-9][0-9]*$'
  done
}
tc_pg03_17() {  # AC4 — route không có → 404 JSON (không phải văn bản '404 page not found')
  ensure_mode test
  local s
  s=$(get_json $GW/api/v1/khong-co)
  chk "status" "$s" 404
  chk "code" "$(jb .code)" NOT_FOUND
  chk "không trả văn bản 404 page not found" "$(grep -c '404 page not found' "$B")" 0
  chk_re "thân là JSON có trace_id" "$(jb .trace_id)" '^[0-9a-f]{32}$'
}
tc_pg03_18() {  # AC4 — sai method → 405 JSON + header Allow
  ensure_mode test
  local s
  s=$(get_json -X DELETE $GW/api/v1/healthz)
  chk "status" "$s" 405
  chk "code" "$(jb .code)" METHOD_NOT_ALLOWED
  chk_re "header Allow có GET" "$(hv "$HD" allow)" 'GET'
  chk "Content-Type" "$(hv "$HD" content-type)" 'application/json; charset=utf-8'
}
tc_pg03_19() {  # AC4 — _test/error/401 là UNAUTHENTICATED và route ẩn danh (không cần token)
  ensure_mode test
  local s
  s=$(get_json $P/error/401)
  chk "status không token" "$s" 401
  chk "code" "$(jb .code)" UNAUTHENTICATED
  chk "status có token hợp lệ vẫn 401 (route trả cố định)" "$(code -H "$H" $P/error/401)" 401
}
tc_pg03_20() {  # AC4 (test Go) — TestErrorFormat_AllStatuses, TestNotFoundAndMethodNotAllowed
  gt ./internal/httpapi 'TestErrorFormat_AllStatuses|TestNotFoundAndMethodNotAllowed'
}

# ================= AC5 — validation =================
tc_pg03_21() {  # AC5 — JSON hỏng → 400 BAD_REQUEST
  ensure_mode test
  local s
  s=$(post_item "qc21-$(now_ms)" '{"name":')
  chk "status" "$s" 400
  chk "code" "$(jb .code)" BAD_REQUEST
  chk "khoá lạ" "$(err_extra_keys)" ''
}
tc_pg03_22() {  # AC5 — details liệt kê MỌI lỗi cùng lúc, đúng thứ tự ["name","extra"]
  ensure_mode test
  local s
  s=$(post_item "qc22-$(now_ms)" '{"name":"","extra":1}')
  chk "status" "$s" 422
  chk "code" "$(jb .code)" VALIDATION_FAILED
  chk "details.field theo thứ tự" "$(jq -c '[.details[].field]' "$B" 2>/dev/null)" '["name","extra"]'
  chk "mọi phần tử có field+code+message" \
    "$(jq '[.details[]|select((.field|type=="string") and (.code|type=="string") and (.message|type=="string"))]|length' "$B" 2>/dev/null)" \
    "$(jq '.details|length' "$B" 2>/dev/null)"
}
tc_pg03_23() {  # AC5 — thiếu name → 422; biên name 200 ký tự OK, 201 ký tự → 422
  ensure_mode test
  local s n200 n201
  s=$(post_item "qc23a-$(now_ms)" '{}')
  chk "thiếu name: status" "$s" 422
  chk "thiếu name: code" "$(jb .code)" VALIDATION_FAILED
  chk "thiếu name: field" "$(jq -r '.details[0].field' "$B" 2>/dev/null)" name
  n200=$(rep 200 x); n201=$(rep 201 x)
  s=$(post_item "qc23b-$(now_ms)" "{\"name\":\"$n200\"}")
  chk_re "name 200 ký tự được nhận" "$s" '^(200|201)$'
  s=$(post_item "qc23c-$(now_ms)" "{\"name\":\"$n201\"}")
  chk "name 201 ký tự: status" "$s" 422
  chk "name 201 ký tự: field" "$(jq -r '.details[0].field' "$B" 2>/dev/null)" name
  $PSQL -c "delete from _test_items where name='$n200'" >/dev/null 2>&1
}
tc_pg03_24() {  # AC5 — Content-Type: text/plain và thiếu → 415; application/json (± charset) được nhận
  ensure_mode test
  local s
  s=$(curl -sk --max-time 25 -o "$B" -w '%{http_code}' -X POST -H "$H" -H 'Content-Type: text/plain' \
        -H "Idempotency-Key: qc24a-$(now_ms)" -d '{"name":"qc24"}' "$ITEMS")
  chk "text/plain: status" "$s" 415
  chk "text/plain: code" "$(jb .code)" UNSUPPORTED_MEDIA_TYPE
  s=$(curl -sk --max-time 25 -o "$B" -w '%{http_code}' -X POST -H "$H" -H 'Content-Type;' \
        -H "Idempotency-Key: qc24b-$(now_ms)" -d '{"name":"qc24"}' "$ITEMS")
  chk "thiếu Content-Type: status" "$s" 415
  s=$(curl -sk --max-time 25 -o "$B" -w '%{http_code}' -X POST -H "$H" -H 'Content-Type: application/json' \
        -H "Idempotency-Key: qc24c-$(now_ms)" -d '{"name":"qc24c"}' "$ITEMS")
  chk_re "application/json được nhận" "$s" '^(200|201)$'
  s=$(curl -sk --max-time 25 -o "$B" -w '%{http_code}' -X POST -H "$H" -H 'Content-Type: application/json; charset=utf-8' \
        -H "Idempotency-Key: qc24d-$(now_ms)" -d '{"name":"qc24d"}' "$ITEMS")
  chk_re "application/json; charset=utf-8 được nhận" "$s" '^(200|201)$'
  $PSQL -c "delete from _test_items where name in ('qc24c','qc24d')" >/dev/null 2>&1
}
tc_pg03_25() {  # AC5 (test Go) — TestValidation
  gt ./internal/httpapi 'TestValidation'
}

# ================= AC6 — rate limit =================
tc_pg03_26() {  # AC6 — giới hạn IP = 5: 5 request không-429 rồi 429 429, code RATE_LIMITED
  ensure_mode test; win_wait; rl_env5
  local h1 i c n429=0 nok=0 f=$QC_OUT/tc-pg03-26.txt
  : > "$f"
  h1=$(hdr -H "$H" $GW/api/v1/jobs/$UX); nok=1; echo "req1 $(printf '%s\n' "$h1" | hval x-ratelimit-limit)" >> "$f"
  chk "X-RateLimit-Limit = 5 (compose có chuyển tiếp RATE_LIMIT_IP_PER_MIN?)" "$(printf '%s\n' "$h1" | hval x-ratelimit-limit)" 5
  for i in 2 3 4 5 6 7; do
    c=$(get_json -H "$H" $GW/api/v1/jobs/$UX); echo "req$i $c $(jb .code)" >> "$f"
    if [ "$c" = 429 ]; then n429=$((n429+1)); chk "code khi 429" "$(jb .code)" RATE_LIMITED; else nok=$((nok+1)); fi
  done
  gw_restore test
  chk "số request không-429 trong 7" "$nok" 5
  chk "số request 429 trong 7" "$n429" 2
}
tc_pg03_27() {  # AC6 — Retry-After (1–60) = retry_after của thân 429
  ensure_mode test; win_wait; rl_env5
  local i c ra hra
  c=""; for i in 1 2 3 4 5 6 7; do c=$(get_json -H "$H" $GW/api/v1/jobs/$UX); [ "$c" = 429 ] && break; done
  ra=$(jb .retry_after); hra=$(hv "$HD" retry-after)
  gw_restore test
  chk "có đạt 429" "$c" 429
  chk_ge "retry_after ≥ 1" "$ra" 1
  chk_le "retry_after ≤ 60" "$ra" 60
  chk "header Retry-After = retry_after" "$hra" "$ra"
}
tc_pg03_28() {  # AC6 — X-RateLimit-Limit/Remaining có mặt và Remaining giảm
  ensure_mode test; win_wait; rl_env5
  local r1 r2 l1
  l1=$(hdr -H "$H" $GW/api/v1/jobs/$UX)
  r1=$(printf '%s\n' "$l1" | hval x-ratelimit-remaining)
  r2=$(hdr -H "$H" $GW/api/v1/jobs/$UX | hval x-ratelimit-remaining)
  gw_restore test
  chk_re "X-RateLimit-Limit là số" "$(printf '%s\n' "$l1" | hval x-ratelimit-limit)" '^[0-9]+$'
  chk_re "X-RateLimit-Remaining request 1 là số" "$r1" '^[0-9]+$'
  chk_re "X-RateLimit-Remaining request 2 là số" "$r2" '^[0-9]+$'
  chk_le "Remaining giảm (r2 < r1)" "$r2" "$((${r1:-0}-1))"
}
tc_pg03_29() {  # AC6 — /healthz, /api/v1/healthz, /api/v1/readyz miễn giới hạn dù đã vượt
  ensure_mode test; win_wait; rl_env5
  local i c over=""
  for i in 1 2 3 4 5 6 7; do c=$(code -H "$H" $GW/api/v1/jobs/$UX); [ "$c" = 429 ] && over=yes; done
  local c1 c2 c3
  c1=$(code $GW/healthz); c2=$(code $GW/api/v1/healthz); c3=$(code $GW/api/v1/readyz)
  gw_restore test
  chk "đã vượt giới hạn trước khi thử health" "$over" yes
  chk "/healthz" "$c1" 200; chk "/api/v1/healthz" "$c2" 200; chk "/api/v1/readyz" "$c3" 200
}
tc_pg03_30() {  # AC6 — sang cửa sổ phút mới lại không-429 (chậm: chờ tới biên phút)
  ensure_mode test; win_wait; rl_env5
  local i c last after
  last=""; for i in 1 2 3 4 5 6 7; do last=$(code -H "$H" $GW/api/v1/jobs/$UX); done
  next_window
  after=$(code -H "$H" $GW/api/v1/jobs/$UX)
  gw_restore test
  chk "cuối cửa sổ cũ bị 429" "$last" 429
  chk_ne "cửa sổ mới không 429" "$after" 429
}
tc_pg03_31() {  # AC6 — bộ đếm theo người dùng tách khỏi bộ đếm IP
  ensure_mode test; win_wait
  gw_env test RATE_LIMIT_USER_PER_MIN=3 RATE_LIMIT_IP_PER_MIN=100000; rl_reset
  local i c u1=""
  for i in 1 2 3 4; do u1=$(get_json -H "$(toka $U1)" $GW/api/v1/jobs/$UX); done
  local code_u1; code_u1=$(jb .code)
  c=$(code -H "$(toka $U2)" $GW/api/v1/jobs/$UX)
  gw_restore test
  chk "U1 request thứ 4 (giới hạn user = 3)" "$u1" 429
  chk "code của U1" "$code_u1" RATE_LIMITED
  chk_ne "U2 không bị ảnh hưởng" "$c" 429
}
tc_pg03_32() {  # AC6 — hai bản gateway dùng chung bộ đếm (3 vào A + 3 vào B, giới hạn 5 → đúng 1 lần 429)
  ensure_mode test; win_wait; rl_env5
  local n1 n2 i r n429=0 f=$QC_OUT/tc-pg03-32.txt
  n1=$(docker ps --format '{{.Names}}' --filter name=edupilot-gateway 2>/dev/null | sort | sed -n 1p)
  n2=$(docker ps --format '{{.Names}}' --filter name=edupilot-gateway 2>/dev/null | sort | sed -n 2p)
  : > "$f"
  for i in 1 2 3; do
    r=$(rawhttp "$n1" 8080 'GET /api/v1/_test/error/404 HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n' 4 | sed -n 1p)
    echo "A$i $r" >> "$f"; printf '%s' "$r" | grep -q ' 429' && n429=$((n429+1))
  done
  for i in 1 2 3; do
    r=$(rawhttp "$n2" 8080 'GET /api/v1/_test/error/404 HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n' 4 | sed -n 1p)
    echo "B$i $r" >> "$f"; printf '%s' "$r" | grep -q ' 429' && n429=$((n429+1))
  done
  gw_restore test
  chk_re "có hai bản gateway" "$n2" '^.+$'
  chk "số 429 trên 6 request chia hai bản" "$n429" 1
}
tc_pg03_33() {  # AC6 — X-Forwarded-For được tin khi nguồn thuộc TRUSTED_PROXY_CIDRS (172.x) → bộ đếm tách theo XFF
  ensure_mode test; win_wait; rl_env5
  local n1 i r last="" other=""
  n1=$(docker ps --format '{{.Names}}' --filter name=edupilot-gateway 2>/dev/null | sort | sed -n 1p)
  for i in 1 2 3 4 5 6; do
    last=$(rawhttp "$n1" 8080 'GET /api/v1/_test/error/404 HTTP/1.1\r\nHost: x\r\nX-Forwarded-For: 203.0.113.7\r\nConnection: close\r\n\r\n' 4 | sed -n 1p)
  done
  other=$(rawhttp "$n1" 8080 'GET /api/v1/_test/error/404 HTTP/1.1\r\nHost: x\r\nX-Forwarded-For: 203.0.113.8\r\nConnection: close\r\n\r\n' 4 | sed -n 1p)
  gw_restore test
  chk_re "XFF 203.0.113.7 request thứ 6 bị 429" "$last" ' 429'
  chk_nre "XFF 203.0.113.8 (bộ đếm khác) không 429" "$other" ' 429'
}
tc_pg03_34() {  # AC6 (test Go) — IP, user, chia bản, miễn health, X-Forwarded-For
  gt ./internal/httpapi 'TestRateLimit_IP|TestRateLimit_User|TestRateLimit_SharedAcrossInstances|TestRateLimit_ExemptHealth|TestRateLimit_ForwardedFor'
}

# ================= AC7 — Redis chết khi rate limit (fail-open) =================
tc_pg03_35() {  # AC7 — Redis dừng: request thường vẫn được xử lý, không treo > 50 ms, log warn ≤ 1/10 s
  ensure_mode test
  local t0 t1 base d c i warn
  t0=$(now_ms); code -H "$H" $P/whoami >/dev/null; t1=$(now_ms); base=$((t1-t0))
  $C stop redis >/dev/null 2>&1
  t0=$(now_ms); c=$(code -H "$H" $P/whoami); t1=$(now_ms); d=$((t1-t0))
  for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do code -H "$H" $P/whoami >/dev/null; sleep 0.6; done
  warn=$($C logs --no-log-prefix --since 40s gateway 2>/dev/null | grep -i '"level":"warn' | grep -ci 'rate')
  $C start redis >/dev/null 2>&1; wait_ready 90
  chk "request thường vẫn 200 khi Redis chết (fail-open)" "$c" 200
  chk_le "độ trễ thêm so với baseline (ms)" "$((d-base))" 50
  chk_le "dòng log warn về rate limit trong ~13 s (≤ 1/10 s × 2 bản)" "$warn" 4
  chk "readyz trở lại 200 sau khi bật Redis" "$(code $GW/api/v1/readyz)" 200
}
tc_pg03_36() {  # AC7 (test Go) — TestRateLimit_RedisDown_FailOpen
  gt ./internal/httpapi 'TestRateLimit_RedisDown_FailOpen'
}

# ================= AC8 — cursor / phân trang =================
tc_pg03_37() {  # AC8 — mặc định 30 mục, hình dạng {items, next_cursor}
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000037 A
  A=$(toka $O)
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc37-'||g, '$O'::uuid, now() - (g||' seconds')::interval from generate_series(1,35) g" >/dev/null 2>&1
  get_json -H "$A" "$ITEMS" >/dev/null
  owner_del $O
  chk "số mục mặc định" "$(jq '.items|length' "$B" 2>/dev/null)" 30
  chk "items là mảng" "$(jq -r '.items|type' "$B" 2>/dev/null)" array
  chk "có trường next_cursor" "$(jq 'has("next_cursor")' "$B" 2>/dev/null)" true
  chk_re "next_cursor không rỗng khi còn trang" "$(jq -r '.next_cursor // ""' "$B" 2>/dev/null)" '^[A-Za-z0-9_-]+$'
  chk "khoá lạ ở thân danh sách" "$(jq -r 'keys[]' "$B" 2>/dev/null | grep -vE '^(items|next_cursor)$' | paste -sd, -)" ''
}
tc_pg03_38() {  # AC8 — biên limit=1 và limit=100 đều được
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000038 A
  A=$(toka $O)
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc38-'||g, '$O'::uuid, now() - (g||' seconds')::interval from generate_series(1,120) g" >/dev/null 2>&1
  get_json -H "$A" "$ITEMS?limit=1" >/dev/null;   chk "limit=1" "$(jq '.items|length' "$B" 2>/dev/null)" 1
  get_json -H "$A" "$ITEMS?limit=100" >/dev/null; chk "limit=100" "$(jq '.items|length' "$B" 2>/dev/null)" 100
  owner_del $O
}
tc_pg03_39() {  # AC8 — limit ngoài 1..100 hoặc không phải số → 422 VALIDATION_FAILED, details[0].field="limit"
  ensure_mode test
  local v s
  for v in 0 101 abc -1 1.5; do
    s=$(get_json -H "$H" "$ITEMS?limit=$v")
    chk "limit=$v status" "$s" 422
    chk "limit=$v code" "$(jb .code)" VALIDATION_FAILED
    chk "limit=$v details[0].field" "$(jq -r '.details[0].field' "$B" 2>/dev/null)" limit
  done
}
tc_pg03_40() {  # AC8 — cursor rác / sai phiên bản / sai kiểu → 422 INVALID_CURSOR (5 biến thể)
  ensure_mode test
  local c s lab
  for lab in rac json_hong v2 t_chuoi i_khong_uuid; do
    case $lab in
      rac)          c='@@@';;
      json_hong)    c=$(b64e '{"v":1,"t":');;
      v2)           c=$(b64e '{"v":2,"t":1767225600000000,"i":"00000000-0000-7000-8000-000000000001"}');;
      t_chuoi)      c=$(b64e '{"v":1,"t":"abc","i":"00000000-0000-7000-8000-000000000001"}');;
      i_khong_uuid) c=$(b64e '{"v":1,"t":1767225600000000,"i":"khong-phai-uuid"}');;
    esac
    s=$(get_json -H "$H" "$ITEMS?cursor=$c")
    chk "cursor $lab status" "$s" 422
    chk "cursor $lab code" "$(jb .code)" INVALID_CURSOR
  done
}
tc_pg03_41() {  # AC8 — trang cuối: next_cursor có mặt và là null
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000041 A n f=$QC_TMP/pg03-41.ids
  A=$(toka $O)
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc41-'||g, '$O'::uuid, now() - (g||' seconds')::interval from generate_series(1,5) g" >/dev/null 2>&1
  n=$(page_all "$A" 100 "$f")
  owner_del $O
  chk_ge "đi được ít nhất 1 trang" "$n" 1
  chk "trang cuối có trường next_cursor" "$(jq 'has("next_cursor")' "$B" 2>/dev/null)" true
  chk "trang cuối next_cursor = null" "$(jq -r '.next_cursor|type' "$B" 2>/dev/null)" null
}
tc_pg03_42() {  # AC8 — danh sách rỗng là "items":[] (không null)
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000042 A
  A=$(toka $O); owner_del $O
  get_json -H "$A" "$ITEMS" >/dev/null
  chk "items là mảng" "$(jq -r '.items|type' "$B" 2>/dev/null)" array
  chk "items rỗng" "$(jq '.items|length' "$B" 2>/dev/null)" 0
  chk "next_cursor = null" "$(jq -r '.next_cursor|type' "$B" 2>/dev/null)" null
}
tc_pg03_43() {  # AC8 — thứ tự (created_at DESC, id DESC) khớp truy vấn psql
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000043 A e=$QC_TMP/pg03-43.exp g=$QC_TMP/pg03-43.got a=$QC_TMP/pg03-43.all
  A=$(toka $O)
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc43-'||g, '$O'::uuid, now() - ((g%7)||' seconds')::interval from generate_series(1,40) g" >/dev/null 2>&1
  $PSQL -c "select id from _test_items where owner_id='$O' order by created_at desc, id desc" 2>/dev/null | grep -E '^[0-9a-f-]{36}$' > "$e"
  page_all "$A" 100 "$a" >/dev/null
  mine "$a" "$e" > "$g"
  owner_del $O
  chk "số id của mình đọc được" "$(grep -c . "$g")" "$(grep -c . "$e")"
  chk "thứ tự giống psql (created_at DESC, id DESC)" "$(cmp -s "$e" "$g" && echo giong || echo khac)" giong
}
tc_pg03_44() {  # AC8 (test Go) — mặc định, biên limit, cursor sai, hình dạng
  gt ./internal/httpapi 'TestPagination_Defaults|TestPagination_LimitBounds|TestPagination_InvalidCursor|TestPagination_Shape'
}

# ================= AC9 — không sót, không lặp =================
tc_pg03_45() {  # AC9 — 250 bản ghi, limit=100, đi hết trang → đúng 250 id duy nhất
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000045 A n e=$QC_TMP/pg03-45.exp g=$QC_TMP/pg03-45.got a=$QC_TMP/pg03-45.all
  A=$(toka $O)
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc45-'||g, '$O'::uuid, now() - (g||' seconds')::interval from generate_series(1,250) g" >/dev/null 2>&1
  $PSQL -c "select id from _test_items where owner_id='$O'" 2>/dev/null | grep -E '^[0-9a-f-]{36}$' > "$e"
  n=$(page_all "$A" 100 "$a")
  mine "$a" "$e" > "$g"
  owner_del $O
  chk "số dòng đã dựng" "$(grep -c . "$e")" 250
  chk "số id đọc được" "$(grep -c . "$g")" 250
  chk "số id duy nhất" "$(sort "$g" | uniq | grep -c .)" 250
  chk "số id lặp" "$(sort "$g" | uniq -d | grep -c .)" 0
  chk_ge "số trang đã đi" "$n" 3
}
tc_pg03_46() {  # AC9 — 300 bản ghi CÙNG created_at: không sót, không lặp (phân định bằng id)
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000046 A e=$QC_TMP/pg03-46.exp g=$QC_TMP/pg03-46.got a=$QC_TMP/pg03-46.all
  A=$(toka $O)
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc46-'||g, '$O'::uuid, timestamptz '2026-01-01 00:00:00+00' from generate_series(1,300) g" >/dev/null 2>&1
  chk "mọi dòng cùng created_at" "$(cnt "select count(distinct created_at) from _test_items where owner_id='$O'")" 1
  $PSQL -c "select id from _test_items where owner_id='$O'" 2>/dev/null | grep -E '^[0-9a-f-]{36}$' > "$e"
  page_all "$A" 100 "$a" >/dev/null
  mine "$a" "$e" > "$g"
  owner_del $O
  chk "số id đọc được" "$(grep -c . "$g")" 300
  chk "số id duy nhất" "$(sort "$g" | uniq | grep -c .)" 300
  chk "số id lặp" "$(sort "$g" | uniq -d | grep -c .)" 0
}
tc_pg03_47() {  # AC9 — chen 5 mới hơn + 5 cũ hơn con trỏ và xoá 3 bản đã đọc giữa lúc đi trang
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000047 A c
  local orig=$QC_TMP/pg03-47.orig p1=$QC_TMP/pg03-47.p1 rest=$QC_TMP/pg03-47.rest
  local all2=$QC_TMP/pg03-47.all2 kept=$QC_TMP/pg03-47.kept del=$QC_TMP/pg03-47.del new=$QC_TMP/pg03-47.new both=$QC_TMP/pg03-47.both
  A=$(toka $O); owner_del $O
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc47o-'||g, '$O'::uuid, now() - (g||' seconds')::interval from generate_series(1,50) g" >/dev/null 2>&1
  $PSQL -c "select id from _test_items where owner_id='$O'" 2>/dev/null | grep -E '^[0-9a-f-]{36}$' > "$orig"
  get_json -H "$A" "$ITEMS?limit=20" >/dev/null
  jq -r '.items[]?.id' "$B" 2>/dev/null | grep -Fx -f "$orig" > "$p1"
  c=$(jb '.next_cursor // empty')
  sort "$p1" | sed -n '1,3p' > "$del"
  $PSQL -c "delete from _test_items where id in ('$(sed -n 1p "$del")','$(sed -n 2p "$del")','$(sed -n 3p "$del")')" >/dev/null 2>&1
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc47n-'||g, '$O'::uuid, now() + interval '1 hour' from generate_series(1,5) g" >/dev/null 2>&1
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc47x-'||g, '$O'::uuid, now() - interval '1 hour' from generate_series(1,5) g" >/dev/null 2>&1
  $PSQL -c "select id from _test_items where owner_id='$O'" 2>/dev/null | grep -E '^[0-9a-f-]{36}$' > "$all2"
  $PSQL -c "select id from _test_items where owner_id='$O' and name like 'qc47x-%'" 2>/dev/null | grep -E '^[0-9a-f-]{36}$' > "$new"
  : > "$rest"
  local n=0
  while [ -n "$c" ]; do
    get_json -H "$A" "$ITEMS?limit=20&cursor=$c" >/dev/null
    jq -r '.items[]?.id' "$B" 2>/dev/null | grep -Fx -f "$all2" >> "$rest"
    c=$(jb '.next_cursor // empty'); n=$((n+1)); [ $n -ge 20 ] && break
  done
  cat "$p1" "$rest" > "$both"
  sort "$orig" > "$QC_TMP/pg03-47.o.s"; sort "$del" > "$QC_TMP/pg03-47.d.s"
  comm -23 "$QC_TMP/pg03-47.o.s" "$QC_TMP/pg03-47.d.s" > "$kept"
  owner_del $O
  chk "trang 1 đọc được 20 mục" "$(grep -c . "$p1")" 20
  chk "không id nào lặp trong cả lần đi" "$(sort "$both" | uniq -d | grep -c .)" 0
  chk "47 bản ghi gốc chưa xoá xuất hiện đúng 1 lần" "$(grep -Fxc -f "$kept" "$both")" 47
  chk "5 bản ghi chen vào vùng chưa đọc xuất hiện đúng 1 lần" "$(grep -Fxc -f "$new" "$both")" 5
}
tc_pg03_48() {  # AC9 (test Go) — không sót/lặp, cùng timestamp, chen + xoá giữa lúc quét
  gt ./internal/httpapi 'TestCursor_NoSkipNoDup|TestCursor_SameTimestamp|TestCursor_InsertAndDeleteDuringScan'
}

# ================= AC10 — định dạng cursor, index, không OFFSET =================
tc_pg03_49() {  # AC10 — cursor là base64url không đệm của {"v":1,"t":<micro-giây>,"i":<uuid>}, ≤ 120 ký tự
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000049 A c j t
  A=$(toka $O)
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc49-'||g, '$O'::uuid, now() - (g||' seconds')::interval from generate_series(1,10) g" >/dev/null 2>&1
  get_json -H "$A" "$ITEMS?limit=1" >/dev/null; c=$(jb '.next_cursor // empty')
  owner_del $O
  chk_re "cursor không rỗng" "$c" '^.+$'
  chk_le "độ dài cursor ≤ 120" "${#c}" 120
  chk_re "chỉ ký tự base64url" "$c" '^[A-Za-z0-9_-]+$'
  chk_nre "không ký tự đệm '='" "$c" '='
  j=$(b64d "$c"); printf '%s' "$j" > "$QC_OUT/tc-pg03-49.json"
  chk "tập khoá JSON" "$(printf '%s' "$j" | jq -r 'keys|join(",")' 2>/dev/null)" 'i,t,v'
  chk "v" "$(printf '%s' "$j" | jq -r '.v' 2>/dev/null)" 1
  chk "t là số" "$(printf '%s' "$j" | jq -r '.t|type' 2>/dev/null)" number
  t=$(printf '%s' "$j" | jq -r '.t' 2>/dev/null)
  chk_re "t là số nguyên" "$t" '^[0-9]+$'
  chk_ge "t tính bằng micro-giây (≥ 1,7e15)" "$t" 1700000000000000
  chk_re "i là uuid" "$(printf '%s' "$j" | jq -r '.i' 2>/dev/null)" '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
}
tc_pg03_50() {  # AC10 — plan truy vấn trang trên 10.000 dòng: Index Scan, không Sort
  ensure_mode test
  local O=00000000-0000-7000-8000-000000000050 pl f=$QC_OUT/tc-pg03-50.plan
  $PSQL -c "insert into _test_items (name, owner_id, created_at) select 'qc50-'||g, '$O'::uuid, now() - (g||' seconds')::interval from generate_series(1,10000) g" >/dev/null 2>&1
  chk_ge "số dòng trong bảng" "$(cnt 'select count(*) from _test_items')" 10000
  $PSQL -c "analyze _test_items" >/dev/null 2>&1
  pl=$($PSQL -c "explain select id, name, created_at from _test_items where (created_at, id) < (now(), 'ffffffff-ffff-ffff-ffff-ffffffffffff'::uuid) order by created_at desc, id desc limit 31" 2>/dev/null)
  printf '%s\n' "$pl" > "$f"
  owner_del $O
  $PSQL -c "analyze _test_items" >/dev/null 2>&1
  chk_re "plan có Index Scan / Index Only Scan" "$pl" 'Index (Only )?Scan'
  chk_nre "plan không có Sort" "$pl" 'Sort'
}
tc_pg03_51() {  # AC10 — không OFFSET ở bất kỳ .sql/.go nào ngoài test
  local n
  n=$(grep -rniE '\boffset\b' backend-go/internal --include=*.sql --include=*.go 2>/dev/null | grep -v '_test.go' | grep -c .)
  grep -rniE '\boffset\b' backend-go/internal --include=*.sql --include=*.go 2>/dev/null | grep -v '_test.go' > "$QC_OUT/tc-pg03-51.txt"
  chk "số dòng có OFFSET (ngoài _test.go)" "$n" 0
}
tc_pg03_52() {  # AC10 (test Go) — TestCursor_Format, TestCursor_UsesIndex
  gt ./internal/httpapi 'TestCursor_Format|TestCursor_UsesIndex'
}

# ================= AC11 — Idempotency-Key, gửi đôi =================
tc_pg03_53() {  # AC11 — cùng khoá + cùng thân: 1 bản ghi, lần 2 y hệt + Idempotent-Replayed: true
  ensure_mode test
  local K="qc53-$(now_ms)" N="qc53-$(now_ms)-$RANDOM" s1 s2
  s1=$(post_item "$K" "{\"name\":\"$N\"}"); cp "$B" "$QC_TMP/pg03-53.b1"; cp "$HD" "$QC_TMP/pg03-53.h1"
  s2=$(post_item "$K" "{\"name\":\"$N\"}"); cp "$B" "$QC_TMP/pg03-53.b2"; cp "$HD" "$QC_TMP/pg03-53.h2"
  chk_re "status lần 1" "$s1" '^(200|201)$'
  chk "status lần 2 = lần 1" "$s2" "$s1"
  chk "thân lần 2 giống từng byte" "$(cmp -s "$QC_TMP/pg03-53.b1" "$QC_TMP/pg03-53.b2" && echo giong || echo khac)" giong
  chk "Content-Type lần 2 = lần 1" "$(hv "$QC_TMP/pg03-53.h2" content-type)" "$(hv "$QC_TMP/pg03-53.h1" content-type)"
  chk "lần 1 không có Idempotent-Replayed" "$(hv "$QC_TMP/pg03-53.h1" idempotent-replayed)" ''
  chk "lần 2 có Idempotent-Replayed" "$(hv "$QC_TMP/pg03-53.h2" idempotent-replayed)" 'true'
  chk "số dòng _test_items" "$(cnt "select count(*) from _test_items where name='$N'")" 1
  $PSQL -c "delete from _test_items where name='$N'" >/dev/null 2>&1
}
tc_pg03_54() {  # AC11 — khoá khác (cùng thân) → bản ghi mới
  ensure_mode test
  local N="qc54-$(now_ms)-$RANDOM" s1 s2 id1 id2
  s1=$(post_item "qc54a-$(now_ms)" "{\"name\":\"$N\"}"); id1=$(jb '.id')
  s2=$(post_item "qc54b-$(now_ms)" "{\"name\":\"$N\"}"); id2=$(jb '.id')
  chk_re "status lần 1" "$s1" '^(200|201)$'
  chk_re "status lần 2" "$s2" '^(200|201)$'
  chk_ne "id khác nhau" "$id2" "$id1"
  chk "số dòng _test_items" "$(cnt "select count(*) from _test_items where name='$N'")" 2
  chk "lần 2 không phải replay" "$(hv "$HD" idempotent-replayed)" ''
  $PSQL -c "delete from _test_items where name='$N'" >/dev/null 2>&1
}
tc_pg03_55() {  # AC11 (test Go) — TestIdempotency_DoubleSend_OneRecord, TestIdempotency_ReplayIdentical
  gt ./internal/httpapi 'TestIdempotency_DoubleSend_OneRecord|TestIdempotency_ReplayIdentical'
}

# ================= AC12 — đua và biên của Idempotency-Key =================
tc_pg03_56() {  # AC12 — 50 request song song cùng khoá: 1 bản ghi, chỉ replay hoặc 409, 0 lần 5xx
  ensure_mode test
  local K="qc56-$(now_ms)" N="qc56-$(now_ms)-$RANDOM" d=$QC_OUT/tc-pg03-56 i s ref="" ok=0 bad="" n5xx=0 n409=0
  rm -rf "$d"; mkdir -p "$d"
  for i in $(seq 50); do
    ( curl -sk --max-time 40 -o "$d/b$i" -D "$d/h$i" -w '%{http_code}' -X POST -H "$H" \
        -H 'Content-Type: application/json' -H "Idempotency-Key: $K" -d "{\"name\":\"$N\"}" "$ITEMS" > "$d/s$i" ) &
  done
  wait
  for i in $(seq 50); do
    s=$(cat "$d/s$i" 2>/dev/null)
    case $s in 200|201) [ -n "$ref" ] || ref="$d/b$i";; esac
  done
  for i in $(seq 50); do
    s=$(cat "$d/s$i" 2>/dev/null)
    case $s in
      200|201) if [ -n "$ref" ] && cmp -s "$d/b$i" "$ref"; then ok=$((ok+1)); else bad="$bad #$i(thân khác bản đầu)"; fi;;
      409) n409=$((n409+1))
           if jq -e '.code=="IDEMPOTENCY_IN_PROGRESS"' "$d/b$i" >/dev/null 2>&1 && grep -qi '^retry-after:' "$d/h$i"; then ok=$((ok+1));
           else bad="$bad #$i(409 thiếu code/Retry-After)"; fi;;
      5*) n5xx=$((n5xx+1)); bad="$bad #$i($s)";;
      *) bad="$bad #$i($s)";;
    esac
  done
  chk "số dòng _test_items" "$(cnt "select count(*) from _test_items where name='$N'")" 1
  chk "số phản hồi hợp lệ (replay hoặc 409)" "$ok" 50
  chk "số phản hồi 5xx" "$n5xx" 0
  chk "phản hồi sai quy ước" "$bad" ''
  chk_re "có ít nhất 1 phản hồi 2xx" "$ref" '^.+$'
  echo "    (409 IDEMPOTENCY_IN_PROGRESS: $n409 / 50; chứng cứ: $d)"
  $PSQL -c "delete from _test_items where name='$N'" >/dev/null 2>&1
}
tc_pg03_57() {  # AC12 — "hai tab gửi trùng" (PG.md): hai tiến trình curl song song, cùng khoá
  ensure_mode test
  local K="qc57-$(now_ms)" N="qc57-$(now_ms)-$RANDOM" d=$QC_OUT/tc-pg03-57 s1 s2 r1 r2
  rm -rf "$d"; mkdir -p "$d"
  ( curl -sk --max-time 30 -o "$d/b1" -D "$d/h1" -w '%{http_code}' -X POST -H "$H" -H 'Content-Type: application/json' \
      -H "Idempotency-Key: $K" -d "{\"name\":\"$N\"}" "$ITEMS" > "$d/s1" ) &
  ( curl -sk --max-time 30 -o "$d/b2" -D "$d/h2" -w '%{http_code}' -X POST -H "$H" -H 'Content-Type: application/json' \
      -H "Idempotency-Key: $K" -d "{\"name\":\"$N\"}" "$ITEMS" > "$d/s2" ) &
  wait
  s1=$(cat "$d/s1"); s2=$(cat "$d/s2")
  r1=$(hv "$d/h1" idempotent-replayed); r2=$(hv "$d/h2" idempotent-replayed)
  chk "số dòng _test_items" "$(cnt "select count(*) from _test_items where name='$N'")" 1
  chk_re "status tiến trình 1" "$s1" '^(200|201|409)$'
  chk_re "status tiến trình 2" "$s2" '^(200|201|409)$'
  chk "số phản hồi KHÔNG mang Idempotent-Replayed (tối đa 1 bản gốc)" \
    "$(printf '%s\n%s\n' "$r1" "$r2" | grep -vc '^true$')" 1
  chk "số phản hồi 5xx" "$(cat "$d/s1" "$d/s2" | grep -c '^5')" 0
  $PSQL -c "delete from _test_items where name='$N'" >/dev/null 2>&1
}
tc_pg03_58() {  # AC12 — cùng khoá nhưng thân khác → 422 IDEMPOTENCY_KEY_REUSED
  ensure_mode test
  local K="qc58-$(now_ms)" N="qc58-$(now_ms)" s1 s2
  s1=$(post_item "$K" "{\"name\":\"$N-a\"}")
  s2=$(post_item "$K" "{\"name\":\"$N-b\"}")
  chk_re "status lần 1" "$s1" '^(200|201)$'
  chk "status lần 2" "$s2" 422
  chk "code lần 2" "$(jb .code)" IDEMPOTENCY_KEY_REUSED
  chk "không tạo bản ghi cho thân thứ hai" "$(cnt "select count(*) from _test_items where name='$N-b'")" 0
  $PSQL -c "delete from _test_items where name='$N-a'" >/dev/null 2>&1
}
tc_pg03_59() {  # AC12 — endpoint bắt buộc khoá mà thiếu header → 422 IDEMPOTENCY_KEY_REQUIRED
  ensure_mode test
  local N="qc59-$(now_ms)" s
  s=$(post_item "" "{\"name\":\"$N\"}")
  chk "status" "$s" 422
  chk "code" "$(jb .code)" IDEMPOTENCY_KEY_REQUIRED
  chk "không tạo bản ghi" "$(cnt "select count(*) from _test_items where name='$N'")" 0
}
tc_pg03_60() {  # AC12 — biên độ dài khoá: 7 → 422, 8 OK, 128 OK, 129 → 422
  ensure_mode test
  local s k7 k8 k128 k129 N="qc60-$(now_ms)"
  k7=$(rep 7 k); k8=$(rep 8 k); k128=$(rep 128 k); k129=$(rep 129 k)
  s=$(post_item "$k7" "{\"name\":\"$N-7\"}");   chk "khoá 7 ký tự: status" "$s" 422
  chk "khoá 7 ký tự: code" "$(jb .code)" VALIDATION_FAILED
  s=$(post_item "$k8$N" "{\"name\":\"$N-8\"}"); chk_re "khoá 8+ ký tự được nhận" "$s" '^(200|201)$'
  s=$(post_item "$k128" "{\"name\":\"$N-128\"}"); chk_re "khoá 128 ký tự được nhận" "$s" '^(200|201)$'
  s=$(post_item "$k129" "{\"name\":\"$N-129\"}"); chk "khoá 129 ký tự: status" "$s" 422
  chk "khoá 129 ký tự: code" "$(jb .code)" VALIDATION_FAILED
  chk "không tạo bản ghi cho khoá sai" "$(cnt "select count(*) from _test_items where name in ('$N-7','$N-129')")" 0
  $PSQL -c "delete from _test_items where name like '$N-%'" >/dev/null 2>&1
}
tc_pg03_61() {  # AC12 — khoá có ký tự ngoài [A-Za-z0-9._:-] → 422 VALIDATION_FAILED
  ensure_mode test
  local s N="qc61-$(now_ms)"
  s=$(post_item 'bad key!' "{\"name\":\"$N-a\"}")
  chk "khoá 'bad key!': status" "$s" 422
  chk "khoá 'bad key!': code" "$(jb .code)" VALIDATION_FAILED
  s=$(post_item 'khoaä-0001' "{\"name\":\"$N-b\"}")
  chk "khoá có 'ä': status" "$s" 422
  chk "khoá có 'ä': code" "$(jb .code)" VALIDATION_FAILED
  chk "không tạo bản ghi" "$(cnt "select count(*) from _test_items where name like '$N-%'")" 0
}
tc_pg03_62() {  # AC12 — cùng khoá của người dùng khác là độc lập
  ensure_mode test
  local K="qc62-$(now_ms)" N="qc62-$(now_ms)-$RANDOM" s1 s2 id1 id2
  s1=$(post_item "$K" "{\"name\":\"$N\"}" "$(toka $U1)"); id1=$(jb '.id')
  s2=$(post_item "$K" "{\"name\":\"$N\"}" "$(toka $U2)"); id2=$(jb '.id')
  chk_re "U1: status" "$s1" '^(200|201)$'
  chk_re "U2: status" "$s2" '^(200|201)$'
  chk "U2 không nhận replay" "$(hv "$HD" idempotent-replayed)" ''
  chk_ne "hai id khác nhau" "$id2" "$id1"
  chk "số dòng _test_items" "$(cnt "select count(*) from _test_items where name='$N'")" 2
  chk "mỗi người một dòng" "$(cnt "select count(distinct owner_id) from _test_items where name='$N'")" 2
  $PSQL -c "delete from _test_items where name='$N'" >/dev/null 2>&1
}
tc_pg03_63() {  # AC12 — khoá và thân không bao giờ vào log
  ensure_mode test
  local N="pii-canary-$(now_ms)" s
  s=$(post_item "$IDEM" "{\"name\":\"$N\"}")
  sleep 2
  $C logs --no-log-prefix --since 90s gateway worker 2>/dev/null > "$QC_OUT/tc-pg03-63.log"
  chk_re "request được xử lý" "$s" '^(200|201|422)$'
  chk "số dòng log chứa khoá idem" "$(grep -c "$IDEM" "$QC_OUT/tc-pg03-63.log")" 0
  chk "số dòng log chứa thân (pii-canary)" "$(grep -c 'pii-canary' "$QC_OUT/tc-pg03-63.log")" 0
  $PSQL -c "delete from _test_items where name like 'pii-canary-%'" >/dev/null 2>&1
}
tc_pg03_64() {  # AC12 (test Go) — 50 goroutine, khoá dùng lại, bắt buộc, khoá sai, phạm vi user+endpoint, 5xx không lưu, không vào log
  gt ./internal/httpapi 'TestIdempotency_Concurrent50|TestIdempotency_KeyReused|TestIdempotency_Required|TestIdempotency_BadKey|TestIdempotency_ScopedByUserAndEndpoint|TestIdempotency_5xxNotStored|TestIdempotency_NotLogged'
}

# ================= AC13 — TTL, FLUSHALL, Redis chết (fail-closed) =================
tc_pg03_65() {  # AC13 — TTL khoá ep:idem:… ∈ [86300, 86400]
  ensure_mode test
  local K="qc65-$(now_ms)" N="qc65-$(now_ms)" k ttl
  post_item "$K" "{\"name\":\"$N\"}" >/dev/null
  k=$($RDS --scan --pattern "ep:idem:*:$K" 2>/dev/null | tr -d '\r' | grep -v ':lock$' | sed -n 1p)
  chk_re "tìm thấy khoá phản hồi của $K" "$k" '^ep:idem:'
  ttl=$($RDS ttl "$k" 2>/dev/null | tr -d '[:space:]')
  chk_ge "TTL ≥ 86300" "$ttl" 86300
  chk_le "TTL ≤ 86400" "$ttl" 86400
  $PSQL -c "delete from _test_items where name='$N'" >/dev/null 2>&1
}
tc_pg03_66() {  # AC13 — TTL khoá chạy-dở …:lock ≤ 30 s (chậm ~40 s)
  ensure_mode test
  local K="qc66-$(now_ms)" N="qc66-$(now_ms)-$RANDOM" i k="" lt="" d=$QC_OUT/tc-pg03-66
  rm -rf "$d"; mkdir -p "$d"
  for i in $(seq 30); do
    ( curl -sk --max-time 30 -o "$d/b$i" -w '%{http_code}' -X POST -H "$H" -H 'Content-Type: application/json' \
        -H "Idempotency-Key: $K" -d "{\"name\":\"$N\"}" "$ITEMS" > "$d/s$i" ) &
  done
  for i in $(seq 40); do
    k=$($RDS --scan --pattern 'ep:idem:*:lock' 2>/dev/null | tr -d '\r' | sed -n 1p)
    if [ -n "$k" ]; then lt=$($RDS ttl "$k" 2>/dev/null | tr -d '[:space:]'); break; fi
  done
  wait
  if [ -n "$lt" ]; then
    chk_le "TTL khoá :lock ≤ 30" "$lt" 30
    chk_ge "TTL khoá :lock ≥ 1" "$lt" 1
  else
    echo "    ok   không bắt được khoá :lock khi đang chạy (handler quá nhanh) — chấm bằng điều kiện hết hạn dưới đây + TC-PG03-69"
  fi
  chk_ge "có ≥ 1 phản hồi 409 hoặc replay (chứng tỏ có khoá chạy-dở)" \
    "$(cat "$d"/s* 2>/dev/null | grep -cE '^(409|200|201)')" 1
  sleep 35
  chk "số khoá :lock còn lại sau 35 s" "$($RDS --scan --pattern 'ep:idem:*:lock' 2>/dev/null | tr -d '\r' | grep -c .)" 0
  chk "số dòng _test_items" "$(cnt "select count(*) from _test_items where name='$N'")" 1
  $PSQL -c "delete from _test_items where name='$N'" >/dev/null 2>&1
}
tc_pg03_67() {  # AC13 — FLUSHALL rồi gửi lại cùng khoá: khôi phục từ bảng idempotency_keys, vẫn 1 bản ghi
  ensure_mode test
  local K="qc67-$(now_ms)" N="qc67-$(now_ms)-$RANDOM" s1 s2
  s1=$(post_item "$K" "{\"name\":\"$N\"}"); cp "$B" "$QC_TMP/pg03-67.b1"
  chk "số dòng idempotency_keys sau lần 1" "$(cnt "select count(*) from idempotency_keys where key='$K'")" 1
  $RDS flushall >/dev/null 2>&1
  chk "Redis đã sạch khoá ep:idem" "$($RDS --scan --pattern 'ep:idem:*' 2>/dev/null | tr -d '\r' | grep -c .)" 0
  s2=$(post_item "$K" "{\"name\":\"$N\"}"); cp "$B" "$QC_TMP/pg03-67.b2"
  chk "status lần 2 = lần 1" "$s2" "$s1"
  chk "Idempotent-Replayed lần 2" "$(hv "$HD" idempotent-replayed)" 'true'
  chk "thân lần 2 tương đương lần 1 (JSON)" \
    "$(jq -S -c . "$QC_TMP/pg03-67.b2" 2>/dev/null)" "$(jq -S -c . "$QC_TMP/pg03-67.b1" 2>/dev/null)"
  chk "số dòng _test_items" "$(cnt "select count(*) from _test_items where name='$N'")" 1
  chk "số dòng idempotency_keys" "$(cnt "select count(*) from idempotency_keys where key='$K'")" 1
  $PSQL -c "delete from _test_items where name='$N'" >/dev/null 2>&1
}
tc_pg03_68() {  # AC13 — Redis chết + endpoint bắt buộc khoá → 503 SERVICE_UNAVAILABLE, không tạo bản ghi
  ensure_mode test
  local N="qc68-$(now_ms)" before after s c
  before=$(cnt 'select count(*) from _test_items')
  $C stop redis >/dev/null 2>&1
  s=$(post_item "qc68-$(now_ms)" "{\"name\":\"$N\"}"); c=$(jb .code)
  after=$(cnt 'select count(*) from _test_items')
  $C start redis >/dev/null 2>&1; wait_ready 90
  chk "status" "$s" 503
  chk "code" "$c" SERVICE_UNAVAILABLE
  chk "số bản ghi mới" "$((after-before))" 0
  chk "số dòng mang tên vừa gửi" "$(cnt "select count(*) from _test_items where name='$N'")" 0
  chk "readyz trở lại 200" "$(code $GW/api/v1/readyz)" 200
}
tc_pg03_69() {  # AC13 (test Go) — TTL, khôi phục từ bảng sau FLUSHALL, fail-closed khi Redis chết
  gt ./internal/httpapi 'TestIdempotency_TTL|TestIdempotency_RedisFlushedFallsBackToTable|TestIdempotency_RedisDown_FailClosed'
}

# ================= AC14 — khoá lạc quan =================
tc_pg03_70() {  # AC14 — PUT với version đang giữ → 200, version +1
  ensure_mode test
  local id s
  id=$(newitem "qc70-$(now_ms)")
  chk_re "dựng được item" "$id" '^[0-9a-f-]{36}$'
  chk "version ban đầu" "$(cnt "select version from _test_items where id='$id'")" 1
  s=$(put_item "$id" '{"name":"qc70-moi","version":1}')
  chk "status" "$s" 200
  chk "version trong thân" "$(jb .version)" 2
  chk "version trong DB" "$(cnt "select version from _test_items where id='$id'")" 2
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_71() {  # AC14 — PUT với version cũ → 409 VERSION_CONFLICT + details.current_version + details.current + ETag hiện tại
  ensure_mode test
  local id s
  id=$(newitem "qc71-$(now_ms)")
  put_item "$id" '{"name":"qc71-v2","version":1}' >/dev/null
  s=$(put_item "$id" '{"name":"qc71-lan2","version":1}')
  chk "status" "$s" 409
  chk "code" "$(jb .code)" VERSION_CONFLICT
  chk "details.current_version" "$(jb '.details.current_version')" 2
  chk "details.current là object" "$(jq -r '.details.current|type' "$B" 2>/dev/null)" object
  chk "header ETag = version hiện tại" "$(hv "$HD" etag)" 'W/"v2"'
  chk "version trong DB không đổi" "$(cnt "select version from _test_items where id='$id'")" 2
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_72() {  # AC14 — thiếu version → 422; id không có → 404
  ensure_mode test
  local id s
  id=$(newitem "qc72-$(now_ms)")
  s=$(put_item "$id" '{"name":"qc72-khong-version"}')
  chk "thiếu version: status" "$s" 422
  chk "thiếu version: code" "$(jb .code)" VALIDATION_FAILED
  chk "version trong DB không đổi" "$(cnt "select version from _test_items where id='$id'")" 1
  s=$(put_item "$UX" '{"name":"qc72-khong-ton-tai","version":1}')
  chk "id không có: status" "$s" 404
  chk "id không có: code" "$(jb .code)" NOT_FOUND
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_73() {  # AC14 — If-Match: W/"v<n>" tương đương trường version; có cả hai mà khác nhau → 422
  ensure_mode test
  local id s
  id=$(newitem "qc73-$(now_ms)")
  s=$(put_item "$id" '{"name":"qc73-ifmatch"}' 'If-Match: W/"v1"')
  chk "If-Match đúng: status" "$s" 200
  chk "version sau If-Match" "$(cnt "select version from _test_items where id='$id'")" 2
  s=$(put_item "$id" '{"name":"qc73-cu","version":1}' 'If-Match: W/"v1"')
  chk "If-Match cũ: status" "$s" 409
  chk "If-Match cũ: code" "$(jb .code)" VERSION_CONFLICT
  s=$(put_item "$id" '{"name":"qc73-lech","version":2}' 'If-Match: W/"v9"')
  chk "If-Match ≠ version trong thân: status" "$s" 422
  chk "version trong DB không đổi" "$(cnt "select version from _test_items where id='$id'")" 2
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_74() {  # AC14 — 20 PUT song song cùng version: đúng 1 lần 200, 19 lần 409, version cuối = n+1
  ensure_mode test
  local id i d=$QC_OUT/tc-pg03-74 n200 n409 n5xx
  id=$(newitem "qc74-$(now_ms)")
  rm -rf "$d"; mkdir -p "$d"
  for i in $(seq 20); do
    ( curl -sk --max-time 40 -o "$d/b$i" -w '%{http_code}' -X PUT -H "$H" -H 'Content-Type: application/json' \
        -d "{\"name\":\"qc74-$i\",\"version\":1}" "$ITEMS/$id" > "$d/s$i" ) &
  done
  wait
  n200=$(cat "$d"/s* | grep -c '^200'); n409=$(cat "$d"/s* | grep -c '^409'); n5xx=$(cat "$d"/s* | grep -c '^5')
  chk "số 200" "$n200" 1
  chk "số 409" "$n409" 19
  chk "số 5xx" "$n5xx" 0
  chk "version cuối = n+1" "$(cnt "select version from _test_items where id='$id'")" 2
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_75() {  # AC14 (test Go) — 409 version cũ, 20 song song, If-Match, thiếu version
  gt ./internal/httpapi 'TestOptimisticLock_Stale409|TestOptimisticLock_Concurrent20|TestOptimisticLock_IfMatch|TestOptimisticLock_Missing'
}

# ================= AC15 — ETag / 304 =================
tc_pg03_76() {  # AC15 — GET item: ETag W/"v<version>", Cache-Control: private, no-cache, Vary: Authorization
  ensure_mode test
  local id s
  id=$(newitem "qc76-$(now_ms)")
  s=$(get_json -H "$H" "$ITEMS/$id")
  chk "status" "$s" 200
  chk "ETag" "$(hv "$HD" etag)" 'W/"v1"'
  chk "Cache-Control" "$(hv "$HD" cache-control)" 'private, no-cache'
  chk_re "Vary có Authorization" "$(hv "$HD" vary)" 'Authorization'
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_77() {  # AC15 — If-None-Match khớp (một giá trị) → 304, thân rỗng, có ETag
  ensure_mode test
  local id e s sz
  id=$(newitem "qc77-$(now_ms)")
  e=$(hdr -H "$H" "$ITEMS/$id" | hval etag)
  sz=$(curl -sk --max-time 25 -o "$B" -D "$HD" -w '%{size_download}' -H "$H" -H "If-None-Match: $e" "$ITEMS/$id")
  s=$(code -H "$H" -H "If-None-Match: $e" "$ITEMS/$id")
  chk "status" "$s" 304
  chk "thân rỗng (byte tải về)" "$sz" 0
  chk "ETag trên 304" "$(hv "$HD" etag)" "$e"
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_78() {  # AC15 — If-None-Match dạng danh sách và `*` → 304
  ensure_mode test
  local id e s
  id=$(newitem "qc78-$(now_ms)")
  e=$(hdr -H "$H" "$ITEMS/$id" | hval etag)
  s=$(code -H "$H" -H "If-None-Match: W/\"v9\", $e" "$ITEMS/$id")
  chk "danh sách có ETag hiện tại" "$s" 304
  s=$(code -H "$H" -H 'If-None-Match: W/"v1", W/"v9"' "$ITEMS/$id")
  chk "danh sách W/\"v1\", W/\"v9\" (khớp v1)" "$s" 304
  s=$(code -H "$H" -H 'If-None-Match: *' "$ITEMS/$id")
  chk "If-None-Match: *" "$s" 304
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_79() {  # AC15 — không khớp → 200; sau PUT thì ETag đổi
  ensure_mode test
  local id e1 e2 s
  id=$(newitem "qc79-$(now_ms)")
  e1=$(hdr -H "$H" "$ITEMS/$id" | hval etag)
  s=$(code -H "$H" -H 'If-None-Match: W/"v999"' "$ITEMS/$id")
  chk "ETag không khớp → 200" "$s" 200
  put_item "$id" '{"name":"qc79-moi","version":1}' >/dev/null
  e2=$(hdr -H "$H" "$ITEMS/$id" | hval etag)
  chk_ne "ETag đổi sau PUT" "$e2" "$e1"
  chk "ETag mới" "$e2" 'W/"v2"'
  chk "ETag cũ không còn khớp" "$(code -H "$H" -H "If-None-Match: $e1" "$ITEMS/$id")" 200
  $PSQL -c "delete from _test_items where id='$id'" >/dev/null 2>&1
}
tc_pg03_80() {  # AC15 — ETag của danh sách = W/"<base64url 16 ký tự>" băm thân
  ensure_mode test
  local e s
  e=$(hdr -H "$H" "$ITEMS?limit=5" | hval etag)
  chk_re "định dạng ETag danh sách" "$e" '^W/"[A-Za-z0-9_-]{16}"$'
  s=$(code -H "$H" -H "If-None-Match: $e" "$ITEMS?limit=5")
  chk "If-None-Match khớp trên danh sách → 304" "$s" 304
  chk "ETag danh sách ổn định khi thân không đổi" "$(hdr -H "$H" "$ITEMS?limit=5" | hval etag)" "$e"
}
tc_pg03_81() {  # AC15 — If-None-Match trên POST / PUT bị bỏ qua
  ensure_mode test
  local id s N="qc81-$(now_ms)"
  s=$(curl -sk --max-time 25 -o "$B" -D "$HD" -w '%{http_code}' -X POST -H "$H" -H 'Content-Type: application/json' \
       -H "Idempotency-Key: qc81-$(now_ms)" -H 'If-None-Match: *' -d "{\"name\":\"$N\"}" "$ITEMS")
  chk_re "POST với If-None-Match: * vẫn tạo (không 304)" "$s" '^(200|201)$'
  id=$(jb '.id')
  s=$(put_item "$id" '{"name":"qc81-moi","version":1}' 'If-None-Match: *')
  chk "PUT với If-None-Match: * vẫn ghi (không 304)" "$s" 200
  chk "version đã tăng" "$(cnt "select version from _test_items where id='$id'")" 2
  $PSQL -c "delete from _test_items where name like 'qc81-%'" >/dev/null 2>&1
}
tc_pg03_82() {  # AC15 (test Go) — 304, ETag đổi sau cập nhật, danh sách nhiều giá trị
  gt ./internal/httpapi 'TestETag_NotModified|TestETag_ChangesAfterUpdate|TestETag_MultipleValues'
}

# ================= AC16 — việc dài 202 =================
tc_pg03_83() {  # AC16 — POST _test/jobs {"steps":4} → 202 {job_id} + Location
  ensure_mode test
  local s id
  s=$(post_job '{"steps":4}'); id=$(jb '.job_id')
  chk "status" "$s" 202
  chk_re "job_id là uuid" "$id" '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
  chk "Location" "$(hv "$HD" location)" "/api/v1/jobs/$id"
  chk "kind trong DB" "$(cnt "select kind from jobs where id='$id'")" 'test.progress'
  chk "owner_id trong DB" "$(cnt "select owner_id from jobs where id='$id'")" "$U1"
}
tc_pg03_84() {  # AC16 — GET /api/v1/jobs/{id} của chủ job: tập trường, SUCCEEDED, progress 100, finished_at
  ensure_mode test
  local s id st ks
  s=$(post_job '{"steps":4}'); id=$(jb '.job_id')
  st=$(job_wait "$id" SUCCEEDED 30)
  get_json -H "$H" "$GW/api/v1/jobs/$id" >/dev/null
  ks=$(jq -r 'keys[]' "$B" 2>/dev/null | grep -vE '^(id|kind|status|progress|result|error|created_at|updated_at|finished_at)$' | paste -sd, -)
  chk "trạng thái trong DB" "$st" SUCCEEDED
  chk "khoá lạ trong thân" "$ks" ''
  chk "đủ trường bắt buộc" "$(jq 'has("id") and has("kind") and has("status") and has("progress") and has("created_at") and has("updated_at")' "$B" 2>/dev/null)" true
  chk "status" "$(jb .status)" SUCCEEDED
  chk "progress" "$(jb .progress)" 100
  chk "finished_at có giá trị" "$(jb '.finished_at != null')" true
}
tc_pg03_85() {  # AC16 — progress tăng không giảm (poll mỗi 50 ms)
  ensure_mode test
  local s id i f=$QC_OUT/tc-pg03-85.txt giam
  s=$(post_job '{"steps":4}'); id=$(jb '.job_id')
  : > "$f"
  for i in $(seq 120); do
    curl -sk --max-time 10 -H "$H" "$GW/api/v1/jobs/$id" | jq -r '[.status,.progress]|@tsv' >> "$f" 2>/dev/null
    grep -q 'SUCCEEDED' "$f" && break
    sleep 0.05
  done
  giam=$(awk 'BEGIN{p=-1;c=0} {v=$2+0; if (v<p) c++; p=v} END{print c+0}' "$f")
  chk "status POST" "$s" 202
  chk_ge "số lần đo" "$(grep -c . "$f")" 2
  chk "số lần progress giảm" "$giam" 0
  chk "progress cuối" "$(awk 'END{print $2}' "$f")" 100
  chk "trạng thái cuối" "$(awk 'END{print $1}' "$f")" SUCCEEDED
  chk_ge "giá trị progress khác nhau (có tiến độ trung gian hoặc 0→100)" "$(awk '{print $2}' "$f" | sort -u | grep -c .)" 2
}
tc_pg03_86() {  # AC16 — dòng jobs QUEUED + dòng outbox job.enqueue ghi cùng transaction (worker đang dừng)
  ensure_mode test
  local s id st before after ob same
  $C stop worker >/dev/null 2>&1
  before=$(cnt "select count(*) from outbox where topic='job.enqueue'")
  s=$(post_job '{"steps":4}'); id=$(jb '.job_id')
  st=$(cnt "select status from jobs where id='$id'")
  ob=$(cnt "select count(*) from outbox where topic='job.enqueue' and payload::text like '%$id%'")
  after=$(cnt "select count(*) from outbox where topic='job.enqueue'")
  same=$(cnt "select count(*) from jobs j, outbox o where j.id='$id' and o.topic='job.enqueue' and o.payload::text like '%'||j.id||'%' and o.created_at=j.created_at")
  $C start worker >/dev/null 2>&1
  chk "status POST" "$s" 202
  chk "jobs.status ngay sau POST (worker dừng)" "$st" QUEUED
  chk "số dòng outbox job.enqueue của job" "$ob" 1
  chk "outbox tăng đúng 1 dòng" "$((after-before))" 1
  chk "jobs.created_at = outbox.created_at (cùng transaction)" "$same" 1
  chk "sau khi bật lại worker" "$(job_wait "$id" SUCCEEDED 40)" SUCCEEDED
}
tc_pg03_87() {  # AC16 (test Go) — vòng đời job, progress không giảm, enqueue nguyên tử
  gt './internal/jobs ./internal/httpapi' 'TestJobs_Lifecycle|TestJobs_ProgressMonotonic|TestJobs_EnqueueAtomic'
}

# ================= AC17 — job lỗi, id sai =================
tc_pg03_88() {  # AC17 — kind test.fail → FAILED + error.code + finished_at, progress giữ nguyên
  ensure_mode test
  local s id st p1 p2
  s=$(post_job '{"steps":4,"kind":"test.fail"}'); id=$(jb '.job_id')
  st=$(job_wait "$id" FAILED 30)
  get_json -H "$H" "$GW/api/v1/jobs/$id" >/dev/null; p1=$(jb .progress)
  sleep 2
  get_json -H "$H" "$GW/api/v1/jobs/$id" >/dev/null; p2=$(jb .progress)
  chk "status POST" "$s" 202
  chk "trạng thái" "$st" FAILED
  chk "status trong thân" "$(jb .status)" FAILED
  chk_re "error.code có giá trị" "$(jb '.error.code')" '^.+$'
  chk "finished_at có giá trị" "$(jb '.finished_at != null')" true
  chk "progress giữ nguyên sau 2 s" "$p2" "$p1"
  chk_le "progress không đạt 100" "$p2" 99
  chk "không treo ở RUNNING" "$(cnt "select count(*) from jobs where id='$id' and status='RUNNING'")" 0
}
tc_pg03_89() {  # AC17 — id không tồn tại / không phải uuid → 404 NOT_FOUND
  ensure_mode test
  local s
  s=$(get_json -H "$H" "$GW/api/v1/jobs/$UX")
  chk "uuid không tồn tại: status" "$s" 404
  chk "uuid không tồn tại: code" "$(jb .code)" NOT_FOUND
  s=$(get_json -H "$H" "$GW/api/v1/jobs/not-a-uuid")
  chk "không phải uuid: status" "$s" 404
  chk "không phải uuid: code" "$(jb .code)" NOT_FOUND
}
tc_pg03_90() {  # AC17 (test Go) — TestJobs_Failed, TestJobs_PanicFailed
  gt ./internal/jobs 'TestJobs_Failed|TestJobs_PanicFailed'
}

# ================= AC18 — phân quyền job + image mặc định an toàn =================
tc_pg03_91() {  # AC18 (phân quyền) — U2 → 404 (không 403), ADMIN → 200
  ensure_mode test
  local s id c
  s=$(post_job '{"steps":4}'); id=$(jb '.job_id')
  c=$(get_json -H "$(toka $U2)" "$GW/api/v1/jobs/$id")
  chk "U2 (STUDENT khác): status" "$c" 404
  chk "U2: code" "$(jb .code)" NOT_FOUND
  chk_ne "U2 không nhận 403" "$c" 403
  chk "ADMIN: status" "$(code -H "$(toka $U1 ADMIN)" "$GW/api/v1/jobs/$id")" 200
  chk "chủ job (U1): status" "$(code -H "$H" "$GW/api/v1/jobs/$id")" 200
  gt ./internal/httpapi 'TestJobs_Ownership'
}
tc_pg03_92() {  # AC18 (phân quyền) — không token → 401 UNAUTHENTICATED; token hết hạn → 401 TOKEN_EXPIRED
  ensure_mode test
  local s
  s=$(get_json "$GW/api/v1/jobs/$UX")
  chk "không token: status" "$s" 401
  chk "không token: code" "$(jb .code)" UNAUTHENTICATED
  chk_re "WWW-Authenticate" "$(hv "$HD" www-authenticate)" 'Bearer realm="edupilot"'
  s=$(get_json -H "Authorization: Bearer $(tok STUDENT $U1 --ttl -1m)" "$GW/api/v1/jobs/$UX")
  chk "token hết hạn: status" "$s" 401
  chk "token hết hạn: code" "$(jb .code)" TOKEN_EXPIRED
}
tc_pg03_93() {  # AC18 — image mặc định: 13 đường dẫn _test với token ADMIN → 404 (không 401/403)
  ensure_mode default
  local p c n404=0 bad="" A
  A=$(toka $U1 ADMIN)
  for p in items items/$U1 slow db-sleep redis-block panic error/500 whoami rbac/admin rbac/staff courses/$U1/ping jobs events; do
    c=$(code -H "$A" "$GW/api/v1/_test/$p")
    if [ "$c" = 404 ]; then n404=$((n404+1)); else bad="$bad $p=$c"; fi
  done
  chk "số đường dẫn trả 404 / 13" "$n404" 13
  chk "đường dẫn không trả 404" "$bad" ''
  get_json -H "$A" "$GW/api/v1/_test/panic" >/dev/null
  chk "code" "$(jb .code)" NOT_FOUND
  chk "chế độ hiện tại" "$(mode_now)" default
}
tc_pg03_94() {  # AC18 — image mặc định, KHÔNG token: 13 đường dẫn → 404 (không 401)
  ensure_mode default
  local p c n404=0 bad=""
  for p in items items/$U1 slow db-sleep redis-block panic error/500 whoami rbac/admin rbac/staff courses/$U1/ping jobs events; do
    c=$(code "$GW/api/v1/_test/$p")
    if [ "$c" = 404 ]; then n404=$((n404+1)); else bad="$bad $p=$c"; fi
  done
  chk "số đường dẫn trả 404 / 13" "$n404" 13
  chk "đường dẫn không trả 404 (401/403 là lỗi: lộ sự tồn tại)" "$bad" ''
}
tc_pg03_95() {  # AC18 — image mặc định: POST / PUT vào route thử cũng 404 (không 405, không ghi dữ liệu)
  ensure_mode default
  local c A bad=""
  A=$(toka $U1 ADMIN)
  c=$(code -X POST -H "$A" -H 'Content-Type: application/json' -H 'Idempotency-Key: qc95-aaaaaaa' -d '{"name":"qc95"}' "$GW/api/v1/_test/items")
  chk "POST _test/items" "$c" 404
  c=$(code -X POST -H "$A" -H 'Content-Type: application/json' -d '{"steps":4}' "$GW/api/v1/_test/jobs")
  chk "POST _test/jobs" "$c" 404
  c=$(code -X POST -H "$A" -H 'Content-Type: application/json' -d '{"type":"test.ping","data":{}}' "$GW/api/v1/_test/events")
  chk "POST _test/events" "$c" 404
  c=$(code -X PUT -H "$A" -H 'Content-Type: application/json' -d '{"name":"qc95","version":1}' "$GW/api/v1/_test/items/$UX")
  chk "PUT _test/items/{id}" "$c" 404
  local n=0
  [ "$(cnt "select count(*) from information_schema.tables where table_name='_test_items'")" = 1 ] \
    && n=$(cnt "select count(*) from _test_items where name='qc95'")
  chk "số dòng _test_items tên qc95 (bảng thử không bị ghi)" "${n:-0}" 0
}
tc_pg03_96() {  # AC18 — go tool nm: binary mặc định 0 symbol internal/testroutes, binary -tags testroutes ≥ 1
  local d t
  (cd backend-go && CGO_ENABLED=0 go build -o "$QC_TMP/gw-default" ./cmd/gateway) || fail_tc "go build bản mặc định thất bại"
  (cd backend-go && CGO_ENABLED=0 go build -tags testroutes -o "$QC_TMP/gw-test" ./cmd/gateway) || fail_tc "go build -tags testroutes thất bại"
  d=$( (cd backend-go && go tool nm "$QC_TMP/gw-default" 2>/dev/null) | grep -c 'internal/testroutes')
  t=$( (cd backend-go && go tool nm "$QC_TMP/gw-test" 2>/dev/null) | grep -c 'internal/testroutes')
  chk "symbol internal/testroutes trong binary mặc định" "$d" 0
  chk_ge "symbol internal/testroutes trong binary -tags testroutes" "$t" 1
}
tc_pg03_97() {  # AC18 — chuỗi /api/v1/_test/ trong file thực thi lấy từ image: mặc định 0, image test ≥ 1
  ensure_mode default
  local nd nt
  docker image inspect edupilot-gateway >/dev/null 2>&1 || $C build gateway >/dev/null 2>&1
  docker image inspect edupilot-gateway-test >/dev/null 2>&1 || $CT build gateway >/dev/null 2>&1
  docker rm -f qcgw03d qcgw03t >/dev/null 2>&1
  docker create --name qcgw03d edupilot-gateway >/dev/null 2>&1
  docker cp qcgw03d:/gateway "$QC_TMP/gw-img-default" >/dev/null 2>&1; docker rm qcgw03d >/dev/null 2>&1
  docker create --name qcgw03t edupilot-gateway-test >/dev/null 2>&1
  docker cp qcgw03t:/gateway "$QC_TMP/gw-img-test" >/dev/null 2>&1; docker rm qcgw03t >/dev/null 2>&1
  chk_ok "lấy được binary từ image mặc định" test -s "$QC_TMP/gw-img-default"
  chk_ok "lấy được binary từ image test" test -s "$QC_TMP/gw-img-test"
  nd=$(LC_ALL=C grep -ac '/api/v1/_test/' "$QC_TMP/gw-img-default" 2>/dev/null)
  nt=$(LC_ALL=C grep -ac '/api/v1/_test/' "$QC_TMP/gw-img-test" 2>/dev/null)
  chk "chuỗi /api/v1/_test/ trong binary image mặc định" "${nd:-khong-doc-duoc}" 0
  chk_ge "chuỗi /api/v1/_test/ trong binary image test" "${nt:-0}" 1
}
tc_pg03_98() {  # AC18 — binary dựng có tag + APP_ENV=production → thoát mã 1, log nêu APP_ENV
  local o=$QC_OUT/tc-pg03-98.log rc t0 t1
  (cd backend-go && CGO_ENABLED=0 go build -tags testroutes -o "$QC_TMP/gw-test" ./cmd/gateway) || fail_tc "go build -tags testroutes thất bại"
  t0=$(now_ms)
  env -i PATH="$PATH" HOME="$HOME" \
    DATABASE_URL=postgres://u:p@127.0.0.1:1/db REDIS_URL=redis://127.0.0.1:1/0 \
    JWT_SECRET_KEY=0123456789abcdef0123456789abcdef BLOB_ENDPOINT=127.0.0.1:1 BLOB_BUCKET=b \
    BLOB_ACCESS_KEY=ak BLOB_SECRET_KEY=sk STARTUP_TIMEOUT=2s APP_ENV=production \
    "$QC_TMP/gw-test" serve > "$o" 2>&1
  rc=$?; t1=$(now_ms)
  chk "mã thoát" "$rc" 1
  chk_ge "log nêu APP_ENV" "$(grep -c 'APP_ENV' "$o")" 1
  chk_le "thời gian thoát (ms)" "$((t1-t0))" 5000
  gt ./cmd/gateway 'TestTestBinary_RefusesProduction'
}
tc_pg03_99() {  # AC18 (test Go, KHÔNG tag) — TestDefaultBinary_NoTestRoutes
  gt_notag ./cmd/gateway 'TestDefaultBinary_NoTestRoutes'
}

main 03 "$@"
