from pa1_lib import *
from pb import wait_job
lec = [d for d in api("teacher", "GET", f"{P}/documents?limit=100")[1]["items"] if d["type"] == "LECTURE" and d["status"] == "READY"][0]
c, j, _ = api("teacher", "POST", f"{P}/questions/generate", {"document_ids": [lec["id"]], "topic": "Mẫu sinh câu", "difficulty": "MEDIUM", "count": 5, "types": ["MCQ_SINGLE"]}); jid = j["job_id"]
J = wait_job(jid); ids = J["result"]["created"]; print("created", len(ids), J["result"]["skipped"][:2])
# duyệt 2 nguyên, 1 sửa rồi duyệt, 2 loại
for q in ids[:2]: review(q, "REQUEST"); r = review(q, "APPROVE"); print("approve", r[0])
d = detail(ids[2]); body = {"type": d["type"], "title": d["title"], "topic": d["topic"], "difficulty": d["difficulty"], "stem": d["stem"] + " (đã sửa)", "options": [{"body": o["body"]} for o in d["options"]], "correct": [i for i, o in enumerate(d["options"]) if o["id"] in d["answer_key"]["option_ids"]], "version": d["version"]}
c, j, _ = api("teacher", "PUT", f"{P}/questions/{ids[2]}", body); print("edit", c, json.dumps(j, ensure_ascii=False)[:100] if c != 200 else "")
review(ids[2], "REQUEST"); print("approve edited", review(ids[2], "APPROVE")[0])
for q in ids[3:]: review(q, "REQUEST"); print("reject", review(q, "REJECT")[0])
c, st, _ = api("teacher", "GET", f"{P}/questions/gen-stats?job_id={jid}"); print("gen-stats", c, st)
ok("TC-03-11 gen-stats {unchanged:2, edited:1, rejected:2}", c == 200 and st.get("unchanged") == 2 and st.get("edited") == 1 and st.get("rejected") == 2, f"{c} {st}")
# TC-03-09: sinh lại → trùng ở mọi trạng thái kể cả REJECTED
c, j, _ = api("teacher", "POST", f"{P}/questions/generate", {"document_ids": [lec["id"]], "topic": "Mẫu sinh câu", "difficulty": "MEDIUM", "count": 5, "types": ["MCQ_SINGLE"]}); J2 = wait_job(j["job_id"])
sk = J2["result"]["skipped"]
ok("TC-03-09 sinh lại: câu trùng bị bỏ qua (kể cả REJECTED), không tạo thêm", not J2["result"]["created"] and all(s["reason"] == "DUPLICATE" for s in sk) and len(sk) == 5, f"created={len(J2['result']['created'])} skipped={[s['reason'] for s in sk]}")
# TC-06-08c với câu GENERATED đã duyệt có trích dẫn: SV thấy citations?
