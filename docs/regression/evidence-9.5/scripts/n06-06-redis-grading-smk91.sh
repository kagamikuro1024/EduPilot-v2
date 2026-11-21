#!/bin/bash
# TC-N06-06 (SmkS4, 2026-11-21): lô chấm BT03 (22 bài) đang chạy → dừng smk91-redis-1 60 s → bật lại → không lô chấm mồ côi (QUEUED/RUNNING còn sau 8 phút, XPENDING > 0, mất ánh xạ lô). Stack smk91.
S=https://localhost:1343/api/v1; PW='Edupilot#Seed-2026'; C=01a1ef53-79d0-75ea-8b95-e44216e58f21; A=01a1ef58-36e3-7679-9e81-8c8903db01cb
P="docker exec -i smk91-postgres-1 sh -c 'psql -U edupilot -d edupilot -At -F|; exit 0'"; R="docker exec smk91-redis-1 redis-cli"
pq(){ echo "$1" | docker exec -i smk91-postgres-1 sh -c 'psql -U edupilot -d edupilot -At -F"|"; exit 0'; }
T=$(curl -sk -X POST $S/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"teacher@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token)
W="submission_id in (select id from submissions where assignment_id='$A')"
st(){ echo "$(date +%T) $(pq "select run_state,count(*) from grading_results where $W group by 1" | tr '\n' ' ') xpending=$($R xpending ep:grading grading | head -1) dead=$($R xlen ep:grading.dead) keys_batch=$($R --scan --pattern '*batch*' | wc -l)"; }
pq "delete from grading_results where $W" >/dev/null
Q=$(curl -sk -X POST "$S/courses/$C/assignments/$A/grade-all" -H "authorization: Bearer $T" -H 'content-type: application/json' -H "Idempotency-Key: n0606-$(date +%s)" -d '{}' | jq -r .queued); echo "grade-all queued=$Q"
sleep 8; st; echo "REDIS STOP $(date +%T)"; docker stop -t 1 smk91-redis-1 >/dev/null
for i in 1 2 3; do sleep 20; st 2>&1 | head -1; done
docker start smk91-redis-1 >/dev/null; echo "REDIS START $(date +%T)"; sleep 8
for i in $(seq 120); do sleep 4; n=$(pq "select count(*) from grading_results where run_state in ('QUEUED','RUNNING') and $W"); [ "$n" = 0 ] && break; done; st
echo "KQ: tổng=$(pq "select count(*) from grading_results where $W") DONE=$(pq "select count(*) from grading_results where run_state='DONE' and $W") FAILED=$(pq "select count(*) from grading_results where run_state='FAILED' and $W") hai_lần=$(pq "select count(*) from (select submission_id, round from grading_results where $W group by 1,2 having count(*)>1) x") mồ_côi=$(pq "select count(*) from grading_results where run_state in ('QUEUED','RUNNING') and $W") chưa_có_kq=$(pq "select count(*) from submissions s where s.assignment_id='$A' and s.match_status='MATCHED' and not exists (select 1 from grading_results r where r.submission_id=s.id)")"
