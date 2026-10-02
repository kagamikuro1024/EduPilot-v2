#!/usr/bin/env bash
# QC sprint 2 (PG) — thư viện dùng chung. KHÔNG chạy trực tiếp; được `source` bởi pg01.sh … pg07.sh, gate-pg.sh.
# Nguồn: docs/specs/FEAT-pg-foundation/US.md mục "Quy ước kiểm chung" (C, CT, GW, PSQL, RDS, tok, tmode, dmode, gotest, envv, PGB, U1, U2, H)
# — thêm các hàm chấm điểm (chk*, gt) và dọn dẹp. Tương thích bash 3.2 (macOS): không mảng kết hợp, không ${x,,}, không `date +%s%N`.
# Chạy từ bất kỳ đâu trong worktree: script tự `cd` về gốc repo (REPO=… để ghi đè).
set -u
REPO=${REPO:-$(git rev-parse --show-toplevel 2>/dev/null)}
cd "$REPO" || { echo "KHÔNG VÀO ĐƯỢC $REPO"; exit 2; }

# ---------- Quy ước chung của US.md (nguyên văn, chỉ đổi cách dựng tok cho nhanh) ----------
C="docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot"
CT="$C -f docker-compose.test.yml"
GW=${GW:-https://localhost}
PSQL="$C exec -T postgres psql -U edupilot -d edupilot -v ON_ERROR_STOP=1 -At"
RDS="$C exec -T redis redis-cli"
envv() { grep "^$1=" .env.local | cut -d= -f2-; }
SECRET=$(envv JWT_SECRET_KEY 2>/dev/null)
PGB="postgres://$(envv PGBOUNCER_STATS_USER 2>/dev/null):$(envv PGBOUNCER_STATS_PASSWORD 2>/dev/null)@pgbouncer:6432/pgbouncer"
U1=00000000-0000-7000-8000-000000000001; U2=00000000-0000-7000-8000-000000000002
UX=00000000-0000-7000-8000-000000000009      # uuid lạ (không có job)
PW='Correct-Horse-9'; IDEM=idem-0001-aaaa

QC_TMP=${QC_TMP:-$(mktemp -d /tmp/epqc.XXXXXX)}
QC_OUT=${QC_OUT:-$QC_TMP/out}; mkdir -p "$QC_OUT"
GWBIN=$QC_TMP/gw-default                       # binary mặc định (không tag) — dùng cho `tok` và các TC chạy tiến trình trần
tok() {  # tok <ROLE> [sub-uuid] [cờ của lệnh token, ví dụ --ttl -1m]  (spec: go run ./cmd/gateway token …)
  [ -x "$GWBIN" ] || (cd backend-go && CGO_ENABLED=0 go build -o "$GWBIN" ./cmd/gateway) || return 1
  JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role "$1" ${2:+--sub "$2"} "${@:3}"; }
H="Authorization: Bearer $(tok STUDENT $U1 2>/dev/null)"     # token STUDENT mặc định (U1)
refresh_h() { H="Authorization: Bearer $(tok STUDENT $U1)"; }                  # token sống 15 phút: main() gọi lại trước mỗi TC

tmode() { $CT up -d --build --force-recreate --scale gateway=2 --wait gateway worker; }   # image *-test (route thử)
dmode() { $C  up -d --build --force-recreate --scale gateway=2 --wait gateway worker; }   # image mặc định (không route thử)
gotest() { (cd backend-go && go test -race -count=1 -tags testroutes "$@"); }

# ---------- Chấm điểm ----------
TC_OK=1; TC_MANUAL=0; N_PASS=0; N_FAIL=0; N_MAN=0; FAILED=""; MANUALS=""
chk()    { if [ "$2" = "$3" ]; then echo "    ok   $1 = [$2]"; else echo "    FAIL $1: thực tế [$2] ≠ mong đợi [$3]"; TC_OK=0; fi; }
chk_ne() { if [ "$2" != "$3" ]; then echo "    ok   $1 = [$2] (≠ [$3])"; else echo "    FAIL $1: thực tế [$2] không được bằng [$3]"; TC_OK=0; fi; }
chk_re() { if printf '%s' "$2" | grep -Eq -- "$3"; then echo "    ok   $1 ~ /$3/"; else echo "    FAIL $1: [$2] không khớp /$3/"; TC_OK=0; fi; }
chk_nre(){ if printf '%s' "$2" | grep -Eq -- "$3"; then echo "    FAIL $1: [$2] không được khớp /$3/"; TC_OK=0; else echo "    ok   $1 !~ /$3/"; fi; }
chk_ge() { if [ "${2:-x}" -ge "$3" ] 2>/dev/null; then echo "    ok   $1 = $2 (≥ $3)"; else echo "    FAIL $1: thực tế [$2] phải ≥ $3"; TC_OK=0; fi; }
chk_le() { if [ "${2:-x}" -le "$3" ] 2>/dev/null; then echo "    ok   $1 = $2 (≤ $3)"; else echo "    FAIL $1: thực tế [$2] phải ≤ $3"; TC_OK=0; fi; }
chk_ok() { local d=$1; shift; if "$@" >/dev/null 2>&1; then echo "    ok   $d"; else echo "    FAIL $d (lệnh trả khác 0): $*"; TC_OK=0; fi; }
chk_no() { local d=$1; shift; if "$@" >/dev/null 2>&1; then echo "    FAIL $d (lệnh phải thất bại): $*"; TC_OK=0; else echo "    ok   $d"; fi; }
fail_tc(){ echo "    FAIL $*"; TC_OK=0; }
manual() { TC_MANUAL=1; echo "    MANUAL $* (QC làm tay theo bước ở tc-*.md, ghi kết quả vào report)"; }

# ---------- Tiện ích ----------
now_ms() { perl -MTime::HiRes=time -e 'printf "%d\n", time*1000'; }
code()   { curl -sk --max-time "${CURL_MAX:-20}" -o /dev/null -w '%{http_code}' "$@"; }          # mã HTTP, 000 nếu lỗi kết nối
hdr()    { curl -sk --max-time "${CURL_MAX:-20}" -D- -o /dev/null "$@" | tr -d '\r'; }           # chỉ header
hval()   { awk -F': ' -v k="$(printf '%s' "$1" | tr 'A-Z' 'a-z')" 'tolower($1)==k{sub(/^[^:]*: /,""); print; exit}'; }   # đọc stdin (header), in giá trị một header
jwt_part() { local p; p=$(printf '%s' "$1" | cut -d. -f"$2" | tr '_-' '/+'); while [ $(( ${#p} % 4 )) -ne 0 ]; do p="$p="; done; printf '%s' "$p" | base64 -d 2>/dev/null; }
wait_ready() { local i=0 max=${1:-60}; while [ $i -lt "$max" ]; do [ "$(code $GW/api/v1/readyz)" = 200 ] && return 0; sleep 1; i=$((i+1)); done; return 1; }
gw_ids() { $C ps -q gateway; }
mode_now() { local c; c=$(code -H "Authorization: Bearer $(tok STUDENT $U1)" $GW/api/v1/_test/whoami); case $c in 200) echo test;; 404) echo default;; *) echo "unknown($c)";; esac; }
ensure_mode() {  # ensure_mode test|default — chỉ dựng lại khi chế độ hiện tại khác
  [ "$(mode_now)" = "$1" ] && return 0
  if [ "$1" = test ]; then tmode >/dev/null 2>&1; else dmode >/dev/null 2>&1; fi; wait_ready 90; [ "$(mode_now)" = "$1" ]; }
gw_env() {  # gw_env test|default VAR=giá_trị … — dựng lại gateway (+worker) với biến môi trường ghi đè (compose phải chuyển tiếp biến)
  local m=$1; shift; local cmd=$C; [ "$m" = test ] && cmd=$CT
  env "$@" $cmd up -d --force-recreate --scale gateway=2 --wait gateway >/dev/null 2>&1; wait_ready 60; }
gw_restore() { if [ "$1" = test ]; then tmode >/dev/null 2>&1; else dmode >/dev/null 2>&1; fi; wait_ready 90; }   # trả về cấu hình chuẩn sau TC đổi env
heal() { $C start postgres redis minio pgbouncer >/dev/null 2>&1; wait_ready 60; }                   # bật lại phụ thuộc đã tắt
rl_reset() { $RDS --scan --pattern 'ep:rl:*' 2>/dev/null | while read -r k; do [ -n "$k" ] && $RDS del "$k" </dev/null >/dev/null 2>&1; done; true; }   # xoá bộ đếm rate limit giữa các TC (không phải cách kiểm — chỉ vệ sinh)
rawhttp() {  # rawhttp <host> <port> '<yêu cầu thô, dùng \r\n>' [giây chờ] — chạy TRONG container postgres (có bash): chạm thẳng gateway:8080, không qua Caddy
  $C exec -T postgres bash -c 'exec 3<>/dev/tcp/$0/$1; printf "%b" "$2" >&3; timeout "${3:-3}" cat <&3' "$1" "$2" "$3" "${4:-3}" 2>/dev/null; }
newitem() {  # newitem <tên> [idem-key] [header Authorization đầy đủ] → in id (route thử, chế độ test)
  curl -sk --max-time 20 -X POST -H "${3:-$H}" -H 'Content-Type: application/json' -H "Idempotency-Key: ${2:-qc-$(now_ms)-$RANDOM}" -d "{\"name\":\"$1\"}" $GW/api/v1/_test/items | jq -r '.id // empty'; }
sse_cap() {  # sse_cap <file> <giây> <token> [header thêm …] — chạy nền, ghi stream thô vào file; in PID
  local f=$1 s=$2 t=$3; shift 3
  ( curl -sk -N --max-time "$s" -H "Authorization: Bearer $t" "$@" $GW/api/v1/events > "$f" 2>/dev/null ) & echo $!; }

# ---------- Chạy test Go và CHỨNG MINH test thật sự chạy ----------
# `go test -run X` thoát 0 cả khi KHÔNG test nào khớp ("no tests to run") hoặc test bị SKIP → gt đòi: mỗi tên ở regex có dòng `--- PASS: <tên>`,
# không có `--- FAIL`, không `--- SKIP`, không "no tests to run", rc=0.
#   gt <pkg> '<A|B|C>' [cờ go test thêm …]          có tag testroutes (như gotest của spec)
#   gt_notag <pkg> '<A|B|C>' [cờ …]                  KHÔNG tag (spec: dùng cho TestDefaultBinary_NoTestRoutes, contract không tag)
_gt() {
  local tags=$1 pkg=$2 pat=$3; shift 3; local log=$QC_OUT/gt-$(printf '%s' "$pkg$pat" | tr -c 'A-Za-z0-9' '_' | cut -c1-80).log rc n
  if [ -n "$tags" ]; then (cd backend-go && go test -race -count=1 -timeout 20m -tags "$tags" -v -run "$pat" "$@" $pkg) > "$log" 2>&1; rc=$?
  else (cd backend-go && go test -race -count=1 -timeout 20m -v -run "$pat" "$@" $pkg) > "$log" 2>&1; rc=$?; fi
  chk "go test $pkg -run '$pat' rc" "$rc" 0
  for n in $(printf '%s' "$pat" | tr -d '^$()' | tr '|' ' '); do
    if grep -Eq "^--- PASS: $n( |\$)" "$log"; then echo "    ok   --- PASS: $n"; else fail_tc "không thấy '--- PASS: $n' (test không tồn tại / bị skip / fail) — log: $log"; fi
  done
  grep -E '^(--- FAIL|--- SKIP|FAIL|panic:)' "$log" | head -5 | while read -r l; do echo "    LOG  $l"; done
  [ "$(grep -cE '^--- FAIL|^--- SKIP' "$log")" = 0 ] || TC_OK=0   # #10(2)/#11: gói không có test khớp in 'no tests to run' là bình thường — bằng chứng là dòng '--- PASS' của từng tên
}
gt() { _gt testroutes "$@"; }
gt_notag() { _gt "" "$@"; }

# ---------- Khung chạy: main "$@" ----------
# Mỗi TC là một hàm tc_pgNN_MM. `bash pgNN.sh` chạy hết; `bash pgNN.sh 03 07` hoặc `TC-PG01-03` chạy chọn lọc; `--list` liệt kê.
main() {  # main <số story hai chữ số | gate> [đối số…]
  local st=$1; shift; local fns f id pre idp
  case $st in [0-9][0-9]) pre=tc_pg${st}_; idp=TC-PG${st}-;; *) pre=tc_${st}_; idp=TC-$(printf %s "$st" | tr a-z A-Z)-;; esac
  fns=$(declare -F | awk '{print $3}' | grep "^${pre}" | sort)
  if [ "${1:-}" = "--list" ]; then printf '%s\n' $fns | sed "s/^${pre}/${idp}/"; return 0; fi
  if [ $# -gt 0 ]; then fns=""; for id in "$@"; do id=${id#$idp}; fns="$fns ${pre}$id"; done; fi
  for f in $fns; do
    id=${idp}${f#$pre}
    declare -F "$f" >/dev/null || { echo "?? $id không có hàm $f"; N_FAIL=$((N_FAIL+1)); FAILED="$FAILED $id"; continue; }
    echo "== $id"; TC_OK=1; TC_MANUAL=0
    if [ -n "$($C ps -q gateway 2>/dev/null)" ] && [ "$(code $GW/api/v1/readyz)" != 200 ]; then heal; fi
    refresh_h
    rl_reset
    "$f"
    if [ $TC_MANUAL = 1 ]; then echo "-- $id MANUAL"; N_MAN=$((N_MAN+1)); MANUALS="$MANUALS $id"
    elif [ $TC_OK = 1 ]; then echo "-- $id PASS"; N_PASS=$((N_PASS+1))
    else echo "-- $id FAIL"; N_FAIL=$((N_FAIL+1)); FAILED="$FAILED $id"; fi
  done
  echo "TỔNG story $st: PASS=$N_PASS FAIL=$N_FAIL MANUAL=$N_MAN"
  [ -n "$FAILED" ] && echo "FAIL:$FAILED"; [ -n "$MANUALS" ] && echo "MANUAL:$MANUALS"
  [ "$N_FAIL" = 0 ]
}
