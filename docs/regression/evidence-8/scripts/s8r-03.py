# s8r 03: US-P6-03 trích công thức bằng stub OpenAI-compatible (cổng 9761). Chạy DƯỚI khoá global: UTILITY → S8R-stub, xong hoàn lại 9router.
import subprocess, signal, copy
from s8r_lib import *
LOCK = "/tmp/qc2-lock-global"; LOG = "/tmp/qc2-global-changes.log"; MODE = "/tmp/qc2-s8r/stub-mode.json"; REQ = "/tmp/qc2-s8r/stub-req.jsonl"
REAL_MODEL = "01a1df2d-a8d7-76a5-9cd8-5dbdb8646c78"
FX = json.load(open(REPO + "/docs/specs/FEAT-gradebook/fixtures/extraction-lop1.json")); FX2 = json.load(open(REPO + "/docs/specs/FEAT-gradebook/fixtures/extraction-lop2.json"))
def logl(m): open(LOG, "a").write(time.strftime("%H:%M:%S") + " SmkK: " + m + "\n")
def acquire():
    for _ in range(80):
        try: os.mkdir(LOCK); open(LOCK + "/owner", "w").write("SmkK"); return True
        except FileExistsError: time.sleep(15)
    return False
def setmode(m, **kw): json.dump({"mode": m, **kw}, open(MODE, "w"))
def routes(): return {i["task"]: i for i in call("GET", "/admin/llm/routes", "admin")[1]["items"]}
def setroute(task, model_id): cur = routes()[task]; return call("PUT", "/admin/llm/routes", "admin", {"task": task, "chain": [model_id], "params": {}, "version": cur["version"]}, headers={"If-Match": 'W/"v%d"' % cur["version"]})
def jobwait(jid, t=120):
    for _ in range(t // 2):
        c, j = call("GET", f"/jobs/{jid}", "teacher"); 
        if isinstance(j, dict) and j.get("status") in ("DONE", "FAILED", "SUCCEEDED", "CANCELLED", "COMPLETED"): return j
        time.sleep(2)
    return j
def sch(c0): return call("GET", f"/courses/{c0}/grade-scheme", "teacher")
def extract(c0, doc, key=None, **kw): return call("POST", f"/courses/{c0}/grade-scheme/extract", "teacher", {"document_id": doc, **kw}, key=key)
def doc_of(c0): return sq1(f"select id from documents where course_id='{c0}' and type='COURSE_POLICY' order by created_at limit 1")
def llm_rows(since): return sql(f"select task, lane, status from llm_audit where created_at >= '{since}'")
def now_db(): return sq1("select now()")
def reset_scheme(c0): sql(f"delete from grade_items where course_id='{c0}'"); sql(f"delete from grade_rules where course_id='{c0}'"); sql(f"delete from grade_components where course_id='{c0}'"); sql(f"delete from grade_schemes where course_id='{c0}'")
if not acquire(): print("không lấy được khoá"); sys.exit(2)
stub = subprocess.Popen([sys.executable, os.path.dirname(os.path.abspath(__file__)) + "/s8r-stub.py", "9761"]); time.sleep(1.5)
logl("LOCK — US-P6-03: UTILITY → S8R-stub (nhà cung cấp OpenAI-compatible giả cổng 9761) để chạy trích công thức; hoàn lại 9router ≤ 10 phút")
try:
    prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub"]
    if not prov:
        c, prov_ = call("POST", "/admin/llm/providers", "admin", {"type": "openai_compatible", "name": "S8R-stub", "base_url": "http://host.docker.internal:9761/v1", "api_key": "good-key", "models": [{"model": "m-ext", "kind": "chat", "price_in": "0", "price_out": "0"}]}); assert c == 201, (c, prov_); prov = [prov_]
    MID = prov[0]["models"][0]["id"]; print("setroute", setroute("UTILITY", MID)[0])
    A2, B1 = cid("QD8A2"), cid("QD8B1")
    for c0 in (A2, B1): reset_scheme(c0)
    # ---- 03-01/02/03/07/08/10/25
    setmode("lop1"); open(REQ, "w").close(); t0 = now_db(); lg0 = subprocess.run("date -u +%Y-%m-%dT%H:%M:%SZ", shell=True, capture_output=True, text=True).stdout.strip()
    c, r = extract(A2, doc_of(A2)); jid = r.get("job_id"); j = jobwait(jid); d_ = sch(A2)[1]
    rows = llm_rows(t0); st = j.get("status")
    C("TC-P6-03-01", c == 202 and jid and st in ("DONE", "SUCCEEDED", "COMPLETED"), f"extract → {c} job_id; job {st}", f"{c} {r} {j}")
    C("TC-P6-03-02", len(rows) == 1 and rows[0][0] == "UTILITY" and rows[0][1] == "NEAR_REALTIME", f"llm_audit của lần trích: {len(rows)} dòng, task {rows[0][0] if rows else None}, lane {rows[0][1] if rows else None}", str(rows))
    lg = subprocess.run(f"docker logs --since {lg0} ep-r-gateway-1 2>&1; docker logs --since {lg0} ep-r-worker-1 2>&1", shell=True, capture_output=True, text=True).stdout
    C("TC-P6-03-03", len(re.findall(r"điều 1|quy chế|Điểm thi cuối kỳ", lg, re.I)) == 0, f"log gateway+worker trong lúc trích: 0 lần nội dung quy chế ('Điều 1', 'Điểm thi cuối kỳ')", f"{len(lg)}")
    dr = d_.get("draft") or {}
    open("/tmp/qc2-s8r/c1.json", "w").write(json.dumps({"citations": [x for it in (dr.get("components", []) + dr.get("rules", [])) for x in it.get("citations", [])]}, ensure_ascii=False))
    print("draft keys", list(dr.keys()), "unclear", dr.get("unclear_items"))
    # 03-07/08 so với fixture
    def comp(x): return sorted([(c_["key"], str(c_.get("weight")).rstrip("0").rstrip(".") if "." in str(c_.get("weight")) else str(c_.get("weight"))) for c_ in x["components"]])
    cd = comp(dr) if dr else None; cf = [(c_["key"], c_["weight"]) for c_ in sorted(FX["components"], key=lambda z: z["key"])]
    rk = sorted((r_["kind"], json.dumps(r_["params"], sort_keys=True)) for r_ in dr.get("rules", [])) if dr else None; rf = sorted((r_["kind"], json.dumps(r_["params"], sort_keys=True)) for r_ in FX["rules"])
    C("TC-P6-03-07", bool(dr) and len(dr["components"]) == 3 and dr.get("unclear_items") == [], f"draft lớp 1: 3 thành phần {cd}, {len(dr.get('rules', []))} luật, unclear_items {dr.get('unclear_items')}", str(d_)[:300])
    C("TC-P6-03-08", cd == cf and rk == rf, f"so khớp fixture extraction-lop1.json: thành phần {cd == cf}, luật+tham số {rk == rf} (chỉ khác id/version/status)", f"{cd} vs {cf}; {rk} vs {rf}")
    rc = subprocess.run([sys.executable, REPO + "/docs/regression/evidence-8/scripts/p6-citation-check.py", REPO + "/docs/specs/FEAT-gradebook/fixtures/quy-che-lop1.md", "/tmp/qc2-s8r/c1.json"], capture_output=True, text=True)
    C("TC-P6-03-10", rc.returncode == 0 and "10/10" in rc.stdout, f"p6-citation-check trên draft.citations: {rc.stdout.strip().splitlines()[-1]} rc={rc.returncode}", rc.stdout[-300:])
    g = sql(f"select status, confirmed_by is null, confirmed_at is null from grade_schemes where course_id='{A2}'")
    C("TC-P6-03-25", g and g[0] == ["DRAFT", "t", "t"], f"sau trích: grade_schemes {g}", str(g))
    # ---- 03-13 lớp 2
    setmode("lop2"); c, r = extract(B1, doc_of(B1)); j = jobwait(r.get("job_id")); d2 = sch(B1)[1].get("draft") or {}
    w = sorted((c_["key"], str(c_["weight"])) for c_ in d2.get("components", [])); rl = {r_["kind"]: r_["params"] for r_ in d2.get("rules", [])}
    C("TC-P6-03-13", len(w) == 3 and [x[1] for x in sorted(w, key=lambda z: z[0])] and sorted(float(x[1]) for x in w) == [25.0, 25.0, 50.0] and "ROUNDING" not in rl, f"lớp 2: thành phần {w}, luật {sorted(rl)}, unclear {[u.get('code') for u in d2.get('unclear_items', [])]}; BONUS {rl.get('BONUS')}, PENALTY {rl.get('PENALTY')}", str(d2)[:400])
    # ---- 03-38 idempotent
    setmode("lop1", delay=0); reset_scheme(A2); K = str(uuid.uuid4()); n0 = int(sq1(f"select count(*) from grade_schemes where course_id='{A2}'"))
    c1, r1 = extract(A2, doc_of(A2), key=K); c2, r2 = extract(A2, doc_of(A2), key=K); jobwait(r1.get("job_id"))
    n1 = int(sq1(f"select count(*) from grade_schemes where course_id='{A2}'"))
    C("TC-P6-03-38", c1 == c2 == 202 and r1.get("job_id") == r2.get("job_id") and n1 - n0 == 1, f"extract hai lần cùng khoá → job_id {'giống' if r1.get('job_id')==r2.get('job_id') else 'KHÁC'}; grade_schemes mới {n1-n0}", f"{r1} {r2} {n0} {n1}")
    # ---- 03-32 / 03-33 (không cần LLM)
    c761 = cid("761987"); oth = sq1(f"select id from documents where course_id='{c761}' and type='OTHER' limit 1")
    ca, ra = call("POST", f"/courses/{A2}/grade-scheme/extract", "teacher", {"document_id": oth}); jb = int(sq1("select count(*) from jobs"))
    T("TC-P6-03-32", "P" if ca in (404, 422) else "F", f"extract với document khác COURSE_POLICY → {ca} {ra.get('code')}")
    # ---- 03-40/41/42 DRAFT_EXISTS
    d = sch(A2)[1]; draft = d["draft"]
    print("draft version", draft.get("version"), list(draft.keys()))
finally:
    stub.terminate()
    print("restore", setroute("UTILITY", REAL_MODEL)[0]); logl("UNLOCK — UTILITY hoàn lại QC 9router (US-P6-03)")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
