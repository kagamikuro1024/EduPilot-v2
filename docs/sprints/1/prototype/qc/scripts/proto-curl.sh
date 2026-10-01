#!/usr/bin/env bash
# QC — kiểm prototype bằng curl + cookie (không cần trình duyệt). Chạy khi stack/dev server đang chạy.
#   F=http://localhost:3000 bash docs/sprints/1/prototype/qc/scripts/proto-curl.sh TC-xx ...
# Các hàm open_as / visible lấy NGUYÊN VĂN từ docs/sprints/1/prototype/spec/US.md (Quy ước kiểm chung).
set -u
F=${F:-http://localhost:3000}
RC=0; TC=""
pass(){ echo "PASS $TC: $*"; }
fail(){ echo "FAIL $TC: $*"; RC=1; }
eq(){ [ "$1" = "$2" ] && pass "$3" || fail "$3 — got [$1] want [$2]"; }
ge(){ [ "$1" -ge "$2" ] && pass "$3 ($1 ≥ $2)" || fail "$3 — got $1 want ≥ $2"; }

open_as() {  # $1 vai (student|ta|teacher|admin)  $2 đường dẫn  $3 người (tuỳ chọn: sv-1..sv-4)
  curl -s -b "ep_demo_role=$1; ep_demo_person=${3:-}; ep_demo_course=int1006-1" "$F$2" \
    | grep -q 'Bạn không có quyền mở trang này' && echo "CHAN $1 $2" || echo "MO   $1 $2"; }
visible() {  # $1 vai $2 đường dẫn $3 người
  curl -s -b "ep_demo_role=$1; ep_demo_person=${3:-}" "$F$2" | perl -0pe 's#<script.*?</script>##gs; s#<[^>]+># #g'; }
# want <MO|CHAN> <vai> <route> [người]  → PASS/FAIL một dòng
want(){ local exp=$1 got; got=$(open_as "$2" "$3" "${4:-}" | awk '{print $1}'); [ "$got" = "$exp" ] && pass "$2 $3 → $exp" || fail "$2 $3 → $got (mong đợi $exp)"; }

# ---------- Ma trận vai × route (SRS mục 2) ----------
ALL='/'
SV='/chat /practice /practice/at-symmetric /practice/at-quiz01 /practice/history /library /me /assignments/bt03 /join /join/BX4P9TW'
SV_TA_GV='/threads /threads/t-cbc /calendar'
TA_GV='/inbox /students /students/sv-3 /attendance /class/members /gradebook /gradebook/scheme /grading /grading/sub-bt03-sv-2 /questions /documents /insights /analytics'
GV_AD='/observability /settings/llm /settings/integrations'
ADMIN='/admin/courses /admin/users'
role_of(){ case "$1" in student) echo SV;; ta) echo TA;; teacher) echo GV;; admin) echo AD;; esac; }
allowed(){ # $1 vai $2 nhóm → 0 nếu mở được
  local r; r=$(role_of "$1")
  case "$2" in
    ALL) return 0;;
    SV) [ "$r" = SV ];;
    SV_TA_GV) [ "$r" = SV ] || [ "$r" = TA ] || [ "$r" = GV ];;
    TA_GV) [ "$r" = TA ] || [ "$r" = GV ];;
    GV_AD) [ "$r" = GV ] || [ "$r" = AD ];;
    ADMIN) [ "$r" = AD ];;
  esac; }
matrix(){ local v g r exp; for v in student ta teacher admin; do
    for g in ALL SV SV_TA_GV TA_GV GV_AD ADMIN; do for r in ${!g}; do
      if allowed $v $g; then exp=MO; else exp=CHAN; fi; want $exp $v "$r" sv-2; done; done; done; }

# ---------- US-PROTO-00 ----------
tc_00_01(){ local u; u=$(curl -s -o /dev/null -w '%{redirect_url}' "$F/"); echo "$u" | grep -q '/login' && pass "/ không cookie → /login ($u)" || fail "/ không cookie: redirect=[$u]"
  for r in /inbox /chat /gradebook /me; do u=$(curl -s -o /dev/null -w '%{redirect_url}' "$F$r"); echo "$u" | grep -q '/login' && pass "$r không cookie → /login" || fail "$r không cookie không về /login ([$u])"; done
  u=$(curl -s -o /dev/null -w '%{redirect_url}|%{http_code}' -b 'ep_demo_role=hacker' "$F/"); echo "cookie vai rác: [$u]"; case "$u" in *login*|*200*) pass "cookie vai rác không gây 5xx ($u)";; *) fail "cookie vai rác → $u";; esac
  ge "$(visible student / sv-2 | grep -c 'Bản mô phỏng')" 1 "dải 'Bản mô phỏng · dữ liệu giả' hiện ở / (SV B)"
  for v in teacher ta admin; do ge "$(visible $v / | grep -c 'Bản mô phỏng')" 1 "dải mô phỏng ở / (vai $v)"; done; }
