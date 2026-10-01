#!/usr/bin/env bash
# QC — TC của US-P0-01 (story tài liệu: docs/DEMO_SCRIPT.md). Dùng: bash docs/sprints/1/qc/scripts/tc-US-P0-01.sh TC-01 ...
set -u
cd "$(git rev-parse --show-toplevel)" || exit 2
D=docs/DEMO_SCRIPT.md; DES=docs/design/DESIGN.md; ARC=docs/ARCHITECTURE.md; FL=docs/FLOWS.md
RC=0; TC=""
pass(){ echo "PASS $TC: $*"; }
fail(){ echo "FAIL $TC: $*"; RC=1; }
eq(){ [ "$1" = "$2" ] && pass "$3" || fail "$3 — got [$1] want [$2]"; }
ROWS='^\| [0-9]{2}:[0-9]{2} \|'
# step N → dòng bắt đầu / kết thúc (trước heading kế tiếp ### hoặc ##)
sect(){ awk -v n="$1" 'BEGIN{p=0} /^### Bước /{p=($0 ~ "^### Bước " n "\\.")} /^### Kết|^## /{p=0} p' $D; }
flow_sect(){ awk -v f="$1" '/^### /{p=($0 ~ "^### " f "\\.")} /^## /{p=0} p' $FL; }
step_f(){ grep -E "^### Bước $1\." $D | grep -oE 'F[0-9]+ \(' | tr -d ' ('; }

tc_01(){ eq "$(grep -oE '^### Bước [0-9]+\..*F[0-9]+ \(' $D | grep -oE 'F[0-9]+ \(' | tr -d ' (' | xargs)" "F2 F3 F5 F7 F9 F10 F17" "thứ tự luồng F2→F3→F5→F7→F9→F10→F17"
  eq "$(grep -cE '^### Bước [0-9]+\.' $D)" 7 "đúng 7 tiêu đề '### Bước'"
  eq "$(grep -oE '^### Bước [0-9]+' $D | grep -oE '[0-9]+$' | xargs)" "1 2 3 4 5 6 7" "số bước liên tục 1..7"
  local o; o=$(grep -oE '^### Bước [0-9]+\..*— F[0-9]+' $D | grep -oE 'F[0-9]+$' | xargs)
  [ "$o" = "F2 F3 F5 F7 F9 F10 F17" ] && pass "lệnh nguyên văn của AC1 ra đúng" || fail "lệnh nguyên văn của AC1 ra [$o]"; }
tc_02(){ eq "$(awk -F'|' '/^\| [0-9]{2}:[0-9]{2} \|/ && NF!=8 {print NR": "$0}' $D | wc -l | xargs)" 0 "mọi hàng có đúng 6 cột (NF=8)"
  local n; n=$(grep -cE "$ROWS" $D); [ "$n" -ge 30 ] && pass "≥ 30 hàng ($n)" || fail "chỉ $n hàng"
  eq "$(awk -F'|' '/^\| [0-9]{2}:[0-9]{2} \|/{for(i=2;i<=7;i++){c=$i; gsub(/ /,"",c); if(c=="") e++}} END{print e+0}' $D)" 0 "không ô nào trống"
  eq "$(grep -cE '^\| Mốc \| Vai trò \| Route \| Thao tác \| Kết quả phải thấy \| Phase \|' $D)" 7 "7 bảng, mỗi bước một bảng đúng 6 tiêu đề cột"
  for i in 1 2 3 4 5 6 7; do [ "$(sect $i | grep -cE "$ROWS")" -ge 3 ] && pass "Bước $i có ≥ 3 hàng" || fail "Bước $i < 3 hàng"; done; }
