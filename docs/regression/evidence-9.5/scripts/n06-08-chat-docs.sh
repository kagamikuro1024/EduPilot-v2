#!/bin/bash
# N06-08: chat hỏi nội dung TÀI LIỆU (không phải dữ liệu điểm) trên smk91 lớp 761987; $1 = nhãn; in mã HTTP, thời gian, số trích dẫn, 150 ký tự đầu của câu trả lời.
S=https://localhost:1343/api/v1; PW='Edupilot#Seed-2026'
tk(){ curl -sk -X POST $S/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"$1@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token; }
SV=$(tk sv.kha); T=$(tk teacher); C=$(docker exec -i smk91-postgres-1 sh -c "psql -U edupilot -d edupilot -At -c \"select id from courses where class_code='761987'\"; exit 0")
Q=${2:-"Theo quy chế môn học, điều kiện để được dự thi cuối kỳ là gì?"}
SID=$(curl -sk -X POST $S/chat/sessions -H "authorization: Bearer $SV" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"course_id\":\"$C\"}" | jq -r .id)
r=$(curl -sk -N --max-time 90 -o /tmp/n0608-body.txt -w '%{http_code} %{time_total}' -X POST $S/chat/sessions/$SID/messages -H "authorization: Bearer $SV" -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "$(jq -n --arg q "$Q" '{content:$q}')")
D=$(grep '^data:' /tmp/n0608-body.txt | grep '"citations"' | tail -1 | sed 's/^data: *//')
echo "$1 | http+giây=$r | trích_dẫn=$(echo "$D" | jq '.citations|length' 2>/dev/null) | degraded=$(echo "$D" | jq -r .degraded 2>/dev/null) low_conf=$(echo "$D" | jq -r .low_confidence 2>/dev/null) | trả lời: $(grep '^data:' /tmp/n0608-body.txt | grep -v citations | head -40 | sed 's/^data: *//' | jq -r '.text // .delta // empty' 2>/dev/null | tr -d '\n' | cut -c1-160)"
