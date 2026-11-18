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




os.makedirs("/tmp/qc2-s8r", exist_ok=True)
if not acquire(): print("không lấy được khoá"); sys.exit(2)
stub = subprocess.Popen([sys.executable, os.path.dirname(os.path.abspath(__file__)) + "/s8r-stub.py", "9761"]); time.sleep(1.5)
logl("LOCK — US-P6-05 flow lớp QD8H2: UTILITY → S8R-stub (lop2); hoàn lại ≤ 6 phút")
try:
    prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub"]
    MID = prov[0]["models"][0]["id"]; print("setroute", setroute("UTILITY", MID)[0]); setmode("lop2")
    r = subprocess.run([sys.executable, os.path.dirname(os.path.abspath(__file__)) + "/s8flow-us05h.py", "flow", "QD8H2"], capture_output=True, text=True); print(r.stdout[-3000:], r.stderr[-1500:])
finally:
    stub.terminate(); setmode("lop1")
    print("restore", setroute("UTILITY", REAL_MODEL)[0]); logl("UNLOCK — UTILITY hoàn lại QC 9router (US-P6-05 flow)")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
