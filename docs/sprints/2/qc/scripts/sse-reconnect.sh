#!/usr/bin/env bash
# QC US-PG-05 — công cụ SSE dùng chung: đọc stream bằng `curl -N`, mô phỏng ngắt / nối lại bằng `Last-Event-ID`, đếm sự kiện.
# Nguồn: docs/specs/FEAT-pg-foundation/US.md v1.1 US-PG-05 AC7, AC8, AC10; SRS mục 6.8 (thứ tự byte, khung sự kiện, thuật toán nối lại).
# Hai cách dùng:
#   1) Từ pg05.sh:   . "$(dirname "$0")/sse-reconnect.sh"      → chỉ nạp hàm, KHÔNG tự chạy.
#   2) Độc lập:      bash docs/sprints/2/qc/scripts/sse-reconnect.sh <lệnh> [đối số …]
#        ids|types|datas|lastid <file>        bóc dòng `id:` / `event:` / `data:` của một stream đã ghi
#        cap <file> <giây> <token> [cờ curl…] mở stream nền, in PID (dừng bằng `kill -9 <PID>`)
#        ac7 [thư mục ra]                     chạy kịch bản AC7 (e1…e14) rồi in tóm tắt một dòng mỗi số đo
#        ac8 [số sự kiện] [số lần ngắt]       chạy kịch bản AC8 (đua, mặc định 1000 / 20) rồi in tóm tắt
#      Cần stack đang chạy ở **chế độ test** (image `*-test`, có `POST /api/v1/_test/events`).
# Bash 3.2 (macOS): không `timeout`, không mảng kết hợp, không `date +%s%N`, không `mapfile`.
SRE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
declare -f chk >/dev/null 2>&1 || . "$SRE_DIR/lib.sh"

# ---------- Mở / đóng stream ----------
# lib.sh có `sse_cap` (chạy trong subshell, chỉ hợp khi để `--max-time` tự kết thúc).
# Các TC ở đây phải CẮT stream giữa chừng nên dùng `sse_bg`: gọi curl trực tiếp để `kill -9 $SSE_PID` giết đúng tiến trình curl.
SSE_PID=""
sse_bg() {  # sse_bg <file> <giây tối đa> <token> [cờ curl thêm, ví dụ -H "Last-Event-ID: 1-0"] — mở stream nền
  local f=$1 s=$2 t=$3; shift 3; : > "$f"
  curl -sk -N --max-time "$s" -H "Authorization: Bearer $t" "$@" "$GW/api/v1/events" >> "$f" 2>/dev/null &
  SSE_PID=$!; }
sse_kill() {  # cắt stream đang mở (mô phỏng client ngắt đột ngột)
  [ -n "$SSE_PID" ] && kill -9 "$SSE_PID" 2>/dev/null
  wait "$SSE_PID" 2>/dev/null; SSE_PID=""; return 0; }

SSE_TS_PID=""
sse_bg_ts() {  # sse_bg_ts <file> <giây> <token> [cờ curl…] — như sse_bg nhưng GẮN MỐC THỜI GIAN từng dòng
  # mỗi dòng: "<epoch giây.mili> <dòng stream>"; dòng cuối "<epoch> __END__" = lúc kết nối đóng.
  local f=$1 s=$2 t=$3; shift 3; : > "$f"
  { curl -sk -N --max-time "$s" -H "Authorization: Bearer $t" "$@" "$GW/api/v1/events" 2>/dev/null \
      | perl -MTime::HiRes=time -ne 'BEGIN{$|=1} printf "%.3f %s", time, $_'
    perl -MTime::HiRes=time -e 'printf "%.3f __END__\n", time'; } >> "$f" &
  SSE_TS_PID=$!; }
sse_ts_wait() { wait "$SSE_TS_PID" 2>/dev/null; SSE_TS_PID=""; return 0; }
sse_plain() {  # sse_plain <file có mốc> → in stream thô (bỏ cột mốc thời gian)
  sed 's/^[0-9][0-9.]* //' "$1" 2>/dev/null; }
