#!/bin/bash
# tạo lớp riêng cho nhóm D: mkcourse.sh <class_code> <tag> -> in id; GV teacher@, TA ta@, SV tham gia bằng mã
source /tmp/ep-r-lib.sh
CC=$1; AT=$(tok admin)
TID=$(psqlr -c "select id from users where email='teacher@edupilot.local'"); TAID=$(psqlr -c "select id from users where email='ta@edupilot.local'")
R=$(curl -sk -X POST $B/admin/courses -H "authorization: Bearer $AT" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"subject_code\":\"QCD\",\"class_code\":\"$CC\",\"name\":\"QC nhóm D $CC\",\"semester\":\"2026-2027-HK1\",\"capacity\":60,\"teacher_id\":\"$TID\"}")
CID=$(echo "$R" | jq -r .id); echo "course $CID $(echo $R | head -c 100)"
curl -sk -X POST $B/admin/courses/$CID/assign -H "authorization: Bearer $AT" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"ta_ids\":[\"$TAID\"]}" | head -c 200; echo
JC=$(psqlr -c "select join_code from courses where id='$CID'"); echo "join_code $JC"
for s in "${@:2}"; do c=$(curl -sk -o /dev/null -w '%{http_code}' -X POST $B/courses/join -H "authorization: Bearer $(tok $s)" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"code\":\"$JC\"}"); echo "join $s $c"; done
