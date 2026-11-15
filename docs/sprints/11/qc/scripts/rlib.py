# QC đợt R (smoke hồi quy F1–F19) — thư viện: gọi API thật ep-qc; kết quả ghi results-r.jsonl
import os, sys, json, time, uuid
sys.path.insert(0, os.path.expanduser("~/Documents/TA_Agent_qc106/docs/sprints/10.5/qc/scripts"))
import dmlib
dmlib.RES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results-r.jsonl")
from dmlib import raw, tok, api, ok, course, sql, sse, PW, B  # noqa
import urllib.request

MAILPIT = "http://localhost:8525"
def mails(to=None, subject=None, limit=50):
    j = json.load(urllib.request.urlopen(f"{MAILPIT}/api/v1/messages?limit={limit}"))
    out = []
    for m in j.get("messages", []):
        addrs = [a["Address"] for a in m.get("To", [])]
        if to and to not in addrs: continue
        if subject and subject.lower() not in m.get("Subject", "").lower(): continue
        out.append(m)
    return out
def mail_text(mid):
    return json.load(urllib.request.urlopen(f"{MAILPIT}/api/v1/message/{mid}")).get("Text", "")
def wait_job(user, jid, t=120):
    for _ in range(t):
        c, j, _ = api(user, "GET", f"/jobs/{jid}")
        if c == 200 and j.get("status") in ("SUCCEEDED", "FAILED", "DEAD"): return j
        time.sleep(1)
    return j
C = lambda user, code="761987": course(user, code)

def mkpdf(lines):
    cs = "BT /F1 14 Tf 72 740 Td 18 TL " + " ".join("(" + l.replace("(", "[").replace(")", "]") + ") Tj T*" for l in lines) + " ET"
    objs = ["<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
            "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
            f"<< /Length {len(cs)} >>\nstream\n{cs}\nendstream", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"]
    out = b"%PDF-1.4\n"; offs = []
    for i, o in enumerate(objs, 1): offs.append(len(out)); out += f"{i} 0 obj\n{o}\nendobj\n".encode("latin-1")
    x = len(out); out += f"xref\n0 {len(objs)+1}\n0000000000 65535 f \n".encode() + b"".join(f"{o:010d} 00000 n \n".encode() for o in offs)
    return out + f"trailer\n<< /Size {len(objs)+1} /Root 1 0 R >>\nstartxref\n{x}\n%%EOF\n".encode()
