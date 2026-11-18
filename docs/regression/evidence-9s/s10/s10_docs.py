# Tải đề mẫu + đáp án mẫu lên QD10A (đổi băm bằng dòng chú thích cuối PDF vì tệp gốc đã có ở lớp 761987: DOCUMENT_DUPLICATE_ELSEWHERE)
from pb import *
import hashlib, urllib.request, time, sys
def up(path, title, typ, tag=b"\n%QC-S10-QD10A\n", C=C1):
    b = open(ROOT + path, "rb").read() + tag; sha = hashlib.sha256(b).hexdigest()
    c, pre, _ = api("teacher", "POST", f"/courses/{C}/uploads/presign", {"purpose": "document", "filename": os.path.basename(path), "mime_type": "application/pdf", "size_bytes": len(b), "sha256": sha}); assert c == 200, (c, pre)
    r = urllib.request.Request(pre["url"], data=b, method=pre["method"], headers=pre["headers"]); urllib.request.urlopen(r, context=CTX).read()
    c, d, _ = api("teacher", "POST", f"/courses/{C}/uploads/complete", {"upload_id": pre["upload_id"], "title": title, "type": typ})
    if c == 409 and (d.get("details") or {}).get("document_id") and "ELSEWHERE" not in d.get("code", ""): return d["details"]["document_id"]
    assert c in (200, 202), (c, d); return d["document"]["id"]
if __name__ == "__main__":
    ids = {"de": up("seed/practice/de-mau-trac-nghiem.pdf", "QC S10 Đề mẫu trắc nghiệm", "EXAM_PAPER"), "key": up("seed/practice/dap-an-mau.pdf", "QC S10 Đáp án mẫu", "ANSWER_KEY")}
    for _ in range(40):
        st = {k: sql(f"select status from documents where id='{v}'")[0][0] for k, v in ids.items()}; print(st)
        if all(s == "READY" for s in st.values()): break
        time.sleep(5)
    print(ids)
