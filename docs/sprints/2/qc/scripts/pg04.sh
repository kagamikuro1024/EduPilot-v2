#!/usr/bin/env bash
# QC US-PG-04 — JWT HS256, 401, RBAC, CourseAccessGuard, bcrypt, không log bí mật, không /auth/*, CLI token, đồng thời.
# Nguồn: docs/specs/FEAT-pg-foundation/US.md v1.1 (US-PG-04 AC1–AC12) + SRS.md 3.4, 4.4, 6.1, 6.2, 6.3, 8.1.
# Hộp đen: chỉ dùng curl / psql / docker / binary CLI / `gt` (test Go có chứng minh). Không đọc mã dev.
#   bash docs/sprints/2/qc/scripts/pg04.sh            # chạy hết, theo thứ tự TC
#   bash docs/sprints/2/qc/scripts/pg04.sh 08 09      # chạy chọn lọc
#   bash docs/sprints/2/qc/scripts/pg04.sh --list
source "$(dirname "$0")/lib.sh"

# ---------- Hằng của story ----------
JOB=$GW/api/v1/jobs/$UX                     # route cần đăng nhập, có ở MỌI chế độ (dùng cho bảng token xấu AC2)
WHO=$GW/api/v1/_test/whoami                 # chỉ chế độ test
RBA=$GW/api/v1/_test/rbac/admin
RBS=$GW/api/v1/_test/rbac/staff
CRS=$GW/api/v1/_test/courses                # $CRS/<courseId>/ping
CID=00000000-0000-7000-8000-0000000000aa    # courseId mẫu của AC7
UUID_RE='^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'
JWT_RE='^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*$'
CLAIM_EMAIL=qc-claim@example.test           # user tạm của TC-PG04-32/33 (dọn ở cuối mỗi TC)
SEC32=0123456789abcdef0123456789abcdef      # secret 32 byte cố định cho các TC chạy binary trần
SEC_OTHER=ffffffffffffffffffffffffffffffff  # secret KHÁC → chữ ký sai

# ---------- Dựng JWT tuỳ ý (không qua CLI) ----------
b64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }     # stdin → base64url không đệm (macOS: không có `base64 -w0`)
mkjwt() {  # mkjwt '<header json>' '<payload json>' [HS256|HS512|none|junk] [secret]
  local hdr pay mode sec data sig
  hdr=$(printf '%s' "$1" | b64url); pay=$(printf '%s' "$2" | b64url)
  mode=${3:-HS256}; sec=${4:-$SECRET}; data="$hdr.$pay"
  case $mode in
    HS256) sig=$(printf '%s' "$data" | openssl dgst -sha256 -hmac "$sec" -binary | b64url);;
    HS512) sig=$(printf '%s' "$data" | openssl dgst -sha512 -hmac "$sec" -binary | b64url);;
    none)  sig="";;
    junk)  sig=$(printf 'chu-ky-rac-khong-phai-RSA-signature-0123456789' | b64url);;
  esac
  printf '%s.%s' "$data" "$sig"
}
HJWT='{"alg":"HS256","typ":"JWT"}'
pay() {  # pay <exp cách now bao nhiêu giây> ['<bộ lọc jq sửa claim>']  → payload JSON một dòng, 9 khoá chuẩn
  local n; n=$(date +%s)
  jq -cn --arg s "$U1" --argjson n "$n" --argjson d "$1" \
    '{sub:$s,role:"STUDENT",email:"qc@example.test",jti:"qcJTI0000000000000000a",iat:$n,nbf:$n,exp:($n+$d),iss:"edupilot",aud:"edupilot-api"}' \
  | { if [ -n "${2:-}" ]; then jq -c "$2"; else cat; fi; }
}

# ---------- Gọi HTTP và chấm ----------
call() {  # call <url> <giá trị header Authorization, hoặc "-" = không gửi header> → RCODE RHDR RBODY
  local url=$1 auth=${2:-}
  if [ "$auth" = "-" ]; then
    RCODE=$(curl -sk --max-time 20 -D "$QC_TMP/h" -o "$QC_TMP/b" -w '%{http_code}' "$url")
  else
    RCODE=$(curl -sk --max-time 20 -D "$QC_TMP/h" -o "$QC_TMP/b" -w '%{http_code}' -H "Authorization: $auth" "$url")
  fi
  RHDR=$(tr -d '\r' < "$QC_TMP/h"); RBODY=$(cat "$QC_TMP/b")
}
jqf() { printf '%s' "$RBODY" | jq -r "$1" 2>/dev/null; }
wwwa() { printf '%s\n' "$RHDR" | hval WWW-Authenticate; }
chk401() {  # chk401 <mô tả> <giá trị header Authorization (hoặc "-")> <code mong đợi> [url]
  local d=$1 w
  call "${4:-$JOB}" "$2"
  chk "$d · status" "$RCODE" 401
  chk "$d · code" "$(jqf '.code // "?"')" "$3"
  w=$(wwwa)
  chk_re "$d · WWW-Authenticate realm" "$w" 'Bearer realm="edupilot"'
  case $3 in
    TOKEN_EXPIRED|TOKEN_INVALID) chk_re "$d · error=invalid_token" "$w" 'error="invalid_token"';;
    UNAUTHENTICATED)             chk_nre "$d · không kèm error=" "$w" 'error=';;
  esac
  chk_re "$d · trace_id 32 hex" "$(jqf '.trace_id // ""')" '^[0-9a-f]{32}$'
  chk "$d · chỉ khoá code/message/trace_id" "$(jqf '[keys[]]|join(",")')" "code,message,trace_id"
}
chk403() {  # chk403 <mô tả> <url> <token> <reason mong đợi: role|course>
  call "$2" "Bearer $3"
  chk "$1 · status" "$RCODE" 403
  chk "$1 · code" "$(jqf '.code // "?"')" FORBIDDEN
  chk "$1 · details.reason" "$(jqf '.details.reason // "?"')" "$4"
  chk_re "$1 · trace_id 32 hex" "$(jqf '.trace_id // ""')" '^[0-9a-f]{32}$'
}
gtlog() { echo "$QC_OUT/gt-$(printf '%s' "$1$2" | tr -c 'A-Za-z0-9' '_' | cut -c1-80).log"; }   # trùng cách đặt tên của lib._gt

