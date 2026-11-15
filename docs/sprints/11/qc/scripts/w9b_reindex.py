from rlib import *
import json, time
c1 = C("teacher")
c, dl, _ = api("teacher", "GET", f"/courses/{c1}/documents?limit=100")
docs = [d for d in dl["items"] if d.get("status") == "READY" and d.get("use_for_rag")]
print("tài liệu READY dùng cho AI:", len(docs))
def calls(): return sql("select count(*), coalesce(sum(attempts),0) from llm_audit where provider='gemini-embed-qcR'")[0]
tot = 0
for d in sorted(docs, key=lambda x: x["title"]):
    n = int(sql(f"select count(*) from content_chunks where document_id='{d['id']}'")[0][0])
    for att in range(4):
        c, j, _ = api("teacher", "POST", f"/courses/{c1}/documents/{d['id']}/reindex", {})
        jid = (j or {}).get("job_id"); st = None
        for _ in range(60):
            time.sleep(2)
            r = sql(f"select status from jobs where id='{jid}'") if jid else []
            if r and r[0][0] in ("DONE", "SUCCEEDED", "COMPLETED", "FAILED", "DEAD"): st = r[0][0]; break
        print(f"  {d['title'][:36]:36} {n:3} chunk  lần {att+1}: HTTP {c} job {st}  | gemini rows/attempts {calls()}", flush=True)
        if st in ("DONE", "SUCCEEDED", "COMPLETED"): break
        time.sleep(65)
    time.sleep(8 if n < 30 else 62)
print("tổng:", calls())
