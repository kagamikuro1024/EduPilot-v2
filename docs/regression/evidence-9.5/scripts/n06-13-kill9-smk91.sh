#!/bin/bash
# N06-13: kill -9 worker giữa lượt chấm, 3 lần liên tiếp (stack smk91, 1 worker ⇒ khởi động lại worker ngay sau kill, như supervisor). Bài đúng một kết quả, không FAILED oan.
S=https://localhost:1343/api/v1; PW='Edupilot#Seed-2026'; C=01a1ef53-79d0-75ea-8b95-e44216e58f21; A=01a1ef58-36e3-7679-9e81-8c8903db01cb
P="docker exec -i smk91-postgres-1 psql -U edupilot -d edupilot -At -F|"; R="docker exec smk91-redis-1 redis-cli"
T=$(curl -sk -X POST $S/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"teacher@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token)
st(){ echo "$(date +%T) $($P -c "select run_state,count(*) from grading_results where submission_id in (select id from submissions where assignment_id='$A') group by 1" | tr '\n' ' ') xpending=$($R xpending ep:grading grading | head -1) dead=$($R xlen ep:grading.dead)"; }
for it in 1 2 3; do
  echo "=== lần $it"; $P -c "delete from grading_results where submission_id in (select id from submissions where assignment_id='$A')" >/dev/null
  Q=$(curl -sk -X POST "$S/courses/$C/assignments/$A/grade-all" -H "authorization: Bearer $T" -H 'content-type: application/json' -H "Idempotency-Key: k9-$it-$(date +%s)" -d '{}' | jq -r .queued); echo "grade-all queued=$Q"
  sleep $((6 + it * 3)); st; docker kill -s KILL smk91-worker-1 >/dev/null; echo "KILL -9 $(date +%T)"; sleep 4; st
  docker start smk91-worker-1 >/dev/null; echo "worker start $(date +%T)"
  for i in $(seq 60); do sleep 4; [ "$($P -c "select count(*) from grading_results where run_state in ('QUEUED','RUNNING') and submission_id in (select id from submissions where assignment_id='$A')")" = 0 ] && break; done; st
  echo "KQ lần $it: tổng=$($P -c "select count(*) from grading_results where submission_id in (select id from submissions where assignment_id='$A')") DONE=$($P -c "select count(*) from grading_results where run_state='DONE' and submission_id in (select id from submissions where assignment_id='$A')") FAILED=$($P -c "select count(*) from grading_results where run_state='FAILED' and submission_id in (select id from submissions where assignment_id='$A')") hai_lần=$($P -c "select count(*) from (select submission_id, round from grading_results where submission_id in (select id from submissions where assignment_id='$A') group by 1,2 having count(*)>1) x") mồ_côi=$($P -c "select count(*) from grading_results where run_state in ('QUEUED','RUNNING') and submission_id in (select id from submissions where assignment_id='$A')") xpending=$($R xpending ep:grading grading | head -1) dead=$($R xlen ep:grading.dead)"
  sleep 5
done
