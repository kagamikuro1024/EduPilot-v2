#!/usr/bin/env bash
# QC P7 — ma trận route × vai (SRS 2 + 6). In từng dòng route|vai|mã thực tế|mong đợi; thoát 1 nếu lệch.
# Dùng: source p7-env.sh && bash p7-rbac.sh        (cần stack + seed; dùng id thật $A3, $SUB_B, $RES_B; thân rỗng chỉ để chạm tầng phân quyền, 403 phải đến TRƯỚC 422)
# CẢNH BÁO: vai TEACHER/TA thật sự gọi `grade-all`, `sync-now`, `teams/sync` (có tác dụng phụ) → chạy trên DB thử đã seed, KHÔNG chạy sau khi chấm/công bố xong.
set -u; source "$(dirname "$0")/p7-env.sh"
NOAUTH=""; OTHER_TCH="$(auth teacher2@edupilot.local 2>/dev/null)"; case "$OTHER_TCH" in *null*|*Bearer\ ) OTHER_TCH="";; esac   # giảng viên lớp khác: seed sprint 9 chỉ có MỘT giảng viên (dạy cả hai lớp) → bỏ qua hàng này
fail=0
# route (method path) | S A T AD  — mã kỳ vọng khi thân rỗng/không hợp lệ: ✓ = "ok" (2xx/4xx nghiệp vụ nhưng KHÔNG 401/403)
rows=(
"GET $Q/assignments|ok ok ok 403"
"POST $Q/assignments|403 403 ok 403"
"GET $Q/assignments/$A3|ok ok ok 403"
"PUT $Q/assignments/$A3|403 403 ok 403"
"DELETE $Q/assignments/$A3|403 403 ok 403"
"POST $Q/assignments/$A3/publish|403 403 ok 403"
"PUT $Q/assignments/$A3/rubric|403 403 ok 403"
"POST $Q/assignments/$A3/rubric/suggest|403 ok ok 403"
"GET $Q/assignments/$A3/mine|ok 403 403 403"
"POST $Q/assignments/$A3/submit/presign|ok 403 403 403"
"POST $Q/assignments/$A3/submit|ok 403 403 403"
"POST $Q/uploads/presign|403 ok ok 403"
"POST $Q/assignments/$A3/submissions/upload|403 ok ok 403"
"GET $Q/submissions|403 ok ok 403"
"PUT $Q/submissions/$SUB_B/match|403 ok ok 403"
"GET $Q/submissions/$SUB_B/file|403 ok ok 403"
"GET $Q/mail-inbound|403 ok ok 403"
"POST $Q/assignments/$A3/grade-all|403 ok ok 403"
"POST $Q/submissions/$SUB_B/grade|403 ok ok 403"
"GET $Q/grading-results/$RES_B|403 ok ok 403"
"PUT $Q/grading-results/$RES_B|403 ok ok 403"
"POST $Q/grading-results/$RES_B/review|403 ok ok 403"
"POST $Q/grading-results/$RES_B/retry|403 ok ok 403"
"POST $Q/grading-results/publish|403 403 ok 403"
"GET $Q/assignments/$A3/stats|403 ok ok 403"
"POST $Q/assignments/$A3/appeal|ok 403 403 403"
"GET $Q/appeals|403 ok ok 403"
"POST $Q/assignments/$A3/teams/sync|403 ok ok 403"
"POST $Q/submissions/$SUB_B/dismiss|403 ok ok 403"
"DELETE $Q/submissions/$SUB_B/dismiss|403 ok ok 403"
"GET $Q/integrations/imap|403 403 ok ok"
"PUT $Q/integrations/imap|403 403 403 ok"
"POST $Q/integrations/imap/test|403 403 403 ok"
"POST $Q/integrations/imap/sync-now|403 ok ok 403"
)
check() { # method url hdr expect
  local m="$1" u="$2" h="$3" e="$4" c
  c=$(curl -sk -o /dev/null -w '%{http_code}' -X "$m" -H "$h" -H 'Content-Type: application/json' -H "Idempotency-Key: $(uuidgen)" -d '{}' "$u")
  if [ "$e" = ok ]; then [ "$c" != 401 ] && [ "$c" != 403 ] || { echo "LỆCH $m $u [$e] thực $c"; fail=1; }
  else [ "$c" = "$e" ] || { echo "LỆCH $m $u [$e] thực $c"; fail=1; }; fi
}
for r in "${rows[@]}"; do
  spec="${r%%|*}"; ex="${r##*|}"; m="${spec%% *}"; u="${spec#* }"; read -r es ea et ead <<<"$ex"
  check "$m" "$u" "$SVA" "$es"; check "$m" "$u" "$TA_" "$ea"; check "$m" "$u" "$TCH" "$et"; check "$m" "$u" "$ADM" "$ead"
  c=$(curl -sk -o /dev/null -w '%{http_code}' -X "$m" "$u"); [ "$c" = 401 ] || { echo "LỆCH không đăng nhập $m $u thực $c"; fail=1; }
  check "$m" "$u" "$SVD" 403          # sinh viên ngoài lớp
  [ -n "$OTHER_TCH" ] && check "$m" "$u" "$OTHER_TCH" 403   # giảng viên lớp khác
done
[ $fail = 0 ] && echo "RESULT: PASS (ma trận ${#rows[@]} route × 7 vai)" || { echo "RESULT: FAIL"; exit 1; }