# ---------- Chạy gateway trần trên host (AC8 biên env, AC11 CLI) ----------
genv() { printf '%s\n' DATABASE_URL=postgres://u:p@127.0.0.1:1/db REDIS_URL=redis://127.0.0.1:1/0 \
  "JWT_SECRET_KEY=$SEC32" BLOB_ENDPOINT=127.0.0.1:1 BLOB_BUCKET=b BLOB_ACCESS_KEY=ak BLOB_SECRET_KEY=sk STARTUP_TIMEOUT=20s; }
serve_rc() {  # serve_rc <VAR=giá_trị> → in "<rc>|<đường dẫn log>"; dùng cho giá trị env SAI (phải thoát ngay)
  local f=$QC_OUT/serve-$(printf '%s' "$1" | tr -c 'A-Za-z0-9' '_').log
  { printf '%s\n' "$1"; genv; } | xargs env -i PATH="$PATH" "$GWBIN" serve >"$f" 2>&1
  echo "$?|$f"
}
serve_alive() {  # serve_alive <VAR=giá_trị> <giây> → 1 nếu tiến trình còn sống (không thoát vì cấu hình)
  local f=$QC_OUT/serve-alive-$(printf '%s' "$1" | tr -c 'A-Za-z0-9' '_').log p
  ( { printf '%s\n' "$1"; genv; } | xargs env -i PATH="$PATH" "$GWBIN" serve >"$f" 2>&1 ) & p=$!
  sleep "$2"
  local r=0; kill -0 $p 2>/dev/null && r=1
  kill $p 2>/dev/null; pkill -f "$GWBIN serve" 2>/dev/null; wait $p 2>/dev/null; sleep 1   # không để lại tiến trình giữ :8080 cho lần đo sau
  echo $r
}

# ================= AC1 — định dạng token =================
tc_pg04_01() {  # AC1 — header JWT: alg=HS256, typ=JWT, đúng 2 khoá
  local t h; t=$(tok STUDENT $U1); h=$(jwt_part "$t" 1)
  chk "token có 3 đoạn" "$(printf '%s' "$t" | awk -F. '{print NF}')" 3
  chk "header.alg" "$(printf '%s' "$h" | jq -r '.alg // "?"')" HS256
  chk "header.typ" "$(printf '%s' "$h" | jq -r '.typ // "?"')" JWT
  chk "header chỉ có alg,typ" "$(printf '%s' "$h" | jq -r '[keys[]]|join(",")')" "alg,typ"
}
tc_pg04_02() {  # AC1 — payload đúng 9 khoá (lệnh Kiểm nguyên văn của AC1)
  local k; k=$(tok STUDENT $U1 | cut -d. -f2 | tr '_-' '/+' | base64 -d 2>/dev/null | jq -c 'keys')
  chk "keys của payload" "$k" '["aud","email","exp","iat","iss","jti","nbf","role","sub"]'
}
tc_pg04_03() {  # AC1 — exp − iat = JWT_EXPIRATION; mặc định 900; --ttl đổi đúng
  local p
  p=$(jwt_part "$(tok STUDENT $U1)" 2);              chk "exp-iat mặc định" "$(printf '%s' "$p" | jq '.exp-.iat')" 900
  p=$(jwt_part "$(tok STUDENT $U1 --ttl 10m)" 2);    chk "exp-iat với --ttl 10m" "$(printf '%s' "$p" | jq '.exp-.iat')" 600
  p=$(jwt_part "$(tok STUDENT $U1 --ttl 1m)" 2);     chk "exp-iat với --ttl 1m" "$(printf '%s' "$p" | jq '.exp-.iat')" 60
  p=$(jwt_part "$(tok STUDENT $U1 --ttl 1h)" 2);     chk "exp-iat với --ttl 1h" "$(printf '%s' "$p" | jq '.exp-.iat')" 3600
}
tc_pg04_04() {  # AC1 — giá trị claim cố định: iss, aud, sub, role (4 vai), email
  local r p
  for r in ADMIN TEACHER TA STUDENT; do
    p=$(jwt_part "$(tok $r $U1)" 2)
    chk "role=$r" "$(printf '%s' "$p" | jq -r .role)" "$r"
    chk "iss (vai $r)" "$(printf '%s' "$p" | jq -r .iss)" edupilot
    chk "aud (vai $r)" "$(printf '%s' "$p" | jq -r .aud)" edupilot-api
    chk "sub (vai $r)" "$(printf '%s' "$p" | jq -r .sub)" "$U1"
    chk_re "email là chuỗi không rỗng (vai $r)" "$(printf '%s' "$p" | jq -r .email)" '.+@.+'
  done
  p=$(jwt_part "$(tok TEACHER $U2)" 2); chk "sub theo --sub U2" "$(printf '%s' "$p" | jq -r .sub)" "$U2"
}
tc_pg04_05() {  # AC1 — iat/nbf: nbf ≤ iat, iat ≈ bây giờ (±120 s), exp > iat
  local p n; n=$(date +%s); p=$(jwt_part "$(tok STUDENT $U1)" 2)
  local iat nbf exp; iat=$(printf '%s' "$p" | jq -r .iat); nbf=$(printf '%s' "$p" | jq -r .nbf); exp=$(printf '%s' "$p" | jq -r .exp)
  chk_le "nbf ≤ iat" "$nbf" "$iat"
  chk_le "|iat - now| ≤ 120 (iat=$iat now=$n)" "$(( iat > n ? iat-n : n-iat ))" 120
  chk_ge "exp > iat" "$exp" "$((iat+1))"
  chk_re "iat là số nguyên" "$iat" '^[0-9]+$'
}
tc_pg04_06() {  # AC1 — jti: base64url 22 ký tự (128 bit), 200 lần cấp = 200 giá trị duy nhất
  local f=$QC_OUT/tc-pg04-06.jti i
  : > "$f"
  for i in $(seq 1 200); do jwt_part "$(tok STUDENT $U1)" 2 | jq -r .jti >> "$f"; done
  chk "số token cấp" "$(grep -c . "$f" | tr -d ' ')" 200
  chk "số jti duy nhất" "$(sort -u "$f" | grep -c . | tr -d ' ')" 200
  chk "số jti dài 22 ký tự base64url" "$(grep -cE '^[A-Za-z0-9_-]{22}$' "$f" | tr -d ' ')" 200
}
tc_pg04_07() {  # AC1 — test Go (10.000 lần cấp)
  gt ./internal/auth 'TestJWT_Claims|TestJWT_JTIUnique'
}

