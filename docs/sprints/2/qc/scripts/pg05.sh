#!/usr/bin/env bash
# QC US-PG-05 — SSE: khung, heartbeat, giới hạn kết nối, Last-Event-ID, chéo bản, tắt gateway, Caddy, Redis chết.
# Nguồn: docs/specs/FEAT-pg-foundation/US.md v1.1 (US-PG-05 AC1…AC15); SRS mục 3.3, 3.4, 5.6, 6.1, 6.2, 6.3, 6.5, 6.8, 8.1, 8.2.
# Chạy ở gốc worktree sau `pnpm dev`:  bash docs/sprints/2/qc/scripts/pg05.sh [MM …|--list]
# TC chậm (≥ 2 phút) mặc định chỉ in MANUAL; chạy thật bằng:  QC_SLOW=1 bash docs/sprints/2/qc/scripts/pg05.sh 25 27
. "$(dirname "$0")/lib.sh"
. "$(dirname "$0")/sse-reconnect.sh"

EVURL="$GW/api/v1/events"
JOBURL="$GW/api/v1/_test/jobs"
uid() { printf '00000000-0000-7000-8000-%012d\n' "$1"; }          # uuid thử riêng cho từng TC nặng (tránh đụng U1/U2)
gw_host() { docker inspect -f '{{.Config.Hostname}}' "$1" 2>/dev/null; }   # = X-Instance-Id mặc định (SRS 6.5)
gw_ip()   { docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$1" 2>/dev/null; }
hold_direct() {  # hold_direct <ip gateway> <token> <giây> → PID nền giữ MỘT stream SSE thẳng vào gateway (không qua Caddy)
  $C exec -T postgres bash -c 'exec 3<>/dev/tcp/$0/8080; printf "GET /api/v1/events HTTP/1.1\r\nHost: g\r\nAuthorization: Bearer $1\r\nAccept: text/event-stream\r\n\r\n" >&3; timeout $2 cat <&3' "$1" "$2" "$3" >/dev/null 2>&1 &
  echo $!; }
direct_sse() {  # direct_sse <ip gateway> <token> [giây] → phản hồi thô của GET /api/v1/events thẳng vào gateway
  rawhttp "$1" 8080 "GET /api/v1/events HTTP/1.1\r\nHost: g\r\nAuthorization: Bearer $2\r\n\r\n" "${3:-3}" | tr -d '\r'; }
sse_open_code() {  # sse_open_code <token> [cờ curl thêm…] → mã HTTP khi mở stream qua Caddy (000 = vẫn đang stream)
  local t=$1; shift; curl -sk -N --max-time 3 -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $t" "$@" "$EVURL"; }
zcard() { $RDS zcard "ep:sse:conn:$1" 2>/dev/null | tr -d '\r'; }
numsub() { $RDS pubsub numsub "ep:sse:ch:$1" 2>/dev/null | tr -d '\r' | awk 'NR==2'; }
wait_slots_free() {  # wait_slots_free <uid> <giây> → 0 khi ZCARD = 0 và NUMSUB = 0
  local u=$1 max=$2 i=0
  while [ "$i" -lt $((max * 5)) ]; do
    [ "$(zcard "$u")" = 0 ] && [ "$(numsub "$u")" = 0 ] && return 0; sleep 0.2; i=$((i + 1)); done; return 1; }

# =============================== AC1 — mở stream, header, độ trễ ===============================
tc_pg05_01() {  # AC1 — thứ tự byte: `retry: 3000` rồi `event: ready` không `id:` (SRS 6.8)
  ensure_mode test
  local t f; t=$(tok STUDENT "$U1"); f=$QC_OUT/tc-pg05-01.sse
  curl -sk -N --max-time 2 -H "Authorization: Bearer $t" "$EVURL" > "$f" 2>/dev/null
  chk "13 byte đầu (retry + dòng trống)" "$(dd if="$f" bs=1 count=13 2>/dev/null | tr '\n' '@')" 'retry: 3000@@'
  chk "dòng 3 = sự kiện trạng thái đầu" "$(awk 'NR==3' "$f")" 'event: ready'
  chk_re "dòng 4 = data JSON một dòng" "$(awk 'NR==4' "$f")" '^data: \{.*\}$'
  chk "dòng 5 rỗng (kết thúc khối)" "$(awk 'NR==5' "$f")" ''
  chk "khối ready KHÔNG có dòng id:" "$(awk 'NR<=5' "$f" | grep -c '^id:')" 0
  chk_ok "data của ready là JSON hợp lệ" jq -e . <<< "$(sse_data_of "$f" ready)"
}
tc_pg05_02() {  # AC1 + SRS 6.8 — header phản hồi qua Caddy (chỉ đọc bằng -D-)
  ensure_mode test
  local t h h1; t=$(tok STUDENT "$U1")
  h=$(curl -sk -N --max-time 2 -D- -o /dev/null -H "Authorization: Bearer $t" "$EVURL" 2>/dev/null | tr -d '\r')
  chk    "mã HTTP" "$(printf '%s\n' "$h" | awk 'NR==1{print $2}')" 200
  chk_re "Content-Type" "$(printf '%s\n' "$h" | hval content-type)" '^text/event-stream'
  chk    "Cache-Control" "$(printf '%s\n' "$h" | hval cache-control)" 'no-cache, no-transform'
  chk    "X-Accel-Buffering" "$(printf '%s\n' "$h" | hval x-accel-buffering)" 'no'
  chk_ne "X-Instance-Id có mặt" "$(printf '%s\n' "$h" | hval x-instance-id)" ''
  chk    "không có Content-Encoding" "$(printf '%s\n' "$h" | grep -ci '^content-encoding')" 0
  h1=$(curl -sk -N --http1.1 --max-time 2 -D- -o /dev/null -H "Authorization: Bearer $t" "$EVURL" 2>/dev/null | tr -d '\r')
  chk_re "Connection (HTTP/1.1)" "$(printf '%s\n' "$h1" | hval connection)" 'keep-alive'
}
tc_pg05_03() {  # AC1 — p95 độ trễ byte đầu ≤ 300 ms trên 50 kết nối (5 lô × 10 song song; 25 user × 2 kết nối)
  ensure_mode test
  local d=$QC_OUT/tc-pg05-03 b i r u t n=0 cnt p95
  rm -rf "$d"; mkdir -p "$d"
  for b in 0 1 2 3 4; do
    for i in 0 1 2 3 4; do
      u=$(uid $((101 + b * 5 + i))); t=$(tok STUDENT "$u")
      for r in 1 2; do
        n=$((n + 1))
        curl -sk -N --max-time 3 -o /dev/null -w '%{time_starttransfer}\n' \
             -H "Authorization: Bearer $t" "$EVURL" > "$d/$n.ttfb" 2>/dev/null &
      done
    done
    wait; sleep 1
  done
  cat "$d"/*.ttfb 2>/dev/null | grep -E '^[0-9]' | sort -n > "$d/all"
  cnt=$(grep -c . "$d/all"); p95=$(awk -v n="$cnt" 'NR==int((n*95+99)/100){printf "%d", $1*1000}' "$d/all")
  echo "    (p50=$(awk -v n="$cnt" 'NR==int(n/2+0.5){printf "%d", $1*1000}' "$d/all") ms · max=$(awk 'END{printf "%d", $1*1000}' "$d/all") ms · số liệu: $d/all)"
  chk    "số kết nối đo được" "$cnt" 50
  chk_le "p95 time_starttransfer (ms)" "${p95:-99999}" 300
}
tc_pg05_04() {  # AC1 — data của `ready`: connection_id duy nhất mỗi kết nối, server_time là mốc UTC hiện tại
  ensure_mode test
  local t f1 f2 c1 c2 st; t=$(tok STUDENT "$U1"); f1=$QC_OUT/tc-pg05-04a.sse; f2=$QC_OUT/tc-pg05-04b.sse
  curl -sk -N --max-time 2 -H "Authorization: Bearer $t" "$EVURL" > "$f1" 2>/dev/null
  curl -sk -N --max-time 2 -H "Authorization: Bearer $t" "$EVURL" > "$f2" 2>/dev/null
  c1=$(sse_data_of "$f1" ready | jq -r '.connection_id // empty'); c2=$(sse_data_of "$f2" ready | jq -r '.connection_id // empty')
  st=$(sse_data_of "$f1" ready | jq -r '.server_time // empty')
  chk_ne "connection_id kết nối 1 không rỗng" "$c1" ''
  chk_ne "connection_id khác nhau giữa 2 kết nối" "$c1" "$c2"
  chk_re "server_time ISO-8601 UTC" "$st" '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?Z$'
  chk_re "server_time đúng phút hiện tại (UTC)" "$st" "^($(date -u +%Y-%m-%dT%H:%M)|$(date -u -v-1M +%Y-%m-%dT%H:%M 2>/dev/null || date -u +%Y-%m-%dT%H:%M))"
}

# =============================== AC2 — khung sự kiện, id, kiểm tra Publish ===============================
tc_pg05_05() {  # AC2 — 20 sự kiện: mỗi khối đúng `id:`/`event:`/`data:` + dòng trống, data là JSON một dòng
  ensure_mode test
  local u t f; u=$(uid 120); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-05.sse; sse_clean "$u"
  sse_bg "$f" 25 "$t"; sse_wait_line "$f" '^event: ready' 10
  sse_burst 20 test.frame "$t" 5; sse_wait_ids "$f" 20 20; sse_kill
  chk "số sự kiện nhận được" "$(sse_nid "$f")" 20
  chk "số khối đúng định dạng (id/event/data/dòng trống)" "$(sse_ok_frames "$f")" 20
  chk "số sự kiện type = test.frame" "$(sse_ntype "$f" test.frame)" 20
  chk "số dòng data KHÔNG phải JSON" "$(sse_datas "$f" | while read -r l; do printf '%s' "$l" | jq -e . >/dev/null 2>&1 || echo x; done | grep -c x)" 0
}
tc_pg05_06() {  # AC2 — id = id Redis Stream `<ms>-<seq>`, TĂNG NGHIÊM NGẶT theo (ms, seq)
  ensure_mode test
  local u t f; u=$(uid 121); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-06.sse; sse_clean "$u"
  sse_bg "$f" 25 "$t"; sse_wait_line "$f" '^event: ready' 10
  sse_burst 20 test.mono "$t" 5; sse_wait_ids "$f" 20 20; sse_kill
  chk "số id sai định dạng ^\\d+-\\d+$" "$(sse_ids "$f" | grep -cvE '^[0-9]+-[0-9]+$')" 0
  chk "id tăng nghiêm ngặt" "$(sse_ids_monotonic "$f")" ok
  chk "số id trùng" "$(sse_ids "$f" | sort | uniq -d | grep -c .)" 0
}
tc_pg05_07() {  # AC2 (biên) — type hợp lệ 1 ký tự và 64 ký tự (^[a-z][a-z0-9_.]{0,63}$) được chấp nhận
  ensure_mode test
  local u t f t64; u=$(uid 122); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-07.sse; sse_clean "$u"
  t64=$(printf 'a%.0s' $(seq 1 64))
  sse_bg "$f" 20 "$t"; sse_wait_line "$f" '^event: ready' 10
  chk_re "mã khi type 1 ký tự 'a'"   "$(sse_pub a "{\"n\":1}" "$t")" '^2[0-9][0-9]$'
  chk_re "mã khi type 64 ký tự"      "$(sse_pub "$t64" "{\"n\":2}" "$t")" '^2[0-9][0-9]$'
  sse_wait_ids "$f" 2 10; sse_kill
  chk "số sự kiện nhận được" "$(sse_nid "$f")" 2
  chk "type 1 ký tự tới nơi" "$(sse_ntype "$f" a)" 1
  chk "type 64 ký tự tới nơi" "$(sse_ntype "$f" "$t64")" 1
}
tc_pg05_08() {  # AC2 (nhánh lỗi) — type sai → `Publish` từ chối (ErrInvalidEventType), không sự kiện nào ra stream
  ensure_mode test
  local u t f t65 c b; u=$(uid 123); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-08.sse; sse_clean "$u"
  t65=$(printf 'a%.0s' $(seq 1 65))
  sse_bg "$f" 20 "$t"; sse_wait_line "$f" '^event: ready' 10
  for v in 'Bad Type' 'Type' '1abc' '' "$t65" 'test..x#'; do
    c=$(sse_pub "$v" '{"n":1}' "$t"); b=$(sse_pub_body "$v" '{"n":1}' "$t" | jq -r '.code // empty')
    chk_re "type [$v] → mã 4xx" "$c" '^4[0-9][0-9]$'
    chk_re "type [$v] → code lỗi theo SRS 6.1" "$b" '^(VALIDATION_FAILED|BAD_REQUEST)$'
  done
  sleep 1; sse_kill
  chk "số sự kiện lọt vào stream" "$(sse_nid "$f")" 0
}
tc_pg05_09() {  # AC2 (biên) — data > 64 KiB bị từ chối (ErrEventTooLarge); data < 64 KiB đi qua
  ensure_mode test
  local u t f big small c; u=$(uid 124); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-09.sse; sse_clean "$u"
  small=$(head -c 60000 /dev/zero | tr '\0' a); big=$(head -c 66000 /dev/zero | tr '\0' a)
  sse_bg "$f" 25 "$t"; sse_wait_line "$f" '^event: ready' 10
  chk_re "data 60 KiB → chấp nhận" "$(sse_pub test.size "{\"s\":\"$small\"}" "$t")" '^2[0-9][0-9]$'
  c=$(sse_pub test.size "{\"s\":\"$big\"}" "$t")
  chk_re "data ≈ 64,5 KiB → từ chối (4xx)" "$c" '^4[0-9][0-9]$'
  chk_re "code lỗi theo SRS 6.1" "$(sse_pub_body test.size "{\"s\":\"$big\"}" "$t" | jq -r '.code // empty')" '^(VALIDATION_FAILED|PAYLOAD_TOO_LARGE|BAD_REQUEST)$'
  sse_wait_ids "$f" 1 10; sleep 1; sse_kill
  chk "số sự kiện nhận được (chỉ cái 60 KiB)" "$(sse_nid "$f")" 1
}
tc_pg05_10() {  # AC2 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_Framing|TestPublish_Validation|TestSSE_IDsMonotonic'
}

# =============================== AC3 — heartbeat ===============================
tc_pg05_11() {  # AC3 — đúng lệnh trong AC: 30 s qua Caddy → đúng 1 dòng `: hb`
  ensure_mode test
  local t n; t=$(tok STUDENT "$U1")
  n=$(curl -sk -N --max-time 30 -H "Authorization: Bearer $t" "$EVURL" 2>/dev/null | grep -c '^: hb')
  chk "số dòng ': hb' trong 30 s (SSE_HEARTBEAT = 25 s)" "$n" 1
}
tc_pg05_12() {  # AC3 — nhịp ĐỀU, không gộp lô: 2 nhịp trong 55 s, khoảng cách 25 s ± 2 s; heartbeat không mang id
  ensure_mode test
  local u t f a b d1 d2; u=$(uid 125); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-12.sse
  sse_bg_ts "$f" 56 "$t"; sse_ts_wait
  sse_plain "$f" > "$f.plain"
  chk_ge "số nhịp ': hb' trong 55 s" "$(grep -c '^: hb' "$f.plain")" 2
  a=$(sse_ts_of "$f" '^event: ready'); b=$(sse_ts_of "$f" '^: hb')
  d1=$(sse_dms "$a" "$b")
  d2=$(sse_dms "$b" "$(awk '{l=$0; sub(/^[0-9.]+ /,"",l); if (l ~ /^: hb/) {c++; if (c==2) {print $1; exit}}}' "$f")")
  chk_ge "nhịp 1 sau ready (ms)" "$d1" 23000; chk_le "nhịp 1 sau ready (ms)" "$d1" 27000
  chk_ge "khoảng cách nhịp 1→2 (ms)" "$d2" 23000; chk_le "khoảng cách nhịp 1→2 (ms)" "$d2" 27000
  chk "stream nhàn rỗi không có id:" "$(grep -c '^id:' "$f.plain")" 0
}
tc_pg05_13() {  # AC3 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_Heartbeat|TestConfig_SSEDefaults'
}

# =============================== AC4 — client ngắt, không rò ===============================
tc_pg05_14() {  # AC4 — 100 chu kỳ mở/ngắt ngẫu nhiên (0,05–1,5 s) → ZCARD = 0 và PUBSUB NUMSUB = 0 trong ≤ 2 s
  ensure_mode test
  local u t i m p1 p2; u=$(uid 140); t=$(tok STUDENT "$u"); sse_clean "$u"
  i=1
  while [ "$i" -le 100 ]; do
    m=$(awk -v s="$RANDOM" 'BEGIN{srand(s); printf "%.2f", 0.05 + rand()*1.45}')
    curl -sk -N --max-time "$m" -o /dev/null -H "Authorization: Bearer $t" "$EVURL" 2>/dev/null & p1=$!
    m=$(awk -v s="$((RANDOM + 7))" 'BEGIN{srand(s); printf "%.2f", 0.05 + rand()*1.45}')
    curl -sk -N --max-time "$m" -o /dev/null -H "Authorization: Bearer $t" "$EVURL" 2>/dev/null & p2=$!
    wait $p1 $p2 2>/dev/null
    i=$((i + 2))
  done
  chk_ok "dọn sạch trong ≤ 2 s sau chu kỳ cuối" wait_slots_free "$u" 2
  chk "ZCARD ep:sse:conn:<uid>" "$(zcard "$u")" 0
  chk "PUBSUB NUMSUB ep:sse:ch:<uid>" "$(numsub "$u")" 0
}
tc_pg05_15() {  # AC4 — ngắt bằng kill -9 ĐÚNG LÚC đang nhận sự kiện → vẫn dọn sạch
  ensure_mode test
  local u t f; u=$(uid 141); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-15.sse; sse_clean "$u"
  sse_bg "$f" 30 "$t"; sse_wait_line "$f" '^event: ready' 10
  sse_burst 30 test.cut "$t" 10 &
  sse_wait_ids "$f" 3 10; sse_kill; wait 2>/dev/null
  chk_ok "ZCARD + NUMSUB về 0 trong ≤ 2 s" wait_slots_free "$u" 2
  chk "ZCARD ep:sse:conn:<uid>" "$(zcard "$u")" 0
}
tc_pg05_16() {  # AC4 — không dòng log lỗi nào sinh ra vì client ngắt
  ensure_mode test
  local u t i n; u=$(uid 142); t=$(tok STUDENT "$u")
  i=1; while [ "$i" -le 10 ]; do
    curl -sk -N --max-time 0.3 -o /dev/null -H "Authorization: Bearer $t" "$EVURL" 2>/dev/null; i=$((i + 1)); done
  sleep 2
  $C logs --no-log-prefix --since 60s gateway > "$QC_OUT/tc-pg05-16.log" 2>/dev/null
  n=$(grep -ci '"level":"error"' "$QC_OUT/tc-pg05-16.log")
  chk "số dòng log mức error trong 60 s qua" "$n" 0
}
tc_pg05_17() {  # AC4 — test Go (goroutine về mức ban đầu — chỉ test Go đo được); `gt` của lib.sh luôn chạy với -race
  gt ./internal/httpapi/sse 'TestSSE_NoLeakOnClientDisconnect'
}

# =============================== AC5 — tối đa 2 kết nối / người ===============================
tc_pg05_18() {  # AC5 — đúng lệnh trong AC: 3 stream của U1 → hai `000` (còn đang stream) rồi một `429`
  ensure_mode test
  local t o; t=$(tok STUDENT "$U1"); sse_clean "$U1"
  o=$(for i in 1 2 3; do
        curl -sk -N -o /dev/null -m 5 -w '%{http_code}\n' -H "Authorization: Bearer $t" "$EVURL" & sleep 0.5
      done; wait)
  echo "    (mã thu được: $(printf '%s' "$o" | tr '\n' ' '))"
  chk "số mã 000 (hai stream được giữ)" "$(printf '%s\n' "$o" | grep -c '^000$')" 2
  chk "số mã 429 (stream thứ 3)"        "$(printf '%s\n' "$o" | grep -c '^429$')" 1
}
tc_pg05_19() {  # AC5 — thân 429 là JSON `SSE_LIMIT_REACHED` + `Retry-After`, TRƯỚC khi mở stream
  ensure_mode test
  local t f1 f2 o h b; t=$(tok STUDENT "$U1"); sse_clean "$U1"
  f1=$QC_OUT/tc-pg05-19a.sse; f2=$QC_OUT/tc-pg05-19b.sse
  sse_bg "$f1" 12 "$t"; sse_wait_line "$f1" '^event: ready' 10
  curl -sk -N --max-time 12 -H "Authorization: Bearer $t" "$EVURL" > "$f2" 2>/dev/null & local p2=$!
  sse_wait_line "$f2" '^event: ready' 10
  o=$(curl -sk --max-time 5 -D "$QC_OUT/tc-pg05-19.hdr" -H "Authorization: Bearer $t" -w '\n%{http_code}' "$EVURL" 2>/dev/null)
  h=$(tr -d '\r' < "$QC_OUT/tc-pg05-19.hdr"); b=$(printf '%s\n' "$o" | sed '$d')
  kill -9 "$p2" 2>/dev/null; sse_kill; wait 2>/dev/null
  chk    "mã HTTP stream thứ 3" "$(printf '%s\n' "$o" | awk 'END{print}')" 429
  chk    "code trong thân" "$(printf '%s' "$b" | jq -r '.code // empty')" SSE_LIMIT_REACHED
  chk_re "trace_id 32 hex" "$(printf '%s' "$b" | jq -r '.trace_id // empty')" '^[0-9a-f]{32}$'
  chk_re "header Retry-After (giây)" "$(printf '%s\n' "$h" | hval retry-after)" '^[0-9]+$'
  chk_re "Content-Type là JSON (không phải event-stream)" "$(printf '%s\n' "$h" | hval content-type)" '^application/json'
  chk    "không mở stream (không có dòng retry:/event:)" "$(printf '%s' "$b" | grep -cE '^(retry|event):')" 0
}
tc_pg05_20() {  # AC5 — giới hạn tính theo người dùng: U1 đầy nhưng U2 vẫn mở được
  ensure_mode test
  local t1 t2 f1 f2 c; t1=$(tok STUDENT "$U1"); t2=$(tok STUDENT "$U2"); sse_clean "$U1"; sse_clean "$U2"
  f1=$QC_OUT/tc-pg05-20a.sse; f2=$QC_OUT/tc-pg05-20b.sse
  sse_bg "$f1" 10 "$t1"; local pa=$SSE_PID; sse_wait_line "$f1" '^event: ready' 10
  sse_bg "$f2" 10 "$t1"; local pb=$SSE_PID; sse_wait_line "$f2" '^event: ready' 10
  c=$(sse_open_code "$t2")
  local cu1; cu1=$(sse_open_code "$t1")
  kill -9 "$pa" "$pb" 2>/dev/null; wait 2>/dev/null
  chk "mã của U2 (đang stream → 000)" "$c" 000
  chk "mã của U1 (stream thứ 3 → 429)" "$cu1" 429
}
tc_pg05_21() {  # AC5 — đóng một stream của U1 → stream mới được nhận trong ≤ 2 s
  ensure_mode test
  local t f1 f2 i ok=0; t=$(tok STUDENT "$U1"); sse_clean "$U1"
  f1=$QC_OUT/tc-pg05-21a.sse; f2=$QC_OUT/tc-pg05-21b.sse
  sse_bg "$f1" 15 "$t"; local pa=$SSE_PID; sse_wait_line "$f1" '^event: ready' 10
  sse_bg "$f2" 15 "$t"; local pb=$SSE_PID; sse_wait_line "$f2" '^event: ready' 10
  chk "trước khi đóng: stream thứ 3" "$(sse_open_code "$t")" 429
  kill -9 "$pa" 2>/dev/null; wait "$pa" 2>/dev/null
  i=0; while [ "$i" -lt 10 ]; do
    [ "$(sse_open_code "$t")" = 000 ] && { ok=1; break; }; sleep 0.2; i=$((i + 1)); done
  kill -9 "$pb" 2>/dev/null; wait 2>/dev/null
  chk "sau khi đóng 1 stream: mở được trong ≤ 2 s" "$ok" 1
}
tc_pg05_22() {  # AC5 — đếm CHUNG giữa hai bản: A giữ 2 stream → stream thứ 3 vào B nhận 429
  ensure_mode test
  local u t ids a b ipa ipb h1 h2 st; u=$(uid 143); t=$(tok STUDENT "$u"); sse_clean "$u"
  ids=$(gw_ids); a=$(printf '%s\n' "$ids" | awk 'NR==1'); b=$(printf '%s\n' "$ids" | awk 'NR==2')
  chk_ne "có hai bản gateway" "$b" ''
  ipa=$(gw_ip "$a"); ipb=$(gw_ip "$b")
  h1=$(hold_direct "$ipa" "$t" 12); h2=$(hold_direct "$ipa" "$t" 12); sleep 2
  st=$(direct_sse "$ipb" "$t" 4 | awk 'NR==1{print $2}')
  kill "$h1" "$h2" 2>/dev/null; wait 2>/dev/null
  chk    "ZCARD sau khi A giữ 2 kết nối" "$(zcard "$u")" 2
  chk    "mã của stream thứ 3 mở thẳng vào bản B" "$st" 429
  wait_slots_free "$u" 20
}
tc_pg05_23() {  # AC5 / SRS 5.6 — ZSET ep:sse:conn:<uid>: 1 member / kết nối, score = hạn ms ≈ now + SSE_CONN_TTL (150 s)
  ensure_mode test
  local u t f now sc d; u=$(uid 144); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-23.sse; sse_clean "$u"
  sse_bg "$f" 12 "$t"; sse_wait_line "$f" '^event: ready' 10
  now=$(now_ms); sc=$($RDS zrange "ep:sse:conn:$u" 0 -1 WITHSCORES 2>/dev/null | tr -d '\r' | awk 'NR==2{printf "%d", $1}')
  sse_kill
  chk    "số member (1 kết nối)" "$(zcard "$u")" 1
  d=$((${sc:-0} - now))
  chk_ge "score − now (ms) ≈ SSE_CONN_TTL" "$d" 140000
  chk_le "score − now (ms) ≈ SSE_CONN_TTL" "$d" 152000
  wait_slots_free "$u" 5
}
tc_pg05_24() {  # AC5 — gateway chết đột ngột (kill -9) → chỗ tự hết sau SSE_CONN_TTL (bản rút gọn: 5 s)
  ensure_mode test
  local u t ids a ipa h1 h2 z0 i free=0
  u=$(uid 145); sse_clean "$u"
  gw_env test SSE_CONN_TTL=5s
  t=$(tok STUDENT "$u")
  ids=$(gw_ids); a=$(printf '%s\n' "$ids" | awk 'NR==1'); ipa=$(gw_ip "$a")
  h1=$(hold_direct "$ipa" "$t" 30); h2=$(hold_direct "$ipa" "$t" 30); sleep 2
  z0=$(zcard "$u")
  docker kill -s KILL "$a" >/dev/null 2>&1
  kill "$h1" "$h2" 2>/dev/null
  i=0; while [ "$i" -lt 12 ]; do [ "$(zcard "$u")" = 0 ] && { free=1; break; }; sleep 1; i=$((i + 1)); done
  gw_restore test
  chk "ZCARD khi bản A giữ 2 kết nối" "$z0" 2
  chk "ZCARD về 0 trong ≤ 12 s với SSE_CONN_TTL=5 (1 = đúng)" "$free" 1
  echo "    (nếu ZCARD vẫn = 2: hoặc chỗ không hết hạn, hoặc compose không chuyển tiếp SSE_CONN_TTL — xem 'Điểm khó kiểm')"
}
tc_pg05_25() {  # AC5 — TAY / CHẬM: SSE_CONN_TTL mặc định 150 s sau kill -9 (QC_SLOW=1 để chạy thật)
  if [ "${QC_SLOW:-0}" != 1 ]; then
    manual "chờ 150 s: kill -9 một bản gateway đang giữ 2 stream, đo lúc ZCARD ep:sse:conn:<uid> về 0 (xem 'Bước tay TC-PG05-25')"; return 0; fi
  ensure_mode test
  local u t ids a ipa h1 h2 t0 t1 i; u=$(uid 146); t=$(tok STUDENT "$u"); sse_clean "$u"
  ids=$(gw_ids); a=$(printf '%s\n' "$ids" | awk 'NR==1'); ipa=$(gw_ip "$a")
  h1=$(hold_direct "$ipa" "$t" 200); h2=$(hold_direct "$ipa" "$t" 200); sleep 2
  t0=$(now_ms); docker kill -s KILL "$a" >/dev/null 2>&1; kill "$h1" "$h2" 2>/dev/null
  t1=0; i=0
  while [ "$i" -lt 200 ]; do [ "$(zcard "$u")" = 0 ] && { t1=$(now_ms); break; }; sleep 1; i=$((i + 1)); done
  gw_restore test
  chk_ge "thời gian chỗ hết hạn (ms)" "$(( ${t1:-0} == 0 ? 0 : t1 - t0 ))" 140000
  chk_le "thời gian chỗ hết hạn (ms)" "$(( ${t1:-0} == 0 ? 999999 : t1 - t0 ))" 160000
}
tc_pg05_26() {  # AC5 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_MaxTwoPerUser|TestSSE_LimitSharedAcrossInstances|TestSSE_SlotExpiresAfterCrash'
}

# =============================== AC6 — thời lượng tối đa, token hết hạn ===============================
tc_pg05_27() {  # AC6 — TAY / CHẬM (2 phút): SSE_MAX_DURATION mặc định 120 s → đúng 1 `reason":"max_duration`
  if [ "${QC_SLOW:-0}" != 1 ]; then
    manual "chờ 130 s theo lệnh của AC6 (xem 'Bước tay TC-PG05-27')"; return 0; fi
  ensure_mode test
  local t f n d; t=$(tok STUDENT "$U1" --ttl 10m); f=$QC_OUT/tc-pg05-27.sse
  sse_bg_ts "$f" 140 "$t"; sse_ts_wait; sse_plain "$f" > "$f.plain"
  n=$(grep -c 'reason":"max_duration' "$f.plain")
  d=$(sse_dms "$(sse_ts_of "$f" '^event: ready')" "$(sse_ts_of "$f" '^event: reconnect')")
  chk    "số lần 'reason\":\"max_duration'" "$n" 1
  chk_ge "ready → reconnect (ms)" "$d" 115000
  chk_le "ready → reconnect (ms)" "$d" 130000
  chk_le "reconnect → đóng kết nối (ms)" "$(sse_dms "$(sse_ts_of "$f" '^event: reconnect')" "$(sse_ts_of "$f" '__END__')")" 2000
}
tc_pg05_28() {  # AC6 (rút gọn) — SSE_MAX_DURATION=5s → `event: reconnect` `{"reason":"max_duration"}` rồi đóng ≈ 5 s
  ensure_mode test
  local t f d r
  gw_env test SSE_MAX_DURATION=5s
  t=$(tok STUDENT "$U1" --ttl 10m); f=$QC_OUT/tc-pg05-28.sse
  sse_bg_ts "$f" 30 "$t"; sse_ts_wait; sse_plain "$f" > "$f.plain"
  d=$(sse_dms "$(sse_ts_of "$f" '^event: ready')" "$(sse_ts_of "$f" '^event: reconnect')")
  r=$(sse_data_of "$f.plain" reconnect | jq -r '.reason // empty')
  gw_restore test
  chk    "reason của event: reconnect" "$r" max_duration
  chk    "reconnect không mang id:" "$(grep -c '^id:' "$f.plain")" 0
  chk_ge "ready → reconnect (ms)" "$d" 3500
  chk_le "ready → reconnect (ms)" "$d" 9000
  chk_le "reconnect → đóng (ms)" "$(sse_dms "$(sse_ts_of "$f" '^event: reconnect')" "$(sse_ts_of "$f" '__END__')")" 2000
}
tc_pg05_29() {  # AC6 — token hết hạn GIỮA stream → `reconnect` `{"reason":"token_expired"}` ≈ lúc exp (leeway 5 s)
  ensure_mode test
  local t f d r; t=$(tok STUDENT "$U1" --ttl 8s); f=$QC_OUT/tc-pg05-29.sse
  sse_bg_ts "$f" 40 "$t"; sse_ts_wait; sse_plain "$f" > "$f.plain"
  d=$(sse_dms "$(sse_ts_of "$f" '^event: ready')" "$(sse_ts_of "$f" '^event: reconnect')")
  r=$(sse_data_of "$f.plain" reconnect | jq -r '.reason // empty')
  chk    "reason" "$r" token_expired
  chk_ge "ready → reconnect (ms), exp = 8 s" "$d" 6000
  chk_le "ready → reconnect (ms), exp = 8 s + leeway 5 s" "$d" 15000
  chk_le "reconnect → đóng (ms)" "$(sse_dms "$(sse_ts_of "$f" '^event: reconnect')" "$(sse_ts_of "$f" '__END__')")" 2000
}
tc_pg05_30() {  # AC6 — nối lại bằng token CŨ (đã hết hạn) → 401 TOKEN_EXPIRED, không mở stream
  ensure_mode test
  local t c b h; t=$(tok STUDENT "$U1" --ttl -1m)
  c=$(curl -sk --max-time 5 -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $t" "$EVURL")
  b=$(curl -sk --max-time 5 -H "Authorization: Bearer $t" "$EVURL" | jq -r '.code // empty')
  h=$(hdr -H "Authorization: Bearer $t" "$EVURL")
  chk    "mã HTTP" "$c" 401
  chk    "code" "$b" TOKEN_EXPIRED
  chk_re "WWW-Authenticate" "$(printf '%s\n' "$h" | hval www-authenticate)" 'Bearer realm="edupilot".*error="invalid_token"'
  chk_re "Content-Type không phải event-stream" "$(printf '%s\n' "$h" | hval content-type)" '^application/json'
}
tc_pg05_31() {  # AC6 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_MaxDuration|TestSSE_TokenExpiryMidStream|TestConfig_SSEDefaults'
}

# =============================== AC7 / AC8 — nối lại bằng Last-Event-ID ===============================
tc_pg05_32() {  # AC7 — ngắt rồi nối lại: nhận đúng e6…e13 theo thứ tự, không nhận lại e5, rồi nhận e14
  ensure_mode test
  local u t d ns; u=$(uid 150); t=$(tok STUDENT "$u"); d=$QC_OUT/tc-pg05-32; rm -rf "$d"; sse_clean "$u"
  sse_ac7_flow "$d" "$t"
  ns=$(sse_datas "$d/s2.log" | sed -n 's/.*"n":\([0-9]*\).*/\1/p' | tr '\n' ' ')
  echo "    (phiên 1 nhận: $(sse_datas "$d/s1.log" | sed -n 's/.*"n":\([0-9]*\).*/\1/p' | tr '\n' ' ')· phiên 2 nhận: $ns)"
  chk "phiên 1 nhận 5 sự kiện" "$(sse_nid "$d/s1.log")" 5
  chk "phiên 2 nhận đúng e6…e14 theo thứ tự" "$ns" '6 7 8 9 10 11 12 13 14 '
  chk "phiên 2 KHÔNG nhận lại e5" "$(sse_ids "$d/s2.log" | grep -c "^$(cat "$d/id5")\$")" 0
  chk "tổng số id trùng giữa hai phiên" "$(cat "$d"/s1.log "$d"/s2.log | grep '^id: ' | sort | uniq -d | grep -c .)" 0
  chk "phiên 2 không có event: resync" "$(sse_ntype "$d/s2.log" resync)" 0
}
tc_pg05_33() {  # AC7 (biên) — Last-Event-ID = id MỚI NHẤT → không đọc bù gì, vẫn nhận sự kiện mới
  ensure_mode test
  local u t f1 f2 last; u=$(uid 151); t=$(tok STUDENT "$u"); sse_clean "$u"
  f1=$QC_OUT/tc-pg05-33a.sse; f2=$QC_OUT/tc-pg05-33b.sse
  sse_bg "$f1" 15 "$t"; sse_wait_line "$f1" '^event: ready' 10
  sse_burst 3 test.tail "$t" 1; sse_wait_ids "$f1" 3 10; sse_kill
  last=$(sse_lastid "$f1")
  sse_bg "$f2" 15 "$t" -H "Last-Event-ID: $last"; sse_wait_line "$f2" '^event: ready' 10
  sleep 1
  chk "số sự kiện đọc bù (phải 0)" "$(sse_nid "$f2")" 0
  sse_pub test.tail '{"n":99}' "$t" >/dev/null; sse_wait_ids "$f2" 1 10; sse_kill
  chk "nhận sự kiện mới sau khi nối lại" "$(sse_nid "$f2")" 1
  chk "không resync" "$(sse_ntype "$f2" resync)" 0
}
tc_pg05_34() {  # AC7 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_LastEventID_NoLossNoDup'
}
tc_pg05_35() {  # AC8 — 1.000 sự kiện + 20 lần ngắt/nối ngẫu nhiên → hợp = 1.000 id duy nhất, theo thứ tự, không trùng
  ensure_mode test
  local u t d n ids; u=$(uid 152); t=$(tok STUDENT "$u"); d=$QC_OUT/tc-pg05-35; rm -rf "$d"; sse_clean "$u"
  sse_ac8_flow "$d" "$t" 1000 20
  cat "$d"/s*.log > "$d/all.sse"
  ids=$(sse_ids "$d/all.sse")
  chk "số sự kiện KHÁC NHAU client nhận" "$(sse_ns "$d"/s*.log)" 1000
  chk "số id duy nhất" "$(printf '%s\n' "$ids" | sort -u | grep -c .)" 1000
  chk "số id trùng (mỗi sự kiện đúng 1 lần)" "$(printf '%s\n' "$ids" | sort | uniq -d | grep -c .)" 0
  chk "id tăng nghiêm ngặt theo thứ tự nhận" "$(sse_ids_monotonic "$d/all.sse")" ok
  chk "số lần phải resync" "$(cat "$d"/s*.log | grep -c '^event: resync')" 0
  rl_reset
}
tc_pg05_36() {  # AC8 — test Go có -race (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_ReconnectRace' -race
}

# =============================== AC9 — bộ đệm và resync ===============================
tc_pg05_37() {  # AC9 — Last-Event-ID SAI ĐỊNH DẠNG → `event: resync` `{"reason":"buffer_exceeded"}` đầu tiên, không 4xx/5xx
  ensure_mode test
  local u t v f c; u=$(uid 160); t=$(tok STUDENT "$u"); sse_clean "$u"
  for v in 'abc' '1-' '-1' '1-2-3' ' ' '0x1-0'; do
    f=$QC_OUT/tc-pg05-37-$(printf '%s' "$v" | tr -c 'A-Za-z0-9' '_').sse
    c=$(curl -sk -N --max-time 3 -o "$f" -w '%{http_code}' -H "Authorization: Bearer $t" -H "Last-Event-ID: $v" "$EVURL" 2>/dev/null)
    chk_re "Last-Event-ID [$v] → không 4xx/5xx" "$c" '^(000|200)$'
    chk    "Last-Event-ID [$v] → sự kiện đầu sau ready là resync" "$(sse_first_event "$f")" resync
    chk    "Last-Event-ID [$v] → reason" "$(sse_data_of "$f" resync | jq -r '.reason // empty')" buffer_exceeded
  done
}
tc_pg05_38() {  # AC9 — Last-Event-ID QUÁ CŨ (cũ hơn sự kiện đầu còn trong bộ đệm) → resync, rồi chuyển sang sự kiện mới
  ensure_mode test
  local u t f; u=$(uid 161); t=$(tok STUDENT "$u"); sse_clean "$u"; f=$QC_OUT/tc-pg05-38.sse
  sse_burst 5 test.old "$t" 5        # bộ đệm có sự kiện mới hơn id 1-0
  sse_bg "$f" 15 "$t" -H "Last-Event-ID: 1-0"
  sse_wait_line "$f" '^event: resync' 10
  sse_pub test.old '{"n":99}' "$t" >/dev/null; sse_wait_ids "$f" 1 10; sse_kill
  chk    "sự kiện đầu sau ready" "$(sse_first_event "$f")" resync
  chk    "reason" "$(sse_data_of "$f" resync | jq -r '.reason // empty')" buffer_exceeded
  chk    "resync không mang id:" "$(awk '/^event: resync/{print prev} {prev=$0}' "$f" | grep -c '^id:')" 0
  chk_ge "sau resync vẫn nhận sự kiện mới" "$(sse_nid "$f")" 1
}
tc_pg05_39() {  # AC9 / SRS 5.6 — bộ đệm MAXLEN ~ 1000 và PTTL ≤ 1 h
  ensure_mode test
  local u t xlen pttl; u=$(uid 162); t=$(tok STUDENT "$u"); sse_clean "$u"
  sse_burst 1500 test.buf "$t" 20
  xlen=$($RDS xlen "ep:sse:buf:$u" 2>/dev/null | tr -d '\r'); pttl=$($RDS pttl "ep:sse:buf:$u" 2>/dev/null | tr -d '\r')
  chk_ge "XLEN ep:sse:buf (MAXLEN ~ 1000)" "${xlen:-0}" 1000
  chk_le "XLEN ep:sse:buf (MAXLEN ~ 1000, cho phép xấp xỉ)" "${xlen:-0}" 1100
  chk_ge "PTTL ep:sse:buf > 0 (không khoá vĩnh viễn)" "${pttl:-0}" 1
  chk_le "PTTL ep:sse:buf ≤ 1 h" "${pttl:-0}" 3600000
  rl_reset
}
tc_pg05_40() {  # AC9 / SRS 5.6 — TTL của ep:sse:conn ≤ 150 s và > 0 khi có stream
  ensure_mode test
  local u t f ttl; u=$(uid 163); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-40.sse; sse_clean "$u"
  sse_bg "$f" 10 "$t"; sse_wait_line "$f" '^event: ready' 10
  ttl=$($RDS ttl "ep:sse:conn:$u" 2>/dev/null | tr -d '\r')
  sse_kill
  chk_ge "TTL ep:sse:conn (giây) > 0" "${ttl:-0}" 1
  chk_le "TTL ep:sse:conn (giây) ≤ SSE_CONN_TTL 150" "${ttl:-0}" 150
}
tc_pg05_41() {  # AC9 (biên) — Last-Event-ID đúng định dạng nhưng mới hơn mọi id → không resync, nhận sự kiện mới
  ensure_mode test
  local u t f; u=$(uid 164); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-41.sse; sse_clean "$u"
  sse_burst 3 test.future "$t" 3
  sse_bg "$f" 15 "$t" -H "Last-Event-ID: 99999999999999-0"; sse_wait_line "$f" '^event: ready' 10
  sleep 1; sse_pub test.future '{"n":99}' "$t" >/dev/null; sse_wait_ids "$f" 1 10; sse_kill
  chk "không resync (id không cũ hơn bộ đệm)" "$(sse_ntype "$f" resync)" 0
  chk "số sự kiện đọc bù" "$(sse_nid "$f")" 1
  chk "sự kiện nhận được là cái mới" "$(sse_datas "$f" | sed -n 's/.*"n":\([0-9]*\).*/\1/p' | awk 'END{print}')" 99
}
tc_pg05_42() {  # AC9 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_ResyncWhenTooOld|TestSSE_ResyncOnMalformedLastEventID'
}

# =============================== AC10 — hai bản gateway ===============================
tc_pg05_43() {  # AC10 — đúng lệnh trong AC: stream qua Caddy + 5 POST → nhận đủ 5 `event: test.ping`
  ensure_mode test
  local t f i; t=$(tok STUDENT "$U1"); f=$QC_OUT/tc-pg05-43.sse; sse_clean "$U1"
  sse_bg "$f" 8 "$t"; sse_wait_line "$f" '^event: ready' 10
  for i in 1 2 3 4 5; do sse_pub test.ping "{\"n\":$i}" "$t" >/dev/null; done
  sse_wait_ids "$f" 5 8; sse_kill
  chk "số 'event: test.ping' nhận được" "$(grep -c '^event: test.ping' "$f")" 5
}
tc_pg05_44() {  # AC10 — chứng minh hai bản cùng tham gia: X-Instance-Id của stream và của 5 POST
  ensure_mode test
  local u t f i inst n other; u=$(uid 170); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-44.sse; sse_clean "$u"
  sse_bg "$f" 12 "$t" -D-; sse_wait_line "$f" '^event: ready' 10
  inst=$(tr -d '\r' < "$f" | hval x-instance-id)
  : > "$QC_OUT/tc-pg05-44.inst"
  for i in 1 2 3 4 5; do
    curl -sk --max-time 10 -D- -o /dev/null -X POST -H "Authorization: Bearer $t" -H 'Content-Type: application/json' \
      -d "{\"type\":\"test.ping\",\"data\":{\"n\":$i}}" "$GW/api/v1/_test/events" 2>/dev/null | tr -d '\r' | hval x-instance-id >> "$QC_OUT/tc-pg05-44.inst"
  done
  sse_wait_ids "$f" 5 10; sse_kill
  n=$(sort -u "$QC_OUT/tc-pg05-44.inst" | grep -c .)
  other=$(grep -vc "^$inst\$" "$QC_OUT/tc-pg05-44.inst")
  echo "    (stream ở bản [$inst]; POST rơi vào: $(sort -u "$QC_OUT/tc-pg05-44.inst" | tr '\n' ' '))"
  chk_ne "X-Instance-Id của stream" "$inst" ''
  chk_ge "số bản gateway nhận POST (round_robin của Caddy)" "$n" 2
  chk_ge "số POST rơi vào bản KHÁC bản giữ stream" "$other" 1
  chk    "stream nhận đủ 5 sự kiện dù POST trúng bản nào" "$(grep -c '^event: test.ping' "$f")" 5
}
tc_pg05_45() {  # AC10 — ép bản: stream ở A, phát THẲNG vào B → nhận trong ≤ 1 s
  ensure_mode test
  local u t f inst a b ipb other d t0; u=$(uid 171); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-45.sse; sse_clean "$u"
  sse_bg_ts "$f" 12 "$t" -D-; sse_wait_line "$f" 'event: ready' 10
  inst=$(sse_plain "$f" | tr -d '\r' | hval x-instance-id)
  other=""
  for a in $(gw_ids); do [ "$(gw_host "$a")" = "$inst" ] || other=$a; done
  chk_ne "tìm được bản KHÁC bản giữ stream" "$other" ''
  ipb=$(gw_ip "$other"); b="{\"type\":\"test.cross\",\"data\":{\"n\":1}}"
  t0=$(now_ms)
  rawhttp "$ipb" 8080 "POST /api/v1/_test/events HTTP/1.1\r\nHost: g\r\nAuthorization: Bearer $t\r\nContent-Type: application/json\r\nContent-Length: ${#b}\r\nConnection: close\r\n\r\n$b" 5 >/dev/null
  sse_wait_line "$f" 'event: test.cross' 5; sse_ts_wait; sse_plain "$f" > "$f.plain"
  d=$(awk -v t0="$t0" '{l=$0; sub(/^[0-9.]+ /,"",l); if (l ~ /^event: test.cross/) {printf "%d", $1*1000 - t0; exit}}' "$f")
  chk    "stream (ở bản A) nhận sự kiện phát từ bản B" "$(sse_ntype "$f.plain" test.cross)" 1
  chk_ge "độ trễ chéo bản (ms) đo được" "${d:--1}" 0
  chk_le "độ trễ chéo bản ≤ 1 s" "${d:-99999}" 1000
}
tc_pg05_46() {  # AC10 — hai kết nối của CÙNG người dùng ở hai bản đều nhận
  ensure_mode test
  local u t ids a b ipa ipb o1 o2 body; u=$(uid 172); t=$(tok STUDENT "$u"); sse_clean "$u"
  ids=$(gw_ids); a=$(printf '%s\n' "$ids" | awk 'NR==1'); b=$(printf '%s\n' "$ids" | awk 'NR==2')
  ipa=$(gw_ip "$a"); ipb=$(gw_ip "$b")
  direct_sse "$ipa" "$t" 6 > "$QC_OUT/tc-pg05-46a.sse" 2>/dev/null &
  direct_sse "$ipb" "$t" 6 > "$QC_OUT/tc-pg05-46b.sse" 2>/dev/null &
  sleep 2
  body='{"type":"test.both","data":{"n":1}}'
  curl -sk --max-time 10 -o /dev/null -X POST -H "Authorization: Bearer $t" -H 'Content-Type: application/json' -d "$body" "$GW/api/v1/_test/events"
  wait 2>/dev/null
  o1=$(grep -c '^event: test.both' "$QC_OUT/tc-pg05-46a.sse"); o2=$(grep -c '^event: test.both' "$QC_OUT/tc-pg05-46b.sse")
  chk "kết nối ở bản A nhận" "$o1" 1
  chk "kết nối ở bản B nhận" "$o2" 1
}
tc_pg05_47() {  # AC10 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_CrossInstance|TestSSE_TwoConnectionsTwoInstances'
}

# =============================== AC11 — phân quyền / cô lập ===============================
tc_pg05_48() {  # AC11 — sự kiện của U1 không lọt sang U2
  ensure_mode test
  local t1 t2 f1 f2 p2; t1=$(tok STUDENT "$U1"); t2=$(tok STUDENT "$U2")
  sse_clean "$U1"; sse_clean "$U2"; f1=$QC_OUT/tc-pg05-48a.sse; f2=$QC_OUT/tc-pg05-48b.sse
  curl -sk -N --max-time 8 -H "Authorization: Bearer $t2" "$EVURL" > "$f2" 2>/dev/null & p2=$!
  sse_bg "$f1" 8 "$t1"; sse_wait_line "$f1" '^event: ready' 10; sse_wait_line "$f2" '^event: ready' 10
  sse_pub test.iso '{"n":1}' "$t1" >/dev/null
  sse_wait_ids "$f1" 1 8; sleep 1; sse_kill; kill -9 "$p2" 2>/dev/null; wait 2>/dev/null
  chk "U1 nhận" "$(sse_ntype "$f1" test.iso)" 1
  chk "U2 nhận (phải 0)" "$(sse_ntype "$f2" test.iso)" 0
  chk "U2 không có id: nào" "$(sse_nid "$f2")" 0
}
tc_pg05_49() {  # AC11 — `?user_id=<U1>` bị BỎ QUA: U2 mở stream kèm tham số đó vẫn chỉ nhận sự kiện của chính mình
  ensure_mode test
  local t1 t2 f; t1=$(tok STUDENT "$U1"); t2=$(tok STUDENT "$U2"); sse_clean "$U1"; sse_clean "$U2"
  f=$QC_OUT/tc-pg05-49.sse
  curl -sk -N --max-time 10 -H "Authorization: Bearer $t2" "$EVURL?user_id=$U1" > "$f" 2>/dev/null & local p=$!
  sse_wait_line "$f" '^event: ready' 10
  sse_pub test.q '{"n":1}' "$t1" >/dev/null            # phát cho U1
  sleep 2
  sse_pub test.q '{"n":2}' "$t2" >/dev/null            # phát cho U2 (chính chủ)
  sse_wait_ids "$f" 1 8; kill -9 "$p" 2>/dev/null; wait 2>/dev/null
  chk "số sự kiện nhận được" "$(sse_nid "$f")" 1
  chk "chỉ nhận sự kiện của chính U2 (n=2)" "$(sse_datas "$f" | sed -n 's/.*"n":\([0-9]*\).*/\1/p' | tr '\n' ' ')" '2 '
}
tc_pg05_50() {  # AC11 — không token → 401 UNAUTHENTICATED (JSON, trước khi mở stream)
  ensure_mode test
  local c b h; c=$(code "$EVURL"); b=$(curl -sk --max-time 5 "$EVURL" | jq -r '.code // empty'); h=$(hdr "$EVURL")
  chk    "mã HTTP" "$c" 401
  chk    "code" "$b" UNAUTHENTICATED
  chk_re "Content-Type là JSON" "$(printf '%s\n' "$h" | hval content-type)" '^application/json'
  chk_re "WWW-Authenticate" "$(printf '%s\n' "$h" | hval www-authenticate)" 'Bearer realm="edupilot"'
  chk    "không có header của SSE" "$(printf '%s\n' "$h" | grep -ci '^x-accel-buffering')" 0
}
tc_pg05_51() {  # AC11 — token hết hạn → 401 TOKEN_EXPIRED (đối chứng với TC-PG05-50)
  ensure_mode test
  local t c b; t=$(tok STUDENT "$U1" --ttl -1m)
  c=$(code -H "Authorization: Bearer $t" "$EVURL"); b=$(curl -sk --max-time 5 -H "Authorization: Bearer $t" "$EVURL" | jq -r '.code // empty')
  chk "mã HTTP" "$c" 401
  chk "code" "$b" TOKEN_EXPIRED
}
tc_pg05_52() {  # AC11 / SRS 6.8 — token qua QUERY STRING không được chấp nhận
  ensure_mode test
  local t c1 c2 b; t=$(tok STUDENT "$U1")
  c1=$(code "$EVURL?token=$t"); c2=$(code "$EVURL?access_token=$t")
  b=$(curl -sk --max-time 5 "$EVURL?token=$t" | jq -r '.code // empty')
  chk "?token= → mã HTTP" "$c1" 401
  chk "?access_token= → mã HTTP" "$c2" 401
  chk "code" "$b" UNAUTHENTICATED
}
tc_pg05_53() {  # AC11 / SRS 6.1 — token sai chữ ký → 401 TOKEN_INVALID
  ensure_mode test
  local t bad c b; t=$(tok STUDENT "$U1"); bad="$(printf '%s' "$t" | cut -d. -f1,2).AAAAinvalidsignatureAAAA"
  c=$(code -H "Authorization: Bearer $bad" "$EVURL"); b=$(curl -sk --max-time 5 -H "Authorization: Bearer $bad" "$EVURL" | jq -r '.code // empty')
  chk "mã HTTP" "$c" 401
  chk "code" "$b" TOKEN_INVALID
}
tc_pg05_54() {  # AC11 / SRS 6.3 — ADMIN phát hộ người khác được; STUDENT chỉ phát cho chính mình
  ensure_mode test
  local ta ts f c; ta=$(tok ADMIN "$(uid 180)"); ts=$(tok STUDENT "$U1"); sse_clean "$U2"
  f=$QC_OUT/tc-pg05-54.sse
  sse_bg "$f" 12 "$(tok STUDENT "$U2")"; sse_wait_line "$f" '^event: ready' 10
  chk_re "ADMIN phát user_id=U2 → 2xx" "$(sse_pub test.adm '{"n":1}' "$ta" "$U2")" '^2[0-9][0-9]$'
  sse_wait_ids "$f" 1 8
  c=$(sse_pub test.stu '{"n":2}' "$ts" "$U2")     # STUDENT U1 cố phát cho U2
  echo "    (mã khi STUDENT chỉ định user_id người khác: $c — spec không nêu mã, xem 'Câu hỏi cho PM')"
  sleep 2; sse_kill
  chk    "U2 nhận sự kiện do ADMIN phát" "$(sse_ntype "$f" test.adm)" 1
  chk    "U2 KHÔNG nhận sự kiện STUDENT khác cố phát hộ" "$(sse_ntype "$f" test.stu)" 0
  chk_re "mã trả về cho STUDENT phát hộ (2xx = bỏ qua user_id, 4xx = từ chối)" "$c" '^(2[0-9][0-9]|4[0-9][0-9])$'
}
tc_pg05_55() {  # AC11 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_IsolationBetweenUsers|TestSSE_UserIDQueryIgnored|TestSSE_Unauthenticated'
}

# =============================== AC12 — tiến độ việc dài qua SSE ===============================
tc_pg05_56() {  # AC12 — job test.progress 4 bước → job.progress 25, 50, 75, 100 không giảm, cuối SUCCEEDED
  ensure_mode test
  local t f jid ps last; t=$(tok STUDENT "$U1"); f=$QC_OUT/tc-pg05-56.sse; sse_clean "$U1"
  sse_bg "$f" 45 "$t"; sse_wait_line "$f" '^event: ready' 10
  jid=$(curl -sk --max-time 15 -X POST -H "Authorization: Bearer $t" -H 'Content-Type: application/json' \
        -d '{"steps":4,"kind":"test.progress"}' "$JOBURL" | jq -r '.job_id // .id // empty')
  echo "$jid" > "$QC_OUT/tc-pg05-56.jobid"
  chk_re "job_id là uuid" "$jid" '^[0-9a-f-]{36}$'
  sse_wait_line "$f" '"progress":100' 40; sleep 1; sse_kill
  ps=$(sse_data_of "$f" job.progress | jq -r '.progress' | tr '\n' ' ')
  last=$(sse_data_of "$f" job.progress | jq -r '.status' | awk 'END{print}')
  chk "dãy progress" "$ps" '25 50 75 100 '
  chk "status của sự kiện cuối" "$last" SUCCEEDED
  chk "mọi sự kiện mang đúng job_id" "$(sse_data_of "$f" job.progress | jq -r '.job_id' | grep -vc "^$jid\$")" 0
  chk "số sự kiện job.progress" "$(sse_ntype "$f" job.progress)" 4
}
tc_pg05_57() {  # AC12 — U2 không nhận tiến độ job của U1
  ensure_mode test
  local t1 t2 f2 p; t1=$(tok STUDENT "$U1"); t2=$(tok STUDENT "$U2"); f2=$QC_OUT/tc-pg05-57.sse; sse_clean "$U2"
  curl -sk -N --max-time 30 -H "Authorization: Bearer $t2" "$EVURL" > "$f2" 2>/dev/null & p=$!
  sse_wait_line "$f2" '^event: ready' 10
  curl -sk --max-time 15 -o /dev/null -X POST -H "Authorization: Bearer $t1" -H 'Content-Type: application/json' \
       -d '{"steps":4,"kind":"test.progress"}' "$JOBURL"
  sleep 12; kill -9 "$p" 2>/dev/null; wait 2>/dev/null
  chk "số job.progress U2 nhận" "$(sse_ntype "$f2" job.progress)" 0
  chk "U2 không nhận sự kiện nào" "$(sse_nid "$f2")" 0
}
tc_pg05_58() {  # AC12 — trạng thái trong DB và GET /api/v1/jobs/{id} khớp sự kiện cuối
  ensure_mode test
  local t jid row st; t=$(tok STUDENT "$U1")
  jid=$(curl -sk --max-time 15 -X POST -H "Authorization: Bearer $t" -H 'Content-Type: application/json' \
        -d '{"steps":4,"kind":"test.progress"}' "$JOBURL" | jq -r '.job_id // .id // empty')
  sleep 12
  row=$($PSQL -c "select status||'|'||progress from jobs where id='$jid'" 2>/dev/null | tr -d '\r')
  st=$(curl -sk --max-time 10 -H "Authorization: Bearer $t" "$GW/api/v1/jobs/$jid" | jq -r '.status+"|"+(.progress|tostring)')
  chk "dòng jobs trong DB (status|progress)" "$row" 'SUCCEEDED|100'
  chk "GET /api/v1/jobs/{id}" "$st" 'SUCCEEDED|100'
}
tc_pg05_59() {  # AC12 — test Go (lệnh trong AC)
  gt './internal/jobs ./internal/httpapi/sse' 'TestJobProgress_SSE|TestJobProgress_OnlyOwner'
}

# =============================== AC13 — tắt một gateway giữa stream ===============================
tc_pg05_60() {  # AC13 — SIGTERM bản đang giữ stream → `event: shutdown` {"reason":"server_shutdown"} rồi đóng ≤ 1 s
  ensure_mode test
  local u t f inst target c d r; u=$(uid 190); t=$(tok STUDENT "$u"); f=$QC_OUT/tc-pg05-60.sse; sse_clean "$u"
  sse_bg_ts "$f" 40 "$t" -D-; sse_wait_line "$f" 'event: ready' 10
  inst=$(sse_plain "$f" | tr -d '\r' | hval x-instance-id); target=""
  for c in $(gw_ids); do [ "$(gw_host "$c")" = "$inst" ] && target=$c; done
  docker stop -t 30 "$target" >/dev/null 2>&1
  sse_ts_wait; sse_plain "$f" > "$f.plain"
  gw_restore test
  r=$(sse_data_of "$f.plain" shutdown | jq -r '.reason // empty')
  d=$(sse_dms "$(sse_ts_of "$f" '^event: shutdown')" "$(sse_ts_of "$f" '__END__')")
  chk_ne "xác định được bản giữ stream" "$target" ''
  chk    "có event: shutdown" "$(grep -c '^event: shutdown' "$f.plain")" 1
  chk    "reason" "$r" server_shutdown
  chk    "shutdown không mang id:" "$(grep -c '^id:' "$f.plain")" 0
  chk_ge "shutdown → đóng (ms) đo được" "$d" 0
  chk_le "shutdown → đóng ≤ 1 s" "$d" 1000
}
tc_pg05_61() {  # AC13 — nối lại bản CÒN LẠI với Last-Event-ID + CÙNG token → 200, không mất, không trùng
  ensure_mode test
  local u t f1 f2 inst target c last ns; u=$(uid 191); t=$(tok STUDENT "$u" --ttl 10m)
  f1=$QC_OUT/tc-pg05-61a.sse; f2=$QC_OUT/tc-pg05-61b.sse; sse_clean "$u"
  sse_bg "$f1" 30 "$t" -D-; sse_wait_line "$f1" 'event: ready' 10
  inst=$(tr -d '\r' < "$f1" | hval x-instance-id); target=""
  for c in $(gw_ids); do [ "$(gw_host "$c")" = "$inst" ] && target=$c; done
  sse_pub test.fo '{"n":1}' "$t" >/dev/null; sse_pub test.fo '{"n":2}' "$t" >/dev/null
  sse_wait_ids "$f1" 2 10
  docker stop -t 30 "$target" >/dev/null 2>&1
  sse_wait_line "$f1" '^event: shutdown' 10; sse_kill
  last=$(sse_lastid "$f1")
  sse_pub test.fo '{"n":3}' "$t" >/dev/null; sse_pub test.fo '{"n":4}' "$t" >/dev/null       # phát trong lúc chuyển
  c=$(curl -sk -N --max-time 2 -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $t" -H "Last-Event-ID: $last" "$EVURL")
  sse_bg "$f2" 15 "$t" -H "Last-Event-ID: $last"; sse_wait_ids "$f2" 2 10
  sse_pub test.fo '{"n":5}' "$t" >/dev/null; sse_wait_ids "$f2" 3 10; sse_kill
  gw_restore test
  ns=$(sse_datas "$f2" | sed -n 's/.*"n":\([0-9]*\).*/\1/p' | tr '\n' ' ')
  chk    "mã HTTP khi nối lại bằng token cũ (000 = đang stream, không 401)" "$c" 000
  chk    "phiên 2 nhận đúng n=3,4,5 theo thứ tự" "$ns" '3 4 5 '
  chk    "không nhận lại sự kiện đã nhận (id trùng)" "$(cat "$f1" "$f2" | grep '^id: ' | sort | uniq -d | grep -c .)" 0
  chk    "phiên 2 không phải resync" "$(sse_ntype "$f2" resync)" 0
}
tc_pg05_62() {  # AC13 — đúng lệnh trong AC: sau khi tắt một bản, phiên kế tiếp qua Caddy vẫn chạy (404, KHÔNG 401)
  ensure_mode test
  local t c target; t=$(tok STUDENT "$U1" --ttl 10m)
  target=$(gw_ids | awk 'NR==1')
  docker stop -t 30 "$target" >/dev/null 2>&1; sleep 3
  c=$(code -H "Authorization: Bearer $t" "$GW/api/v1/jobs/$UX")
  gw_restore test
  chk "GET /api/v1/jobs/<uuid lạ> sau khi tắt một bản" "$c" 404
}
tc_pg05_63() {  # AC13 (chiều còn lại) — tắt bản KHÔNG giữ stream → stream vẫn sống, không `shutdown`
  ensure_mode test
  local u t f inst other c; u=$(uid 192); t=$(tok STUDENT "$u" --ttl 10m); f=$QC_OUT/tc-pg05-63.sse; sse_clean "$u"
  sse_bg "$f" 30 "$t" -D-; sse_wait_line "$f" 'event: ready' 10
  inst=$(tr -d '\r' < "$f" | hval x-instance-id); other=""
  for c in $(gw_ids); do [ "$(gw_host "$c")" = "$inst" ] || other=$c; done
  docker stop -t 30 "$other" >/dev/null 2>&1; sleep 3
  sse_pub test.alive '{"n":1}' "$t" >/dev/null
  sse_wait_ids "$f" 1 10; sse_kill
  gw_restore test
  chk_ne "tìm được bản KHÔNG giữ stream" "$other" ''
  chk    "stream không nhận shutdown" "$(grep -c '^event: shutdown' "$f")" 0
  chk    "stream vẫn nhận sự kiện sau khi bản kia tắt" "$(sse_ntype "$f" test.alive)" 1
}
tc_pg05_64() {  # AC13 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_GatewayShutdownFailover'
}

# =============================== AC14 — qua Caddy không nén, không đệm ===============================
tc_pg05_65() {  # AC14 — đúng lệnh trong AC: `Accept-Encoding: gzip` → không `Content-Encoding`, chỉ `event: ready`
  ensure_mode test
  local t n; t=$(tok STUDENT "$U1")
  n=$(curl -sk -N -m 3 -D- -H 'Accept-Encoding: gzip' -H "Authorization: Bearer $t" "$EVURL" 2>/dev/null \
      | grep -ciE '^content-encoding|^event: ready')
  chk "số dòng khớp '^content-encoding|^event: ready' (chỉ còn event: ready)" "$n" 1
}
tc_pg05_66() {  # AC14 — với gzip: `ready` vẫn đến ≤ 300 ms và thân KHÔNG bị nén (đọc được chữ thô)
  ensure_mode test
  local t f ttfb; t=$(tok STUDENT "$U1"); f=$QC_OUT/tc-pg05-66.sse
  ttfb=$(curl -sk -N --max-time 3 -H 'Accept-Encoding: gzip' -H "Authorization: Bearer $t" \
         -o "$f" -w '%{time_starttransfer}' "$EVURL" 2>/dev/null)
  chk_le "ttfb (ms) khi xin gzip" "$(awk -v v="$ttfb" 'BEGIN{printf "%d", v*1000}')" 300
  chk "dòng đầu đọc được dạng văn bản" "$(awk 'NR==1' "$f")" 'retry: 3000'
  chk "có event: ready dạng văn bản" "$(grep -c '^event: ready' "$f")" 1
  chk "không có byte gzip magic 1f8b" "$(dd if="$f" bs=1 count=2 2>/dev/null | od -An -tx1 | tr -d ' \n' | grep -c '^1f8b$')" 0
}

# =============================== AC15 — Redis chết ===============================
tc_pg05_67() {  # AC15 — Redis chết GIỮA stream → `reconnect` {"reason":"upstream_unavailable"} rồi đóng (không treo)
  ensure_mode test
  local t f r d; t=$(tok STUDENT "$U1" --ttl 10m); f=$QC_OUT/tc-pg05-67.sse
  sse_bg_ts "$f" 45 "$t"; sse_wait_line "$f" 'event: ready' 10
  $C stop redis >/dev/null 2>&1
  sse_ts_wait
  $C start redis >/dev/null 2>&1; wait_ready 60
  sse_plain "$f" > "$f.plain"
  r=$(sse_data_of "$f.plain" reconnect | jq -r '.reason // empty')
  d=$(sse_dms "$(sse_ts_of "$f" '^event: reconnect')" "$(sse_ts_of "$f" '__END__')")
  chk    "có event: reconnect" "$(grep -c '^event: reconnect' "$f.plain")" 1
  chk    "reason" "$r" upstream_unavailable
  chk_ge "reconnect → đóng (ms) đo được" "$d" 0
  chk_le "reconnect → đóng ≤ 2 s (không treo)" "$d" 2000
  chk_le "stream không treo tới hết --max-time 45 s" "$(sse_dms "$(sse_ts_of "$f" '^event: ready')" "$(sse_ts_of "$f" '__END__')")" 40000
}
tc_pg05_68() {  # AC15 — mở kết nối MỚI khi Redis còn chết → 503 SERVICE_UNAVAILABLE
  ensure_mode test
  local t c b h; t=$(tok STUDENT "$U1" --ttl 10m)
  $C stop redis >/dev/null 2>&1; sleep 2
  c=$(curl -sk --max-time 10 -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $t" "$EVURL")
  b=$(curl -sk --max-time 10 -H "Authorization: Bearer $t" "$EVURL" | jq -r '.code // empty')
  h=$(curl -sk --max-time 10 -D- -o /dev/null -H "Authorization: Bearer $t" "$EVURL" | tr -d '\r')
  $C start redis >/dev/null 2>&1; wait_ready 60
  chk    "mã HTTP" "$c" 503
  chk    "code" "$b" SERVICE_UNAVAILABLE
  chk_re "Content-Type là JSON (không mở stream)" "$(printf '%s\n' "$h" | hval content-type)" '^application/json'
}
tc_pg05_69() {  # AC15 — Redis về → nối lại được ngay, stream chạy bình thường
  ensure_mode test
  local t f; t=$(tok STUDENT "$U1" --ttl 10m); f=$QC_OUT/tc-pg05-69.sse
  $C stop redis >/dev/null 2>&1; sleep 2
  $C start redis >/dev/null 2>&1; wait_ready 60
  sse_clean "$U1"
  sse_bg "$f" 12 "$t"; sse_wait_line "$f" '^event: ready' 15
  sse_pub test.back '{"n":1}' "$t" >/dev/null; sse_wait_ids "$f" 1 10; sse_kill
  chk "có event: ready sau khi Redis về" "$(grep -c '^event: ready' "$f")" 1
  chk "nhận được sự kiện mới" "$(sse_ntype "$f" test.back)" 1
}
tc_pg05_70() {  # AC15 — test Go (lệnh trong AC)
  gt ./internal/httpapi/sse 'TestSSE_RedisDownMidStream|TestSSE_RedisDownOnConnect'
}

main 05 "$@"
