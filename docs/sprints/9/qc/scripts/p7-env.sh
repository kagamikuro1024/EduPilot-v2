# QC P7 — nguồn (source) biến dùng chung cho mọi TC. Cần stack đang chạy + seed (`pnpm dev`).
# Dùng:  source docs/sprints/9/qc/scripts/p7-env.sh
# GW mặc định theo SRS 9.1 (http://localhost:8080); đổi qua env nếu stack test ở https://localhost.
GW="${GW:-http://localhost:8080}"; PW="${PW:-Edupilot#Seed-2026}"
PSQL="${PSQL:-psql $DATABASE_URL -tAc}"
j() { curl -sk -H 'Content-Type: application/json' "$@"; }
idem() { echo "Idempotency-Key: $(uuidgen)"; }
login() { j -X POST "$GW/api/v1/auth/login" -d "{\"email\":\"$1\",\"password\":\"$PW\"}" | jq -r '.access_token // .data.access_token'; }
auth() { echo "Authorization: Bearer $(login "$1")"; }
TCH="$(auth teacher@edupilot.local)"; TA_="$(auth ta@edupilot.local)"; ADM="$(auth admin@edupilot.local)"
SVA="$(auth sv.gioi@edupilot.local)"; SVB="$(auth sv.kha@edupilot.local)"; SVC="$(auth sv.nguyco@edupilot.local)"
C1="$($PSQL "select id from courses where class_code='761987'")"; C2="$($PSQL "select id from courses where class_code='761988'")"
Q="$GW/api/v1/courses/$C1"; Q2="$GW/api/v1/courses/$C2"
SVB_ID="$($PSQL "select u.id from users u where u.email='sv.kha@edupilot.local'")"
SVC_ID="$($PSQL "select u.id from users u where u.email='sv.nguyco@edupilot.local'")"
POLICY1="$($PSQL "select id from documents where course_id='$C1' and type='COURSE_POLICY' limit 1")"
# code <lệnh curl…> → in mã HTTP
code() { curl -sk -o /dev/null -w '%{http_code}' "$@"; }
# --- bổ sung P7 (SRS 9.1) ---
MP="${MP:-http://127.0.0.1:8025}"
SVD="$(auth sv.khac@edupilot.local)"     # D: tài khoản chưa vào lớp (seed sprint 3)
A3="$($PSQL "select id from assignments where course_id='$C1' and submission_code='BT03'")"
A4="$($PSQL "select id from assignments where course_id='$C1' and submission_code='BT04'")"
SUB_B="$($PSQL "select id from submissions where assignment_id='$A3' and student_id='$SVB_ID' order by version desc limit 1")"
RES_B="$($PSQL "select id from grading_results where submission_id='$SUB_B' order by round desc limit 1")"
LLMADM_PSQL="$PSQL"                       # llm_audit đọc bằng SQL (QC không có route đọc cho ADMIN trong script)
