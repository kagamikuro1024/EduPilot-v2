#!/usr/bin/env bash
# QC US-PG-07 — hạ tầng chạy (compose 10 service, Caddy, PgBouncer, image, CI, k6, baseline, env thiếu, bề mặt mạng, Redis, route thử ở target thử)
# Nguồn: docs/specs/FEAT-pg-foundation/US.md v1.2 US-PG-07 AC1…AC20 + SRS.md 8.2, 8.3, 8.4, 8.5, 9.3.
# Chạy ở gốc worktree sau `pnpm dev`:  bash docs/sprints/2/qc/scripts/pg07.sh [MM …|--list]
# Trạng thái chuẩn của story: stack đầy đủ, CHẾ ĐỘ DEFAULT (image edupilot-gateway/worker), `--scale gateway=2`.
# Mọi TC đổi chế độ / tắt service / sửa .env.local đều khôi phục trạng thái chuẩn ngay sau lệnh đo, trước khi chấm.
source "$(dirname "$0")/lib.sh"

SVC10="caddy frontend gateway mailpit migrate minio pgbouncer postgres redis worker"

# ---------- tiện ích riêng của story ----------
_pg07_cfgjson() { $C config --format json 2>/dev/null; }                 # compose config đã hợp nhất, dạng JSON
_pg07_ctcfgjson() { $CT config --format json 2>/dev/null; }
_pg07_norm_ts() {  # chuẩn hoá mốc thời gian của docker inspect: chỉ giữ chữ số, đệm '0' đủ 24 ký tự → so sánh được bằng thứ tự từ điển
  printf '%s' "$1" | tr -dc '0-9' | awk '{printf "%-24s\n", $0}' | tr ' ' '0'; }
_pg07_jlog() {  # $1=service, $2=biểu thức jq trên từng dòng log JSON
  $C logs --no-log-prefix "$1" 2>/dev/null | jq -R "fromjson? | $2" -r 2>/dev/null; }
_pg07_redis_wait() { local i=0; while [ $i -lt 30 ]; do [ "$($RDS ping 2>/dev/null)" = PONG ] && return 0; sleep 1; i=$((i+1)); done; return 1; }

# ---------- AC1: 10 service, không Python ----------
tc_pg07_01() {  # AC1 — đúng 10 tên service
  local svc; svc=$($C config --services 2>/dev/null | sort | paste -sd' ' -)
  chk "danh sách service của docker-compose.local.yml" "$svc" "$SVC10"
}
tc_pg07_02() {  # AC1 — không service Python / docling
  chk "service có tên python|docling" "$($C config --services 2>/dev/null | grep -ciE 'python|docling')" 0
  chk "chuỗi docling trong compose" "$(grep -ci 'docling' docker-compose.local.yml)" 0
}

# ---------- AC2: up --wait ----------
tc_pg07_03() {  # AC2 — `$C up -d --scale gateway=2 --wait` từ máy sạch thoát 0
  local s e rc
  $C down --remove-orphans >/dev/null 2>&1           # "máy sạch": bỏ container, GIỮ volume dữ liệu
  s=$(now_ms)
  $C up -d --scale gateway=2 --wait > "$QC_OUT/tc-PG07-03.log" 2>&1; rc=$?
  e=$(now_ms); echo "    (thời gian up --wait: $(( (e-s)/1000 )) s — xem $QC_OUT/tc-PG07-03.log)"
  wait_ready 120
  chk "rc của up -d --scale gateway=2 --wait" "$rc" 0
}
tc_pg07_04() {  # AC2 — bảng trạng thái từng service
  local f=$QC_OUT/tc-PG07-04.txt s
  $C ps -a --format '{{.Service}} {{.State}} {{.Health}} {{.ExitCode}}' | sort > "$f"
  chk "số dòng 'gateway running healthy 0'" "$(grep -c '^gateway running healthy 0$' "$f")" 2
  for s in postgres redis minio mailpit pgbouncer caddy worker frontend; do
    chk "$s running healthy 0" "$(grep -c "^$s running healthy 0$" "$f")" 1
  done
  chk "migrate <state> <exit>" "$(awk '$1=="migrate"{print $2, $NF; exit}' "$f")" "exited 0"
}
tc_pg07_05() {  # AC2 — migrate kết thúc TRƯỚC khi gateway/worker khởi động
  local mid fin id st
  mid=$($C ps -aq migrate | head -1)
  fin=$(_pg07_norm_ts "$(docker inspect -f '{{.State.FinishedAt}}' "$mid" 2>/dev/null)")
  chk_re "migrate có FinishedAt thật (không phải mốc rỗng 0001-01-01)" "$fin" '^20[0-9]{12}'
  for id in $($C ps -q gateway) $($C ps -q worker); do
    st=$(_pg07_norm_ts "$(docker inspect -f '{{.State.StartedAt}}' "$id")")
    if [ "$st" \> "$fin" ]; then echo "    ok   $id StartedAt ($st) > migrate FinishedAt ($fin)"
    else fail_tc "$id StartedAt ($st) không sau migrate FinishedAt ($fin)"; fi
  done
}
tc_pg07_06() {  # AC2 — depends_on: migrate service_completed_successfully
  local j; j=$(_pg07_cfgjson)
  chk "gateway depends_on.migrate.condition" "$(printf '%s' "$j" | jq -r '.services.gateway.depends_on.migrate.condition // "KHÔNG CÓ"')" service_completed_successfully
  chk "worker depends_on.migrate.condition"  "$(printf '%s' "$j" | jq -r '.services.worker.depends_on.migrate.condition  // "KHÔNG CÓ"')" service_completed_successfully
  chk "gateway depends_on.pgbouncer.condition" "$(printf '%s' "$j" | jq -r '.services.gateway.depends_on.pgbouncer.condition // "KHÔNG CÓ"')" service_healthy
  chk "gateway depends_on.redis.condition"     "$(printf '%s' "$j" | jq -r '.services.gateway.depends_on.redis.condition // "KHÔNG CÓ"')" service_healthy
}

