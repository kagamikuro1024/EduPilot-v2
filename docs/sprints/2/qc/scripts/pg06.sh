#!/usr/bin/env bash
# QC US-PG-06 — hợp đồng API: openapi.yaml / openapi.test.yaml + contract test hai chiều
# (nguồn US.md v1.2 US-PG-06 AC1–AC8 + dòng "Làm rõ (QC questions #Q-QC-06-1…5)"; SRS 6.1, 6.2, 6.3, 6.5, 9.2, 9.4). Hộp đen: chỉ grep artefact hợp đồng (gồm internal/contract/exempt.go, xem AC5) + gọi stack thật.
# Chạy: bash docs/sprints/2/qc/scripts/pg06.sh [MM …|--list]
source "$(dirname "$0")/lib.sh"

API=backend-go/api/openapi.yaml
APIT=backend-go/api/openapi.test.yaml
CPKG=./internal/contract
# 8 test của gói contract (tên lấy nguyên văn từ AC1, AC3…AC7; TestStatusParity theo US.md v1.2 AC4 — QC questions #Q-QC-06-5)
C8='TestSpec_LoadsAndValidates|TestRouteSpecParity|TestStatusParity|TestSpec_ErrorResponsesDeclared|TestSpec_EveryDocumentedStatusExercised|TestValidator_RejectsBadResponses|TestSpec_SecurityDeclared|TestContract_UnauthenticatedMatchesSpec'
# 5 đường dẫn sản xuất (AC2 / SRS 6.2), đã sắp xếp
PROD5='/api/v1/events /api/v1/healthz /api/v1/jobs/{id} /api/v1/readyz /healthz'
# 13 đường dẫn thử (SRS 6.3), chuẩn hoá {x} -> {}, đã sắp xếp
TEST13='/api/v1/_test/courses/{}/ping /api/v1/_test/db-sleep /api/v1/_test/error/{} /api/v1/_test/events /api/v1/_test/items /api/v1/_test/items/{} /api/v1/_test/jobs /api/v1/_test/panic /api/v1/_test/rbac/admin /api/v1/_test/rbac/staff /api/v1/_test/redis-block /api/v1/_test/slow /api/v1/_test/whoami'

# ---------- tiện ích đọc YAML bằng awk thuần (bash 3.2, không yq, không python) ----------
need() {  # need <tệp…> — có đủ tệp hợp đồng mới kiểm tiếp
  local f; for f in "$@"; do [ -f "$f" ] || { fail_tc "không có tệp hợp đồng $f"; return 1; }; done; return 0; }
paths_of() {  # in các đường dẫn khai ở cột 2 (`^  /…:`)
  awk '{t=$0; sub(/[ \t]+$/,"",t)} t ~ /^  \// && t ~ /:$/ {sub(/:$/,"",t); sub(/^  /,"",t); print t}' "$1"; }
norm() { sed 's/{[^}]*}/{}/g'; }
flat() { LC_ALL=C sort | tr '\n' ' ' | sed 's/ $//'; }
block_of() {  # block_of <tệp> <đường dẫn> — in khối YAML của một đường dẫn
  awk -v p="$2" '{t=$0; sub(/[ \t]+$/,"",t)
    if (t == "  " p ":") { inb=1; print; next }
    if (inb && t ~ /^  [^ ]/) inb=0
    if (inb) print }' "$1"; }
methods_of() {  # methods_of <tệp> <đường dẫn> — in method của một đường dẫn
  block_of "$1" "$2" | awk '{t=$0; sub(/[ \t]+$/,"",t)
    if (t ~ /^    [a-z]+:$/) { sub(/^    /,"",t); sub(/:$/,"",t)
      if (t=="get"||t=="post"||t=="put"||t=="patch"||t=="delete"||t=="head"||t=="options") print t } }'; }
comp_resp() {  # in khối components.responses (khoá `  responses:` ở cột 2)
  awk '{t=$0; sub(/[ \t]+$/,"",t)
    if (t == "  responses:") { inb=1; next }
    if (inb && t ~ /^  [^ ]/) inb=0
    if (inb) print }' "$1"; }
drop_path() {  # drop_path <tệp> <đường dẫn> — in tệp đã xoá khối đường dẫn đó
  awk -v p="$2" '{t=$0; sub(/[ \t]+$/,"",t)
    if (t == "  " p ":") { sk=1; next }
    if (sk) { if ($0 ~ /^  [^ ]/ || $0 ~ /^[^ ]/) sk=0; else next }
    print }' "$1"; }
