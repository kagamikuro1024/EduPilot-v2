#!/bin/bash
# N05-02 (lần 2, trọng số như k6 (a)+(d): đọc SV nhiều gấp ~10 lần dashboard GV). Dùng: n05-browse-mix.sh [số vòng SV]
S=https://localhost:1343/api/v1; PW='Edupilot#Seed-2026'; N=${1:-300}
tk(){ curl -sk -X POST $S/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"$1@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token; }
T=$(tk teacher); SV=$(tk sv.gioi); SV2=$(tk sv.kha)
PSQL="docker exec -i smk91-postgres-1 psql -U edupilot -d edupilot -At"
C=$(curl -sk "$S/me/courses?limit=20" -H "authorization: Bearer $T" | jq -r '.items[]|select(.course.class_code=="761987").course.id')
F=$(curl -sk "$S/courses/$C/students?limit=5" -H "authorization: Bearer $T" | jq -r '.items[0].student.id')
$PSQL -c "select pg_stat_statements_reset()" >/dev/null
export S T SV SV2 C F
sv(){ for u in /me/today /courses/$C/me/grades "/courses/$C/me/attendance?limit=100" "/courses/$C/threads?limit=30" /courses/$C/library "/notifications?limit=20"; do for t in $SV $SV2; do curl -sk -o /dev/null "$S$u" -H "authorization: Bearer $t"; done; done; }
tc(){ for u in /me/today /courses/$C/today "/courses/$C/students?limit=100" /courses/$C/gradebook /courses/$C/students/$F/profile; do curl -sk -o /dev/null "$S$u" -H "authorization: Bearer $T"; done; }
export -f sv tc
seq $N | xargs -P4 -I{} bash -c 'sv'
seq $((N/10)) | xargs -P2 -I{} bash -c 'tc'
$PSQL -F '|' -c "SELECT calls, round(total_exec_time::numeric,1) total_ms, round(mean_exec_time::numeric,3) mean_ms, left(regexp_replace(query,'\\s+',' ','g'),120) q FROM pg_stat_statements WHERE dbid=(select oid from pg_database where datname='edupilot') ORDER BY total_exec_time DESC LIMIT 10"
