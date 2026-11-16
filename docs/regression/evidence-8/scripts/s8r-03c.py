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
logl("LOCK — US-P6-03 (03-36/37 đo lại): UTILITY → S8R-stub; hoàn lại ≤ 5 phút")
try:
    prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub"]
    MID = prov[0]["models"][0]["id"]; print("setroute", setroute("UTILITY", MID)[0])
    A2 = cid("QD8A2")
    # 36: đo số lời gọi + llm_audit
    reset_scheme(A2); open(REQ, "w").close(); setmode("invalid"); t0 = sq1("select now()")
    c, r = extract(A2, doc_of(A2)); j = jobwait(r.get("job_id"), 150); err = j.get("error"); ecode = err.get("code") if isinstance(err, dict) else err
    aud = sql(f"select attempts, status, error_kind from llm_audit where created_at >= '{t0}' and task='UTILITY'")
    n = nreq()
    T("TC-P6-03-36", "P" if (ecode == "EXTRACTION_INVALID" and n == 1) else "F", f"đầu ra không phải JSON (stub 'invalid'): job {j.get('status')} {ecode} (spec: EXTRACTION_INVALID); stub nhận {n} lời gọi (spec: đúng 1); llm_audit {aud}. Gateway coi đầu ra sai schema là BAD_RESPONSE → ErrAllProvidersFailed → extract.callLLM thử lại ×3 → LLM_UNAVAILABLE")
    # 19a đối chiếu cùng nguyên nhân (weight 'abc')
    # 37: bản CONFIRMED không bị phá
    reset_scheme(A2); setmode("lop1"); c, r = extract(A2, doc_of(A2)); jobwait(r.get("job_id")); d = draft_of(A2)
    cc, rc = call("POST", f"/courses/{A2}/grade-scheme/confirm", "teacher", {"version": d["version"]})
    st0 = sql(f"select status, scheme_no from grade_schemes where course_id='{A2}' order by scheme_no")
    setmode("fail503"); c1, r1 = extract(A2, doc_of(A2)); j1 = jobwait(r1.get("job_id"), 150) if c1 == 202 else {}
    setmode("invalid"); c2, r2 = extract(A2, doc_of(A2)); j2 = jobwait(r2.get("job_id"), 150) if c2 == 202 else {}
    st1 = sql(f"select status, scheme_no from grade_schemes where course_id='{A2}' order by scheme_no"); gb = call("GET", f"/courses/{A2}/gradebook", "teacher")
    C("TC-P6-03-37", cc == 200 and st0 == st1 and "CONFIRMED" in json.dumps(st1) and gb[0] == 200 and j1.get("status") == "FAILED" and j2.get("status") == "FAILED", f"QD8A2 CONFIRMED (confirm→{cc}): grade_schemes trước {st0} = sau {st1} sau 2 lần trích lỗi ({j1.get('status')}/{j2.get('status')}); /gradebook {gb[0]}", f"{cc} {rc} {st0} {st1} {gb[0]} {j1.get('status')} {j2.get('status')}")
finally:
    stub.terminate(); setmode("lop1")
    print("restore", setroute("UTILITY", REAL_MODEL)[0]); logl("UNLOCK — UTILITY hoàn lại QC 9router (03-36/37)")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