sse_ts_of() {  # sse_ts_of <file có mốc> <regex dòng> → epoch giây (số thực) của dòng ĐẦU khớp, rỗng nếu không có
  awk -v re="$2" '{ line=$0; sub(/^[0-9.]+ /,"",line); if (line ~ re) { print $1; exit } }' "$1" 2>/dev/null; }
sse_ts_last() {  # sse_ts_last <file có mốc> <regex dòng> → epoch của dòng CUỐI khớp
  awk -v re="$2" '{ line=$0; sub(/^[0-9.]+ /,"",line); if (line ~ re) v=$1 } END{ if (v!="") print v }' "$1" 2>/dev/null; }
sse_dms() {  # sse_dms <epoch1> <epoch2> → (epoch2 − epoch1) tính bằng mili giây, số nguyên
  awk -v a="${1:-0}" -v b="${2:-0}" 'BEGIN{ if (a==0 || b==0) print -1; else printf "%d", (b-a)*1000 }'; }

# ---------- Bóc khung sự kiện (SRS 6.8: `id:` / `event:` / `data:` + dòng trống) ----------
sse_ids()    { grep '^id: '    "$1" 2>/dev/null | sed 's/^id: //'; }
sse_types()  { grep '^event: ' "$1" 2>/dev/null | sed 's/^event: //'; }
sse_datas()  { grep '^data: '  "$1" 2>/dev/null | sed 's/^data: //'; }
sse_lastid() { sse_ids "$1" | awk 'END{ if (NR>0) print }'; }
sse_nid()    { sse_ids "$1" | grep -c .; }
sse_ntype()  { sse_types "$1" | grep -c "^$2\$"; }
sse_nhb()    { grep -c '^: hb' "$1" 2>/dev/null; }
sse_data_of() {  # sse_data_of <file> <type> → các dòng data của sự kiện có type đó (một JSON mỗi dòng)
  awk -v t="event: $2" '$0==t { getline; sub(/^data: /,""); print }' "$1" 2>/dev/null; }
sse_first_event() {  # sự kiện đầu tiên SAU `ready` (để kiểm `resync` đứng đầu)
  sse_types "$1" | awk 'NR==2'; }