drop_status() {  # drop_status <tệp> <đường dẫn> <status> — xoá khoá status (và các dòng con) trong khối đường dẫn
  awk -v p="$2" -v s="$3" 'BEGIN{q=sprintf("%c",39)}
    {t=$0; sub(/[ \t]+$/,"",t)
    if (t == "  " p ":") { inb=1; print; next }
    if (inb && t ~ /^  [^ ]/) inb=0
    if (sk) { if (t == "") next
      match($0, /^[ ]*/); if (RLENGTH > ind) next; sk=0 }
    if (inb) { u=t; gsub(q,"",u); gsub(/"/,"",u)
      if (u ~ /^[ ]+[0-9][0-9][0-9]:$/) { v=u; gsub(/[ :]/,"",v)
        if (v == s) { match($0, /^[ ]*/); ind=RLENGTH; sk=1; next } } }
    print }' "$1"; }
add_path() {  # add_path <tệp> — in tệp có thêm khối GET tối thiểu /api/v1/khong-co ngay sau `paths:`
  awk '{print
    if (!done && $0 ~ /^paths:[ \t]*$/) { done=1
      print "  /api/v1/khong-co:"
      print "    get:"
      print "      operationId: qcKhongCoTC23"
      print "      summary: QC - duong dan thua de kiem parity"
      print "      responses:"
      print "        \"200\":"
      print "          description: ok" } }' "$1"; }
parity_run() {  # parity_run <spec tuyệt đối> <tag|notag> <tệp log> [cờ -run …] → in rc
  local spec=$1 m=$2 log=$3; shift 3
  if [ "$m" = tag ]; then
    (cd backend-go && OPENAPI_PATH="$spec" go test -race -count=1 -timeout 20m -tags testroutes -v "$@" ./internal/contract) > "$log" 2>&1
  else
    (cd backend-go && OPENAPI_PATH="$spec" go test -count=1 -timeout 20m -v "$@" ./internal/contract) > "$log" 2>&1
  fi
  echo $?; }
jqv() { printf '%s' "$1" | jq -r "$2" 2>/dev/null; }

# ---------- AC1 — spec hợp lệ ----------
tc_pg06_01() {  # AC1 — redocly lint hai tệp, rc=0
  need $API $APIT || return
  local v out rc
  v=$(pnpm exec redocly --version 2>&1 | tr -d '\r' | tail -1)
  if [ -z "$v" ] || printf '%s' "$v" | grep -qiE 'not found|no such|ERR_PNPM|command'; then
    fail_tc "KHÔNG KIỂM ĐƯỢC: không có 'pnpm exec redocly' ([$v]) — QC không cài thêm gói"; return; fi
  echo "    info redocly version = [$v]"
  out=$(pnpm exec redocly lint $API $APIT 2>&1); rc=$?
  printf '%s\n' "$out" > "$QC_OUT/tc-pg06-01.log"
  chk "redocly lint rc" "$rc" 0
  chk_nre "đầu ra redocly không có lỗi" "$out" '[1-9][0-9]* error'
}
tc_pg06_02() {  # AC1 — kin-openapi nạp + Validate cả hai tệp
  gt $CPKG 'TestSpec_LoadsAndValidates'
}
tc_pg06_03() {  # AC1 (Q5) — phiên bản openapi: thuộc {3.1.0, 3.0.3} và hai tệp bằng nhau
  need $API $APIT || return
  local v1 v2
  v1=$(grep -m1 '^openapi:' $API  | awk '{print $2}' | tr -d "\"' \r")
  v2=$(grep -m1 '^openapi:' $APIT | awk '{print $2}' | tr -d "\"' \r")
  printf 'openapi.yaml=%s\nopenapi.test.yaml=%s\n' "$v1" "$v2" > "$QC_OUT/tc-pg06-03.txt"
  chk_re "openapi: của openapi.yaml"      "$v1" '^(3\.1\.0|3\.0\.3)$'
  chk_re "openapi: của openapi.test.yaml" "$v2" '^(3\.1\.0|3\.0\.3)$'
  chk "hai tệp cùng phiên bản" "$v1" "$v2"
}

# ---------- AC2 — đường dẫn và operationId ----------
tc_pg06_04() {  # AC2 — openapi.yaml đúng 5 đường dẫn, đúng tên
  need $API || return
  chk "grep -cE '^  /' api/openapi.yaml" "$(grep -cE '^  /' $API)" 5
  chk "danh sách đường dẫn" "$(paths_of $API | flat)" "$PROD5"
}
tc_pg06_05() {  # AC2 — không rò route thử; giữ /healthz của scaffold
  need $API || return
  chk "grep -c '_test' api/openapi.yaml" "$(grep -c '_test' $API)" 0
  chk "khai /healthz (giữ từ FEAT-scaffold)" "$(grep -cE '^  /healthz:' $API)" 1
}
tc_pg06_06() {  # AC2 / SRS 6.3 — openapi.test.yaml có đúng 13 đường dẫn thử (US.md v1.2, QC questions #Q-QC-06-1)
  need $APIT || return
  local got p
  chk "grep -cE '^  /api/v1/_test' api/openapi.test.yaml" "$(grep -cE '^  /api/v1/_test' $APIT)" 13
  got=$(paths_of $APIT | norm | flat)
  chk "13 đường dẫn thử (SRS 6.3, chuẩn hoá {x}→{})" "$got" "$TEST13"
  for p in $TEST13; do
    chk "có đúng một đường dẫn $p" "$(paths_of $APIT | norm | grep -cFx -- "$p")" 1
  done
}
tc_pg06_07() {  # AC2 — mọi thao tác có operationId; operationId duy nhất trên hai tệp
  need $API $APIT || return
  local m='^    (get|post|put|patch|delete|head|options):'
  chk "số thao tác openapi.yaml"       "$(grep -cE "$m" $API)"  5
  chk "số operationId openapi.yaml"    "$(grep -cE '^ +operationId:' $API)" 5
  chk "operationId = số thao tác (openapi.test.yaml)" \
      "$(grep -cE '^ +operationId:' $APIT)" "$(grep -cE "$m" $APIT)"
  chk "số thao tác openapi.test.yaml (SRS 6.3: đúng 15 route)" "$(grep -cE "$m" $APIT)" 15
  chk "operationId trùng nhau" \
      "$(grep -hE '^ +operationId:' $API $APIT | sed 's/.*operationId:[ ]*//' | tr -d "\"' \r" | LC_ALL=C sort | uniq -d | tr '\n' ' ' | sed 's/ $//')" ""
}

# ---------- AC3 — contract test hai chế độ ----------
tc_pg06_08() {  # AC3 — lệnh nguyên văn: gotest ./internal/contract/... rc=0
  local out rc
  out=$( (cd backend-go && go test -race -count=1 -timeout 20m -tags testroutes ./internal/contract/...) 2>&1 ); rc=$?
  printf '%s\n' "$out" > "$QC_OUT/tc-pg06-08.log"
  chk "gotest ./internal/contract/... rc" "$rc" 0
  chk_re "có gói contract chạy thật" "$out" 'ok[ \t]+.*internal/contract'
  chk_nre "không gói nào rỗng test" "$out" 'internal/contract.*no test files'
}
tc_pg06_09() {  # AC3 — 8 test của gói contract thật sự PASS (có tag)
  gt $CPKG "$C8"
}
tc_pg06_10() {  # AC3 — lệnh nguyên văn không tag: go test -count=1 ./internal/contract/... rc=0
  local out rc
  out=$( (cd backend-go && go test -count=1 -timeout 20m ./internal/contract/...) 2>&1 ); rc=$?
  printf '%s\n' "$out" > "$QC_OUT/tc-pg06-10.log"
  chk "go test (không tag) ./internal/contract/... rc" "$rc" 0
  chk_re "có gói contract chạy thật" "$out" 'ok[ \t]+.*internal/contract'
}
tc_pg06_11() {  # AC3 — 8 test PASS ở bản KHÔNG tag
  gt_notag $CPKG "$C8"
}
tc_pg06_12() {  # AC3 — chế độ default: mọi đường dẫn của openapi.test.yaml trả 404
  need $APIT || return
  ensure_mode default || { fail_tc "không về được chế độ default"; return; }
  local p m u c n=0 bad=0
  for p in $(paths_of $APIT); do
    for m in $(methods_of $APIT "$p"); do
      u=$(printf '%s' "$p" | sed -e 's#{status}#500#g' -e "s#{[^}]*}#$UX#g")
      c=$(code -X "$(printf '%s' "$m" | tr 'a-z' 'A-Z')" -H "$H" "$GW$u")
      n=$((n+1)); [ "$c" = 404 ] || { bad=$((bad+1)); echo "    LOG  $m $u → $c (mong đợi 404)"; }
    done
  done
  chk_ge "số (method, đường dẫn) thử đã gọi" "$n" 15
  chk "số phản hồi khác 404 ở image mặc định" "$bad" 0
}

# ---------- AC7 (hành vi thật, chế độ default) ----------
tc_pg06_13() {  # AC7 — không token: jobs/{id} và events → 401 + WWW-Authenticate
  ensure_mode default || { fail_tc "không về được chế độ default"; return; }
  local b h
  chk "GET /api/v1/jobs/{id} không token" "$(code $GW/api/v1/jobs/$UX)" 401
  b=$(curl -sk --max-time 20 $GW/api/v1/jobs/$UX)
  chk "code của thân lỗi" "$(jqv "$b" .code)" UNAUTHENTICATED
  h=$(hdr $GW/api/v1/jobs/$UX | hval WWW-Authenticate)
  chk_re "WWW-Authenticate" "$h" '^Bearer realm="edupilot"'
  chk "GET /api/v1/events không token" "$(code $GW/api/v1/events)" 401
  b=$(curl -sk --max-time 20 $GW/api/v1/events)
  chk "code của thân lỗi (events)" "$(jqv "$b" .code)" UNAUTHENTICATED
}
tc_pg06_14() {  # AC7 — ba endpoint sức khoẻ mở ẩn danh
  ensure_mode default || { fail_tc "không về được chế độ default"; return; }
  chk "GET /healthz ẩn danh"         "$(code $GW/healthz)"         200
  chk "GET /api/v1/healthz ẩn danh"  "$(code $GW/api/v1/healthz)"  200
  chk "GET /api/v1/readyz ẩn danh"   "$(code $GW/api/v1/readyz)"   200
}

# ---------- AC3 (hành vi thật, chế độ test) ----------
tc_pg06_15() {  # AC3 — 5 thao tác sản xuất khớp hình dạng spec
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local b
  chk "GET /healthz"        "$(curl -sk --max-time 20 $GW/healthz | jq -cS .)"        '{"status":"ok"}'
  chk "GET /api/v1/healthz" "$(curl -sk --max-time 20 $GW/api/v1/healthz | jq -r .status)" ok
  b=$(curl -sk --max-time 20 $GW/api/v1/readyz)
  chk "readyz status/db/redis" "$(jqv "$b" '.status+"/"+.db+"/"+.redis')" "ready/up/up"
  chk "GET /api/v1/jobs/<uuid lạ> (có token)" "$(code -H "$H" $GW/api/v1/jobs/$UX)" 404
  b=$(curl -sk --max-time 20 -H "$H" $GW/api/v1/jobs/$UX)
  chk "code" "$(jqv "$b" .code)" NOT_FOUND
  chk_re "trace_id 32 hex" "$(jqv "$b" .trace_id)" '^[0-9a-f]{32}$'
  chk "khoá lạ ngoài hợp đồng lỗi (SRS 6.1)" \
      "$(jqv "$b" '[keys[]|select(.!="code" and .!="message" and .!="trace_id" and .!="details" and .!="retry_after")]|join(",")')" ""
  chk "GET /api/v1/events mã" \
      "$(curl -sk -N --max-time 3 -o /dev/null -w '%{http_code}' -H "$H" $GW/api/v1/events 2>/dev/null)" 200
  chk_re "Content-Type của SSE" \
      "$(curl -sk -N --max-time 3 -D- -o /dev/null -H "$H" $GW/api/v1/events 2>/dev/null | tr -d '\r' | hval Content-Type)" '^text/event-stream'
}
tc_pg06_16() {  # AC3 — header bắt buộc (ETag, Location, Retry-After) khai trong spec và có thật
  need $API $APIT || return
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local hh et loc ra c
  chk_ge "spec khai ETag"        "$(cat $API $APIT | grep -c 'ETag')"        1
  chk_ge "spec khai Location"    "$(cat $API $APIT | grep -c 'Location')"    1
  chk_ge "spec khai Retry-After" "$(cat $API $APIT | grep -c 'Retry-After')" 1
  et=$(hdr -H "$H" $GW/api/v1/_test/items | hval ETag)
  chk_re "ETag thật của danh sách" "$et" '^W/"'
  hh=$(curl -sk --max-time 20 -D- -o /dev/null -X POST -H "$H" -H 'Content-Type: application/json' \
       -d '{"steps":2,"kind":"test.progress"}' $GW/api/v1/_test/jobs | tr -d '\r')
  chk "POST /api/v1/_test/jobs mã" "$(printf '%s\n' "$hh" | awk '/^HTTP/{c=$2} END{print c}')" 202
  loc=$(printf '%s\n' "$hh" | hval Location)
  chk_re "Location của việc dài" "$loc" '^/api/v1/jobs/[0-9a-f-]{36}$'
  ra=$(hdr $GW/api/v1/_test/error/429 | hval Retry-After)
  chk_re "Retry-After của 429" "$ra" '^[1-9][0-9]*$'
}
tc_pg06_17() {  # AC3 — hình dạng phân trang {items, next_cursor}
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local b
  newitem "qc-pg06-a-$(now_ms)" >/dev/null; newitem "qc-pg06-b-$(now_ms)" >/dev/null
  b=$(curl -sk --max-time 20 -H "$H" "$GW/api/v1/_test/items?limit=1")
  chk "kiểu của items"      "$(jqv "$b" '.items|type')"   array
  chk "số phần tử (limit=1)" "$(jqv "$b" '.items|length')" 1
  chk "có khoá next_cursor" "$(jqv "$b" 'has("next_cursor")')" true
  chk_ne "next_cursor khi còn trang sau" "$(jqv "$b" '.next_cursor')" null
  chk "khoá lạ ngoài {items,next_cursor}" \
      "$(jqv "$b" '[keys[]|select(.!="items" and .!="next_cursor")]|join(",")')" ""
}
tc_pg06_18() {  # AC5 (thật) — 8 status khai trong components.responses được sinh thật, đúng code SRS 6.1
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local pair s want b
  for pair in 401:UNAUTHENTICATED 403:FORBIDDEN 404:NOT_FOUND 409:CONFLICT 422:VALIDATION_FAILED \
              429:RATE_LIMITED 503:SERVICE_UNAVAILABLE 504:DEADLINE_EXCEEDED; do
    s=${pair%%:*}; want=${pair#*:}
    chk "GET /api/v1/_test/error/$s mã" "$(code $GW/api/v1/_test/error/$s)" "$s"
    b=$(curl -sk --max-time 20 $GW/api/v1/_test/error/$s)
    chk "code của $s" "$(jqv "$b" .code)" "$want"
    case $s in 429|503)
      chk "retry_after của $s" "$(jqv "$b" .retry_after)" 7
      chk_re "header Retry-After của $s" "$(hdr $GW/api/v1/_test/error/$s | hval Retry-After)" '^[0-9]+$';;
    esac
  done
}
tc_pg06_19() {  # AC7 (thật) — rbac/admin: 401 / 403 / 200 đúng như spec khai
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  local b ta
  chk "rbac/admin không token" "$(code $GW/api/v1/_test/rbac/admin)" 401
  chk_re "WWW-Authenticate" "$(hdr $GW/api/v1/_test/rbac/admin | hval WWW-Authenticate)" '^Bearer realm="edupilot"'
  chk "rbac/admin với STUDENT" "$(code -H "$H" $GW/api/v1/_test/rbac/admin)" 403
  b=$(curl -sk --max-time 20 -H "$H" $GW/api/v1/_test/rbac/admin)
  chk "code" "$(jqv "$b" .code)" FORBIDDEN
  chk "details.reason" "$(jqv "$b" '.details.reason')" role
  ta=$(tok ADMIN $U2)
  chk "rbac/admin với ADMIN" "$(code -H "Authorization: Bearer $ta" $GW/api/v1/_test/rbac/admin)" 200
}

# ---------- AC4 — lệch hai chiều thì đỏ (chạy trên bản spec sao chép, KHÔNG sửa repo) ----------
tc_pg06_20() {  # AC4 — đối chứng dương: spec nguyên vẹn → parity xanh
  need $API || return
  local log=$QC_OUT/tc-pg06-20.log rc
  cp $API "$QC_TMP/ok.yaml" || { fail_tc "không chép được spec"; return; }
  rc=$(parity_run "$QC_TMP/ok.yaml" tag "$log" -run TestRouteSpecParity)
  chk "rc (spec nguyên vẹn)" "$rc" 0
  chk_ge "có --- PASS: TestRouteSpecParity" "$(grep -c '^--- PASS: TestRouteSpecParity' "$log")" 1
}
tc_pg06_21() {  # AC4 — xoá /api/v1/readyz khỏi spec → đỏ, nêu GET /api/v1/readyz (có tag)
  need $API || return
  local log=$QC_OUT/tc-pg06-21.log rc
  drop_path $API '/api/v1/readyz' > "$QC_TMP/m1.yaml"
  chk "đường dẫn còn lại sau khi xoá" "$(paths_of "$QC_TMP/m1.yaml" | grep -c '^/api/v1/readyz$')" 0
  rc=$(parity_run "$QC_TMP/m1.yaml" tag "$log" -run TestRouteSpecParity)
  chk_ne "rc (spec thiếu readyz)" "$rc" 0
  chk_ge "có --- FAIL: TestRouteSpecParity" "$(grep -c '^--- FAIL: TestRouteSpecParity' "$log")" 1
  chk_ge "thông báo nêu GET /api/v1/readyz" "$(grep -c 'GET /api/v1/readyz' "$log")" 1
}
tc_pg06_22() {  # AC4 — cùng phép thử ở bản KHÔNG tag
  need $API || return
  local log=$QC_OUT/tc-pg06-22.log rc
  [ -f "$QC_TMP/m1.yaml" ] || drop_path $API '/api/v1/readyz' > "$QC_TMP/m1.yaml"
  rc=$(parity_run "$QC_TMP/m1.yaml" notag "$log" -run TestRouteSpecParity)
  chk_ne "rc (không tag, spec thiếu readyz)" "$rc" 0
  chk_ge "có --- FAIL: TestRouteSpecParity" "$(grep -c '^--- FAIL: TestRouteSpecParity' "$log")" 1
  chk_ge "thông báo nêu GET /api/v1/readyz" "$(grep -c 'GET /api/v1/readyz' "$log")" 1
}
tc_pg06_23() {  # AC4 — thêm /api/v1/khong-co vào spec → đỏ, nêu đường dẫn thừa
  need $API || return
  local log=$QC_OUT/tc-pg06-23.log rc
  add_path $API > "$QC_TMP/m2.yaml"
  chk "spec thử có đường dẫn thừa" "$(paths_of "$QC_TMP/m2.yaml" | grep -c '^/api/v1/khong-co$')" 1
  rc=$(parity_run "$QC_TMP/m2.yaml" tag "$log" -run TestRouteSpecParity)
  chk_ne "rc (spec thừa đường dẫn)" "$rc" 0
  chk_ge "có --- FAIL: TestRouteSpecParity" "$(grep -c '^--- FAIL: TestRouteSpecParity' "$log")" 1
  chk_ge "thông báo nêu /api/v1/khong-co" "$(grep -c '/api/v1/khong-co' "$log")" 1
}
tc_pg06_24() {  # AC4 — xoá status 404 của /api/v1/jobs/{id} → TestStatusParity đỏ (US.md v1.2, QC questions #Q-QC-06-5)
  need $API || return
  local log=$QC_OUT/tc-pg06-24.log rc before after
  before=$(block_of $API '/api/v1/jobs/{id}' | grep -cE "^ +['\"]?404['\"]?:")
  drop_status $API '/api/v1/jobs/{id}' 404 > "$QC_TMP/m3.yaml"
  after=$(block_of "$QC_TMP/m3.yaml" '/api/v1/jobs/{id}' | grep -cE "^ +['\"]?404['\"]?:")
  chk_ge "spec gốc khai 404 cho jobs/{id}" "$before" 1
  chk "bản sửa đã xoá 404" "$after" 0
  rc=$(parity_run "$QC_TMP/m3.yaml" tag "$log" -run TestStatusParity)
  chk_ne "rc (status thật không có trong spec)" "$rc" 0
  chk_ge "có --- FAIL: TestStatusParity" "$(grep -c '^--- FAIL: TestStatusParity' "$log")" 1
  chk_ge "thông báo nêu GET /api/v1/jobs/{id}" "$(grep -c 'GET /api/v1/jobs/{id}' "$log")" 1
  chk_ge "thông báo nêu 404" "$(grep -c '404' "$log")" 1
}

# ---------- AC5 — components.responses ----------
tc_pg06_25() {  # AC5 — đủ 8 response dùng chung
  need $API || return
  local n
  for n in Unauthorized Forbidden NotFound Conflict ValidationFailed RateLimited Unavailable DeadlineExceeded; do
    chk "components.responses.$n" "$(comp_resp $API | grep -cE "^    $n:")" 1
  done
}
tc_pg06_26() {  # AC5 — mỗi status dùng đúng response đã khai
  need $API $APIT || return
  local pair s n
  for pair in 401:Unauthorized 403:Forbidden 404:NotFound 409:Conflict 422:ValidationFailed \
              429:RateLimited 503:Unavailable 504:DeadlineExceeded; do
    s=${pair%%:*}; n=${pair#*:}
    chk_ge "status $s → responses/$n" \
      "$(grep -hA3 -E "^ +['\"]?$s['\"]?:" $API $APIT | grep -c "responses/$n")" 1
  done
}
tc_pg06_27() {  # AC5 — mọi response lỗi $ref schema Error
  need $API || return
  chk_ge "số \$ref schemas/Error trong components.responses" "$(comp_resp $API | grep -c 'schemas/Error')" 8
  chk "định nghĩa schema Error" "$(grep -cE '^    Error:' $API)" 1
}
tc_pg06_28() {  # AC5 — hai test khai báo lỗi / mọi status được gọi thật + mọi miễn trừ có reason (QC questions #Q-QC-06-4)
  gt $CPKG 'TestSpec_ErrorResponsesDeclared|TestSpec_EveryDocumentedStatusExercised'
  local f=backend-go/internal/contract/exempt.go nop nst nre bad
  [ -f "$f" ] || { fail_tc "không có artefact miễn trừ $f (AC5 đòi danh sách miễn trừ nằm ở đây)"; return; }
  # mỗi mục là {operation, status, reason} — đếm theo khoá trường, không phụ thuộc cách xuống dòng
  nop=$(grep -cE '[Oo]peration:' "$f"); nst=$(grep -cE '[Ss]tatus:' "$f"); nre=$(grep -cE '[Rr]eason:' "$f")
  bad=$(grep -cE '[Rr]eason:[[:space:]]*(""|``|,|$)' "$f")
  printf 'operation=%s status=%s reason=%s reason-rong=%s\n' "$nop" "$nst" "$nre" "$bad" > "$QC_OUT/tc-pg06-28.txt"
  grep -nE '[Oo]peration:|[Ss]tatus:|[Rr]eason:' "$f" >> "$QC_OUT/tc-pg06-28.txt" 2>/dev/null
  chk "số trường status = số trường operation" "$nst" "$nop"
  chk "số trường reason = số trường operation" "$nre" "$nop"
  chk "số reason rỗng" "$bad" 0
  echo "    info $nop mục miễn trừ — trích vào $QC_OUT/tc-pg06-28.txt (đối chiếu handoff: TC-PG06-38, tay)"
}

# ---------- AC6 — đối chứng âm của validator ----------
tc_pg06_29() {  # AC6 — validator bắt được response sai
  gt $CPKG 'TestValidator_RejectsBadResponses'
  local log; log=$(ls -t "$QC_OUT"/gt-*RejectsBadResponses*.log 2>/dev/null | sed -n 1p)
  [ -n "$log" ] && grep -E '^ +--- (PASS|FAIL): TestValidator_RejectsBadResponses/' "$log" | while read -r l; do echo "    info $l"; done
  true
}

# ---------- AC7 — khai báo security trong spec ----------
tc_pg06_30() {  # AC7 — jobs/{id} và events: bearerAuth + 401
  need $API || return
  local p
  for p in '/api/v1/jobs/{id}' '/api/v1/events'; do
    # spec v1.5 (#12): security HIỆU LỰC — thao tác tự khai bearerAuth, hoặc không khai `security` riêng và gốc tài liệu khai bearerAuth
    chk_ge "$p có hiệu lực bearerAuth (riêng hoặc kế thừa gốc)" "$(b=$(block_of $API "$p"); if printf '%s' "$b" | grep -q 'bearerAuth'; then echo 1; elif ! printf '%s' "$b" | grep -qE '^ +security:' && grep -qE '^security:' $API && awk '/^security:/{f=1;next} f&&/^[a-z]/{exit} f' $API | grep -q bearerAuth; then echo 1; else echo 0; fi)" 1
    chk_ge "$p khai 401" "$(block_of $API "$p" | grep -cE "(^ +['\"]?401['\"]?:|responses/Unauthorized)")" 1
  done
}
tc_pg06_31() {  # AC7 — ba endpoint sức khoẻ: security: []
  need $API || return
  local p
  for p in '/healthz' '/api/v1/healthz' '/api/v1/readyz'; do
    chk "$p khai security: []" "$(block_of $API "$p" | grep -cE '^ +security: \[\]')" 1
    chk "$p không khai bearerAuth" "$(block_of $API "$p" | grep -c 'bearerAuth')" 0
  done
}
tc_pg06_32() {  # AC7 — route chỉ vai trò khai thêm 403
  need $APIT || return
  local p
  for p in '/api/v1/_test/rbac/admin' '/api/v1/_test/rbac/staff'; do
    chk_ge "$p khai 403" "$(block_of $APIT "$p" | grep -cE "(^ +['\"]?403['\"]?:|responses/Forbidden)")" 1
    chk_ge "$p khai 401" "$(block_of $APIT "$p" | grep -cE "(^ +['\"]?401['\"]?:|responses/Unauthorized)")" 1
  done
}
tc_pg06_33() {  # AC7 — hai test security
  gt $CPKG 'TestSpec_SecurityDeclared|TestContract_UnauthenticatedMatchesSpec'
}

# ---------- AC8 (D52) — kin-openapi chỉ trong test ----------
tc_pg06_34() {  # AC8 — binary gateway/worker không phụ thuộc kin-openapi (hai cấu hình biên dịch)
  local d dt
  d=$( (cd backend-go && go list -deps ./cmd/gateway ./cmd/worker) 2>"$QC_OUT/tc-pg06-34.err" )
  dt=$( (cd backend-go && go list -tags testroutes -deps ./cmd/gateway ./cmd/worker) 2>>"$QC_OUT/tc-pg06-34.err" )
  chk_ge "số gói phụ thuộc (không tag)" "$(printf '%s\n' "$d" | grep -c .)" 50
  chk_ge "số gói phụ thuộc (-tags testroutes)" "$(printf '%s\n' "$dt" | grep -c .)" 50
  chk "kin-openapi trong deps (không tag)" "$(printf '%s\n' "$d" | grep -c kin-openapi)" 0
  chk "kin-openapi trong deps (-tags testroutes)" "$(printf '%s\n' "$dt" | grep -c kin-openapi)" 0
}
tc_pg06_35() {  # AC8 — kin-openapi có trong cây test của internal/contract
  chk_ge "kin-openapi trong go list -deps -test ./internal/contract" \
    "$( (cd backend-go && go list -deps -test ./internal/contract) 2>/dev/null | grep -c kin-openapi )" 1
}
tc_pg06_36() {  # AC8 — lint sạch hai cấu hình (SRS 9.4) + ca âm depguard của dev (QC questions #Q-QC-06-3)
  if ! command -v golangci-lint >/dev/null 2>&1; then
    fail_tc "KHÔNG KIỂM ĐƯỢC: không có golangci-lint trên PATH — QC không cài"; return; fi
  local out rc log=$QC_OUT/tc-pg06-36.log
  out=$( (cd backend-go && golangci-lint run) 2>&1 ); rc=$?
  printf '%s\n' "$out" > "$log"
  chk "golangci-lint run rc" "$rc" 0
  chk_nre "không vi phạm depguard" "$out" 'depguard'
  out=$( (cd backend-go && golangci-lint run --build-tags testroutes) 2>&1 ); rc=$?
  printf '%s\n' "$out" >> "$log"
  chk "golangci-lint run --build-tags testroutes rc" "$rc" 0
  chk_nre "không vi phạm depguard (có tag)" "$out" 'depguard'
  # ca âm do dev để sẵn: gói tạm internal/zz_depguard_probe/ phải bị lint chặn rồi bị xoá; lệnh thoát 0 khi chặn đúng
  out=$(make -C backend-go lint-depguard-negative 2>&1); rc=$?
  printf '%s\n' "$out" >> "$log"
  chk "make -C backend-go lint-depguard-negative rc" "$rc" 0
  chk_re "ca âm nêu depguard" "$out" 'depguard'
  chk_ok "gói tạm đã bị xoá (test ! -e backend-go/internal/zz_depguard_probe)" test ! -e backend-go/internal/zz_depguard_probe
  out=$(git status --porcelain backend-go 2>&1)
  printf 'git status --porcelain backend-go:\n%s\n' "$out" >> "$log"
  chk "git status --porcelain backend-go không còn vết gói tạm" "$(printf '%s\n' "$out" | grep -c 'zz_depguard_probe')" 0
}
tc_pg06_37() {  # AC8 — đối chứng trên binary đã dựng: go version -m không có getkin/kin-openapi
  local b cmd tag out
  for cmd in gateway worker; do
    for tag in notag testroutes; do
      b=$QC_TMP/pg06-$cmd-$tag
      if [ "$tag" = testroutes ]; then
        (cd backend-go && CGO_ENABLED=0 go build -tags testroutes -o "$b" ./cmd/$cmd) >/dev/null 2>&1
      else
        (cd backend-go && CGO_ENABLED=0 go build -o "$b" ./cmd/$cmd) >/dev/null 2>&1
      fi
      if [ ! -x "$b" ]; then fail_tc "không dựng được $cmd ($tag)"; continue; fi
      out=$(go version -m "$b" 2>/dev/null)
      chk_ge "số dòng dep của $cmd ($tag)" "$(printf '%s\n' "$out" | grep -c '	dep	')" 5
      chk "getkin/kin-openapi trong $cmd ($tag)" "$(printf '%s\n' "$out" | grep -c 'getkin/kin-openapi')" 0
      rm -f "$b"
    done
  done
}
tc_pg06_38() {  # AC5 (TAY) — đối chiếu danh sách miễn trừ exempt.go với bản chép trong handoff
  manual "mở $QC_OUT/tc-pg06-28.txt (trích từ backend-go/internal/contract/exempt.go) và bản chép danh sách miễn trừ trong handoff của dev:" \
         "cùng số mục, cùng từng cặp (operation, status), mỗi lý do nêu được vì sao status đó không gọi thật được; thừa / thiếu / lý do vô nghĩa → FAIL"
}

main 06 "$@"