# ================= AC2 — token sai (SRS 3.4 "Token hết hạn / sai") =================
tc_pg04_08() {  # AC2 — token hết hạn → 401 TOKEN_EXPIRED
  chk401 "hết hạn (--ttl -1m)" "Bearer $(tok STUDENT $U1 --ttl -1m)" TOKEN_EXPIRED
  chk401 "hết hạn 1 giờ (mkjwt exp=now-3600)" "Bearer $(mkjwt "$HJWT" "$(pay -3600)")" TOKEN_EXPIRED
}
tc_pg04_09() {  # AC2 — sai chữ ký → TOKEN_INVALID (hai cách: sửa 1 ký tự, ký bằng secret khác)
  local t s; t=$(tok STUDENT $U1)
  s=$(printf '%s' "$t" | cut -d. -f3); case $s in *A) s="${s%A}B";; *) s="${s%?}A";; esac
  chk401 "chữ ký đổi 1 ký tự" "Bearer $(printf '%s' "$t" | cut -d. -f1,2).$s" TOKEN_INVALID
  chk401 "ký bằng secret khác" "Bearer $(mkjwt "$HJWT" "$(pay 900)" HS256 "$SEC_OTHER")" TOKEN_INVALID
  chk401 "chữ ký rỗng (3 đoạn, đoạn 3 trống)" "Bearer $(printf '%s' "$t" | cut -d. -f1,2)." TOKEN_INVALID
}
tc_pg04_10() {  # AC2 — alg=none → TOKEN_INVALID
  chk401 'alg=none' "Bearer $(mkjwt '{"alg":"none","typ":"JWT"}' "$(pay 900)" none)" TOKEN_INVALID
}
tc_pg04_11() {  # AC2 — alg=HS512 ký ĐÚNG bằng chính secret → vẫn TOKEN_INVALID (chỉ chấp nhận HS256)
  chk401 'alg=HS512 (chữ ký hợp lệ theo HS512)' "Bearer $(mkjwt '{"alg":"HS512","typ":"JWT"}' "$(pay 900)" HS512)" TOKEN_INVALID
}
tc_pg04_12() {  # AC2 — alg=RS256 → TOKEN_INVALID
  chk401 'alg=RS256 (chữ ký rác)' "Bearer $(mkjwt '{"alg":"RS256","typ":"JWT"}' "$(pay 900)" junk)" TOKEN_INVALID
}
tc_pg04_13() {  # AC2 — thiếu exp → TOKEN_INVALID
  chk401 'thiếu claim exp' "Bearer $(mkjwt "$HJWT" "$(pay 900 'del(.exp)')")" TOKEN_INVALID
}
tc_pg04_14() {  # AC2 — role ngoài danh sách → TOKEN_INVALID
  chk401 'role=SUPERUSER' "Bearer $(mkjwt "$HJWT" "$(pay 900 '.role="SUPERUSER"')")" TOKEN_INVALID
  chk401 'role="student" (chữ thường, ngoài 4 giá trị)' "Bearer $(mkjwt "$HJWT" "$(pay 900 '.role="student"')")" TOKEN_INVALID
  chk401 'thiếu claim role' "Bearer $(mkjwt "$HJWT" "$(pay 900 'del(.role)')")" TOKEN_INVALID
}
tc_pg04_15() {  # AC2 — sai iss → TOKEN_INVALID
  chk401 'iss="khac"' "Bearer $(mkjwt "$HJWT" "$(pay 900 '.iss="khac"')")" TOKEN_INVALID
  chk401 'thiếu iss' "Bearer $(mkjwt "$HJWT" "$(pay 900 'del(.iss)')")" TOKEN_INVALID
}
tc_pg04_16() {  # AC2 — sai aud → TOKEN_INVALID
  chk401 'aud="edupilot-web"' "Bearer $(mkjwt "$HJWT" "$(pay 900 '.aud="edupilot-web"')")" TOKEN_INVALID
  chk401 'thiếu aud' "Bearer $(mkjwt "$HJWT" "$(pay 900 'del(.aud)')")" TOKEN_INVALID
}
tc_pg04_17() {  # AC2 — nbf ở tương lai → TOKEN_INVALID
  chk401 'nbf = iat + 600' "Bearer $(mkjwt "$HJWT" "$(pay 900 '.nbf=(.iat+600)')")" TOKEN_INVALID
}
tc_pg04_18() {  # AC2 — chuỗi hỏng → TOKEN_INVALID
  local t; t=$(tok STUDENT $U1)
  chk401 'chỉ 2 đoạn' "Bearer $(printf '%s' "$t" | cut -d. -f1,2)" TOKEN_INVALID
  chk401 'rác "garbage"' "Bearer garbage" TOKEN_INVALID
  chk401 '4 đoạn' "Bearer $t.$(printf '%s' "$t" | cut -d. -f3)" TOKEN_INVALID
  chk401 'payload không phải base64url hợp lệ' "Bearer $(printf '%s' "$t" | cut -d. -f1).!!!.$(printf '%s' "$t" | cut -d. -f3)" TOKEN_INVALID
}
tc_pg04_19() {  # AC2 — không / sai kiểu header Authorization → UNAUTHENTICATED (và WWW-Authenticate KHÔNG có error=)
  chk401 'không gửi header Authorization' "-" UNAUTHENTICATED
  chk401 'Authorization: Basic …' "Basic $(printf 'u:p' | b64url)" UNAUTHENTICATED
  chk401 'Bearer rỗng ("Bearer ")' "Bearer " UNAUTHENTICATED
  chk401 'Authorization rỗng' "" UNAUTHENTICATED
}
tc_pg04_20() {  # AC2 — thân lỗi không lộ lý do: mọi TOKEN_INVALID cùng message, không nêu alg/iss/aud/signature
  local b f=$QC_OUT/tc-pg04-20.msg m
  : > "$f"
  for b in "$(mkjwt "$HJWT" "$(pay 900)" HS256 "$SEC_OTHER")" \
           "$(mkjwt '{"alg":"none","typ":"JWT"}' "$(pay 900)" none)" \
           "$(mkjwt "$HJWT" "$(pay 900 '.iss="khac"')")" \
           "$(mkjwt "$HJWT" "$(pay 900 '.aud="khac"')")" \
           "$(mkjwt "$HJWT" "$(pay 900 'del(.exp)')")" \
           garbage; do
    call "$JOB" "Bearer $b"
    chk "mã lỗi" "$(jqf '.code // "?"')" TOKEN_INVALID
    jqf '.message' >> "$f"
  done
  chk "số message khác nhau giữa các TOKEN_INVALID" "$(sort -u "$f" | grep -c . | tr -d ' ')" 1
  m=$(sort -u "$f" | head -1)
  chk_nre "message không nêu lý do kỹ thuật" "$m" '[Aa][Ll][Gg]|HS256|HS512|RS256|[Ss]ignature|chữ ký|"iss"|"aud"|"nbf"|"exp"|base64|JWT|[Cc]laim'
  chk_nre "message không chứa token" "$m" 'eyJ'
}
tc_pg04_21() {  # AC2 + SRS 3.4 "không truy DB" — Postgres dừng, 401 vẫn đúng mã
  local pg o1 o2 o3; pg=$($C ps -q postgres)
  docker pause "$pg" >/dev/null 2>&1
  o1=$(curl -sk --max-time 20 -H "Authorization: Bearer $(tok STUDENT $U1 --ttl -1m)" "$JOB" | jq -r '.code // "?"')
  o2=$(curl -sk --max-time 20 -H "Authorization: Bearer garbage" "$JOB" | jq -r '.code // "?"')
  o3=$(curl -sk --max-time 20 "$JOB" | jq -r '.code // "?"')
  docker unpause "$pg" >/dev/null 2>&1; wait_ready 60
  chk "DB dừng · token hết hạn" "$o1" TOKEN_EXPIRED
  chk "DB dừng · token hỏng" "$o2" TOKEN_INVALID
  chk "DB dừng · không token" "$o3" UNAUTHENTICATED
}
tc_pg04_22() {  # AC2 — test Go bảng ≥ 14 dòng
  gt ./internal/auth 'TestVerify_Table'
  local l n; l=$(gtlog ./internal/auth 'TestVerify_Table'); n=$(grep -c -- '--- PASS: TestVerify_Table/' "$l" 2>/dev/null | tr -d ' ')
  chk_ge "số dòng bảng (subtest PASS của TestVerify_Table)" "${n:-0}" 14
}

