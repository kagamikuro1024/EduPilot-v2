#!/bin/bash
# N06-04: tải chat 1 QPS (SV) + dừng Redis 70 s (smk91-redis-1, stack riêng); mỗi lời gọi chat: mã HTTP + thời gian; kỳ vọng: suy giảm có kiểm soát (không 5xx thô, ≤ 10 s mỗi lời gọi, không treo)
S=https://localhost:1343/api/v1; PW='Edupilot#Seed-2026'; O=${1:-/tmp/smks/n0604}; mkdir -p $O; rm -f $O/*
tk(){ curl -sk -X POST $S/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"$1@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token; }
SV=$(tk sv.kha); T=$(tk teacher); C=$(curl -sk "$S/me/courses?limit=20" -H "authorization: Bearer $T" | jq -r '.items[]|select(.course.class_code=="761987").course.id')
SID=$(curl -sk -X POST $S/chat/sessions -H "authorization: Bearer $SV" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"course_id\":\"$C\"}" | jq -r .id)
t0=$(date +%s)
one(){ n=$1; s=$(date +%s); r=$(curl -sk -N --max-time 90 -o $O/body-$n.txt -w '%{http_code} %{time_total}' -X POST $S/chat/sessions/$SID/messages -H "authorization: Bearer $SV" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d '{"content":"Điểm quá trình chiếm bao nhiêu phần trăm?"}'); echo "t=$((s-t0)) n=$n kq=$r $(grep -c '^event: done' $O/body-$n.txt)done $(grep -o '"degraded":[a-z]*' $O/body-$n.txt | head -1) $(grep -o '"code":"[A-Z_]*"' $O/body-$n.txt | head -1)" >> $O/calls.log; }
for i in $(seq 1 100); do
  [ $i = 11 ] && { docker stop -t 1 smk91-redis-1 >/dev/null; echo "t=$(( $(date +%s)-t0 )) REDIS STOP" >> $O/calls.log; }
  [ $i = 81 ] && { docker start smk91-redis-1 >/dev/null; echo "t=$(( $(date +%s)-t0 )) REDIS START" >> $O/calls.log; }
  one $i &
  sleep 1
done; wait
sort -t= -k2 -n $O/calls.log | cat
