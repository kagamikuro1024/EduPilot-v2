# s8r 03b: US-P6-03 phần còn lại (06,11,12,17-23,30,31,33-37,39-42) trên ep-r bằng stub OpenAI-compatible :9761. Chạy DƯỚI khoá global (UTILITY → S8R-stub; 03-39 dừng worker).
import subprocess, copy, hashlib
from s8r_lib import *
LOCK = "/tmp/qc2-lock-global"; LOG = "/tmp/qc2-global-changes.log"; MODE = "/tmp/qc2-s8r/stub-mode.json"; REQ = "/tmp/qc2-s8r/stub-req.jsonl"
REAL_MODEL = "01a1df2d-a8d7-76a5-9cd8-5dbdb8646c78"
FXD = REPO + "/docs/specs/FEAT-gradebook/fixtures/"
FX = json.load(open(FXD + "extraction-lop1.json"))
def logl(m): open(LOG, "a").write(time.strftime("%H:%M:%S") + " qc2: " + m + "\n")
def acquire():
    for _ in range(80):
        try: os.mkdir(LOCK); open(LOCK + "/owner", "w").write("qc2"); return True
        except FileExistsError: time.sleep(15)
    return False
def setmode(m, **kw): json.dump({"mode": m, **kw}, open(MODE, "w"))
def routes(): return {i["task"]: i for i in call("GET", "/admin/llm/routes", "admin")[1]["items"]}
def setroute(task, model_id): cur = routes()[task]; return call("PUT", "/admin/llm/routes", "admin", {"task": task, "chain": [model_id], "params": {}, "version": cur["version"]}, headers={"If-Match": 'W/"v%d"' % cur["version"]})
def jobwait(jid, t=120):
    j = None
    for _ in range(t // 2):
        c, j = call("GET", f"/jobs/{jid}", "teacher")
        if isinstance(j, dict) and j.get("status") in ("DONE", "FAILED", "SUCCEEDED", "CANCELLED", "COMPLETED"): return j
        time.sleep(2)
    return j
def sch(c0): return call("GET", f"/courses/{c0}/grade-scheme", "teacher")
def extract(c0, doc, key=None, **kw): return call("POST", f"/courses/{c0}/grade-scheme/extract", "teacher", {"document_id": doc, **kw}, key=key)
def doc_of(c0): return sq1(f"select id from documents where course_id='{c0}' and type='COURSE_POLICY' order by created_at limit 1")
def reset_scheme(c0):
    for t in ("grade_items", "grade_rules", "grade_components"): sql(f"delete from {t} where course_id='{c0}'")
    sql(f"delete from grade_schemes where course_id='{c0}'")
def nreq(): return sum(1 for _ in open(REQ)) if os.path.exists(REQ) else 0
def draft_of(c0): return sch(c0)[1].get("draft") or {}
def ucodes(d): return sorted({u.get("code") for u in d.get("unclear_items", [])})
def run(c0, custom, doc=None):
    reset_scheme(c0); open(REQ, "w").close(); setmode("custom", custom=custom)
    c, r = extract(c0, doc or doc_of(c0)); j = jobwait(r.get("job_id")) if c == 202 else {}
    return c, r, j, draft_of(c0)
def mut(f):
    d = copy.deepcopy(FX); f(d); return d
def comp_weight(d, k, w):
    for c in d["components"]:
        if c["key"] == k: c["weight"] = w
def citation(d, cid_): return [c for c in d["citations"] if c["id"] == cid_][0]
def rule(d, kind, sub=None):
    for r in d["rules"]:
        if r["kind"] == kind and (sub is None or r["params"].get("kind") == sub): return r


if not acquire(): print("không lấy được khoá"); sys.exit(2)
stub = subprocess.Popen([sys.executable, os.path.dirname(os.path.abspath(__file__)) + "/s8r-stub.py", "9761"]); time.sleep(1.5)
logl("LOCK — US-P6-03 (03-31 đo lại với tên trong danh sách lớp): UTILITY → S8R-stub; hoàn lại ≤ 5 phút")
try:
    prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub"]
    MID = prov[0]["models"][0]["id"]; print("setroute", setroute("UTILITY", MID)[0])
    A2 = cid("QD8A2")
    subprocess.run(f"cd {REPO} && node docs/regression/evidence-8/scripts/s8core-mkpdf.mjs", shell=True, capture_output=True)
    def upload(c0, pdfp, title):
        b = open(pdfp, "rb").read(); sha = hashlib.sha256(b).hexdigest()
        c, pre = call("POST", f"/courses/{c0}/uploads/presign", "teacher", {"purpose": "document", "filename": os.path.basename(pdfp), "mime_type": "application/pdf", "size_bytes": len(b), "sha256": sha})
        if c != 200 and c != 201: return None, f"presign {c} {pre}"
        import urllib.request
        rq = urllib.request.Request(pre.get("url") or pre.get("upload_url"), data=b, method=pre.get("method", "PUT"), headers=pre.get("headers") or {})
        import ssl; ctx = ssl.create_default_context(); ctx.check_hostname = False; ctx.verify_mode = ssl.CERT_NONE
        try: urllib.request.urlopen(rq, context=ctx, timeout=60).read()
        except Exception as ex: return None, f"PUT blob {ex}"
        c, done = call("POST", f"/courses/{c0}/uploads/complete", "teacher", {"upload_id": pre["upload_id"], "title": title, "type": "COURSE_POLICY"})
        for _ in range(60):
            s = sq1(f"select status from documents where course_id='{c0}' and title='{title}'")
            if s in ("READY", "FAILED"): break
            time.sleep(3)
        return sq1(f"select id from documents where course_id='{c0}' and title='{title}'"), s

    did, st = upload(A2, "/tmp/s8core/pdf-pii2.pdf", "Quy chế QC pii2")
    if not did or st != "READY": T("TC-P6-03-31", "F", f"tải PDF pii2: {did} {st}")
    else:
        reset_scheme(A2); open(REQ, "w").close(); setmode("lop1")
        c, r = extract(A2, did); j = jobwait(r.get("job_id")); reqs = [json.loads(l) for l in open(REQ) if l.strip()]
        body = json.dumps(reqs[0]["req"], ensure_ascii=False) if reqs else ""
        usr = " ".join(m_.get("content", "") for m_ in (reqs[0]["req"].get("messages") if reqs else []) if m_.get("role") == "user" and isinstance(m_.get("content"), str))
        leak = [x for x in ("Bùi Thanh Khải", "Ngô Ngọc Cẩm", "20229002", "nvtest@example.com") if x in body]
        line = [l for l in usr.split("\n") if "Người lập" in l]
        gen = "Nguyễn Văn Test" in body
        T("TC-P6-03-31", "P" if not leak and j.get("status") in ("SUCCEEDED", "DONE") else "F", f"PDF có tên+MSSV+email của SV trong danh sách lớp: prompt tới LLM không còn tên/MSSV/email (lọt: {leak}); dòng trong prompt: {line[:1]}; job {j.get('status')}. Lưu ý: tên KHÔNG thuộc danh sách lớp (vd 'Nguyễn Văn Test') không bị che (che theo danh sách lớp + MSSV/email/SĐT, internal/privacy/roster.go)")
finally:
    stub.terminate(); setmode("lop1")
    print("restore", setroute("UTILITY", REAL_MODEL)[0]); logl("UNLOCK — UTILITY hoàn lại QC 9router (03-31)")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