sse_ok_frames() {  # sse_ok_frames <file> → số khối `id:`+`event:`+`data:`+dòng trống đúng định dạng, data là JSON một dòng
  awk '
    /^id: /    { id=$0; sub(/^id: /,"",id); st=1; next }
    st==1 && /^event: / { ty=$0; sub(/^event: /,"",ty); st=2; next }
    st==2 && /^data: /  { d=$0; sub(/^data: /,"",d); st=3; next }
    st==3 && /^$/ {
        if (id ~ /^[0-9]+-[0-9]+$/ && ty ~ /^[a-z][a-z0-9_.]{0,63}$/ && d ~ /^[\[{]/) ok++
        st=0; next }
    { st=0 }
    END { print ok+0 }' "$1" 2>/dev/null; }
sse_ids_monotonic() {  # sse_ids_monotonic <file> → "ok" nếu id tăng NGHIÊM NGẶT theo (ms, seq), ngược lại "LOI:<id>"
  sse_ids "$1" | awk -F- '
    { if (NR>1 && !($1>pm || ($1==pm && $2>ps))) { printf "LOI:%s-%s\n", $1, $2; bad=1; exit } pm=$1; ps=$2 }
    END { if (!bad) print "ok" }'; }
sse_ns() {  # sse_ns <file…> → số giá trị "n" KHÁC NHAU trong data (data dạng {"n":<số>})
  cat "$@" 2>/dev/null | grep '^data: ' | sed -n 's/.*"n":\([0-9][0-9]*\).*/\1/p' | sort -n -u | grep -c .; }

# ---------- Phát sự kiện (route thử, chế độ test) ----------
sse_pub() {  # sse_pub <type> <json data> <token> [user_id] → in mã HTTP
  local ty=$1 d=$2 t=$3 u=${4:-} body
  if [ -n "$u" ]; then body="{\"type\":\"$ty\",\"data\":$d,\"user_id\":\"$u\"}"
  else body="{\"type\":\"$ty\",\"data\":$d}"; fi
  curl -sk --max-time 15 -o /dev/null -w '%{http_code}' -X POST \
    -H "Authorization: Bearer $t" -H 'Content-Type: application/json' -d "$body" "$GW/api/v1/_test/events"; }
sse_pub_body() {  # như sse_pub nhưng in THÂN phản hồi (để đọc `code` khi bị từ chối)
  local ty=$1 d=$2 t=$3 u=${4:-} body
  if [ -n "$u" ]; then body="{\"type\":\"$ty\",\"data\":$d,\"user_id\":\"$u\"}"
  else body="{\"type\":\"$ty\",\"data\":$d}"; fi
  curl -sk --max-time 15 -X POST \
    -H "Authorization: Bearer $t" -H 'Content-Type: application/json' -d "$body" "$GW/api/v1/_test/events"; }
sse_burst() {  # sse_burst <n> <type> <token> [song song=10] — phát n sự kiện data={"n":i}; tự `rl_reset` mỗi 150 lần
  local n=$1 ty=$2 t=$3 par=${4:-10} i=1 c=0                       # (giới hạn mặc định 300/phút/IP, 600/phút/user — SRS 6.5)
  while [ "$i" -le "$n" ]; do
    sse_pub "$ty" "{\"n\":$i}" "$t" >/dev/null &
    c=$((c+1))
    [ $((c % par)) -eq 0 ] && wait
    [ $((c % 150)) -eq 0 ] && rl_reset >/dev/null 2>&1
    i=$((i+1))
  done; wait; return 0; }

# ---------- Dọn khoá SSE của một người dùng (vệ sinh giữa các TC, KHÔNG phải phép kiểm) ----------
sse_clean() { $RDS del "ep:sse:buf:$1" "ep:sse:conn:$1" >/dev/null 2>&1; return 0; }

# ---------- Chờ ----------
sse_wait_ids() {  # sse_wait_ids <file> <n> <giây tối đa> → 0 nếu stream đã có ≥ n sự kiện có id
  local f=$1 n=$2 max=$3 i=0
  while [ "$i" -lt $((max * 5)) ]; do
    [ "$(sse_nid "$f")" -ge "$n" ] && return 0; sleep 0.2; i=$((i + 1)); done; return 1; }
sse_wait_line() {  # sse_wait_line <file> <regex> <giây tối đa>
  local f=$1 re=$2 max=$3 i=0
  while [ "$i" -lt $((max * 5)) ]; do
    grep -Eq "$re" "$f" 2>/dev/null && return 0; sleep 0.2; i=$((i + 1)); done; return 1; }

# ---------- Kịch bản AC7: ngắt rồi nối lại bằng Last-Event-ID ----------
# Phát e1…e5 → client đọc đủ 5 → CẮT → phát e6…e13 lúc client đang ngắt → nối lại `Last-Event-ID: <id e5>`
# → phải nhận đúng e6…e13 theo thứ tự, không nhận lại e5 → phát e14 → nhận tiếp.
# (US.md viết "phát e1…e10, đọc 5 rồi ngắt": curl không điều tiết được tốc độ đọc nên QC cố định điểm cắt
#  bằng cách phát 5 trước — điều kiện tương đương: id cuối client đã nhận = id của e5. Ghi ở "Điểm khó kiểm".)
sse_ac7_flow() {  # sse_ac7_flow <thư mục> <token> → file s1.log, s2.log, id5
  local d=$1 t=$2 i
  mkdir -p "$d"
  sse_bg "$d/s1.log" 25 "$t"; sse_wait_line "$d/s1.log" '^event: ready' 10 || return 1
  for i in 1 2 3 4 5; do sse_pub test.e "{\"n\":$i}" "$t" >/dev/null; done
  sse_wait_ids "$d/s1.log" 5 15 || { sse_kill; return 1; }
  sse_kill
  sse_ids "$d/s1.log" | awk 'NR==5' > "$d/id5"
  for i in 6 7 8 9 10 11 12 13; do sse_pub test.e "{\"n\":$i}" "$t" >/dev/null; done
  sse_bg "$d/s2.log" 25 "$t" -H "Last-Event-ID: $(cat "$d/id5")"
  sse_wait_ids "$d/s2.log" 8 15
  sse_pub test.e '{"n":14}' "$t" >/dev/null
  sse_wait_ids "$d/s2.log" 9 15
  sse_kill; return 0; }

# ---------- Kịch bản AC8: đua — phát liên tục trong khi client ngắt / nối lại nhiều lần ----------
sse_ac8_flow() {  # sse_ac8_flow <thư mục> <token> <số sự kiện> <số lần ngắt> → s01.log … s99.log
  local d=$1 t=$2 n=$3 br=$4 i=1 last="" f pp l k
  mkdir -p "$d"
  # Phiên 1 phải mở TRƯỚC khi phát (không có Last-Event-ID thì không đọc bù được sự kiện phát trước lúc nối).
  while [ "$i" -le "$br" ]; do
    f=$d/s$(printf '%02d' "$i").log
    if [ -n "$last" ]; then sse_bg "$f" 12 "$t" -H "Last-Event-ID: $last"; else sse_bg "$f" 12 "$t"; fi
    if [ "$i" = 1 ]; then
      sse_wait_line "$f" '^event: ready' 10 || { sse_kill; return 1; }
      sse_burst "$n" test.seq "$t" 20 >"$d/pub.log" 2>&1 &
      pp=$!
    fi
    sleep "0.$((RANDOM % 5 + 2))"
    sse_kill
    l=$(sse_lastid "$f"); [ -n "$l" ] && last=$l
    i=$((i + 1))
  done
  wait "$pp" 2>/dev/null
  f=$d/s99.log
  if [ -n "$last" ]; then sse_bg "$f" 60 "$t" -H "Last-Event-ID: $last"; else sse_bg "$f" 60 "$t"; fi
  k=0
  while [ "$k" -lt 250 ]; do
    [ "$(sse_ns "$d"/s*.log)" -ge "$n" ] && break; sleep 0.2; k=$((k + 1)); done
  sse_kill; return 0; }

# ---------- Chạy độc lập ----------
sre_main() {
  local cmd=${1:-}; shift 2>/dev/null || true
  case "$cmd" in
    ids|types|datas|lastid) "sse_$cmd" "$1";;
    cap) local f=$1 s=$2 t=$3; shift 3; sse_bg "$f" "$s" "$t" "$@"; echo "$SSE_PID";;
    ac7) local d=${1:-$QC_OUT/sre-ac7} t; t=$(tok STUDENT "$U1"); sse_clean "$U1"; sse_ac7_flow "$d" "$t"
         echo "id5=$(cat "$d/id5" 2>/dev/null)"
         echo "s1_ids=$(sse_ids "$d/s1.log" | tr '\n' ' ')"
         echo "s2_ids=$(sse_ids "$d/s2.log" | tr '\n' ' ')"
         echo "s2_ns=$(sse_datas "$d/s2.log" | sed -n 's/.*"n":\([0-9]*\).*/\1/p' | tr '\n' ' ')";;
    ac8) local n=${1:-1000} br=${2:-20} d=$QC_OUT/sre-ac8 t; t=$(tok STUDENT "$U1"); sse_clean "$U1"
         sse_ac8_flow "$d" "$t" "$n" "$br"
         echo "nhan_duy_nhat=$(sse_ns "$d"/s*.log) mong_doi=$n"
         echo "id_trung=$(cat "$d"/s*.log | grep '^id: ' | sort | uniq -d | grep -c .)";;
    *) sed -n '2,14p' "$SRE_DIR/sse-reconnect.sh";;
  esac; }

case "${0##*/}" in sse-reconnect.sh) sre_main "$@";; esac
