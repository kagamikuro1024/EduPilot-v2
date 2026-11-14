# Kiểm lại L1: trích đề docling thật → 24 câu EXTRACTED PENDING, đáp án khớp bảng, có citations
from pb import *
def get_or_upload(path, title, typ):
    try: return docid(title)["id"]
    except KeyError: d = upload(path, title, typ)["document"]["id"]; wait_doc(d); return d
de = get_or_upload("seed/practice/de-mau-trac-nghiem.pdf", "QC Đề mẫu", "EXAM_PAPER"); dk = get_or_upload("seed/practice/dap-an-mau.pdf", "QC Đáp án mẫu", "ANSWER_KEY")
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": de, "answer_key_document_id": dk}); print(c, j)
J = wait_job(j["job_id"]); r = J.get("result") or {}; print(J["status"], len(r.get("created", [])), "created;", [(s["number"], s["reason"]) for s in r.get("skipped", [])][:6])
ex = [q for q in bank() if q["origin"] == "EXTRACTED"]
exp = json.load(open(ROOT + "seed/practice/de-mau.json"))["questions"]
by = {}; nocit = 0
for q in ex:
    d = detail(q["id"]); by[d["stem"]] = d
    if not d.get("citations") or not all(isinstance(x.get("page"), int) and x.get("document_id") == de for x in d["citations"]): nocit += 1
good = bad = miss = 0
for e in exp:
    hit = next((d for s, d in by.items() if e["stem"][:30] in s or s[:30] in e["stem"]), None)
    if not hit: miss += 1; continue
    pos = sorted(o["position"] - 1 for o in hit["options"] if o["id"] in hit["answer_key"]["option_ids"]); cor = sorted(e["correct"] if isinstance(e["correct"], list) else [e["correct"]])
    good += pos == cor; bad += pos != cor
ok("RECHECK-L1 trích 24 câu, đáp án khớp bảng, trích dẫn đúng", len(ex) == 24 and good == 24 and nocit == 0 and all(q["review_status"] == "PENDING" for q in ex), f"EXTRACTED={len(ex)} khớp={good}/24 thiếu={miss} lệch={bad} thiếu citations={nocit}")
# idempotent: trích lại khoá mới → không thêm
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": de, "answer_key_document_id": dk}); J2 = wait_job(j["job_id"]); n2 = len([q for q in bank() if q["origin"] == "EXTRACTED"])
ok("RECHECK-L1b trích lại: không thêm câu, DUPLICATE", n2 == len(ex) and all(s["reason"] == "DUPLICATE" for s in J2["result"]["skipped"]), f"{len(ex)}→{n2}; {[s['reason'] for s in J2['result']['skipped']][:3]}")
# không đáp án
d0 = get_or_upload("seed/practice/de-mau-trac-nghiem.pdf", "QC Đề mẫu", "EXAM_PAPER")
