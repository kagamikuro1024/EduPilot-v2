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
        c, j = call("GET", f"/jobs/{jid}", "gv.s8f")
        if isinstance(j, dict) and j.get("status") in ("DONE", "FAILED", "SUCCEEDED", "CANCELLED", "COMPLETED"): return j
        time.sleep(2)
    return j
def sch(c0): return call("GET", f"/courses/{c0}/grade-scheme", "gv.s8f")
def extract(c0, doc, key=None, **kw): return call("POST", f"/courses/{c0}/grade-scheme/extract", "gv.s8f", {"document_id": doc, **kw}, key=key)
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



C1 = "QD8C1"
os.makedirs("/tmp/qc2-s8r", exist_ok=True)
if not acquire(): print("không lấy được khoá"); sys.exit(2)
stub = subprocess.Popen([sys.executable, os.path.dirname(os.path.abspath(__file__)) + "/s8r-stub.py", "9761"]); time.sleep(1.5)
logl("LOCK — 04-29/30/31(API)/09-33: UTILITY → S8R-stub; ≤ 6 phút")
try:
    prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub"]
    MID = prov[0]["models"][0]["id"]; print("setroute", setroute("UTILITY", MID)[0])
    c0 = sq1(f"select id from courses where class_code='{C1}'")
    def hp():
        out = {}
        cur = ""
        while True:
            c, g = call("GET", f"/courses/{c0}/gradebook?limit=100" + (f"&cursor={cur}" if cur else ""), "gv.s8f")
            for r in g["rows"]: out[r["student"]["student_code"]] = (r["result"].get("hp"), r["result"].get("letter"))
            cur = g.get("next_cursor") or ""
            if not cur: return out
    hp0 = hp(); st0 = call("GET", f"/courses/{c0}/grade-scheme/status", "gv.s8f")[1]
    # 04-29: tải quy chế khác (PDF) → ingest → trích nền → bản nháp mới cạnh CONFIRMED
    pdf = REPO + "/seed/demo/quy-che-lop1-v2.pdf"; b = open(pdf, "rb").read(); sha = hashlib.sha256(b).hexdigest()
    c, pre = call("POST", f"/courses/{c0}/uploads/presign", "gv.s8f", {"purpose": "document", "filename": "quy-che-lop1-v2.pdf", "mime_type": "application/pdf", "size_bytes": len(b), "sha256": sha})
    import urllib.request, ssl; ctx = ssl.create_default_context(); ctx.check_hostname = False; ctx.verify_mode = ssl.CERT_NONE
    urllib.request.urlopen(urllib.request.Request(pre.get("url") or pre.get("upload_url"), data=b, method=pre.get("method", "PUT"), headers=pre.get("headers") or {}), context=ctx, timeout=60).read()
    c, done = call("POST", f"/courses/{c0}/uploads/complete", "gv.s8f", {"upload_id": pre["upload_id"], "title": "Quy chế QC v2", "type": "COURSE_POLICY"})
    s = ""
    for _ in range(60):
        s = sq1(f"select status from documents where course_id='{c0}' and title='Quy chế QC v2'")
        if s in ("READY", "FAILED"): break
        time.sleep(3)
    doc = sq1(f"select id from documents where course_id='{c0}' and title='Quy chế QC v2'")
    def v2(d): comp_weight(d, "midterm", "15"); comp_weight(d, "weekly", "25"); rule(d, "BONUS")["params"]["cap"] = "1.5"
    open(REQ, "w").close(); setmode("custom", custom=mut(v2))
    c, r = extract(c0, doc); j = jobwait(r.get("job_id"), 150)
    prog = sql(f"select status, progress from jobs where id='{r.get('job_id')}'") if c == 202 else None
    cs, sc = sch(c0); drf = sc.get("draft") or {}; st1 = call("GET", f"/courses/{c0}/grade-scheme/status", "gv.s8f")[1]; hp1 = hp()
    T("TC-P6-04-29", "P" if s == "READY" and c == 202 and j.get("status") in ("DONE", "SUCCEEDED", "COMPLETED") and drf and st1.get("state") == "CONFIRMED" and st1.get("pending_draft") and hp0 == hp1 else "F",
      f"upload presign→PUT→complete→doc {s}; extract 202 job {j.get('status')} jobs.progress {prog}; nháp mới v{drf.get('version')} cạnh CONFIRMED (state {st1.get('state')}, pending_draft {st1.get('pending_draft')}); HP 30 SV không đổi {hp0 == hp1}")
    # 04-30: diff do code
    cd, df = call("GET", f"/courses/{c0}/grade-scheme?diff=confirmed", "gv.s8f")
    dj = json.dumps(df, ensure_ascii=False); print(dj[:1500])
    open("../logs/s8r-04-30-diff.json", "w").write(dj)
    # 09-33: xác nhận nháp (cap 1.5), hỏi lại
    cb, rb = call("POST", f"/courses/{c0}/grade-scheme/confirm", "gv.s8f", {"version": drf.get("version")}); print("confirm", cb, str(rb)[:200])
    json.dump({"c0": c0, "confirm": cb, "v": drf.get("version")}, open("/tmp/qc2-s8r/c1.json", "w"))
finally:
    stub.terminate(); setmode("lop1")
    print("restore", setroute("UTILITY", REAL_MODEL)[0]); logl("UNLOCK — UTILITY hoàn lại QC 9router (04-29/30)")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
