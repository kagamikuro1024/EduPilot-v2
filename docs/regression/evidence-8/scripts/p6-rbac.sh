#!/usr/bin/env bash
# QC P6 — ma trận phân quyền SRS 2 qua HTTP thật (stack + seed đang chạy).
# Vai bị cấm phải nhận 403 TRƯỚC khi chạm nghiệp vụ. Vai được phép gửi thân rỗng/sai: đạt nếu KHÔNG phải 401/403/5xx
# (400/404/409/422 là đạt — bằng chứng đã qua cổng quyền). Không đăng nhập → 401.
# Dùng:  bash docs/sprints/8/qc/scripts/p6-rbac.sh    (rc=0 khi hết FAIL)
set -u; source "$(dirname "$0")/p6-env.sh"
BAD=0; ID0=00000000-0000-0000-0000-000000000001
chk() { # chk "tên" mong mã   — mong = 401 | 403 | pass (không 401/403/5xx)
  case "$2" in
    pass) if [ "$3" != 401 ] && [ "$3" != 403 ] && [ "${3:0:1}" != 5 ] && [ "$3" != 000 ]; then echo "PASS $1 ($3)"; else echo "FAIL $1 mong qua cổng quyền, thật $3"; BAD=1; fi;;
    *)    if [ "$2" = "$3" ]; then echo "PASS $1 ($3)"; else echo "FAIL $1 mong $2 thật $3"; BAD=1; fi;;
  esac; }
# phương thức|URL|thân|nhóm quyền   — RD = TA+TEACHER đọc; WR = chỉ TEACHER; ME = chỉ STUDENT
ROUTES=(
 "GET|$Q/grade-scheme||RD" "GET|$Q/grade-scheme/status||RD" "POST|$Q/grade-scheme/impact|{}|RD"
 "GET|$Q/gradebook||RD" "GET|$Q/gradebook/preview||RD" "GET|$Q/gradebook/students/$SVB_ID/explain||RD" "GET|$Q/gradebook/students/$SVB_ID/history||RD"
 "GET|$Q/gradebook/snapshots||RD"
 "PUT|$Q/grade-scheme|{}|WR" "POST|$Q/grade-scheme/extract|{}|WR" "POST|$Q/grade-scheme/confirm|{}|WR"
 "PUT|$Q/grade-entries|{}|WR" "POST|$Q/grade-entries/import|{}|WR" "POST|$Q/uploads/presign|{\"purpose\":\"grade_import\"}|WR"
 "POST|$Q/grade-items/$ID0/publish|{}|WR" "POST|$Q/gradebook/finalize|{}|WR" "POST|$Q/gradebook/unlock|{}|WR" "GET|$Q/gradebook/export.xlsx||WR"
 "GET|$Q/me/grades||ME" "POST|$Q/me/grades/what-if|{}|ME"
)
for r in "${ROUTES[@]}"; do IFS='|' read -r m u b g <<<"$r"; short="$m ${u#*/courses/*/}"
  case "$g" in RD) s=403; t=pass; h=pass; a=403;; WR) s=403; t=403; h=pass; a=403;; ME) s=pass; t=403; h=403; a=403;; esac
  for who in "SVA:$s" "TA_:$t" "TCH:$h" "ADM:$a"; do n=${who%%:*}
    chk "$short [$n]" "${who##*:}" "$(code -X "$m" -H "${!n}" -H 'Content-Type: application/json' -H "$(idem)" ${b:+-d "$b"} "$u")"; done
  chk "$short [không đăng nhập]" 401 "$(code -X "$m" -H 'Content-Type: application/json' ${b:+-d "$b"} "$u")"
done
echo; if [ "$BAD" = 0 ]; then echo "RBAC: PASS"; else echo "RBAC: FAIL"; exit 1; fi
