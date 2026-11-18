#!/bin/bash
# N06-05: client SSE (/events) mở TRƯỚC khi dừng Redis; dừng Redis ~20 s; sau khi Redis lên nối lại bằng Last-Event-ID. Stack smk91.
G=https://localhost:1343; PW='Edupilot#Seed-2026'; O=${1:-/tmp/qc2-r95b/sse}; mkdir -p $O; rm -f $O/*
tk(){ curl -sk -X POST $G/api/v1/auth/login -H 'content-type: application/json' -d "{\"email\":\"$1@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token; }
A=$(tk admin); T=$(tk sv.gioi); UID_=$(curl -sk -H "Authorization: Bearer $T" $G/api/v1/me/profile | jq -r '.id // .user.id')
push(){ curl -sk -o /dev/null -w '%{http_code}' -X POST -H "Authorization: Bearer $A" -H 'content-type: application/json' -d "{\"type\":\"test.ping\",\"data\":{\"n\":$1},\"user_id\":\"$UID_\"}" $G/api/v1/_test/events; }
curl -skN --max-time 150 -H "Authorization: Bearer $T" $G/api/v1/events > $O/f1.txt 2>/dev/null &
P1=$!; sleep 2
for n in 1 2 3 4 5; do echo "push $n $(push $n)" >> $O/push.log; sleep 0.5; done
sleep 1; echo "== stop redis $(date +%T)" >> $O/push.log; docker stop -t 1 smk91-redis-1 >/dev/null
for n in 6 7 8; do echo "push $n (redis tắt) $(push $n)" >> $O/push.log; sleep 2; done
sleep 8; docker start smk91-redis-1 >/dev/null; echo "== start redis $(date +%T)" >> $O/push.log; sleep 12
echo "client f1 còn mở: $(kill -0 $P1 2>/dev/null && echo có || echo không)" >> $O/push.log
for n in 9 10 11; do echo "push $n $(push $n)" >> $O/push.log; sleep 1; done
sleep 3; kill -9 $P1 2>/dev/null; wait $P1 2>/dev/null
LAST=$(grep -E '^id: ' $O/f1.txt | tail -1 | sed 's/^id: //' | tr -d '\r'); echo "Last-Event-ID=$LAST" >> $O/push.log
for n in 12 13; do echo "push $n $(push $n)" >> $O/push.log; sleep 1; done
curl -skN --max-time 8 -H "Authorization: Bearer $T" -H "Last-Event-ID: $LAST" $G/api/v1/events > $O/f2.txt 2>/dev/null
echo "f1 ids: $(grep -c '^id: ' $O/f1.txt) · n nhận: $(grep -o '"n":[0-9]*' $O/f1.txt | tr '\n' ' ')" >> $O/push.log
echo "f2 ids: $(grep -c '^id: ' $O/f2.txt) · n nhận: $(grep -o '"n":[0-9]*' $O/f2.txt | tr '\n' ' ')" >> $O/push.log
cat $O/push.log