tc_03(){ grep -oE "$ROWS" $D | tr -d '| ' | sort -c && pass "mốc không giảm (lệnh AC3)" || fail "mốc giảm"
  eq "$(grep -oE "$ROWS" $D | tr -d '| ' | uniq -d | wc -l | xargs)" 0 "không hai hàng trùng mốc (kỹ hơn AC: tăng chặt)"
  local last; last=$(grep -oE "$ROWS" $D | tr -d '| ' | tail -1); [ "$last" \< "15:00" ] && pass "hàng cuối $last < 15:00" || fail "hàng cuối $last ≥ 15:00"
  # tổng thời lượng bảng mục 1
  local s; s=$(awk -F'|' '/^## 1\./{p=1} /^## 2\./{p=0} p && $5 ~ /^ [0-9]+:[0-9]+ $/ {split($5,t,":"); sec+=t[1]*60+t[2]} END{printf "%d", sec}' $D)
  [ "$s" -le 900 ] && pass "tổng thời lượng bảng mục 1 = $((s/60)):$(printf %02d $((s%60))) ≤ 15:00" || fail "tổng $s s > 900"
  eq "$s" 825 "tổng khớp mốc 'Hết kịch bản' 13:45 (825 s trừ mở đầu = 13:45 tính cả 0:30)"
  grep -qE 'Hết kịch bản.*13:45' $D && pass "dòng 'Hết kịch bản' ghi 13:45 ≤ 15:00" || fail "không thấy dòng 'Hết kịch bản'"
  # hàng của mỗi bước nằm trong khoảng ghi ở tiêu đề
  local i hd a b first lastr; for i in 1 2 3 4 5 6 7; do hd=$(grep -E "^### Bước $i\." $D | grep -oE '\([0-9:–]+\)' | tr -d '()'); a=${hd%%–*}; b=${hd##*–}
    first=$(sect $i | grep -oE "$ROWS" | tr -d '| ' | head -1); lastr=$(sect $i | grep -oE "$ROWS" | tr -d '| ' | tail -1)
    { [ "$first" = "$a" ] || [ "$first" \> "$a" ]; } && [ "$lastr" \< "$b" ] && pass "Bước $i: hàng ${first}…${lastr} nằm trong ${a}–${b}" || fail "Bước $i: hàng ${first}…${lastr} ngoài ${a}–${b}"; done; }
tc_04(){ local rs miss=0 r; rs=$(grep -E "$ROWS" $D | awk -F'|' '{print $4}' | grep -oE '`/[^`]*`' | tr -d '`' | sed -E 's#/BX4P9TW#/[code]#' | sort -u)
  [ "$(echo "$rs" | wc -l)" -ge 8 ] && pass "$(echo "$rs" | wc -l | xargs) route khác nhau được trích" || fail "trích được quá ít route"
  for r in $rs; do grep -qF "\`$r\`" $DES $ARC || { echo "THIẾU $r"; miss=1; }; done; [ $miss = 0 ] && pass "mọi route có ở DESIGN hoặc ARCHITECTURE (lệnh AC4)" || fail "route bịa (lệnh AC4)"
  # kỹ hơn: phải nằm trong DESIGN §14 (từ dòng '# 14.') hoặc ARCHITECTURE §7 (## 7. … ## 8.)
  local d14 a7; d14=$(awk '/^# 14\./{p=1} p' $DES); a7=$(awk '/^## 7\./{p=1} /^## 8\./{p=0} p' $ARC); miss=0
  for r in $rs; do { echo "$d14"; echo "$a7"; } | grep -qF "\`$r\`" || { echo "NGOÀI §14/§7: $r"; miss=1; }; done; [ $miss = 0 ] && pass "mọi route nằm đúng DESIGN §14 hoặc ARCHITECTURE §7" || fail "route chỉ xuất hiện ngoài §14/§7 (xem tay)"
  # vai trò trong cột 'Vai trò' hợp lệ
  eq "$(grep -E "$ROWS" $D | awk -F'|' '{gsub(/^ +| +$/,"",$3); print $3}' | sed -E 's/ \(375 px\)//; s/ [A-D]$//' | sort -u | xargs)" "Admin Giảng viên Người trình bày Sinh viên" "vai trò chỉ gồm Admin / Giảng viên / Sinh viên (+ A–D) / Người trình bày (chỉ cho Mailpit, công cụ dev)"; }
tc_05(){ local o; o=$(grep -E '^\| [0-9:]+ \| Sinh viên' $D | grep -inE 'RAG|PII|fallback|trace|provider|confidence|escalat|prompt|LLM|model|token|placeholder|embedding'); [ -z "$o" ] && pass "lệnh AC5: không từ cấm" || { fail "từ cấm ở hàng Sinh viên"; echo "$o" | cut -c1-200; }
  [ "$(grep -cE '^\| [0-9:]+ \| Sinh viên' $D)" -ge 5 ] && pass "có ≥ 5 hàng vai Sinh viên để kiểm" || fail "quá ít hàng Sinh viên"
  o=$(grep -E '^\| [0-9:]+ \| Sinh viên' $D | grep -inE 'ticket|\bAI\b|chatbot|DEMO_MODE|\bfake\b|audit|outbox|SSE|JWT|nhãn rủi ro|điểm nháp')
  [ -z "$o" ] && pass "hàng Sinh viên không có từ kỹ thuật mở rộng (ticket, AI, audit, SSE…) hay thông tin cấm (điểm nháp, nhãn rủi ro) ở dạng lộ" || { echo "xem tay (có thể là câu 'không thấy' hợp lệ):"; echo "$o" | cut -c1-240; }
  grep -qE 'Cần giảng viên hỗ trợ' $D && pass "dùng đúng cụm DESIGN §13 'Cần giảng viên hỗ trợ'" || fail "thiếu cụm 'Cần giảng viên hỗ trợ'"
  grep -qE 'Đã ẩn [0-9]+ thông tin cá nhân|Đã ẩn thông tin cá nhân' $D && pass "dùng đúng cụm 'Đã ẩn … thông tin cá nhân'" || fail "thiếu cụm PII đúng §13"; }
tc_06(){ local s; s=$(awk '/^## 5\./{p=1} /^## 6\./{p=0} p' $D); local k miss=0
  for k in 'DEMO_MODE=true' 'fake' 'Video' 'Ai bật' '.env.local' 'up -d gateway' 'lỗi 2 lần'; do grep -qF "$k" $D || { echo "THIẾU $k"; miss=1; }; done; [ $miss = 0 ] && pass "lệnh AC6" || fail "lệnh AC6"
  miss=0; for k in 'DEMO_MODE=true' 'fake' 'Video' 'Ai bật' '.env.local' 'up -d gateway' 'lỗi 2 lần'; do echo "$s" | grep -qF "$k" || { echo "THIẾU trong mục 5: $k"; miss=1; }; done; [ $miss = 0 ] && pass "cả 7 khoá nằm trong mục 5 (không chỉ đâu đó trong file)" || fail "khoá không nằm trong mục 5"
  eq "$(echo "$s" | grep -cE '^\| [123]\. ')" 3 "3 tầng 1/2/3"
  echo "$s" | grep -qE 'docker compose --env-file \.env\.local -f docker-compose\.local\.yml -p edupilot up -d gateway' && pass "lệnh bật tầng 2 khớp stack US-P0-02 (--env-file .env.local -f docker-compose.local.yml -p edupilot)" || fail "lệnh bật tầng 2 không khớp tên stack thật"
  echo "$s" | grep -qiE 'Chủ dự án' && pass "ghi ai bật (Chủ dự án)" || fail "không ghi ai bật"
  grep -q 'DEMO_MODE' $ARC && pass "ARCHITECTURE có DEMO_MODE" || echo "info: ARCHITECTURE §8 chưa có DEMO_MODE (kịch bản nói P1 thêm — mục 6)"; }
tc_07(){ local n miss=0; n=$(grep -c '^Nếu hội đồng hỏi' $D); [ "$n" -ge 5 ] && pass "≥ 5 dòng 'Nếu hội đồng hỏi' ($n)" || fail "chỉ $n"
  local s; for s in $(grep -oE '[a-z-]+\.spec\.ts' $D | sort -u); do grep -qF "$s" $FL || { echo "THIẾU $s"; miss=1; }; done; [ $miss = 0 ] && pass "mọi *.spec.ts có trong FLOWS (lệnh AC7)" || fail "spec E2E không có trong FLOWS"
  local i f names; for i in 1 2 3 4 5 6; do f=$(step_f $i); names=$(sect $i | grep 'Nếu hội đồng hỏi' | grep -oE '[a-z-]+\.spec\.ts' | sort -u)
    [ -n "$(sect $i | grep 'Nếu hội đồng hỏi')" ] && pass "Bước $i ($f) có 'Nếu hội đồng hỏi'" || fail "Bước $i ($f) thiếu 'Nếu hội đồng hỏi'"
    [ -n "$(sect $i | grep '^Nếu hội đồng hỏi')" ] && pass "  … ở đầu dòng (đếm được bằng lệnh AC7)" || echo "info: Bước $i ($f): 'Nếu hội đồng hỏi' không ở đầu dòng nên lệnh '^Nếu hội đồng hỏi' không đếm — lỗi trình bày nhẹ"
    [ -n "$names" ] && pass "Bước $i nêu spec E2E: $(echo $names)" || fail "Bước $i không nêu tên spec E2E"
    for s in $names; do flow_sect $f | grep -qF "$s" && pass "  $s nằm trong mục $f của FLOWS" || fail "  $s không nằm trong mục $f của FLOWS"; done; done
  local t5; t5=$(awk '/^## 5\./{p=1} /^## 6\./{p=0} p' $D)
  echo "$t5" | grep -qiE 'Mạng hoặc API LLM lỗi' && pass "mục 5: tầng 2 dùng khi mạng/API LLM lỗi" || fail "mục 5 không nêu mạng/API LLM lỗi"
  echo "$t5" | grep -qE 'Stack không lên, hoặc tầng 2 cũng lỗi' && pass "mục 5: tầng 3 khi stack không lên / tầng 2 lỗi" || fail "mục 5 không nêu điều kiện sang tầng 3"
  echo "$t5" | grep -qE 'không cần mạng|Không cần mạng|không cần mạng' && pass "mục 5: tầng 2 không cần mạng ngoài" || fail "mục 5 không nói tầng 2 chạy khi không mạng"; }
tc_08(){ local rows; rows=$(grep -E "$ROWS" $D)
  echo "$(sect 2)" | grep -E "$ROWS" | grep -q 'chỉ trả lời được thông tin của chính bạn' && pass "Bước 2 (hàng bảng): sinh viên hỏi người khác bị từ chối" || fail "Bước 2 thiếu hàng 'chỉ trả lời được thông tin của chính bạn'"
  echo "$(sect 5)" | grep -E "$ROWS" | grep -q 'không có điểm nháp' && pass "Bước 5 (hàng bảng): sinh viên không thấy điểm nháp" || fail "Bước 5 thiếu hàng 'không có điểm nháp'"
  { sect 5; sect 6; } | grep -E "$ROWS" | grep -qE 'chỉ TEACHER' && pass "Bước 5/6 (hàng bảng): chỉ TEACHER công bố" || fail "Bước 5/6 thiếu 'chỉ TEACHER'"
  { sect 5; sect 6; } | grep -E "$ROWS" | grep -qE 'Chốt điểm. bị khoá' && pass "Bước 5/6 (hàng bảng): 'Chốt điểm' bị khoá khi công thức chưa xác nhận" || fail "Bước 5/6 thiếu 'Chốt điểm bị khoá'"
  local n; n=$(grep -cE 'chỉ trả lời được thông tin của chính bạn|không có điểm nháp|chỉ TEACHER|Chốt điểm. bị khoá' $D); [ "$n" -ge 4 ] && pass "lệnh AC8: $n ≥ 4" || fail "lệnh AC8: $n < 4"
  echo "$(sect 6)" | grep -E "$ROWS" | grep -qiE 'chưa xác nhận|Xác nhận công thức' && pass "Bước 6 có thao tác xác nhận công thức (chỉ giảng viên làm — CLAUDE.md cấm tự xác nhận)" || fail "Bước 6 không có xác nhận công thức bởi giảng viên"
  echo "$(sect 5)" | grep -E "$ROWS" | grep -qE 'Giảng viên.*(Công bố|công bố)' && pass "Bước 5: giảng viên là người công bố" || echo "info: không thấy hàng 'Giảng viên … công bố' rõ ở Bước 5 (xem tay)"; }
tc_09(){ eq "$(grep -E '^\| P(1|2|3|4|5|6|7|10) \|' $D | wc -l | xargs)" 8 "lệnh AC9: 8 hàng P1..P7,P10"
  eq "$(grep -c '^## 7. Nhật ký chạy lại' $D)" 1 "đúng 1 mục 7 nhật ký"
  eq "$(awk '/^## 6\./{p=1} /^## 7\./{p=0} p' $D | grep -oE '^\| P[0-9]+ ' | tr -d '| ' | xargs)" "P1 P2 P3 P4 P5 P6 P7 P10" "mục 6 liệt kê đúng P1–P7, P10"
  eq "$(awk '/^## 7\./{p=1} p' $D | grep -cE '^\| Ngày \| Phase vừa xong \| Bước xa nhất chạy thật \| Tổng thời gian \| Tầng \| Ghi chú \|')" 1 "mục 7 có bảng nhật ký đủ 6 cột"
  # mọi phase nêu ở cột Phase của bảng bước phải có trong mục 6
  local ph p miss=0; ph=$(grep -E "$ROWS" $D | awk -F'|' '{print $7}' | sed -E 's/P0 L1//g' | grep -oE 'P[0-9]+' | sort -u)  # P0 đã xong, không cần yêu cầu ở mục 6
  for p in $ph; do awk '/^## 6\./{p=1} /^## 7\./{p=0} p' $D | grep -qE "^\| $p " || { echo "phase $p dùng ở bước nhưng không có yêu cầu ở mục 6"; miss=1; }; done
  [ $miss = 0 ] && pass "mọi phase trong cột Phase ($(echo $ph)) đều có yêu cầu ở mục 6" || fail "phase thiếu ở mục 6"
  # cột Phase có phase nào không tồn tại trong docs/phases?
  for p in $ph; do ls docs/phases/$p.md >/dev/null 2>&1 && pass "docs/phases/$p.md tồn tại" || fail "docs/phases/$p.md không tồn tại"; done; }

# ---------- Kiểm chéo ----------
tc_10(){ eq "$(grep -oE '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[a-z]{2,}' $D | grep -vc '@edupilot\.local$')" 0 "mọi email thuộc tên miền giả @edupilot.local"
  eq "$(grep -cE '\b[0-9]{8,10}\b' $D)" 0 "không chuỗi giống MSSV 8–10 số (xem tay nếu >0)"
  eq "$(grep -cE 'sk-[A-Za-z0-9]{10,}|AKIA[0-9A-Z]{12}' $D)" 0 "không khoá thật"
  eq "$(grep -oE 'mật khẩu `[^`]*`' $D | grep -vc 'SEED_DEFAULT_PASSWORD')" 0 "mật khẩu chỉ được nhắc bằng tên biến SEED_DEFAULT_PASSWORD, không giá trị"; }
tc_11(){ local f miss=0; for f in $(grep -oE '`(docs|seed|scripts)/[A-Za-z0-9_./-]+`' $D | tr -d '`' | sort -u); do [ -e "$f" ] || { echo "không tồn tại: $f"; miss=1; }; done
  [ $miss = 0 ] && pass "mọi đường dẫn file được trích đều tồn tại (seed/demo/* do P1+ tạo thì liệt kê ở trên)" || echo "info: file ở trên chưa có — kiểm xem tài liệu có ghi rõ là việc phase sau không (mục 6)"; }
tc_12(){ local c miss=0; for c in $(grep -oE '\b[0-9]{6}\b' $D | sort -u); do grep -qF "$c" $ARC || { echo "mã lớp $c không có ở ARCHITECTURE §9"; miss=1; }; done; [ $miss = 0 ] && pass "mã lớp trong kịch bản có ở ARCHITECTURE" || fail "mã lớp lệch seed"
  miss=0; for c in $(grep -oE '`[A-Z0-9]{7}`' $D | tr -d '`' | sort -u); do grep -qF "$c" $ARC $FL docs/PRD.md docs/phases/*.md || { echo "mã tham gia $c không thấy ở tài liệu nào khác"; miss=1; }; done; [ $miss = 0 ] && pass "mã tham gia (7 ký tự) có ở tài liệu khác" || echo "info: mã tham gia là seed do chính kịch bản đặt (Q2) — không FAIL"
  local f; for f in F2 F3 F5 F7 F9 F10 F17; do grep -qE "^### $f\." $FL && pass "FLOWS có mục $f" || fail "FLOWS thiếu mục $f"; done; }
tc_13(){ eq "$(grep -ciE 'TODO|TBD|\.\.\.\.|XXX|lorem' $D)" 0 "không placeholder TODO/TBD/lorem"
  grep -qE '^Phiên bản 1' $D && pass "có dòng phiên bản" || fail "thiếu dòng phiên bản"
  ok_sp(){ [ -f docs/specs/FEAT-demo-script/QUESTIONS.md ]; }; ok_sp && pass "QUESTIONS.md tồn tại (Q1–Q9 không chặn)" || fail "thiếu QUESTIONS.md"; }

tc_14(){ local a miss=0; for a in $(grep -oE '[a-z.]+@edupilot\.local' $D | sort -u); do grep -qF "$a" $ARC || { echo "tài khoản $a không có ở ARCHITECTURE §9"; miss=1; }; done; [ $miss = 0 ] && pass "mọi tài khoản seed trong kịch bản khớp ARCHITECTURE §9" || fail "tài khoản lệch seed"
  grep -E 'sv\.kha@' $D | grep -q 'Sinh viên B' && pass "sv.kha = Sinh viên B (khớp ARCHITECTURE)" || fail "sv.kha không phải Sinh viên B"
  grep -E 'sv\.moi@' $D | grep -q 'Sinh viên D' && pass "sv.moi = Sinh viên D" || fail "sv.moi không phải Sinh viên D"
  grep -E 'sv\.nguyco@' $D | grep -q 'Sinh viên C' && pass "sv.nguyco = Sinh viên C" || fail "sv.nguyco không phải Sinh viên C"; }

[ $# -eq 0 ] && { echo "dùng: $0 TC-01 [TC-02 ...]"; exit 2; }
for t in "$@"; do TC=$t; f="tc_${t#TC-}"; if declare -F "$f" >/dev/null; then "$f"; else echo "FAIL $t: không có TC này"; RC=1; fi; done
exit $RC