tc_00_02(){ ge "$(visible student / sv-2 | grep -c 'Trần Thu Uyên')" 1 "tên 'Trần Thu Uyên' hiện cho Sinh viên B"; }
tc_00_04(){ ge "$(visible student / sv-4 | grep -c 'mã tham gia')" 1 "SV D: 'Hôm nay' có ô nhập 'mã tham gia'"; }
tc_00_05(){ local s n
  for s in empty error; do n=$(visible teacher "/inbox?state=$s" | grep -cE 'Thử lại|Không còn câu hỏi'); ge "$n" 1 "/inbox?state=$s có 'Thử lại'/'Không còn câu hỏi'"; done
  for r in /students /gradebook /documents /insights /analytics /questions /grading; do n=$(visible teacher "$r?state=error" | grep -c 'Thử lại'); ge "$n" 1 "$r?state=error có 'Thử lại'"; done
  for r in /threads /library /calendar /me /practice/history; do n=$(visible student "$r?state=error" sv-2 | grep -c 'Thử lại'); ge "$n" 1 "(SV) $r?state=error có 'Thử lại'"; done
  for r in /inbox /students /gradebook; do eq "$(visible teacher "$r?state=loading" | grep -ciE 'spinner|đang tải trang')" 0 "$r?state=loading không có chữ spinner (xem mắt để chắc)"; done; }
tc_00_06(){ for r in /chat /me /practice /library /join; do want CHAN teacher $r; done
  for r in /inbox /attendance /gradebook /grading /admin/courses /observability; do want CHAN student $r sv-2; done
  for r in /chat /inbox /gradebook /calendar; do want CHAN admin $r; done
  want MO ta /class/members; want CHAN ta /admin/users
  local t; t=$(curl -s -b 'ep_demo_role=student; ep_demo_person=sv-2; ep_demo_course=int1006-1' "$F/inbox" | perl -0pe 's#<script.*?</script>##gs; s#<[^>]+># #g')
  echo "$t" | grep -q 'Về Hôm nay' && pass "màn chặn có nút 'Về Hôm nay'" || fail "màn chặn thiếu 'Về Hôm nay'"
  echo "$t" | grep -qE 'Trang này dành cho' && pass "màn chặn có một câu lý do theo vai" || fail "màn chặn thiếu câu lý do 'Trang này dành cho …'"
  [ "$(echo "$t" | grep -c 'Nhận')" = 0 ] && pass "màn chặn không lộ nút của /inbox" || fail "màn chặn lộ nội dung trang"; }
tc_00_matrix(){ matrix; }
tc_00_static(){ cd "$(git rev-parse --show-toplevel)"; local o
  o=$(grep -rn 'fetch(' frontend/src --include=*.ts --include=*.tsx | grep -v 'src/shared/'); [ -z "$o" ] && pass "không fetch( ngoài src/shared" || { fail "fetch( ngoài shared"; echo "$o" | head -3; }
  o=$(git diff main -- frontend/package.json | grep -E '^\+ ' | grep -v lucide-react | grep -vE '^\+\+\+'); [ -z "$o" ] && pass "package.json không thêm phụ thuộc ngoài lucide-react" || { fail "thêm phụ thuộc"; echo "$o"; }
  o=$(git grep -nE "localStorage\.(setItem|getItem)\(['\"]" -- frontend/src | grep -v ep_demo_state); [ -z "$o" ] && pass "localStorage chỉ dùng khoá ep_demo_state" || { fail "localStorage khoá khác"; echo "$o" | head -3; }
  o=$(git grep -nE "localStorage|sessionStorage" -- frontend/src | grep -iE 'token|jwt|password'); [ -z "$o" ] && pass "không lưu token ở storage" || { fail "token ở storage"; echo "$o"; }
  o=$(git grep -nE '#[0-9a-fA-F]{3,8}\b|rgb\(|hsl\(' -- 'frontend/src/**/*.tsx' 'frontend/src/**/*.css' ':!frontend/src/shared/styles/tokens.css' | grep -v 'ui-allow'); [ -z "$o" ] && pass "không màu viết cứng ngoài tokens.css" || { fail "màu cứng"; echo "$o" | head -3; }
}

