#!/bin/bash
# TC-N06-04 trên ep-r: Redis tắt → chat phải fail nhanh (kỳ vọng ≤ 10 s, OVERLOADED). Chạy trong khoá toàn cục, Redis tắt ≤ 40 s, khôi phục bằng trap.
S=https://localhost:1143/api/v1; PW='Edupilot#Seed-2026'; L=/tmp/qc2-global-changes.log
mkdir /tmp/qc2-lock-global 2>/dev/null || { echo "khoá toàn cục đang bị giữ"; exit 3; }
echo "$(date +%T) DefR: LOCK dừng ep-r-redis-1 ≤ 40 s (TC-N06-04)" >> $L
restore(){ docker start ep-r-redis-1 >/dev/null 2>&1; rmdir /tmp/qc2-lock-global; echo "$(date +%T) DefR: UNLOCK redis start" >> $L; }
trap restore EXIT
curl -sk "$S/../readyz" 2>/dev/null | head -c 100; echo
tk(){ curl -sk -X POST $S/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"$1@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token; }
SV=$(tk sv.kha); T=$(tk teacher)
C=$(curl -sk "$S/me/courses?limit=20" -H "authorization: Bearer $T" | jq -r '.items[]|select(.course.class_code=="761987").course.id')
mk(){ curl -sk -X POST $S/chat/sessions -H "authorization: Bearer $SV" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"course_id\":\"$C\"}" | jq -r .id; }
send(){ curl -sk -N --max-time 60 -o /tmp/qc2-redis-out.txt -w '%{http_code} %{time_total}' -X POST $S/chat/sessions/$1/messages -H "authorization: Bearer $SV" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d '{"content":"Điểm quá trình chiếm bao nhiêu phần trăm?"}'; echo " $(head -c 200 /tmp/qc2-redis-out.txt | tr '\n' ' ')"; }
S1=$(mk); S2=$(mk); S3=$(mk); echo "phiên: $S1 $S2 $S3"
echo "-- Redis chạy (đối chứng)"; send $S1
echo "$(date +%T) REDIS STOP"; docker stop -t 2 ep-r-redis-1 >/dev/null
for i in 1 2 3; do echo "-- t=$(date +%T) lượt $i"; send $([ $i = 1 ] && echo $S2 || echo $S3); done
echo "-- readyz khi Redis tắt: $(curl -sk -o /dev/null -w '%{http_code}' https://localhost:1143/readyz)"
echo "-- tạo phiên khi Redis tắt:"; curl -sk -o /dev/null -w '%{http_code} %{time_total}\n' -X POST $S/chat/sessions -H "authorization: Bearer $SV" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"course_id\":\"$C\"}"
docker start ep-r-redis-1 >/dev/null; echo "$(date +%T) REDIS START"
for i in $(seq 1 20); do r=$(curl -sk https://localhost:1143/readyz); echo "$r" | grep -q '"redis":"up"' && { echo "readyz sau start: $r (+${i}s)"; break; }; sleep 1; done