# ================= AC3 — leeway 5 s =================
tc_pg04_23() { gt ./internal/auth 'TestVerify_Leeway'; }   # AC3 — test Go (đồng hồ giả)
tc_pg04_24() {  # AC3 — biên thật qua HTTP: hết hạn 3 s / 4 s → chấp nhận; 7 s / 10 s → TOKEN_EXPIRED
  ensure_mode test || fail_tc "không vào được chế độ test"
  local c
  c=$(code -H "Authorization: Bearer $(mkjwt "$HJWT" "$(pay 900)")" "$WHO");  chk "token mkjwt còn hạn (đối chứng dương)" "$c" 200
  c=$(code -H "Authorization: Bearer $(mkjwt "$HJWT" "$(pay -3)")" "$WHO");   chk "hết hạn 3 s (trong leeway)" "$c" 200
  c=$(code -H "Authorization: Bearer $(mkjwt "$HJWT" "$(pay -4)")" "$WHO");   chk "hết hạn 4 s (trong leeway)" "$c" 200
  call "$WHO" "Bearer $(mkjwt "$HJWT" "$(pay -7)")"
  chk "hết hạn 7 s · status" "$RCODE" 401; chk "hết hạn 7 s · code" "$(jqf '.code // "?"')" TOKEN_EXPIRED
  call "$WHO" "Bearer $(mkjwt "$HJWT" "$(pay -10)")"
  chk "hết hạn 10 s · status" "$RCODE" 401; chk "hết hạn 10 s · code" "$(jqf '.code // "?"')" TOKEN_EXPIRED
}

