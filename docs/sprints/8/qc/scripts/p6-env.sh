# QC P6 — nguồn (source) biến dùng chung cho mọi TC. Cần stack đang chạy + seed (`pnpm dev`).
# Dùng:  source docs/sprints/8/qc/scripts/p6-env.sh
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
