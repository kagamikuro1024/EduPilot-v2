from pb import *
# Tái hiện L1 + L2 trên stack sạch (seed mới)
d = {}
for key, path, title, typ in [("de", "seed/practice/de-mau-trac-nghiem.pdf", "QC Đề mẫu", "EXAM_PAPER"), ("dapan", "seed/practice/dap-an-mau.pdf", "QC Đáp án mẫu", "ANSWER_KEY")]:
    d[key] = upload(path, title, typ)["document"]["id"]
for k in d: print(k, wait_doc(d[k])["status"])
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": d["de"], "answer_key_document_id": d["dapan"]}); J = wait_job(j["job_id"]); print("L1:", c, J["status"], json.dumps(J["result"], ensure_ascii=False))
print("EXTRACTED:", sql("select count(*) from question_bank where origin='EXTRACTED'"), "chunks:", sql(f"select count(*), sum(length(text)) from content_chunks where document_id='{d['de']}'"))
l2 = api("teacher", "GET", f"/courses/{C2}/documents?limit=100")[1]["items"]; ep = [x for x in l2 if x["type"] == "EXAM_PAPER"]
print("lớp 2 EXAM_PAPER:", [(x["title"], x["id"] in [y["id"] for y in api("teacher","GET",P+"/documents?limit=100")[1]["items"]]) for x in ep])
if ep:
    c, j, _ = api("ta", "POST", f"{P}/questions/extract", {"document_id": ep[0]["id"]}); print("L2: TA lớp 1 + tài liệu lớp 2 ->", c, j)
    if c == 202: J = wait_job(j["job_id"], "ta"); print("   job", J["status"], json.dumps(J["result"], ensure_ascii=False))