# ================= AC4 — danh tính từ claim, không truy DB =================
tc_pg04_25() {  # AC4 — Postgres dừng, whoami vẫn 200 và đúng sub/role cho cả 4 vai
  ensure_mode test || fail_tc "không vào được chế độ test"
  local pg r out; pg=$($C ps -q postgres); : > "$QC_OUT/tc-pg04-25.out"
  docker pause "$pg" >/dev/null 2>&1
  for r in ADMIN TEACHER TA STUDENT; do
    printf '%s %s %s\n' "$r" \
      "$(code -H "Authorization: Bearer $(tok $r $U1)" "$WHO")" \
      "$(curl -sk --max-time 20 -H "Authorization: Bearer $(tok $r $U1)" "$WHO" | jq -c '[.sub,.role]')" >> "$QC_OUT/tc-pg04-25.out"
  done
  printf '%s %s %s\n' "STUDENT-U2" \
    "$(code -H "Authorization: Bearer $(tok STUDENT $U2)" "$WHO")" \
    "$(curl -sk --max-time 20 -H "Authorization: Bearer $(tok STUDENT $U2)" "$WHO" | jq -c '[.sub,.role]')" >> "$QC_OUT/tc-pg04-25.out"
  docker unpause "$pg" >/dev/null 2>&1; wait_ready 60
  while read -r r c body; do
    chk "DB dừng · whoami vai $r · status" "$c" 200
    case $r in
      STUDENT-U2) chk "DB dừng · whoami $r · [sub,role]" "$body" "[\"$U2\",\"STUDENT\"]";;
      *)          chk "DB dừng · whoami $r · [sub,role]" "$body" "[\"$U1\",\"$r\"]";;
    esac
  done < "$QC_OUT/tc-pg04-25.out"
}
tc_pg04_26() {  # AC4 — 0 truy vấn DB: xact_commit không tăng quá mức nền trong cửa sổ 20 request whoami
  ensure_mode test || fail_tc "không vào được chế độ test"
  local xq="select xact_commit from pg_stat_database where datname='edupilot'"
  local a b c d t0 t1 t2 t3 i rate exp
  t0=$(now_ms); a=$($PSQL -c "$xq"); sleep 5; b=$($PSQL -c "$xq"); t1=$(now_ms)
  rate=$(( (b-a) * 1000 / (t1-t0) ))                      # giao dịch nền mỗi giây (worker poll outbox, healthcheck…)
  t2=$(now_ms); c=$($PSQL -c "$xq")
  for i in $(seq 1 20); do curl -sk --max-time 20 -o /dev/null -H "Authorization: Bearer $(tok TA $U1)" "$WHO"; done
  d=$($PSQL -c "$xq"); t3=$(now_ms)
  exp=$(( rate * (t3-t2) / 1000 + 3 ))                    # +3: hai lệnh psql của phép đo + 1 dung sai
  echo "    info nền=${rate}/s  cửa sổ=$(( (t3-t2) ))ms  delta=$((d-c))  ngưỡng=$exp"
  chk_le "giao dịch DB trong 20 request whoami" "$((d-c))" "$exp"
}
tc_pg04_27() { gt ./internal/auth 'TestAuth_NoDBQueryPerRequest'; }   # AC4 — bộ đếm tracer = 0

# ================= AC5 — RBAC (phân quyền) =================
tc_pg04_28() {  # AC5 — ma trận 4 vai × 2 route
  ensure_mode test || fail_tc "không vào được chế độ test"
  local r t
  for r in ADMIN TEACHER TA STUDENT; do
    t=$(tok $r $U1)
    case $r in
      ADMIN)   chk "rbac/admin vai ADMIN" "$(code -H "Authorization: Bearer $t" "$RBA")" 200
               chk "rbac/staff vai ADMIN" "$(code -H "Authorization: Bearer $t" "$RBS")" 403;;
      TEACHER) chk "rbac/admin vai TEACHER" "$(code -H "Authorization: Bearer $t" "$RBA")" 403
               chk "rbac/staff vai TEACHER" "$(code -H "Authorization: Bearer $t" "$RBS")" 200;;
      TA)      chk "rbac/admin vai TA" "$(code -H "Authorization: Bearer $t" "$RBA")" 403
               chk "rbac/staff vai TA" "$(code -H "Authorization: Bearer $t" "$RBS")" 200;;
      STUDENT) chk "rbac/admin vai STUDENT" "$(code -H "Authorization: Bearer $t" "$RBA")" 403
               chk "rbac/staff vai STUDENT" "$(code -H "Authorization: Bearer $t" "$RBS")" 403;;
    esac
  done
}
tc_pg04_29() {  # AC5 — ẩn danh ở cả hai route RBAC → 401 UNAUTHENTICATED (không phải 403/404)
  ensure_mode test || fail_tc "không vào được chế độ test"
  chk401 "ẩn danh · rbac/admin" "-" UNAUTHENTICATED "$RBA"
  chk401 "ẩn danh · rbac/staff" "-" UNAUTHENTICATED "$RBS"
}
tc_pg04_30() {  # AC5 — thân 403: code=FORBIDDEN, details.reason="role"
  ensure_mode test || fail_tc "không vào được chế độ test"
  chk403 "STUDENT → rbac/admin" "$RBA" "$(tok STUDENT $U1)" role
  chk403 "ADMIN → rbac/staff"   "$RBS" "$(tok ADMIN $U1)"   role
  chk "khoá của thân 403" "$(jqf '[keys[]]|sort|join(",")')" "code,details,message,trace_id"
  chk_nre "message 403 không nêu vai cần có" "$(jqf '.message')" 'ADMIN|TEACHER|STUDENT|[Aa]dmin|[Tt]eacher|[Ss]tudent'
}
tc_pg04_31() { gt ./internal/auth 'TestRBAC_Matrix'; }   # AC5 — test Go

# ================= AC6 — claim thắng DB (phân quyền) =================
tc_pg04_32() {  # AC6 — DB ghi ADMIN, token STUDENT → 403
  ensure_mode test || fail_tc "không vào được chế độ test"
  $PSQL -c "insert into users (id,email,full_name,role) values ('$U1','$CLAIM_EMAIL','QC Claim','ADMIN')
            on conflict (id) do update set role='ADMIN', email='$CLAIM_EMAIL'" >/dev/null 2>&1
  chk "users.role trong DB" "$($PSQL -c "select role from users where id='$U1'")" ADMIN
  chk403 "DB=ADMIN, claim=STUDENT → rbac/admin" "$RBA" "$(tok STUDENT $U1)" role
  $PSQL -c "delete from users where id='$U1' and email='$CLAIM_EMAIL'" >/dev/null 2>&1
  chk "đã dọn user tạm" "$($PSQL -c "select count(*) from users where email='$CLAIM_EMAIL'")" 0
}
tc_pg04_33() {  # AC6 (chiều ngược) — DB ghi STUDENT, token ADMIN → 200
  ensure_mode test || fail_tc "không vào được chế độ test"
  $PSQL -c "insert into users (id,email,full_name,role) values ('$U1','$CLAIM_EMAIL','QC Claim','STUDENT')
            on conflict (id) do update set role='STUDENT', email='$CLAIM_EMAIL'" >/dev/null 2>&1
  local c; c=$(code -H "Authorization: Bearer $(tok ADMIN $U1)" "$RBA")
  $PSQL -c "delete from users where id='$U1' and email='$CLAIM_EMAIL'" >/dev/null 2>&1
  chk "DB=STUDENT, claim=ADMIN → rbac/admin" "$c" 200
  chk "đã dọn user tạm" "$($PSQL -c "select count(*) from users where email='$CLAIM_EMAIL'")" 0
}
tc_pg04_34() { gt ./internal/auth 'TestRBAC_ClaimWinsOverDB'; }   # AC6 — test Go

