#!/usr/bin/env bash
# QC — kiểm prototype bằng curl + cookie (không cần trình duyệt). Chạy khi stack/dev server đang chạy.
#   F=http://localhost:3000 bash docs/sprints/1.5/qc/scripts/proto-curl.sh TC-xx ...
# Các hàm open_as / visible lấy NGUYÊN VĂN từ docs/sprints/1.5/spec/US.md (Quy ước kiểm chung).
set -u
F=${F:-http://localhost:3000}
RC=0; TC=""
pass(){ echo "PASS $TC: $*"; }
fail(){ echo "FAIL $TC: $*"; RC=1; }
eq(){ [ "$1" = "$2" ] && pass "$3" || fail "$3 — got [$1] want [$2]"; }
ge(){ [ "$1" -ge "$2" ] && pass "$3 ($1 ≥ $2)" || fail "$3 — got $1 want ≥ $2"; }

open_as() {  # $1 vai (student|ta|teacher|admin)  $2 đường dẫn  $3 người (tuỳ chọn: sv-1..sv-4)
  curl -s -b "ep_demo_role=$1; ep_demo_person=${3:-}; ep_demo_course=int1006-1" "$F$2" \
    | grep -qE 'Bạn không có quyền (mở trang|xem màn) này' && echo "CHAN $1 $2" || echo "MO   $1 $2"; }  # sprint 3 (US-PU-04 AC10): lời mới "xem màn này"
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
  o=$(grep -rnE '(^|[^A-Za-z_])fetch\(' frontend/src --include=*.ts --include=*.tsx | grep -v 'src/shared/'); [ -z "$o" ] && pass "không fetch( ngoài src/shared" || { fail "fetch( ngoài shared"; echo "$o" | head -3; }
  # góp ý #13 (sprint 3): cho phép thư viện trong ARCHITECTURE §3 (kiểm thử / UI), không chỉ lucide-react
  o=$(git diff main -- frontend/package.json | grep -E '^\+ ' | grep -E '"[^"]+": *"[~^]?[0-9]' | grep -vE 'lucide-react|@playwright/test|@axe-core/playwright|@lhci/cli|@tanstack/react-(query|virtual)|"recharts"|"zustand"' | grep -vE '^\+\+\+'); [ -z "$o" ] && pass "package.json không thêm phụ thuộc ngoài bảng ARCHITECTURE §3" || { fail "thêm phụ thuộc ngoài bảng §3"; echo "$o"; }
  o=$(git grep -nE "localStorage\.(setItem|getItem)\(['\"]" -- frontend/src | grep -v ep_demo_state); [ -z "$o" ] && pass "localStorage chỉ dùng khoá ep_demo_state" || { fail "localStorage khoá khác"; echo "$o" | head -3; }
  # (sprint 3, US-PU-03) bỏ dòng chú thích: tokenStore.ts nói rõ "không localStorage…" ở dòng 1
  o=$(git grep -nE "localStorage|sessionStorage" -- frontend/src | grep -vE ':[0-9]+:[[:space:]]*(//|\*|/\*)' | grep -iE 'token|jwt|password'); [ -z "$o" ] && pass "không lưu token ở storage" || { fail "token ở storage"; echo "$o"; }
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

# ---------- Spec v5 (#17 UI polish, #18 Threads như thật) ----------
hook(){ # hook <vai> <route> <data-part> <min> [người]  → đếm thuộc tính data-part trong HTML
  local n; n=$(curl -s -b "ep_demo_role=$1; ep_demo_person=${5:-sv-2}; ep_demo_course=int1006-1" "$F$2" | grep -o "data-part=\"$3\"" | wc -l | tr -d ' '); ge "$n" "$4" "$1 $2 có data-part=$3"; }
tc_00_07(){ local v; for v in student ta teacher admin; do hook $v / brand 1; done
  ge "$(curl -s "$F/login" | grep -c 'alt="EduPilot"')" 1 "/login có logo alt=EduPilot (AC7)"; }
tc_00_08(){ ge "$(curl -s -b "ep_demo_role=teacher; ep_demo_course=int1006-1" $F/ | grep -c 'title="761987 · An ninh mạng"')" 1 "bộ chọn lớp có title đầy đủ (AC8)"
  ge "$(curl -s -b "ep_demo_role=teacher; ep_demo_course=int1006-1" $F/ | grep -c 'aria-label="[^"]*761987 · An ninh mạng')" 1 "bộ chọn lớp có aria-label đầy đủ (SRS 4.7 b2)"; }
tc_00_09(){ local p v n
  for p in "teacher:TS. Lê Thu Hà" "ta:Phạm Quốc Bảo" "admin:Đỗ Hoàng Nam" "student:Trần Thu Uyên"; do
    ge "$(curl -s -b "ep_demo_role=${p%%:*}; ep_demo_person=sv-2; ep_demo_course=int1006-1" $F/ | grep -c "Tài khoản: ${p#*:}")" 1 "aria-label Tài khoản: ${p#*:}"; done
  for p in sv-1:'Nguyễn Minh Trung' sv-3:'Lê Quang Huy' sv-4:'Phạm Ngọc Linh'; do
    ge "$(curl -s -b "ep_demo_role=student; ep_demo_person=${p%%:*}; ep_demo_course=int1006-1" $F/ | grep -c "Tài khoản: ${p#*:}")" 1 "SV ${p%%:*} → Tài khoản: ${p#*:}"; done
  for v in teacher ta admin; do eq "$(curl -s -b "ep_demo_role=$v; ep_demo_person=sv-2; ep_demo_course=int1006-1" $F/ | grep -c 'Tài khoản: \(Giảng viên\|Trợ giảng\|Admin\)"')" 0 "$v: aria-label không ghi tên vai"; done; }
tc_00_hooks(){ hook student /chat chat-history 1; hook student /chat chat-thread 1; hook student /chat chat-composer 1; hook teacher /inbox inbox-list 1; }  # sprint 3 (US-P1-05): hook provider-status/-action của /settings/llm là bản mock → đã thay bằng TC-P105-04 (màn thật, cần token)
THREADS_N='t-cbc:4 t-salt:3 t-sqli:3 t-pin-rubric:4 t-pin-lab:3 t-rsa-key:2 t-xss:2 t-vpn:2 t-pki:2 t-phishing:2 t-firewall:2 t-wifi:2'
tc_01_10(){ local p id n v   # 01-AC10: không còn hai tiêu đề cũ; "Thảo luận (n)" khớp SRS 4.3.1 B, mọi vai được vào thread
  for v in student ta teacher; do for p in $THREADS_N; do id=${p%%:*}; n=${p#*:}
    eq "$(visible $v /threads/$id sv-2 | grep -cE 'Thảo luận của lớp|Phản hồi trong thread này')" 0 "$v $id không còn tiêu đề cũ"
    ge "$(visible $v /threads/$id sv-2 | grep -c "Thảo luận ($n)")" 1 "$v $id có 'Thảo luận ($n)'"; done; done; }
tc_01_11(){ local t   # 01-AC10: seed t-cbc / t-salt đúng bảng B (nếu curl không thấy → so tay bằng trình duyệt)
  t=$(visible student /threads/t-cbc sv-2)
  for s in 'Đặng Gia An' 'Dương Thanh Hiếu' 'Vậy IV có cần giữ bí mật không ạ' 'Phạm Quốc Bảo' 'Em hiểu rồi: IV cố định' 'Nguyễn Thị Giang' 'Còn CTR thì sao ạ' 'Nguồn tham khảo (2)' 'Chờ xác nhận'; do ge "$(echo "$t" | grep -c "$s")" 1 "t-cbc có «${s}»"; done
  eq "$(echo "$t" | grep -c 'Xác nhận\b.*Chỉnh sửa\|Loại khỏi tri thức')" 0 "SV không có nút Xác nhận/Chỉnh sửa/Loại (t-cbc)"
  t=$(visible student /threads/t-salt sv-2)
  for s in 'Đã được giảng viên xác nhận' 'Lê Thu Hà' 'Muối lưu ở đâu ạ' 'bcrypt còn cố tình chậm' 'Em nghĩ là để mỗi lần đoán thử'; do ge "$(echo "$t" | grep -c "$s")" 1 "t-salt có «${s}»"; done
  for id in t-firewall t-wifi; do eq "$(visible student /threads/$id sv-2 | grep -c 'Trợ lý AI của lớp')" 0 "$id không có câu AI"; done; }
tc_04_07(){ eq "$(visible teacher /settings/llm | grep -c 'Test kết nối')" 0 "GV không có Test kết nối"; }
# ---------- Spec v5.1 (#19) — lệnh Kiểm lấy từ US.md; mỗi hàm một AC ----------
ROOT(){ git rev-parse --show-toplevel; }
tc_00_13(){ local v   # 00-AC13 / FR-X16
  eq "$(curl -s $F/khong-co-trang | grep -c 'This page could')" 0 "404 không lộ trang tiếng Anh của Next.js (không cookie)"
  for v in teacher student ta admin; do ge "$(curl -s -b "ep_demo_role=$v; ep_demo_person=sv-2" $F/khong-co-trang | grep -c 'Không tìm thấy trang')" 1 "$v: 404 tiếng Việt"
    ge "$(curl -s -b "ep_demo_role=$v; ep_demo_person=sv-2" $F/khong-co-trang | grep -c 'Về Hôm nay')" 1 "$v: 404 có nút Về Hôm nay"; done
  [ -f "$(ROOT)/frontend/src/app/error.tsx" ] && [ -f "$(ROOT)/frontend/src/app/not-found.tsx" ] && pass "có app/error.tsx và app/not-found.tsx" || fail "thiếu app/error.tsx hoặc app/not-found.tsx"; }
tc_00_14(){ local o r   # 00-AC14 / FR-X13 (N6): không chuỗi thời gian cứng
  o=$(grep -rnE '[0-9]+ (ngày|giờ|phút) trước' "$(ROOT)/frontend/src" --include=*.ts --include=*.tsx | grep -v 'mock/derive.ts' | grep -v 'app/dev/'); [ -z "$o" ] && pass "không chuỗi 'N … trước' ngoài mock/derive.ts" || { fail "còn chuỗi thời gian cứng"; echo "$o" | cut -c1-150 | head -5; }
  eq "$(visible student / sv-2 | grep -c '12 ngày trước')" 0 "SV B không còn '12 ngày trước'"
  for r in / /admin/courses; do ge "$(visible admin $r | grep -ci 'hôm qua 16:40')" 1 "Admin $r có 'hôm qua 16:40'"; done
  eq "$(visible admin / | grep -c '08:30')" 0 "Admin / không còn '08:30'"; }
tc_00_15(){ local o   # 00-AC12 tĩnh: không glass; chạy được ngay trên cây sạch
  o=$(grep -rnE 'backdrop-filter|transparent\)' "$(ROOT)/frontend/src/shared/shell" "$(ROOT)/frontend/src/shared/ui" 2>/dev/null); [ -z "$o" ] && pass "shared/shell, shared/ui không có backdrop-filter / transparent)" || { fail "còn glass"; echo "$o" | cut -c1-150 | head -5; }
  grep -qE 'backdrop-filter' "$(ROOT)/scripts/ui-antipatterns.sh" && pass "ui-antipatterns.sh có mẫu backdrop-filter" || fail "ui-antipatterns.sh thiếu mẫu backdrop-filter"; }
tc_01_17(){ local r a b   # 01-AC17 / FR-X18: SV D
  for r in /chat /threads /threads/t-cbc /practice /practice/at-symmetric /practice/history /library /calendar /me /assignments/bt03; do
    a=$(visible student $r sv-4 | grep -c 'Bạn chưa vào lớp nào'); b=$(visible student $r sv-4 | grep -ciE 'phiên trước|Cách chọn độ dài khoá RSA|Nộp muộn Bài tập 03|CBC khác ECB|Chương 3 — Mật mã')
    ge "$a" 1 "D $r: màn 'Bạn chưa vào lớp nào'"; eq "$b" 0 "D $r: không dữ liệu lớp / phiên của B"
    eq "$(open_as student $r sv-4 | awk '{print $1}')" MO "D $r: không phải màn chặn quyền (FR-X4)"; done
  eq "$(visible student / sv-4 | grep -cE 'Chat riêng|Luyện đề|Thư viện|Lịch')" 0 "D: sidebar chỉ Hôm nay"
  ge "$(visible student / sv-4 | grep -ci 'mã tham gia')" 1 "D: ô nhập 'mã tham gia'"
  for r in /inbox /attendance /gradebook /observability; do want CHAN student $r sv-4; done; }
tc_01_20(){ eq "$(visible student /threads/t-rsa-key sv-2 | grep -c 'lan truyền lỗi')" 0 "t-rsa-key không có nội dung CBC 'lan truyền lỗi' (01-AC20)"; }
tc_01_24(){ local t; t=$(visible student /library sv-2)   # 01-AC24 / N5
  ge "$(echo "$t" | grep -c 'Chương 5 — Quản lý khoá và PKI')" 1 "/library có Chương 5 — Quản lý khoá và PKI"; ge "$(echo "$t" | grep -c 'Modern Network Security Threats')" 1 "/library có Modern Network Security Threats"
  eq "$(echo "$t" | grep -cE 'Đáp án đề|Mordern')" 0 "/library không Đáp án đề / Mordern"
  for s in 'Chương 1 — Tổng quan an ninh mạng và mô hình đe doạ' 'Chương 2 — Tấn công mạng phổ biến' 'Chương 3 — Mật mã đối xứng và chế độ vận hành' 'Chương 4 — Hàm băm và chữ ký số' 'Quy chế đào tạo của trường' 'Quy chế môn học An ninh mạng – 761987' 'Đề thi cuối kỳ An ninh mạng — HK1 2025–2026' 'Đề thi giữa kỳ An ninh mạng — HK1 2024–2025'; do ge "$(echo "$t" | grep -c "$s")" 1 "/library có «${s}»"; done; }
tc_02_14(){ local o; o=$(grep -n 'badge:' "$(ROOT)/frontend/src/shared/shell/nav.ts" 2>/dev/null | grep -E 'badge: *[0-9]'); [ -z "$o" ] && pass "nav.ts không còn badge số cứng (02-AC14)" || { fail "nav.ts còn badge cứng"; echo "$o" | head -3; }; }
tc_02_18(){ eq "$(visible teacher /students | grep -c 'Theo dõi')" 0 "/students không còn 'Theo dõi' (02-AC18)"; }
tc_02_20(){ local k; for k in '1 lệch hai lượt chấm' '1 bài ngắn bất thường' '1 trùng đoạn với bài khác' '1 AI không chắc ở một tiêu chí'; do ge "$(visible teacher / | grep -c "$k")" 1 "Hôm nay có lý do «${k}» (02-AC20)"; done; }
tc_03_08(){ local t; t=$(visible teacher /documents)   # 03-AC8 / N5
  ge "$(echo "$t" | grep -c 'Chương 3 — Mật mã đối xứng và chế độ vận hành')" 1 "/documents có tên hiển thị Chương 3"
  eq "$(echo "$t" | grep -cE 'Chuong[0-9]_|Mordern')" 0 "/documents không tên tệp gạch dưới / Mordern ngoài Drawer"
  eq "$(echo "$t" | grep -c 'Hạ tầng khoá công khai')" 0 "/documents không còn tài liệu 'Hạ tầng khoá công khai' (v4)"; ge "$(echo "$t" | grep -c 'Không hiển thị cho sinh viên')" 1 "/documents ghi 'Không hiển thị cho sinh viên' cho đáp án"; }
tc_04_08(){ eq "$(visible teacher /inbox | grep -o 'Quá 24 giờ' | wc -l | tr -d ' ')" 3 "/inbox có 3 hàng 'Quá 24 giờ' (04-AC8)"; ge "$(visible teacher /analytics | grep -c 'AI tự trả lời 98%')" 1 "/analytics 'AI tự trả lời 98%'"; }
tc_04_10(){ local r; for r in / /admin/courses; do ge "$(visible admin $r | grep -ci 'hôm qua 16:40')" 1 "Admin $r có 'hôm qua 16:40' (04-AC10)"; done; eq "$(visible admin / | grep -c '08:30')" 0 "Admin / không '08:30'"; }
tc_00_16(){ local d f n; d="$(ROOT)/frontend/src/mock/derive.ts"   # FR-X13: một nguồn số liệu (SRS 4.8)
  [ -f "$d" ] && pass "có mock/derive.ts" || { fail "thiếu mock/derive.ts"; return; }
  for f in ticketStats attentionSet navBadges ago; do ge "$(grep -c "$f" "$d")" 1 "derive.ts có $f"; done
  n=$(grep -rnE 'overdue: *\[|answeredByAi|escalated: *\[' "$(ROOT)/frontend/src" --include=*.ts --include=*.tsx | wc -l | tr -d ' '); eq "$n" 0 "không còn số cứng overdue / answeredByAi / escalated (N1, N2)"
  n=$(grep -rnE 'color-mix\([^)]*transparent' "$(ROOT)/frontend/src/shared/shell" "$(ROOT)/frontend/src/shared/ui" 2>/dev/null | wc -l | tr -d ' '); eq "$n" 0 "shell / ui không color-mix(… transparent) (FR-X15)"; }


[ $# -eq 0 ] && { echo "dùng: F=… $0 tc_00_01 …  (hoặc: $0 all)"; grep -oE '^tc_[0-9a-z_]+' "$0" | xargs; exit 2; }
for t in "$@"; do TC=$t; if [ "$t" = all ]; then for f in $(grep -oE '^tc_[0-9a-z_]+' "$0"); do TC=$f; $f; done; elif declare -F "$t" >/dev/null; then "$t"; else echo "FAIL $t: không có TC này"; RC=1; fi; done
exit $RC
