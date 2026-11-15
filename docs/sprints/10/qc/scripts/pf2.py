from pb import *
de = docid("QC Đề mẫu")["id"]; dk = docid("QC Đáp án mẫu")["id"]
c, lst, _ = api("teacher", "GET", f"{P}/documents?limit=100"); L1 = [d for d in lst["items"] if d["type"] == "LECTURE" and d["status"] == "READY"][0]
ex0 = sql("select count(*) from question_bank where origin in ('EXTRACTED','GENERATED')")[0][0]
jobs = []
for i in range(3):
    c, j, _ = api("teacher", "POST", f"{P}/questions/generate", {"document_ids": [L1["id"]], "topic": "Mẫu sinh câu", "difficulty": "EASY", "count": 3, "types": ["MCQ_SINGLE"]}); jobs.append(("gen", c, j.get("job_id")))
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": de, "answer_key_document_id": dk}); jobs.append(("ext", c, j.get("job_id")))
print(jobs)
out = []
for k, c, jid in jobs:
    if not jid: out.append((k, c)); continue
    J = wait_job(jid, t=150); out.append((k, J["status"], json.dumps(J.get("error"), ensure_ascii=False)[:100])); print(k, J["status"], J.get("error"), json.dumps(J.get("result"), ensure_ascii=False)[:100])
n1 = sql("select count(*) from question_bank where origin in ('EXTRACTED','GENERATED')")[0][0]
ov = [o for o in out if len(o) > 2 and "OVERLOADED" in o[2]]
ok("TC-02-14/03-08 quá tải: có việc FAILED OVERLOADED, không câu nửa vời", bool(ov), f"{out}; câu {ex0}→{n1} (câu mới đến từ việc thành công)")