# ================= AC7 — CourseAccessGuard =================
tc_pg04_35() {  # AC7 — mặc định từ chối tất cả, kể cả ADMIN
  ensure_mode test || fail_tc "không vào được chế độ test"
  local r
  for r in ADMIN TEACHER TA STUDENT; do
    chk "guard · vai $r → 403" "$(code -H "Authorization: Bearer $(tok $r $U1)" "$CRS/$CID/ping")" 403
  done
  chk "guard · vai STUDENT, sub khác (U2) → 403" "$(code -H "Authorization: Bearer $(tok STUDENT $U2)" "$CRS/$CID/ping")" 403
}
tc_pg04_36() {  # AC7 — thân 403 của guard: details.reason="course" (khác "role" của AC5)
  ensure_mode test || fail_tc "không vào được chế độ test"
  chk403 "guard ADMIN" "$CRS/$CID/ping" "$(tok ADMIN $U1)" course
  chk403 "guard STUDENT" "$CRS/$CID/ping" "$(tok STUDENT $U1)" course
}
tc_pg04_37() {  # AC7 — courseId không phải uuid → 404 NOT_FOUND (JSON), cả hai phía biên độ dài uuid
  ensure_mode test || fail_tc "không vào được chế độ test"
  local t id; t=$(tok ADMIN $U1)
  for id in abc 123 00000000-0000-7000-8000-0000000000a 00000000-0000-7000-8000-0000000000aaa not-a-uuid-at-all; do
    call "$CRS/$id/ping" "Bearer $t"
    chk "courseId='$id' · status" "$RCODE" 404
    chk "courseId='$id' · code" "$(jqf '.code // "?"')" NOT_FOUND
    chk_re "courseId='$id' · trace_id" "$(jqf '.trace_id // ""')" '^[0-9a-f]{32}$'
  done
  chk "courseId uuid hợp lệ vẫn vào guard (403, không 404)" "$(code -H "Authorization: Bearer $t" "$CRS/$CID/ping")" 403
}
tc_pg04_38() {  # AC7 + AC2 — ẩn danh ở route guard → 401 (xác thực trước guard). Chờ Q-QC-04-3.
  ensure_mode test || fail_tc "không vào được chế độ test"
  chk401 "ẩn danh · guard ping" "-" UNAUTHENTICATED "$CRS/$CID/ping"
  chk401 "token hỏng · guard ping" "Bearer garbage" TOKEN_INVALID "$CRS/$CID/ping"
}
tc_pg04_39() {  # AC7 — resolver giả / lỗi (503) / không cache / gọi đúng 1 lần: chỉ kiểm được bằng test Go
  gt ./internal/auth 'TestCourseAccessGuard_DefaultDenyAll|TestCourseAccessGuard_Membership|TestCourseAccessGuard_BadID|TestCourseAccessGuard_ResolverError|TestCourseAccessGuard_NoCache'
}

