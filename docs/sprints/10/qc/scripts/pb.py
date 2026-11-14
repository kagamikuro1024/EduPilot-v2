# US-P9-02 (trích đề), 04 (bulk), 03 (sinh câu) — API
from pa1_lib import *
import hashlib, urllib.request
ROOT = os.path.expanduser("~/Documents/TA_Agent_qc5/")
def upload(path, title, typ, C=C1, user="teacher", name=None):
    b = open(ROOT + path, "rb").read(); sha = hashlib.sha256(b).hexdigest()
    c, pre, _ = api(user, "POST", f"/courses/{C}/uploads/presign", {"purpose": "document", "filename": name or os.path.basename(path), "mime_type": "application/pdf", "size_bytes": len(b), "sha256": sha}); assert c == 200, (c, pre)
    r = urllib.request.Request(pre["url"], data=b, method=pre["method"], headers=pre["headers"]); urllib.request.urlopen(r, context=CTX).read()
    c, d, _ = api(user, "POST", f"/courses/{C}/uploads/complete", {"upload_id": pre["upload_id"], "title": title, "type": typ}); assert c in (200, 202), (c, d)
    return d
def wait_job(jid, user="teacher", t=300):
    t0 = time.time()
    while time.time() - t0 < t:
        c, j, _ = api(user, "GET", f"/jobs/{jid}")
        if c == 200 and j.get("status") in ("SUCCEEDED", "DONE", "FAILED", "COMPLETED", "DEAD"): return j
        time.sleep(1.5)
    return j
def wait_doc(did, C=C1, t=240):
    t0 = time.time()
    while time.time() - t0 < t:
        c, d, _ = api("teacher", "GET", f"/courses/{C}/documents/{did}")
        if c == 200 and d.get("status") in ("READY", "FAILED"): return d
        time.sleep(2)
    return d
import json as _j
if __name__ == "__main__":
    docs = {}
    for key, path, title, typ in [("de", "seed/practice/de-mau-trac-nghiem.pdf", "QC Đề mẫu", "EXAM_PAPER"), ("dapan", "seed/practice/dap-an-mau.pdf", "QC Đáp án mẫu", "ANSWER_KEY"), ("thieu", "seed/practice/dap-an-thieu.pdf", "QC Đáp án thiếu", "ANSWER_KEY"), ("loicau", "seed/practice/de-mau-loi-cau.pdf", "QC Đề lỗi câu", "EXAM_PAPER"), ("loischema", "seed/practice/de-mau-loi-schema.pdf", "QC Đề lỗi schema", "EXAM_PAPER")]:
        d = upload(path, title, typ); docs[key] = d; print("upload", key, d.get("document_id") or d)
    print(docs["de"])
    

def docid(title):
    c, j, _ = api("teacher", "GET", f"{P}/documents?limit=100"); return {d["title"]: d for d in j["items"]}[title]
