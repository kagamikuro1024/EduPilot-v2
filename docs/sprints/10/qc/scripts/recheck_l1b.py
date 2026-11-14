# L1 (tiếp): đề lỗi câu / lỗi schema / không có bảng đáp án sau bản vá chunk
from pb import *
def ensure(path, title, typ):
    import hashlib
    b = open(ROOT + path, "rb").read(); sha = hashlib.sha256(b).hexdigest()
    c, pre, _ = api("teacher", "POST", f"{P}/uploads/presign", {"purpose": "document", "filename": os.path.basename(path), "mime_type": "application/pdf", "size_bytes": len(b), "sha256": sha})
    urllib.request.urlopen(urllib.request.Request(pre["url"], data=b, method=pre["method"], headers=pre["headers"]), context=CTX).read()
    c, d, _ = api("teacher", "POST", f"{P}/uploads/complete", {"upload_id": pre["upload_id"], "title": title, "type": typ})
    if c == 409: return d["details"]["existing_id"]
    did = d["document"]["id"]; wait_doc(did); return did
n0 = len(bank())
lc = ensure("seed/practice/de-mau-loi-cau.pdf", "QC Đề lỗi câu", "EXAM_PAPER")
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": lc}); print(c, j); J = wait_job(j["job_id"]); r = J["result"]
print("loi-cau", J["status"], len(r["created"]), [(s["number"], s["reason"]) for s in r["skipped"]])
ok("RECHECK-TC-02-07a đề lỗi câu: lưu 1, INVALID ở 2,3,4, NO_ANSWER ở 5", len(r["created"]) == 1 and {(s["number"], s["reason"]) for s in r["skipped"]} >= {(2, "INVALID"), (3, "INVALID"), (4, "INVALID"), (5, "NO_ANSWER")}, f"created={len(r['created'])} skipped={[(s['number'], s['reason']) for s in r['skipped']]}")
ls = ensure("seed/practice/de-mau-loi-schema.pdf", "QC Đề lỗi schema", "EXAM_PAPER")
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": ls}); J = wait_job(j["job_id"]); n1 = sql("select count(*) from question_bank where origin='EXTRACTED'")[0][0]
print("loi-schema", J["status"], J.get("error"))
ok("RECHECK-TC-02-07b lỗi schema: job FAILED, không câu nào lưu", J["status"] in ("FAILED", "DEAD") and "LLM_UNAVAILABLE" in json.dumps(J.get("error")), f"status={J['status']} error={json.dumps(J.get('error'), ensure_ascii=False)[:100]}")
