# TC-N08-11: «escalation theo độ tin cậy» cần ≥ 3 mẫu + 1 dòng đánh giá định tính/mẫu. Harness chat chỉ có Q10, Q11.
# Gửi thêm 4 câu hỏi ngoài tài liệu lớp qua đúng đường chat (chat_once của harness) rồi đọc low_confidence / ticket.
import os, sys, json
os.environ.update(BASE="https://localhost:1143", INSECURE_TLS="1", OUT=os.path.abspath("out"),
                  AUDIT_PSQL="docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -tA -v ON_ERROR_STOP=1")
sys.path.insert(0, "../../../../benchmarks/real-llm")
import real_llm as R
api = R.Api() if hasattr(R, "Api") else None
Q = ["Học phí học kỳ sau của em là bao nhiêu tiền ạ?", "Giảng viên có cho dời hạn nộp bài tập 03 sang tuần sau không ạ?",
     "Lịch bảo vệ đồ án tốt nghiệp của khoa mình vào ngày nào ạ?", "Thầy dạy lớp mình tên đầy đủ là gì và văn phòng thầy ở phòng nào?"]
course = R.c1_id(api)
out = []
for i, t in enumerate(Q, 1):
    r = R.chat_once(api, "sv.kha", course, t)
    out.append({"id": f"E{i}", "text": t, "intent": (r.get("db") or {}).get("intent"), "low_confidence": r["low_confidence"], "ticket": r["ticket"],
                "confidence": (r.get("db") or {}).get("confidence"), "citations": r["citations"], "error": r["error"], "content": r["content"]})
    print(out[-1]["id"], {k: out[-1][k] for k in ("intent", "low_confidence", "ticket", "confidence", "citations", "error")}); R.pause(6)
json.dump(out, open("out/escalate.json", "w"), ensure_ascii=False, indent=1)