# ---------- AC3: Caddy (mỗi mệnh đề một TC) ----------
tc_pg07_07() {  # AC3 — /api/v1/healthz qua Caddy: 200 {"status":"ok"}
  chk "mã HTTP" "$(code $GW/api/v1/healthz)" 200
  chk "status trong thân" "$(curl -fsSk --max-time 20 $GW/api/v1/healthz | jq -r '.status // "KHÔNG CÓ"')" ok
}
tc_pg07_08() {  # AC3 — `/` trả HTML của frontend
  chk_ge "số lần '<html' ở trang gốc" "$(curl -sk --max-time 30 $GW/ | grep -c '<html')" 1
  chk_re "content-type của /" "$(hdr $GW/ | hval content-type)" 'text/html'
}
tc_pg07_09() {  # AC3 — http://localhost chuyển hướng sang https
  local h; h=$(curl -sI --max-time 10 http://localhost/ | tr -d '\r')
  chk_re "dòng trạng thái" "$(printf '%s\n' "$h" | head -1)" '^HTTP/[0-9.]+ (301|308)'
  chk_re "header location" "$(printf '%s\n' "$h" | hval location)" '^https://localhost/?$'
}
tc_pg07_10() {  # AC3 — HSTS + nosniff (đo bằng cả GET và HEAD)
  local g i; g=$(hdr $GW/api/v1/healthz); i=$(curl -skI --max-time 20 $GW/api/v1/healthz | tr -d '\r')
  chk "GET: số header (HSTS|nosniff)" "$(printf '%s\n' "$g" | grep -ciE '^(strict-transport-security|x-content-type-options: nosniff)')" 2
  chk_re "max-age của HSTS" "$(printf '%s\n' "$g" | hval strict-transport-security)" 'max-age=31536000'
  echo "    (HEAD trả: $(printf '%s\n' "$i" | head -1))"
  chk "HEAD: số header (HSTS|nosniff)" "$(printf '%s\n' "$i" | grep -ciE '^(strict-transport-security|x-content-type-options: nosniff)')" 2
}
tc_pg07_11() {  # AC3 — HTML lớn được nén
  local h; h=$(curl -sk --max-time 30 -D- -o /dev/null -H 'Accept-Encoding: gzip, zstd' $GW/ | tr -d '\r')
  chk_re "content-encoding của /" "$(printf '%s\n' "$h" | hval content-encoding)" '^(zstd|gzip)$'
}
tc_pg07_12() {  # AC3 — JSON nhỏ và text/event-stream KHÔNG nén dù client xin
  local hj hs
  hj=$(curl -sk --max-time 20 -D- -o /dev/null -H 'Accept-Encoding: gzip, zstd' $GW/api/v1/healthz | tr -d '\r')
  chk "số header content-encoding của JSON nhỏ" "$(printf '%s\n' "$hj" | grep -ci '^content-encoding')" 0
  hs=$(curl -sk -N --max-time 4 -D- -o /dev/null -H 'Accept-Encoding: gzip, zstd' -H "$H" $GW/api/v1/events | tr -d '\r')
  chk_re "content-type của SSE" "$(printf '%s\n' "$hs" | hval content-type)" 'text/event-stream'
  chk "số header content-encoding của SSE" "$(printf '%s\n' "$hs" | grep -ci '^content-encoding')" 0
}
tc_pg07_13() {  # AC3 / SRS 8.2 — header `-Server` bị gỡ
  chk "header Server ở /api/v1/healthz" "$(hdr $GW/api/v1/healthz | grep -ci '^server:')" 0
  chk "header Server ở /"               "$(hdr $GW/ | grep -ci '^server:')" 0
}

# ---------- AC4: chứng chỉ TLS nội bộ ----------
tc_pg07_14() {  # AC4 — issuer Caddy Local Authority, SAN localhost, có hạn
  local f=$QC_OUT/tc-PG07-14.txt
  echo | openssl s_client -connect localhost:443 -servername localhost 2>/dev/null \
    | openssl x509 -noout -issuer -subject -dates -text 2>/dev/null > "$f"
  chk_ge "số dòng chứa 'Caddy Local Authority'" "$(grep -c 'Caddy Local Authority' "$f")" 1
  chk_ge "SAN DNS:localhost" "$(grep -c 'DNS:localhost' "$f")" 1
  chk_ge "có notAfter (hạn chứng chỉ)" "$(grep -c '^notAfter=' "$f")" 1
}

# ---------- AC5: hai bản gateway, không trạng thái phiên ----------
tc_pg07_15() {  # AC5 — 40 GET /api/v1/healthz: 40×200, đúng 2 X-Instance-Id, mỗi bản > 0 lần
  local f=$QC_OUT/tc-PG07-15.txt i
  : > "$f"
  for i in $(seq 40); do
    curl -sk --max-time 20 -D- -o /dev/null -w 'S %{http_code}\n' $GW/api/v1/healthz | tr -d '\r' \
      | awk -F': ' 'tolower($1)=="x-instance-id"{print "I "$2} /^S [0-9]/{print}' >> "$f"   # #5/#11: Caddy phục vụ HTTP/2 → lấy mã bằng -w
  done
  chk "số lần mã 200" "$(grep -c '^S 200$' "$f")" 40
  chk "số lần mã khác 200" "$(grep '^S ' "$f" | grep -vc '^S 200$')" 0
  chk "số giá trị X-Instance-Id khác nhau" "$(grep '^I ' "$f" | sort -u | grep -c .)" 2
  chk_ge "số lần của bản ít được chọn nhất" "$(grep '^I ' "$f" | sort | uniq -c | awk '{print $1}' | sort -n | head -1)" 1
}
tc_pg07_16() {  # AC5 — X-Instance-Id khớp container gateway thật
  local names v i ids
  names=$(for c in $(gw_ids); do docker inspect -f '{{.Config.Hostname}} {{.Name}} {{.Id}}' "$c"; done)
  ids=$(for i in $(seq 12); do hdr $GW/api/v1/healthz | hval x-instance-id; done | sort -u)
  chk_ge "số bản gateway đang chạy" "$(printf '%s\n' "$names" | grep -c .)" 2
  for v in $ids; do
    if printf '%s\n' "$names" | grep -q -- "$v"; then echo "    ok   X-Instance-Id [$v] khớp container"
    else fail_tc "X-Instance-Id [$v] không khớp container nào: $names"; fi
  done
}
tc_pg07_17() {  # AC5 — một token (ký một lần) dùng 20 lần: toàn 404, không bao giờ 401
  local T out f=$QC_OUT/tc-PG07-17.txt
  T=$(tok STUDENT $U1); : > "$f"
  for i in $(seq 20); do code -H "Authorization: Bearer $T" $GW/api/v1/jobs/$UX >> "$f"; echo >> "$f"; done
  out=$(grep -c '^404$' "$f")
  chk "số lần 404" "$out" 20
  chk "số lần 401" "$(grep -c '^401$' "$f")" 0
}

# ---------- AC6 (nhánh lỗi): tắt một bản gateway ----------
_pg07_kill_round() {  # $1 = số vòng; in "<số dòng không-200> <số dòng không-200 ở 60 dòng cuối>"
  local f=$QC_OUT/tc-PG07-18-r$1.codes bg i
  rl_reset
  ( for i in $(seq 200); do curl -sk --max-time 5 -o /dev/null -w '%{http_code}\n' $GW/api/v1/healthz; sleep 0.1; done > "$f" ) &
  bg=$!
  sleep 5
  docker stop "$($C ps -q gateway | head -1)" >/dev/null 2>&1
  wait $bg
  $C up -d --scale gateway=2 --wait gateway >/dev/null 2>&1; wait_ready 90     # khôi phục NGAY, trước khi chấm
  echo "$(grep -vc '^200$' "$f") $(tail -60 "$f" | grep -vc '^200$')"
}
tc_pg07_18() {  # AC6 — 200 request 10/s, tắt một gateway ở giây 5; lặp 3 vòng, pass khi cả 3 đạt
  # "lỗi" = mọi response không-200 HOẶC kết nối 000; ≤ 4/200 và 0 ở 60 dòng cuối (US.md 07-AC6, QC questions #Q-QC-07-4)
  local r res bad tail60
  for r in 1 2 3; do
    res=$(_pg07_kill_round $r); bad=${res% *}; tail60=${res#* }
    echo "    vòng $r: không-200 = $bad, 60 dòng cuối không-200 = $tail60 ($QC_OUT/tc-PG07-18-r$r.codes)"
    chk_le "vòng $r: số dòng không-200 (≤ 2 % của 200)" "$bad" 4
    chk "vòng $r: số dòng không-200 ở 60 dòng cuối (sau ~6 s)" "$tail60" 0
  done
}
tc_pg07_19() {  # AC6 — không ai mất đăng nhập: cùng token trước và sau khi tắt một bản
  local T c1 c2
  T=$(tok STUDENT $U1)
  c1=$(code -H "Authorization: Bearer $T" $GW/api/v1/jobs/$UX)
  docker stop "$($C ps -q gateway | head -1)" >/dev/null 2>&1
  sleep 7
  c2=$(code -H "Authorization: Bearer $T" $GW/api/v1/jobs/$UX)
  $C up -d --scale gateway=2 --wait gateway >/dev/null 2>&1; wait_ready 90
  chk "trước khi tắt" "$c1" 404
  chk "sau khi tắt, cùng token (không 401)" "$c2" 404
}
tc_pg07_20() {  # AC6 — khôi phục đủ hai bản sau khi tắt một bản
  docker stop "$($C ps -q gateway | head -1)" >/dev/null 2>&1
  $C up -d --scale gateway=2 --wait gateway >/dev/null 2>&1; wait_ready 90
  chk "số container gateway 'running healthy 0'" "$($C ps -a --format '{{.Service}} {{.State}} {{.Health}} {{.ExitCode}}' | grep -c '^gateway running healthy 0$')" 2
}

# ---------- AC7: PgBouncer ----------
tc_pg07_21() {  # AC7 / SRS 8.3 — SHOW CONFIG đúng từng khoá
  local cfg f=$QC_OUT/tc-PG07-21.txt
  $C exec -T postgres psql "$PGB" -Atc 'SHOW CONFIG' > "$f" 2>&1
  cfg=$(cat "$f")
  _v() { printf '%s\n' "$cfg" | awk -F'|' -v k="$1" '$1==k{print $2; exit}'; }
  chk "pool_mode"                 "$(_v pool_mode)" transaction
  chk "listen_port"               "$(_v listen_port)" 6432
  chk "max_client_conn"           "$(_v max_client_conn)" 200
  chk "default_pool_size"         "$(_v default_pool_size)" 20
  chk "reserve_pool_size"         "$(_v reserve_pool_size)" 5
  chk "reserve_pool_timeout"      "$(_v reserve_pool_timeout)" 3
  chk "query_wait_timeout"        "$(_v query_wait_timeout)" 30
  chk "server_lifetime"           "$(_v server_lifetime)" 1800
  chk "server_idle_timeout"       "$(_v server_idle_timeout)" 600
  chk "auth_type"                 "$(_v auth_type)" scram-sha-256
  chk "max_prepared_statements"   "$(_v max_prepared_statements)" 0
  chk_re "ignore_startup_parameters" "$(_v ignore_startup_parameters)" 'extra_float_digits'
}
tc_pg07_22() {  # AC7 — 6432 chỉ trong mạng compose, không công bố ra host
  chk "số cổng công bố của pgbouncer" "$($C ps -a --format '{{.Service}} {{.Ports}}' | awk '$1=="pgbouncer"' | grep -c -- '->')" 0
  chk_ok "đối chứng: cổng 5433 (postgres) mở được từ host" nc -z -w 2 127.0.0.1 5433
  chk_no "cổng 6432 KHÔNG mở được từ host" nc -z -w 2 127.0.0.1 6432
}
tc_pg07_23() {  # AC7 — mở được từ container postgres bằng tài khoản thống kê
  chk_ok "psql \"\$PGB\" -Atc 'SHOW VERSION' từ container postgres" $C exec -T postgres psql "$PGB" -Atc 'SHOW VERSION'
  chk_re "chuỗi phiên bản" "$($C exec -T postgres psql "$PGB" -Atc 'SHOW VERSION' 2>/dev/null | head -1 | tr 'A-Z' 'a-z')" 'pgbouncer'
}

# ---------- AC8: runtime qua PgBouncer, migrate đi thẳng ----------
tc_pg07_24() {  # AC8 — log `config loaded` có db_via=pgbouncer ở cả gateway và worker
  chk "gateway db_via" "$(_pg07_jlog gateway 'select(.msg=="config loaded") | .db_via' | sort -u | paste -sd, -)" pgbouncer
  chk "worker db_via"  "$(_pg07_jlog worker  'select(.msg=="config loaded") | .db_via' | sort -u | paste -sd, -)" pgbouncer
}
tc_pg07_25() {  # AC8 — SHOW CLIENTS ≥ 3 client ứng dụng, không có client migrate
  local cl f=$QC_OUT/tc-PG07-25.txt
  code -H "$H" $GW/api/v1/jobs/$UX >/dev/null      # bảo đảm gateway đã chạm DB
  $C exec -T postgres psql "$PGB" -Atc 'SHOW CLIENTS' > "$f" 2>&1; cl=$(cat "$f")
  chk_ge "số client edupilot" "$(printf '%s\n' "$cl" | grep -c edupilot)" 3
  chk "số client migrate" "$(printf '%s\n' "$cl" | grep -ci migrate)" 0
}
tc_pg07_26() {  # AC8 — migrate nối thẳng Postgres (không qua PgBouncer), kể cả khi chạy lại
  local envstr i hit=0
  envstr=$(_pg07_cfgjson | jq -r '.services.migrate.environment | tostring')
  chk_re "env của migrate dùng Postgres trực tiếp" "$envstr" 'postgres:5432'
  chk_nre "env của migrate không trỏ PgBouncer" "$envstr" 'pgbouncer:6432'
  $C up -d --force-recreate migrate >/dev/null 2>&1
  i=0; while [ $i -lt 12 ]; do
    $C exec -T postgres psql "$PGB" -Atc 'SHOW CLIENTS' 2>/dev/null | grep -qi migrate && hit=1
    i=$((i+1))
  done
  chk "số lần thấy client migrate trên PgBouncer (12 lần lấy mẫu)" "$hit" 0
  chk "migrate chạy lại vẫn thoát 0" "$($C ps -a --format '{{.Service}} {{.State}} {{.ExitCode}}' | awk '$1=="migrate"{print $2, $NF; exit}')" "exited 0"
}

# ---------- AC9: pgx hợp transaction mode ----------
tc_pg07_27() {  # AC9 — test Go qua PgBouncer thật
  gt ./internal/platform/db 'TestPgBouncer_TransactionMode|TestPgBouncer_NoPreparedStatements|TestVector_ThroughPgBouncer'
}
tc_pg07_28() {  # AC9 (hộp đen) — 20 vòng × 50 request song song đi qua PgBouncer, không lỗi giao thức
  local T f=$QC_OUT/tc-PG07-28.codes r i
  ensure_mode default || fail_tc "không về được chế độ default"
  T=$(tok STUDENT $U1); : > "$f"
  r=0; while [ $r -lt 20 ]; do
    rl_reset                                   # 1.000 request > 300/phút/IP: xoá bộ đếm giữa các vòng
    i=0; while [ $i -lt 50 ]; do
      curl -sk --max-time 20 -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $T" $GW/api/v1/jobs/$UX >> "$f" &
      i=$((i+1))
    done
    wait; r=$((r+1))
  done
  chk "số request trả 404 (có truy vấn DB)" "$(grep -c '^404$' "$f")" 1000
  chk "số dòng log gateway có lỗi prepared statement / bind message" \
      "$($C logs --since 10m gateway 2>/dev/null | grep -ciE 'prepared statement|bind message|unnamed prepared')" 0
}

# ---------- AC10: image ----------
tc_pg07_29() {  # AC10 — build hai target từ cùng một Dockerfile
  local t s e rc
  for t in gateway worker; do
    s=$(now_ms); docker build -q --target $t -t edupilot-$t backend-go > "$QC_OUT/tc-PG07-29-$t.log" 2>&1; rc=$?; e=$(now_ms)
    echo "    (build --target $t: $(( (e-s)/1000 )) s)"
    chk "rc của docker build --target $t" "$rc" 0
  done
  chk_ge "số dòng FROM trong backend-go/Dockerfile" "$(grep -cE '^FROM ' backend-go/Dockerfile)" 2
}
tc_pg07_30() {  # AC10 — mỗi image < 40.000.000 byte
  local t sz
  for t in gateway worker; do
    sz=$(docker image inspect -f '{{.Size}}' edupilot-$t 2>/dev/null)
    echo "    (edupilot-$t = ${sz:-?} byte)"
    chk_le "kích thước edupilot-$t (byte)" "${sz:-999999999}" 39999999
  done
}
tc_pg07_31() {  # AC10 — chạy bằng người dùng không phải root
  local t u
  for t in gateway worker; do
    u=$(docker image inspect -f '{{.Config.User}}' edupilot-$t 2>/dev/null)
    chk_re "Config.User của edupilot-$t" "$u" '^(nonroot|65532)(:(nonroot|65532))?$'
  done
}
tc_pg07_32() {  # AC10 — không có shell trong image
  local t
  for t in gateway worker; do
    chk_no "edupilot-$t không chạy được sh" docker run --rm --entrypoint sh edupilot-$t -c true
    chk_no "edupilot-$t không chạy được /bin/sh" docker run --rm --entrypoint /bin/sh edupilot-$t -c true
  done
}
tc_pg07_33() {  # AC10 / SRS 8.4 — migrate dùng lại image gateway, entrypoint+command đúng y hệt (QC questions #Q-QC-07-1)
  local j; j=$(_pg07_cfgjson)
  chk "image của service migrate" "$(printf '%s' "$j" | jq -r '.services.migrate.image // "KHÔNG CÓ"')" edupilot-gateway
  chk "entrypoint của migrate" "$(printf '%s' "$j" | jq -cr '.services.migrate.entrypoint // "KHÔNG CÓ"')" '["/gateway","migrate"]'
  chk "command của migrate"    "$(printf '%s' "$j" | jq -cr '.services.migrate.command // "KHÔNG CÓ"')" '["up"]'
}

# ---------- AC11: không trạng thái ----------
tc_pg07_34() {  # AC11a — read_only rootfs cho mọi container gateway + worker
  local ids id v n=0 bad=0
  ids="$($C ps -q gateway) $($C ps -q worker)"
  for id in $ids; do
    v=$(docker inspect -f '{{.HostConfig.ReadonlyRootfs}}' "$id"); n=$((n+1))
    [ "$v" = true ] || { bad=$((bad+1)); echo "    (container $id: ReadonlyRootfs=$v)"; }
  done
  chk_ge "số container gateway+worker đã kiểm" "$n" 3
  chk "số container KHÔNG read_only" "$bad" 0
}
tc_pg07_35() {  # AC11b — mã sản xuất không ghi đĩa cục bộ (kiểm tĩnh do AC chỉ định)
  local out
  out=$(cd backend-go && grep -rnE 'os\.(Create|CreateTemp|WriteFile|OpenFile|Mkdir|MkdirAll|MkdirTemp)|ioutil\.(WriteFile|TempFile)' internal cmd --include=*.go 2>/dev/null | grep -v '_test.go')
  printf '%s\n' "$out" > "$QC_OUT/tc-PG07-35.txt"
  chk "số chỗ ghi đĩa trong mã sản xuất" "$(printf '%s' "$out" | grep -c .)" 0
}
tc_pg07_36() {  # AC11c — golangci-lint sạch (gồm gochecknoglobals)
  local rc
  (cd backend-go && golangci-lint run) > "$QC_OUT/tc-PG07-36.log" 2>&1; rc=$?
  chk "rc của golangci-lint run" "$rc" 0
  chk "số dòng báo lỗi lint" "$(grep -cE '^[^ ].*\.go:[0-9]+' "$QC_OUT/tc-PG07-36.log")" 0
}
tc_pg07_37() {  # AC11b — chứng minh động: sau tải, lớp ghi của container không đổi
  local id i d ids
  for i in $(seq 60); do code $GW/api/v1/healthz >/dev/null; done
  code -H "$H" $GW/api/v1/jobs/$UX >/dev/null
  ids="$($C ps -q gateway) $($C ps -q worker)"
  for id in $ids; do
    d=$(docker diff "$id" | grep -c .)
    [ "$d" = 0 ] || docker diff "$id" > "$QC_OUT/tc-PG07-37-$id.txt"
    chk "số thay đổi hệ thống file của $id" "$d" 0
  done
}

# ---------- AC12: CI ----------
_pg07_runid() {  # id của run ci.yml mới nhất trên nhánh $1
  gh run list --workflow ci.yml --branch "$1" --limit 1 --json databaseId --jq '.[0].databaseId' 2>/dev/null; }
tc_pg07_38() {  # AC12 — CI xanh ở HEAD nhánh sprint/2-pg
  local sha j
  git fetch -q origin sprint/2-pg 2>/dev/null
  sha=$(git rev-parse origin/sprint/2-pg)
  j=$(gh run list --workflow ci.yml --branch sprint/2-pg --limit 1 --json databaseId,headSha,conclusion 2>/dev/null)
  printf '%s\n' "$j" > "$QC_OUT/tc-PG07-38.json"
  chk "headSha của run mới nhất" "$(printf '%s' "$j" | jq -r '.[0].headSha // "KHÔNG CÓ"')" "$sha"
  chk "conclusion" "$(printf '%s' "$j" | jq -r '.[0].conclusion // "KHÔNG CÓ"')" success
}
tc_pg07_39() {  # AC12 — ≥ 6 bước tên chứa sqlc|race|vet|golangci
  local id n
  id=$(_pg07_runid sprint/2-pg); chk_re "tìm được run id" "${id:-}" '^[0-9]+$'
  gh run view "$id" --json jobs --jq '.jobs[].steps[].name' > "$QC_OUT/tc-PG07-39.txt" 2>/dev/null
  n=$(grep -ciE 'sqlc|race|vet|golangci' "$QC_OUT/tc-PG07-39.txt")
  chk_ge "số bước khớp sqlc|race|vet|golangci" "$n" 6
}
tc_pg07_40() {  # AC12 — go vet và golangci-lint chạy HAI lần (hai cấu hình tag)
  local id f
  id=$(_pg07_runid sprint/2-pg); f=$QC_OUT/tc-PG07-40.txt
  gh run view "$id" --json jobs --jq '.jobs[].steps[].name' > "$f" 2>/dev/null
  chk_ge "số bước có 'vet'"      "$(grep -ci 'vet' "$f")" 2
  chk_ge "số bước có 'golangci'" "$(grep -ci 'golangci' "$f")" 2
  chk_ge "số bước có 'sqlc'"     "$(grep -ci 'sqlc' "$f")" 1
}
tc_pg07_41() {  # AC12 — job Frontend còn nguyên và xanh
  local id j
  id=$(_pg07_runid sprint/2-pg)
  j=$(gh run view "$id" --json jobs --jq '.jobs[] | "\(.name)|\(.conclusion)"' 2>/dev/null)
  printf '%s\n' "$j" > "$QC_OUT/tc-PG07-41.txt"
  chk_ge "số job tên chứa Frontend" "$(printf '%s\n' "$j" | grep -ci '^frontend')" 1
  chk "job Frontend không success" "$(printf '%s\n' "$j" | grep -i '^frontend' | grep -vc '|success$')" 0
  chk "job Go không success"       "$(printf '%s\n' "$j" | grep -i '^go'       | grep -vc '|success$')" 0
}
tc_pg07_42() {  # AC12 — ci.yml nhắc testroutes ≥ 3 lần, không đọc legacy, không dùng secret
  chk_ge "số lần 'testroutes' trong ci.yml" "$(grep -c 'testroutes' .github/workflows/ci.yml)" 3
  chk "số dòng khớp 'legacy|secrets\\.'" "$(grep -nE 'legacy|secrets\.' .github/workflows/ci.yml | grep -c .)" 0
}

# ---------- AC13 (nhánh lỗi): CI chặn lệch sqlc ----------
tc_pg07_43() {  # AC13 — run của nhánh ci/sqlc-drift đỏ đúng chỗ
  local id j f=$QC_OUT/tc-PG07-43.json
  id=${RUN_ID:-$(_pg07_runid ci/sqlc-drift)}
  if [ -z "${id:-}" ]; then fail_tc "không tìm thấy run ci.yml của nhánh ci/sqlc-drift (chạy lại với RUN_ID=<id> lấy từ handoff; nhánh bị xoá sớm hơn lúc QC chấm = FAIL)"; return; fi
  gh run view "$id" --json conclusion,headSha,headBranch,jobs > "$f" 2>/dev/null
  j=$(cat "$f")
  chk "conclusion của run" "$(printf '%s' "$j" | jq -r '.conclusion // "KHÔNG CÓ"')" failure
  chk "headBranch" "$(printf '%s' "$j" | jq -r '.headBranch // "KHÔNG CÓ"')" ci/sqlc-drift
  chk "job Go"       "$(printf '%s' "$j" | jq -r '.jobs[] | select(.name|test("^Go";"i")) | .conclusion' | sort -u | paste -sd, -)" failure
  chk "job Frontend" "$(printf '%s' "$j" | jq -r '.jobs[] | select(.name|test("^Frontend";"i")) | .conclusion' | sort -u | paste -sd, -)" success
  chk_ge "số bước thất bại có tên chứa sqlc" \
    "$(printf '%s' "$j" | jq -r '.jobs[].steps[] | select(.conclusion=="failure") | .name' | grep -ci sqlc)" 1
}
tc_pg07_44() {  # AC13 — tính xác thực: headSha = SHA nhánh, và chỉ đụng internal/store/queries/*.sql
  local id sha remote files
  id=${RUN_ID:-$(_pg07_runid ci/sqlc-drift)}
  if [ -z "${id:-}" ]; then fail_tc "không có run ci/sqlc-drift để đối chiếu"; return; fi
  sha=$(gh run view "$id" --json headSha --jq '.headSha' 2>/dev/null)
  remote=$(git ls-remote --heads origin ci/sqlc-drift | awk '{print $1}')
  chk "headSha của run = SHA đầu nhánh trên origin" "$sha" "$remote"
  git fetch -q origin sprint/2-pg ci/sqlc-drift 2>/dev/null
  files=$(git diff --name-only "origin/sprint/2-pg...$sha" 2>/dev/null)
  printf '%s\n' "$files" > "$QC_OUT/tc-PG07-44.txt"
  chk_ge "số tệp internal/store/queries/*.sql bị đổi" "$(printf '%s\n' "$files" | grep -c 'internal/store/queries/.*\.sql$')" 1
  chk "số tệp đổi NGOÀI internal/store/queries/*.sql" "$(printf '%s\n' "$files" | grep -v 'internal/store/queries/.*\.sql$' | grep -c .)" 0
}
tc_pg07_45() {  # AC13 (Tay) — sau khi QC chấm xong, dev xoá nhánh ci/sqlc-drift
  manual "chạy sau TC-PG07-43/44: báo dev xoá nhánh, rồi 'git ls-remote --heads origin ci/sqlc-drift' phải không in gì (xem Bước tay TC-PG07-45)"
}

# ---------- AC14: k6 ----------
tc_pg07_46() {  # AC14 — k6 smoke trên stack mặc định: rc=0, mọi ngưỡng ✓
  local rc T f=$QC_OUT/k6.txt
  command -v k6 >/dev/null 2>&1 || { fail_tc "k6 chưa cài trên máy chạy QC"; return; }
  ensure_mode default || fail_tc "không về được chế độ default"
  T=$(tok STUDENT $U1)
  k6 run -e BASE=https://localhost -e TOKEN="$T" benchmarks/load/smoke.js > "$f" 2>&1; rc=$?
  grep -E 'p\(95\)|http_req_failed|checks|✓|✗' "$f" | head -20
  chk "rc của k6 run" "$rc" 0
  chk "số ngưỡng ✗ (không đạt)" "$(grep -c '✗' "$f")" 0
  chk_ge "số dòng ngưỡng ✓" "$(grep -c '✓' "$f")" 3
}
tc_pg07_47() {  # AC14 — ngưỡng nằm trong options.thresholds và KHÔNG bị nới
  local f=benchmarks/load/smoke.js
  chk_ge "số lần 'p(95)<300'" "$(grep -c 'p(95)<300' "$f")" 3
  chk "tập ngưỡng p(95) có trong tệp" "$(grep -oE 'p\(95\)<[0-9]+' "$f" | sort -u | paste -sd, -)" "p(95)<300,p(95)<500"
  chk_ge "ngưỡng http_req_failed rate<0.005" "$(grep -c 'rate<0.005' "$f")" 1
  chk_ge "ngưỡng checks rate>0.99" "$(grep -cE 'rate>0\.99' "$f")" 1
}
tc_pg07_48() {  # AC14 — kịch bản ghi với -e TEST_ROUTES=1 ở chế độ test (p95 ≤ 500)
  local rc T f=$QC_OUT/k6-testroutes.txt
  command -v k6 >/dev/null 2>&1 || { fail_tc "k6 chưa cài trên máy chạy QC"; return; }
  ensure_mode test || fail_tc "không dựng được chế độ test"
  T=$(tok STUDENT $U1)
  k6 run -e BASE=https://localhost -e TOKEN="$T" -e TEST_ROUTES=1 benchmarks/load/smoke.js > "$f" 2>&1; rc=$?
  ensure_mode default >/dev/null 2>&1                       # trả stack về trạng thái chuẩn trước khi chấm
  grep -E 'p\(95\)|http_req_failed|checks|✓|✗' "$f" | head -20
  chk "rc của k6 run (TEST_ROUTES=1, tmode)" "$rc" 0
  chk "số ngưỡng ✗" "$(grep -c '✗' "$f")" 0
  chk_ge "có ngưỡng p(95)<500 của kịch bản ghi" "$(grep -c 'p(95)<500' "$f")" 1
}
tc_pg07_49() {  # AC14 — TEST_ROUTES=1 trên stack mặc định: rc≠0 + đúng chuỗi thông báo (QC questions #Q-QC-07-5)
  local rc T f=$QC_OUT/k6-testroutes-default.txt
  command -v k6 >/dev/null 2>&1 || { fail_tc "k6 chưa cài trên máy chạy QC"; return; }
  ensure_mode default || fail_tc "không về được chế độ default"
  T=$(tok STUDENT $U1)
  k6 run -e BASE=https://localhost -e TOKEN="$T" -e TEST_ROUTES=1 benchmarks/load/smoke.js > "$f" 2>&1; rc=$?
  tail -15 "$f"
  chk_ne "rc của k6 run (TEST_ROUTES=1, dmode)" "$rc" 0
  chk_ge "đầu ra chứa đúng chuỗi 'TEST_ROUTES=1 cần stack dựng bằng docker-compose.test.yml'" \
      "$(grep -cF 'TEST_ROUTES=1 cần stack dựng bằng docker-compose.test.yml' "$f")" 1
}

# ---------- AC15: baseline ----------
tc_pg07_50() {  # AC15 — pg-baseline.md đủ 4 số và startup_ms là số
  local f=benchmarks/reports/pg-baseline.md s
  chk_ok "tệp benchmarks/reports/pg-baseline.md tồn tại" test -f "$f"
  chk_ge "số dòng khớp RAM|khởi động|image|p95" "$(grep -ciE 'RAM|khởi động|image|p95' "$f" 2>/dev/null)" 4
  chk_ge "số con số trong báo cáo" "$(grep -oE '[0-9]+([.,][0-9]+)?' "$f" 2>/dev/null | grep -c .)" 4
  s=$(_pg07_jlog gateway 'select(.msg=="gateway ready") | .startup_ms' | head -1)
  chk_re "startup_ms trong log là số" "${s:-}" '^[0-9]+$'
}
tc_pg07_51() {  # AC15 — QC tự đo lại 4 số (RAM nghỉ sau 60 s, kích thước image, p95 healthz)
  local f=$QC_OUT/tc-PG07-51.txt g w sg sw p
  sleep 60                                              # 60 s rảnh theo AC
  docker stats --no-stream --format '{{.Name}} {{.MemUsage}}' > "$f" 2>&1
  cat "$f"
  g=$(grep -i 'gateway' "$f" | head -1 | awk '{print $2}')
  w=$(grep -i 'worker'  "$f" | head -1 | awk '{print $2}')
  sg=$(docker image inspect -f '{{.Size}}' edupilot-gateway 2>/dev/null)
  sw=$(docker image inspect -f '{{.Size}}' edupilot-worker 2>/dev/null)
  p=$(grep -E 'http_req_duration' "$QC_OUT/k6.txt" 2>/dev/null | head -1)
  echo "    ĐO LẠI: RAM gateway=$g, RAM worker=$w, image gateway=$sg B, image worker=$sw B, k6: $p"
  chk_re "RAM nghỉ gateway đo được" "${g:-}" '^[0-9]'
  chk_re "RAM nghỉ worker đo được"  "${w:-}" '^[0-9]'
  chk_re "kích thước image gateway đo được" "${sg:-}" '^[0-9]+$'
  chk_re "kích thước image worker đo được"  "${sw:-}" '^[0-9]+$'
  echo "    (PM chốt góp ý #2: chỉ ghi nhận số đo vào report, KHÔNG FAIL vì lệch so với benchmarks/reports/pg-baseline.md)"
}

# ---------- AC16 (nhánh lỗi): thiếu env trong compose ----------
_pg07_drop_jwt() { cp .env.local "$QC_TMP/env.local.bak" && grep -v '^JWT_SECRET_KEY=' "$QC_TMP/env.local.bak" > .env.local; }
_pg07_restore_env() {
  [ -f "$QC_TMP/env.local.bak" ] && cp "$QC_TMP/env.local.bak" .env.local
  $C up -d --force-recreate --scale gateway=2 --wait gateway >/dev/null 2>&1; wait_ready 120; }
tc_pg07_52() {  # AC16 — gateway thoát mã 1 và log nêu tên biến thiếu
  local miss exits
  _pg07_drop_jwt
  trap '_pg07_restore_env' INT TERM
  $C up -d --force-recreate gateway >/dev/null 2>&1
  sleep 8
  miss=$(_pg07_jlog gateway 'select(.missing) | .missing[]' | sort -u | paste -sd, -)
  exits=$($C ps -a --format '{{.Service}} {{.ExitCode}}' | awk '$1=="gateway"{print $2}' | sort -u | paste -sd, -)
  trap - INT TERM; _pg07_restore_env                    # khôi phục trước khi chấm
  chk_ok ".env.local giống hệt bản gốc" cmp -s "$QC_TMP/env.local.bak" .env.local
  chk_re "danh sách missing trong log" "$miss" 'JWT_SECRET_KEY'
  chk_re "mã thoát của container gateway" "$exits" '(^|,)1(,|$)'
}
tc_pg07_53() {  # AC16 — restart tối đa 3 lần rồi dừng (không lặp vô hạn)
  local id n max=0 running=""
  _pg07_drop_jwt
  trap '_pg07_restore_env' INT TERM
  $C up -d --force-recreate gateway >/dev/null 2>&1
  sleep 60
  for id in $($C ps -aq gateway); do
    n=$(docker inspect -f '{{.RestartCount}}' "$id" 2>/dev/null)
    [ "${n:-0}" -gt "$max" ] 2>/dev/null && max=$n
    running="$running $(docker inspect -f '{{.State.Running}}' "$id" 2>/dev/null)"
  done
  trap - INT TERM; _pg07_restore_env
  echo "    (RestartCount lớn nhất = $max; State.Running =$running)"
  chk_le "RestartCount lớn nhất sau 60 s" "$max" 3
  chk "số container gateway còn đang chạy" "$(printf '%s' "$running" | tr ' ' '\n' | grep -c '^true$')" 0
}
tc_pg07_54() {  # AC16 — service migrate không bị ảnh hưởng
  local mid before after state
  mid=$($C ps -aq migrate | head -1)
  before=$(docker inspect -f '{{.State.FinishedAt}}' "$mid" 2>/dev/null)
  _pg07_drop_jwt
  trap '_pg07_restore_env' INT TERM
  $C up -d --no-deps --force-recreate gateway >/dev/null 2>&1   # #7/#11: không tạo lại migrate
  sleep 8
  mid=$($C ps -aq migrate | head -1)
  after=$(docker inspect -f '{{.State.FinishedAt}}' "$mid" 2>/dev/null)
  state=$($C ps -a --format '{{.Service}} {{.State}} {{.ExitCode}}' | awk '$1=="migrate"{print $2, $NF; exit}')
  trap - INT TERM; _pg07_restore_env
  chk "migrate FinishedAt không đổi" "$after" "$before"
  chk "trạng thái migrate" "$state" "exited 0"
}

# ---------- AC17 (phân quyền / bề mặt mạng) ----------
tc_pg07_55() {  # AC17 — gateway, worker, pgbouncer, migrate không công bố cổng
  local f=$QC_OUT/tc-PG07-55.txt
  $C ps -a --format '{{.Service}} {{.Ports}}' | sort > "$f"; cat "$f"
  chk "số dòng gateway|worker|pgbouncer|migrate có '->'" \
      "$(grep -E '^(gateway|worker|pgbouncer|migrate) ' "$f" | grep -c -- '->')" 0
}
tc_pg07_56() {  # AC17 / SRS 8.4 — chỉ caddy công bố 80/443; bảng cổng khớp SRS 8.4
  local f=$QC_OUT/tc-PG07-56.txt
  $C ps -a --format '{{.Service}} {{.Ports}}' | sort > "$f"
  chk "tập service có cổng công bố" "$(awk '/->/{print $1}' "$f" | sort -u | paste -sd' ' -)" "caddy frontend mailpit minio postgres redis"
  chk_ge "caddy công bố 80"   "$(grep '^caddy '    "$f" | grep -c ':80->')" 1
  chk_ge "caddy công bố 443"  "$(grep '^caddy '    "$f" | grep -c ':443->')" 1
  chk_ge "postgres 5433"      "$(grep '^postgres ' "$f" | grep -c '5433->')" 1
  chk_ge "redis 6380"         "$(grep '^redis '    "$f" | grep -c '6380->')" 1
  chk_ge "minio 9000"         "$(grep '^minio '    "$f" | grep -Ec '9000(-[0-9]+)?->')" 1
  chk_ge "mailpit 8025"       "$(grep '^mailpit '  "$f" | grep -c '8025->')" 1
  chk_ge "frontend 3000"      "$(grep '^frontend ' "$f" | grep -c '3000->')" 1
}
tc_pg07_57() {  # AC17 — không bí mật trong compose (mọi giá trị nhạy cảm là ${BIEN}). QC v2: neo regex ở đầu dòng — `${JWT_SECRET_KEY:-}` bị khớp nhầm ở "KEY:-" bên trong tham chiếu biến
  chk "số dòng bí mật trong docker-compose.local.yml" "$(grep -nE '^ *[A-Za-z_]*(PASSWORD|SECRET|KEY)[A-Za-z_]*: *[^$ ]' docker-compose.local.yml | grep -c .)" 0
  chk "số dòng bí mật trong docker-compose.test.yml"  "$(grep -nE '^ *[A-Za-z_]*(PASSWORD|SECRET|KEY)[A-Za-z_]*: *[^$ ]' docker-compose.test.yml 2>/dev/null | grep -c .)" 0
}
tc_pg07_58() {  # AC17 — .env.example chỉ có giá trị dev giả: đo TRỰC TIẾP bằng quy ước '-dev' (PM chốt góp ý #2, QC questions #Q-QC-07-3)
  local ex
  chk_ge "số biến bí mật trong .env.example (không đo rỗng)" "$(grep -cE '^[A-Z_]*(PASSWORD|SECRET|ACCESS_KEY)[A-Z_]*=' .env.example)" 1
  chk "số giá trị bí mật của .env.example KHÔNG chứa '-dev'" \
      "$(grep -E '^[A-Z_]*(PASSWORD|SECRET|ACCESS_KEY)[A-Z_]*=' .env.example | grep -vc -- '-dev')" 0
  chk_ok ".env.local bị git bỏ qua" git check-ignore -q .env.local
  chk "số chỗ lộ JWT_SECRET_KEY ngoài .env.example/docs" "$(git grep -nE 'JWT_SECRET_KEY=[^ ]{20,}' -- ':!.env.example' ':!docs' ':!legacy' | grep -c .)" 0
  chk_ge ".env.example có mục JWT_SECRET_KEY" "$(grep -c '^JWT_SECRET_KEY=' .env.example)" 1
}
tc_pg07_59() {  # AC17 — 8080/8081 không chạm được từ host, nhưng vẫn sống trong mạng compose
  chk "GET http://localhost:8080/healthz từ host" "$(CURL_MAX=5 code http://localhost:8080/healthz)" 000
  chk "GET http://localhost:8081/healthz từ host" "$(CURL_MAX=5 code http://localhost:8081/healthz)" 000
  chk_re "đối chứng trong mạng compose (rawhttp gateway:8080)" \
      "$(rawhttp gateway 8080 'GET /healthz HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n' 5 | head -1 | tr -d '\r')" '^HTTP/1\.1 200'
}

# ---------- AC18: Redis bền + volume ----------
tc_pg07_60() {  # AC18 — cấu hình Redis
  chk "appendonly"        "$($RDS config get appendonly       | tail -1 | tr -d '\r')" yes
  chk "appendfsync"       "$($RDS config get appendfsync      | tail -1 | tr -d '\r')" everysec
  chk "maxmemory-policy"  "$($RDS config get maxmemory-policy | tail -1 | tr -d '\r')" noeviction
}
tc_pg07_61() {  # AC18 — volume tên cố định cho postgres, redis, minio, caddy
  local v d; v=$($C config --volumes 2>/dev/null | sort | paste -sd' ' -); d=$(docker volume ls --format '{{.Name}}')
  echo "    (compose volumes: $v)"
  chk_ge "volume postgres" "$(printf '%s' "$v" | grep -c 'postgres')" 1
  chk_ge "volume redis"    "$(printf '%s' "$v" | grep -c 'redis')" 1
  chk_ge "volume minio"    "$(printf '%s' "$v" | grep -c 'minio')" 1
  chk_ge "volume caddy_data" "$(printf '%s' "$v" | grep -c 'caddy_data')" 1
  chk_ge "volume thật của project edupilot" "$(printf '%s\n' "$d" | grep -c '^edupilot_')" 4
}
tc_pg07_62() {  # AC18 — AOF giữ dữ liệu qua restart redis
  local val
  $RDS set ep:qc:pg07:aof v1 EX 600 >/dev/null 2>&1
  sleep 2                                               # appendfsync everysec
  $C restart redis >/dev/null 2>&1
  _pg07_redis_wait; wait_ready 90                       # khôi phục trước khi chấm
  val=$($RDS get ep:qc:pg07:aof 2>/dev/null | tr -d '\r')
  $RDS del ep:qc:pg07:aof >/dev/null 2>&1
  chk "giá trị khoá sau khi restart redis" "$val" v1
}

# ---------- AC19: cổng PG (tham chiếu GATE) ----------
tc_pg07_63() {  # AC19 — cổng nghiệm thu PG chạy trọn: do GATE-PG chấm (TC-GATE-01…13, mỗi TC một lệnh của AC19)
  manual "AC19 = TC-GATE-01…13 của tc-GATE-PG.md đều PASS (bash docs/sprints/2/qc/scripts/gate-pg.sh 01 02 03 04 05 06 07 08 09 10 11 12 13); chép kết luận vào report-US-PG-07 mục AC19"
}

# ---------- AC20: route thử chỉ ở target thử ----------
tc_pg07_64() {  # AC20 — Dockerfile có 2 target thử, đúng 2 dòng testroutes, target mặc định không tag
  chk "số 'FROM … AS (gateway-test|worker-test)'" "$(grep -cE '^FROM .* AS (gateway-test|worker-test)$' backend-go/Dockerfile)" 2
  chk "số dòng chứa 'testroutes'" "$(grep -c 'testroutes' backend-go/Dockerfile)" 2
  chk "số 'FROM … AS (gateway|worker)'" "$(grep -cE '^FROM .* AS (gateway|worker)$' backend-go/Dockerfile)" 2
}
tc_pg07_65() {  # AC20 — compose local không tham chiếu target/image -test
  chk "số lần 'gateway-test|worker-test' trong docker-compose.local.yml" "$(grep -c 'gateway-test\|worker-test' docker-compose.local.yml)" 0
}
tc_pg07_66() {  # AC20 — docker-compose.test.yml là override, chỉ đổi image/target/APP_ENV
  local j
  chk "danh sách service khi có override" "$($CT config --services 2>/dev/null | sort | paste -sd' ' -)" "$SVC10"
  chk "số dòng 'image: edupilot-(gateway|worker)-test'" "$($CT config 2>/dev/null | grep -cE 'image: edupilot-(gateway|worker)-test')" 2
  chk "số dòng 'target: (gateway|worker)-test'"         "$($CT config 2>/dev/null | grep -cE 'target: (gateway|worker)-test')" 2
  j=$(_pg07_ctcfgjson)
  chk_re "APP_ENV của gateway ở override" "$(printf '%s' "$j" | jq -r '.services.gateway.environment | tostring')" 'APP_ENV[^,]*test'
  chk_re "APP_ENV của worker ở override"  "$(printf '%s' "$j" | jq -r '.services.worker.environment  | tostring')" 'APP_ENV[^,]*test'
}
tc_pg07_67() {  # AC20 — tmode: /_test/whoami 200; dmode: 404 (kết thúc ở dmode)
  local c1 c2
  tmode > "$QC_OUT/tc-PG07-67-tmode.log" 2>&1; wait_ready 120
  c1=$(code -H "Authorization: Bearer $(tok STUDENT $U1)" $GW/api/v1/_test/whoami)
  dmode > "$QC_OUT/tc-PG07-67-dmode.log" 2>&1; wait_ready 120      # trả về trạng thái chuẩn
  c2=$(code -H "Authorization: Bearer $(tok STUDENT $U1)" $GW/api/v1/_test/whoami)
  chk "whoami ở image *-test" "$c1" 200
  chk "whoami ở image mặc định" "$c2" 404
}
tc_pg07_68() {  # AC20 — image thử cũng < 40.000.000 byte
  local t sz
  ensure_mode test || fail_tc "không dựng được chế độ test"
  for t in gateway-test worker-test; do
    sz=$(docker image inspect -f '{{.Size}}' edupilot-$t 2>/dev/null)
    echo "    (edupilot-$t = ${sz:-?} byte)"
    chk_le "kích thước edupilot-$t (byte)" "${sz:-999999999}" 39999999
  done
  ensure_mode default >/dev/null 2>&1                   # trả về trạng thái chuẩn
}
tc_pg07_69() {  # AC20 — CI không đẩy image -test đi đâu
  local f=.github/workflows/ci.yml
  chk "số dòng đẩy image (docker push / --push / build-push-action)" "$(grep -cE 'docker +push|--push|docker/build-push-action' "$f")" 0
  chk "số dòng vừa nhắc '-test' vừa nhắc push" "$(grep -nE '(gateway|worker)-test' "$f" | grep -ci push)" 0
}

main 07 "$@"
