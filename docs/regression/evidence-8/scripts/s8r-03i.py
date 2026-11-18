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
logl("LOCK — US-P6-03 chạy lại sau R8-A1 (03-36, 03-19): UTILITY → S8R-stub; hoàn lại ≤ 5 phút")
try:
    prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub"]
    if not prov:
        c, pr = call("POST", "/admin/llm/providers", "admin", {"type": "openai_compatible", "name": "S8R-stub", "base_url": "http://host.docker.internal:9761/v1", "api_key": "good-key", "models": [{"model": "m-ext", "kind": "chat", "price_in": "0", "price_out": "0"}]}); print("create prov", c)
        prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub"]
    MID = prov[0]["models"][0]["id"]; print("setroute", setroute("UTILITY", MID)[0])
    A2 = cid("QD8A2")
    def one(mode, **kw):
        reset_scheme(A2); open(REQ, "w").close(); setmode(mode, **kw); t0 = sq1("select now()")
        c, r = extract(A2, doc_of(A2)); j = jobwait(r.get("job_id"), 150); err = j.get("error"); ec = err.get("code") if isinstance(err, dict) else err
        return c, j, ec, nreq(), sql(f"select attempts, status, coalesce(error_kind,'') from llm_audit where created_at >= '{t0}' and task='UTILITY'"), draft_of(A2)
    # 03-19: weight 'abc' bị schema strict (pattern số) chặn → EXTRACTION_INVALID; PARAM_INVALID từng mục cho các trường hợp qua được schema
    def sneaky(d): comp_weight(d, "midterm", "40"); comp_weight(d, "weekly", "150")
    c, j, ec, n, aud, dr = one("custom", custom=mut(sneaky)); u = dr.get("unclear_items", [])
    pi = [x for x in u if x.get("code") == "PARAM_INVALID"]
    cc, rc = call("POST", f"/courses/{A2}/grade-scheme/confirm", "teacher", {"version": dr.get("version")})
    c2, j2, ec2, n2, aud2, dr2 = one("custom", custom=mut(lambda d: comp_weight(d, "weekly", "20.123"))); pi2 = [x for x in dr2.get("unclear_items", []) if x.get("code") == "PARAM_INVALID"]
    c3, j3, ec3, n3, aud3, dr3 = one("custom", custom=mut(lambda d: comp_weight(d, "final", "0"))); b3 = ucodes(dr3)
    c4, j4, ec4, n4, aud4, dr4 = one("custom", custom=mut(lambda d: comp_weight(d, "weekly", "abc")))
    ok = bool(pi) and "SCHEME_HAS_UNCLEAR" in json.dumps(rc) and cc == 422 and bool(pi2) and "PARAM_INVALID" in b3 and ec4 == "EXTRACTION_INVALID"
    T("TC-P6-03-19", "P" if ok else "F", f"RERUN sau R8-A3 (7d2de170), bản nháp trích MỚI: weight 150 → PARAM_INVALID {[x.get('target') for x in pi]} «{(pi[0].get('message') if pi else '')}», confirm → {cc} {rc.get('code') if isinstance(rc, dict) else ''}; 20.123 → {len(pi2)} mục PARAM_INVALID; CK=0 → {b3}; weight abc → job FAILED {ec4} (schema strict chặn, 1 lời gọi)")
finally:
    stub.terminate(); setmode("lop1")
    print("restore", setroute("UTILITY", REAL_MODEL)[0]); logl("UNLOCK — UTILITY hoàn lại QC 9router (03-36/03-19)")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
