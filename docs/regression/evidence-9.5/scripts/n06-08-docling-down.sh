#!/bin/bash
# TC-N06-08 (SmkS4, 2026-11-21): trong lúc smk91-docling-1 tắt, chat hỏi về tài liệu đã READY → 200 + trích dẫn.
# Câu hỏi = nguyên văn một đoạn (content_chunks.text) của tài liệu READY (embedding giả theo băm ⇒ chỉ trùng nguyên văn mới vượt sàn cosine; mỗi câu một đoạn khác nhau vì câu lặp bị trả từ bộ nhớ đệm, 0 trích dẫn).
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-9.5/scripts
ck(){ docker exec smk91-postgres-1 sh -c "psql -U edupilot -d edupilot -At -c \"select c.text from content_chunks c join documents d on d.id=c.document_id where d.title='$1' and c.ord=$2 limit 1\"; exit 0" 2>/dev/null | tr '\n' ' '; }
echo "Docling trước: $(docker inspect -f '{{.State.Status}}' smk91-docling-1)"; ./n06-08-chat-docs.sh "trước khi tắt  [Hỏi đáp đã được giảng viên xác nhận #0]" "$(ck 'Hỏi đáp đã được giảng viên xác nhận' 0)"
docker stop smk91-docling-1 >/dev/null; echo "DOCLING STOP $(date +%T) trạng thái=$(docker inspect -f '{{.State.Status}}' smk91-docling-1)"
./n06-08-chat-docs.sh "docling tắt 1 [Quy chế học phần #1]" "$(ck 'Quy chế học phần' 1)"; sleep 15
./n06-08-chat-docs.sh "docling tắt 2 [Quy chế học phần #2]" "$(ck 'Quy chế học phần' 2)"; sleep 15
./n06-08-chat-docs.sh "docling tắt 3 [Đề tham khảo tuần 5 #0]" "$(ck 'Đề tham khảo tuần 5' 0)"; sleep 15
./n06-08-chat-docs.sh "docling tắt 4 [Hỏi đáp #1]" "$(ck 'Hỏi đáp đã được giảng viên xác nhận' 1)"
echo "docling vẫn tắt: $(docker inspect -f '{{.State.Status}}' smk91-docling-1)"
docker start smk91-docling-1 >/dev/null; echo "DOCLING START $(date +%T)"