# ---------- US-PROTO-01 ----------
S1='/ /chat /threads /threads/t-cbc /practice /practice/at-symmetric /practice/history /library /calendar /me /assignments/bt03 /join'
tc_01_01(){ for r in $S1; do want MO student "$r" sv-2; done; }
tc_01_05(){ local r o; for r in / /chat /threads /threads/t-cbc /practice /library /calendar /me /assignments/bt03; do o=$(visible student "$r" sv-2 | grep -inE 'RAG|PII|fallback|trace|provider|confidence|redaction|độ tin cậy|cần chú ý|rủi ro'); [ -z "$o" ] && pass "(AC5) $r sạch từ cấm" || { fail "(AC5) $r có từ cấm"; echo "$o" | cut -c1-160 | head -3; }; done
  for r in / /chat /threads /practice/history /library /me /join /join/BX4P9TW; do o=$(visible student "$r" sv-2 | grep -E '\bRAG\b|\bPII\b|embedding|\bprompt\b|\bLLM\b|\btoken\b|placeholder|escalat|\[\[SV_'); [ -z "$o" ] && pass "(mở rộng) $r không RAG/PII/LLM/prompt/placeholder" || { fail "(mở rộng) $r"; echo "$o" | cut -c1-160 | head -2; }; done
  o=$(visible student /assignments/bt03 sv-2 | grep -E '8,5|7,0 *(/|điểm)|điểm nháp'); [ -z "$o" ] && pass "/assignments/bt03 trước công bố không có điểm nháp (8,5, 7,0)" || { fail "bt03 lộ điểm nháp"; echo "$o" | cut -c1-160; }; }
tc_01_08(){ ge "$(visible student /join/BX4P9TW sv-4 | grep -c 761988)" 1 "/join/BX4P9TW xem trước lớp 761988"; }
tc_01_09(){ local v r; for v in ta teacher admin; do for r in /chat /me /practice /library /assignments/bt03 /join; do want CHAN $v "$r"; done; done; want CHAN admin /threads; want CHAN admin /calendar; }
# ---------- US-PROTO-02 ----------
tc_02_01(){ local r; for r in / /inbox /students /students/sv-3 /attendance /class/members; do want MO teacher "$r"; done; }
tc_02_07(){ local v r; for v in student admin; do for r in /inbox /students /students/sv-3 /attendance /class/members; do want CHAN $v "$r" sv-2; done; done
  eq "$(visible ta /class/members | grep -c 'Tạo lại mã')" 0 "TA không có 'Tạo lại mã'"; eq "$(visible ta /class/members | grep -c 'Mời ra khỏi lớp')" 0 "TA không có 'Mời ra khỏi lớp'"; }
# ---------- US-PROTO-03 ----------
tc_03_01(){ local r; for r in /gradebook /gradebook/scheme /grading /grading/sub-bt03-sv-2 /questions /documents; do want MO teacher "$r"; done; }
tc_03_06(){ ge "$(visible ta /grading | grep -c 'Chỉ giảng viên công bố điểm')" 1 "TA ở /grading thấy 'Chỉ giảng viên công bố điểm'"
  eq "$(visible ta /gradebook/scheme | grep -c 'Xác nhận công thức')" 0 "TA ở /gradebook/scheme không có 'Xác nhận công thức'"
  local v r; for v in student admin; do for r in /gradebook /gradebook/scheme /grading /questions /documents; do want CHAN $v "$r" sv-2; done; done; }
# ---------- US-PROTO-04 ----------
tc_04_01(){ local r; for r in /insights /analytics /observability /settings/llm /settings/integrations; do want MO teacher "$r"; done; for r in / /observability /settings/llm /settings/integrations /admin/courses /admin/users; do want MO admin "$r"; done; }
tc_04_02(){ eq "$(visible teacher /insights | grep -cE '2022[0-9]{4}|Trần Thu Uyên|Lê Quang Huy')" 0 "/insights không tên/MSSV"; }
tc_04_03(){ eq "$(visible teacher /observability | grep -c '\[\[SV_')" 0 "GV ở /observability không thấy [[SV_"; }
tc_04_04(){ eq "$(visible teacher /settings/llm | grep -c 'Test kết nối')" 0 "GV ở /settings/llm không có 'Test kết nối'"; }
tc_04_06(){ local v r; for v in student ta; do for r in /observability /settings/llm /settings/integrations /admin/courses /admin/users; do want CHAN $v "$r" sv-2; done; done; want CHAN teacher /admin/courses
  eq "$(visible ta /analytics | grep -ci 'chi phí')" 0 "TA ở /analytics không có 'chi phí'"; }

[ $# -eq 0 ] && { echo "dùng: F=… $0 tc_00_01 …  (hoặc: $0 all)"; grep -oE '^tc_[0-9a-z_]+' "$0" | xargs; exit 2; }
for t in "$@"; do TC=$t; if [ "$t" = all ]; then for f in $(grep -oE '^tc_[0-9a-z_]+' "$0"); do TC=$f; $f; done; elif declare -F "$t" >/dev/null; then "$t"; else echo "FAIL $t: không có TC này"; RC=1; fi; done
exit $RC