# ================= AC8 — bcrypt =================
tc_pg04_40() {  # AC8 — test Go + chứng minh có nhánh 72/73 byte trong log
  gt ./internal/auth 'TestBcrypt_Format|TestBcrypt_DefaultCost|TestBcrypt_TooLong|TestBcrypt_Check'
  local l n
  l=$(gtlog ./internal/auth 'TestBcrypt_Format|TestBcrypt_DefaultCost|TestBcrypt_TooLong|TestBcrypt_Check')
  n=$(grep -c -- '--- PASS: TestBcrypt_' "$l" 2>/dev/null | tr -d ' ')
  chk_ge "số dòng PASS TestBcrypt_* (4 test + subtest)" "${n:-0}" 4
  chk_no "log test không chứa mật khẩu rõ '$PW'" grep -q -- "$PW" "$l"
  chk_no "log test không chứa chuỗi bcrypt" grep -qE '\$2[aby]\$[0-9]{2}\$' "$l"
}
tc_pg04_41() {  # AC8 (biên env) — BCRYPT_COST: 3 và 15 → thoát 1 nêu tên biến; 4 và 14 → chạy tiếp. Chờ Q-QC-04-4.
  local r rc l v
  for v in 3 15; do
    r=$(serve_rc "BCRYPT_COST=$v"); rc=${r%%|*}; l=${r#*|}
    chk "BCRYPT_COST=$v · rc" "$rc" 1
    chk_ok "BCRYPT_COST=$v · log nêu tên biến" grep -q BCRYPT_COST "$l"
  done
  for v in 4 14; do
    chk "BCRYPT_COST=$v · tiến trình không thoát vì cấu hình (sống sau 3 s)" "$(serve_alive "BCRYPT_COST=$v" 3)" 1
  done
}

# ================= AC9 — không log bí mật =================
tc_pg04_42() { gt ./internal/auth ./internal/httpapi 'TestAuth_NoSecretsInLogs'; }   # AC9 — test Go
tc_pg04_43() {  # AC9 — hộp đen: 20 request có token rồi soi log container
  ensure_mode test || fail_tc "không vào được chế độ test"
  local t jti i lg=$QC_OUT/tc-pg04-43.log
  t=$(tok STUDENT $U1); jti=$(jwt_part "$t" 2 | jq -r .jti)
  for i in $(seq 1 20); do curl -sk --max-time 20 -o /dev/null -H "Authorization: Bearer $t" "$WHO"; done
  curl -sk --max-time 20 -o /dev/null -H "Authorization: Bearer $(tok STUDENT $U1 --ttl -1m)" "$JOB"
  curl -sk --max-time 20 -o /dev/null -H "Authorization: Bearer garbage" "$JOB"
  sleep 2
  $C logs --no-log-prefix --since 5m gateway worker > "$lg" 2>&1
  chk_ge "log có nội dung để soi" "$(grep -c . "$lg" | tr -d ' ')" 1
  chk "số dòng log chứa token đầy đủ" "$(grep -c -- "$t" "$lg" | tr -d ' ')" 0
  chk "số dòng log chứa đoạn chữ ký của token" "$(grep -c -- "$(printf '%s' "$t" | cut -d. -f3)" "$lg" | tr -d ' ')" 0
  chk "số dòng log chứa JWT_SECRET_KEY" "$(grep -c -- "$SECRET" "$lg" | tr -d ' ')" 0
  chk "số dòng log chứa mật khẩu thử '$PW'" "$(grep -c -- "$PW" "$lg" | tr -d ' ')" 0
  chk "số dòng log chứa hash bcrypt" "$(grep -cE '\$2[aby]\$[0-9]{2}\$' "$lg" | tr -d ' ')" 0
  chk "số dòng log chứa jti đầy đủ (22 ký tự)" "$(grep -c -- "$jti" "$lg" | tr -d ' ')" 0
  chk "số dòng log chứa 'Bearer '" "$(grep -c -- 'Bearer ' "$lg" | tr -d ' ')" 0
}

# ================= AC10 — không làm trùng P2 =================
tc_pg04_44() {  # AC10 — openapi.yaml không có /auth/, /me/, /admin/
  local f=backend-go/api/openapi.yaml
  chk_ok "có tệp $f" test -f "$f"
  chk "grep -c '/auth/' $f" "$(grep -c '/auth/' "$f" 2>/dev/null | tr -d ' ')" 0
  chk "grep -c '/me/' $f" "$(grep -c '/me/' "$f" 2>/dev/null | tr -d ' ')" 0
  chk "grep -c '/admin/' $f" "$(grep -c '/admin/' "$f" 2>/dev/null | tr -d ' ')" 0
}
tc_pg04_45() {  # AC10 — chế độ TEST: /auth/*, /me*, /admin/* đều 404 JSON NOT_FOUND
  ensure_mode test || fail_tc "không vào được chế độ test"
  local p
  for p in auth/login auth/register auth/refresh auth/verify me me/profile admin/x admin/users; do
    call "$GW/api/v1/$p" "Bearer $(tok ADMIN $U1)"
    chk "GET /api/v1/$p (chế độ test) · status" "$RCODE" 404
    chk "GET /api/v1/$p · code" "$(jqf '.code // "?"')" NOT_FOUND
  done
  chk "POST /api/v1/auth/login (chế độ test)" "$(code -X POST -H "Authorization: Bearer $(tok ADMIN $U1)" "$GW/api/v1/auth/login")" 404
}
tc_pg04_46() {  # AC10 — chế độ DEFAULT (image mặc định): lệnh Kiểm nguyên văn + thân JSON NOT_FOUND
  ensure_mode default || fail_tc "không vào được chế độ default"
  chk "POST /api/v1/auth/login" "$(code -X POST "$GW/api/v1/auth/login")" 404
  call "$GW/api/v1/auth/login" "-"
  chk "GET /api/v1/auth/login · code" "$(jqf '.code // "?"')" NOT_FOUND
  chk_re "GET /api/v1/auth/login · trace_id" "$(jqf '.trace_id // ""')" '^[0-9a-f]{32}$'
  local p
  for p in me admin/users auth/refresh; do
    chk "GET /api/v1/$p (chế độ default)" "$(code "$GW/api/v1/$p")" 404
  done
}

# ================= AC11 — CLI `gateway token` =================
tc_pg04_47() {  # AC11 — chạy bình thường: rc=0, stdout đúng 1 dòng là JWT, stderr không chứa token
  local o=$QC_OUT/tc-pg04-47.out e=$QC_OUT/tc-pg04-47.err rc
  JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role ADMIN --sub "$U1" >"$o" 2>"$e"; rc=$?
  chk "rc" "$rc" 0
  chk "số dòng stdout" "$(grep -c . "$o" | tr -d ' ')" 1
  chk_re "stdout là JWT 3 đoạn" "$(cat "$o")" "$JWT_RE"
  chk "stderr không chứa token" "$(grep -c -- "$(cat "$o")" "$e" | tr -d ' ')" 0
  chk "claim role của token in ra" "$(jwt_part "$(cat "$o")" 2 | jq -r .role)" ADMIN
}
tc_pg04_48() {  # AC11 — APP_ENV=production → rc=1 và KHÔNG in token (cả stdout lẫn stderr)
  local o=$QC_OUT/tc-pg04-48.out e=$QC_OUT/tc-pg04-48.err rc
  APP_ENV=production JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role ADMIN >"$o" 2>"$e"; rc=$?
  chk "rc" "$rc" 1
  chk "số dòng stdout khớp dạng JWT" "$(grep -cE "$JWT_RE" "$o" | tr -d ' ')" 0
  chk "số dòng stderr khớp dạng JWT" "$(grep -cE "$JWT_RE" "$e" | tr -d ' ')" 0
  chk_ge "có thông báo lỗi" "$(cat "$o" "$e" | grep -c . | tr -d ' ')" 1
}
tc_pg04_49() {  # AC11 — thiếu JWT_SECRET_KEY → rc=1 và nêu TÊN biến (không in giá trị nào)
  local o=$QC_OUT/tc-pg04-49.out e=$QC_OUT/tc-pg04-49.err rc
  env -i PATH="$PATH" "$GWBIN" token --role ADMIN >"$o" 2>"$e"; rc=$?
  chk "rc" "$rc" 1
  chk_ok "thông báo nêu tên biến JWT_SECRET_KEY" grep -q JWT_SECRET_KEY "$o" "$e"
  chk "không in token" "$(cat "$o" "$e" | grep -cE "$JWT_RE" | tr -d ' ')" 0
}
tc_pg04_50() {  # AC11 — --role sai / thiếu → rc≠0, không in token
  local o=$QC_OUT/tc-pg04-50.out e=$QC_OUT/tc-pg04-50.err rc
  JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role ROOT >"$o" 2>"$e"; rc=$?
  chk "--role ROOT · rc" "$rc" 1
  chk "--role ROOT · không in token" "$(cat "$o" "$e" | grep -cE "$JWT_RE" | tr -d ' ')" 0
  JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role student >"$o" 2>"$e"; rc=$?
  chk "--role student (chữ thường) · rc" "$rc" 1
  JWT_SECRET_KEY="$SECRET" "$GWBIN" token >"$o" 2>"$e"; rc=$?
  chk_ne "thiếu --role · rc" "$rc" 0
  chk "thiếu --role · không in token" "$(cat "$o" "$e" | grep -cE "$JWT_RE" | tr -d ' ')" 0
}
tc_pg04_51() {  # AC11 — --ttl hỏng → không âm thầm cấp token. Chờ Q-QC-04-2.
  local o=$QC_OUT/tc-pg04-51.out e=$QC_OUT/tc-pg04-51.err rc
  JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role ADMIN --ttl abc >"$o" 2>"$e"; rc=$?
  chk_ne "--ttl abc · rc" "$rc" 0
  chk "--ttl abc · không in token" "$(cat "$o" "$e" | grep -cE "$JWT_RE" | tr -d ' ')" 0
  JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role ADMIN --ttl '' >"$o" 2>"$e"; rc=$?
  chk_ne "--ttl rỗng · rc" "$rc" 0
}
tc_pg04_52() {  # AC11 + AC1 — cờ tuỳ chọn: bỏ --sub → sub vẫn là uuid; --email vào đúng claim
  local t p
  t=$(JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role TEACHER); p=$(jwt_part "$t" 2)
  chk_re "bỏ --sub → claim sub là uuid" "$(printf '%s' "$p" | jq -r .sub)" "$UUID_RE"
  t=$(JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role TEACHER); chk_ne "hai lần cấp cho hai jti khác nhau" \
    "$(jwt_part "$t" 2 | jq -r .jti)" "$(printf '%s' "$p" | jq -r .jti)"
  t=$(JWT_SECRET_KEY="$SECRET" "$GWBIN" token --role TA --sub "$U2" --email qc-tc52@example.test)
  p=$(jwt_part "$t" 2)
  chk "--email vào claim email" "$(printf '%s' "$p" | jq -r .email)" qc-tc52@example.test
  chk "--sub vào claim sub" "$(printf '%s' "$p" | jq -r .sub)" "$U2"
}
tc_pg04_53() { gt ./cmd/gateway 'TestTokenCommand'; }   # AC11 — test Go

# ================= AC12 — đồng thời, không trạng thái toàn cục =================
tc_pg04_54() { gt ./internal/auth 'TestPrincipal_ContextOnly|TestAuth_ParallelRequests' -count=20; }   # AC12 — -count=20 đè -count=1 của lib
tc_pg04_55() {  # AC12 — 200 request song song, 4 token (4 vai, 2 sub): không lẫn danh tính
  ensure_mode test || fail_tc "không vào được chế độ test"
  local w i
  local tA tT tTA tS
  tA=$(tok ADMIN $U1); tT=$(tok TEACHER $U1); tTA=$(tok TA $U2); tS=$(tok STUDENT $U2)
  rm -f "$QC_OUT"/tc-pg04-55.*
  for w in 1 2 3 4 5; do                     # 5 đợt × 40 request song song = 200
    for i in $(seq 1 10); do
      ( curl -sk --max-time 25 -H "Authorization: Bearer $tA"  "$WHO" | jq -r '[.sub,.role]|join("|")' >> "$QC_OUT/tc-pg04-55.ADMIN" ) &
      ( curl -sk --max-time 25 -H "Authorization: Bearer $tT"  "$WHO" | jq -r '[.sub,.role]|join("|")' >> "$QC_OUT/tc-pg04-55.TEACHER" ) &
      ( curl -sk --max-time 25 -H "Authorization: Bearer $tTA" "$WHO" | jq -r '[.sub,.role]|join("|")' >> "$QC_OUT/tc-pg04-55.TA" ) &
      ( curl -sk --max-time 25 -H "Authorization: Bearer $tS"  "$WHO" | jq -r '[.sub,.role]|join("|")' >> "$QC_OUT/tc-pg04-55.STUDENT" ) &
    done
    wait
  done
  chk "ADMIN · số phản hồi" "$(grep -c . "$QC_OUT/tc-pg04-55.ADMIN" | tr -d ' ')" 50
  chk "ADMIN · giá trị duy nhất" "$(sort -u "$QC_OUT/tc-pg04-55.ADMIN")" "$U1|ADMIN"
  chk "TEACHER · số phản hồi" "$(grep -c . "$QC_OUT/tc-pg04-55.TEACHER" | tr -d ' ')" 50
  chk "TEACHER · giá trị duy nhất" "$(sort -u "$QC_OUT/tc-pg04-55.TEACHER")" "$U1|TEACHER"
  chk "TA · số phản hồi" "$(grep -c . "$QC_OUT/tc-pg04-55.TA" | tr -d ' ')" 50
  chk "TA · giá trị duy nhất" "$(sort -u "$QC_OUT/tc-pg04-55.TA")" "$U2|TA"
  chk "STUDENT · số phản hồi" "$(grep -c . "$QC_OUT/tc-pg04-55.STUDENT" | tr -d ' ')" 50
  chk "STUDENT · giá trị duy nhất" "$(sort -u "$QC_OUT/tc-pg04-55.STUDENT")" "$U2|STUDENT"
}

main 04 "$@"
