#!/bin/bash
# N05-02: kịch bản nhẹ (dashboard GV + đọc SV, như benchmarks/load/t1.js) trên smk91 (pg_stat_statements bật) rồi top 10 theo total_exec_time. Dùng: n05-light.sh [vòng]
S=https://localhost:1343/api/v1; PW='Edupilot#Seed-2026'; N=${1:-25}
tk(){ curl -sk -X POST $S/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"$1@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token; }
T=$(tk teacher); SV=$(tk sv.gioi)
PSQL="docker exec -i smk91-postgres-1 psql -U edupilot -d edupilot -At"
$PSQL -c "select pg_stat_statements_reset()" >/dev/null   # bỏ nhiễu đăng nhập
g(){ curl -sk -o /dev/null -w '%{http_code} ' "$S$1" -H "authorization: Bearer $2"; }
C=$(curl -sk "$S/me/courses?limit=20" -H "authorization: Bearer $T" | jq -r '.items[]|select(.course.class_code=="761987").course.id')
for i in $(seq $N); do
  for c in $C; do
    g /me/today $T; g /courses/$c/today $T; g "/courses/$c/students?limit=100" $T; g /courses/$c/gradebook $T
    F=$(curl -sk "$S/courses/$c/students?limit=5" -H "authorization: Bearer $T" | jq -r '.items[0].student.id'); g /courses/$c/students/$F/profile $T
    g "/courses/$c/attendance/sessions" $T
  done
  g /me/today $SV; g "/courses/$C/me/grades" $SV; g "/courses/$C/me/attendance?limit=100" $SV; g "/courses/$C/threads?limit=30" $SV; g "/courses/$C/library" $SV; g "/notifications?limit=20" $SV
  if [ $((i % 10)) = 1 ]; then  # RAG: 1 câu chat mỗi 10 vòng (nhà cung cấp fake)
    SID=$(curl -sk -X POST $S/chat/sessions -H "authorization: Bearer $SV" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"course_id\":\"$C\"}" | jq -r .id)
    curl -sk -N --max-time 40 -X POST $S/chat/sessions/$SID/messages -H "authorization: Bearer $SV" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d '{"content":"Điểm quá trình chiếm bao nhiêu phần trăm điểm học phần?"}' -o /dev/null
  fi
done; echo
$PSQL -F '|' -c "SELECT queryid, calls, round(total_exec_time::numeric,1) total_ms, round(mean_exec_time::numeric,3) mean_ms, left(regexp_replace(query,'\s+',' ','g'),110) q FROM pg_stat_statements WHERE dbid=(select oid from pg_database where datname='edupilot') ORDER BY total_exec_time DESC LIMIT 10"
