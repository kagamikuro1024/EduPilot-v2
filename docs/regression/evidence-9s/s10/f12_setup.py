import sys, os, hashlib, urllib.request; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pb import *
src = open("mkclass10.py", encoding="utf8").read().split('mk("QD10A"')[0]; exec(src)
CODE = os.environ.get("QC_F12_CLASS", "QD10G"); mk(CODE, ["sv36", "sv37"], 0)
CG = sql(f"select id from courses where class_code='{CODE}'")[0][0]
def up2(path, title, typ):
    b = open(ROOT + path, "rb").read() + f"\n%QC-F12-{CODE}\n".encode(); sha = hashlib.sha256(b).hexdigest()
    c, pre, _ = api("teacher", "POST", f"/courses/{CG}/uploads/presign", {"purpose": "document", "filename": os.path.basename(path), "mime_type": "application/pdf", "size_bytes": len(b), "sha256": sha}); assert c == 200, (c, pre)
    urllib.request.urlopen(urllib.request.Request(pre["url"], data=b, method=pre["method"], headers=pre["headers"]), context=CTX).read()
    c, d, _ = api("teacher", "POST", f"/courses/{CG}/uploads/complete", {"upload_id": pre["upload_id"], "title": title, "type": typ}); assert c in (200, 202), (c, d); return d["document"]["id"]
ids = [up2("seed/practice/de-mau-trac-nghiem.pdf", "Đề mẫu F12", "EXAM_PAPER"), up2("seed/practice/dap-an-mau.pdf", "Đáp án F12", "ANSWER_KEY")]
for i in ids: print("doc", wait_doc(i, C=CG)["status"])
print("CG", CG, "câu:", sql(f"select count(*) from question_bank where course_id='{CG}'")[0][0])
