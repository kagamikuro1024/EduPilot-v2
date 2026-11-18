#!/bin/bash
# TC-P6-05-46: hai worker cùng chạy → mỗi (lớp, tuần, GV) đúng một chuông. Dưới khoá global ≤ 5 phút.
source /tmp/ep-r-lib.sh
LOCK=/tmp/qc2-lock-global; until mkdir $LOCK 2>/dev/null; do sleep 10; done; echo qc2 > $LOCK/owner
echo "$(date +%H:%M:%S) qc2: LOCK — 05-46: dừng worker, thêm worker thứ hai tạm (ep-r-worker-qc2b), chạy 2 worker, hoàn lại; ≤ 5 phút" >> /tmp/qc2-global-changes.log
R="docker exec ep-r-redis-1 redis-cli"
cleanup() { docker rm -f ep-r-worker-qc2b >/dev/null 2>&1; docker start ep-r-worker-1 >/dev/null 2>&1; echo "$(date +%H:%M:%S) qc2: UNLOCK — 05-46 xong, worker-1 chạy lại, worker phụ đã xoá" >> /tmp/qc2-global-changes.log; rm -f $LOCK/owner; rmdir $LOCK; }
trap cleanup EXIT
IMG=$(docker inspect ep-r-worker-1 --format '{{.Config.Image}}')
docker inspect ep-r-worker-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -v '^$' > /tmp/qc2-w.env
docker stop ep-r-worker-1 >/dev/null
WK=$($R --scan --pattern 'ep:grade.remind:*' | head -1); echo "week key: $WK"
$R del $WK ep:lock:grade.remind >/dev/null
psqlr -c "delete from notifications where type='GRADE_SCHEME_REMINDER' and created_at > now() - interval '7 days'" 
N0=$(psqlr -Atc "select count(*) from notifications where type='GRADE_SCHEME_REMINDER'")
CMD=$(docker inspect ep-r-worker-1 --format '{{json .Config.Cmd}} {{json .Config.Entrypoint}}'); echo "cmd: $CMD"
docker run -d --name ep-r-worker-qc2b --network ep-r_default --env-file /tmp/qc2-w.env $IMG >/dev/null
docker start ep-r-worker-1 >/dev/null; date +%T
sleep 140
echo "workers up: $(docker ps --format '{{.Names}}' | grep -c 'ep-r-worker')"
psqlr -c "select user_id, course_id, count(*) c from notifications where type='GRADE_SCHEME_REMINDER' group by 1,2 having count(*)>1" 
TOTAL=$(psqlr -Atc "select count(*) from notifications where type='GRADE_SCHEME_REMINDER'")
DUP=$(psqlr -Atc "select count(*) from (select 1 from notifications where type='GRADE_SCHEME_REMINDER' group by user_id, course_id having count(*)>1) x")
DISTINCT=$(psqlr -Atc "select count(*) from (select 1 from notifications where type='GRADE_SCHEME_REMINDER' group by user_id, course_id) x")
echo "total=$TOTAL distinct_pairs=$DISTINCT dup_pairs=$DUP before=$N0"
echo "logsA: $(docker logs ep-r-worker-1 2>&1 | grep -ci 'remind')  logsB: $(docker logs ep-r-worker-qc2b 2>&1 | grep -ci 'remind')"
echo "RESULT total=$TOTAL distinct=$DISTINCT dup=$DUP"
