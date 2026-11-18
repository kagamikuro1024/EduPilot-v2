"""Thư viện QC smoke-9 trên ep-qc: gọi API https://localhost:943, SQL qua docker exec ep-qc-postgres-1, khoá global /tmp/qc-global.lock, nhà cung cấp stub (host :9751).
Mật khẩu seed đọc từ env SEED_DEFAULT_PASSWORD (. ~/Documents/ep-qc-env/env.sh) — KHÔNG in / ghi."""
import copy, hashlib, json, os, re, ssl, subprocess, sys, time, urllib.request, uuid
GW = os.environ.get("GW", "https://localhost:943"); PW = os.environ["SEED_DEFAULT_PASSWORD"]; ctx = ssl._create_unverified_context()
FX = os.path.expanduser("~/Documents/TA_Agent_qc9sub/docs/specs/FEAT-assignments/fixtures"); TMP = "/tmp/q9s"; os.makedirs(TMP, exist_ok=True)
MP = "http://localhost:8525"; PG = "ep-qc-postgres-1"; RD = "ep-qc-redis-1"
def raw(m, url, h=None, data=None):
    try:
        r = urllib.request.urlopen(urllib.request.Request(url, data=data, method=m, headers=h or {}), context=ctx); return r.status, r.read(), r.headers
    except urllib.error.HTTPError as e: return e.code, e.read(), e.headers
def call(m, p, tok=None, body=None, key=None, hdr=None):
    h = {"Content-Type": "application/json", **(hdr or {})}
    if tok: h["Authorization"] = "Bearer " + tok
    if key or m in ("POST", "PUT", "PATCH", "DELETE"): h["Idempotency-Key"] = key or str(uuid.uuid4())
    s, t, hd = raw(m, GW + p, h, None if body is None else json.dumps(body).encode())
    return s, (json.loads(t) if t[:1] in (b"{", b"[") else t.decode()), hd
def _tc():
    try: return json.load(open(f"{TMP}/tok.json"))
    except Exception: return {}
def tok(user, fresh=False):
    e = user if "@" in user else user + "@edupilot.local"; c = _tc(); n = time.time()
    if not fresh and e in c and n - c[e][1] < 200: return c[e][0]
    for i in range(5):
        s, j, _ = call("POST", "/api/v1/auth/login", body={"email": e, "password": PW})
        if s == 200: break
        time.sleep(3 + 3 * i)
    c = _tc(); c[e] = [j["access_token"], n]; json.dump(c, open(f"{TMP}/tok.json", "w")); return j["access_token"]
def psql(q):
    r = subprocess.run(["docker", "exec", PG, "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True)
    if r.returncode: raise RuntimeError(r.stderr)
    return r.stdout.strip()
def redis(*a): return subprocess.run(["docker", "exec", RD, "redis-cli", *a], capture_output=True, text=True).stdout.strip()
idem = lambda: str(uuid.uuid4())
class Rep:
    def __init__(s, name): s.name = name; s.fails = 0; s.res = {}
    def chk(s, tc, n, ok, x=""):
        s.fails += (not ok); print("PASS" if ok else "FAIL", tc, n, "" if ok else str(x)[:300], flush=True); s.res.setdefault(tc, []).append(bool(ok))
    def note(s, tc, t): print("NOTE", tc, t, flush=True)
    def end(s):
        print("SUMMARY", json.dumps({k: all(v) for k, v in s.res.items()}), flush=True); print("RESULT:", "FAIL" if s.fails else "PASS", f"({s.fails} lỗi)"); return 1 if s.fails else 0
# ---- khoá global (mkdir /tmp/qc-global.lock) + tuyến LLM
LOCKD = "/tmp/qc-global.lock"; WHO = "Smoke9Runner"
def acquire(wait=1200):
    t0 = time.time()
    while time.time() - t0 < wait:
        try: os.mkdir(LOCKD); open(LOCKD + "/owner", "w").write(WHO); open("/tmp/qc-global-changes.log", "a").write(f"{time.strftime('%H:%M:%S')} {WHO}: LOCK\n"); return True
        except FileExistsError: time.sleep(10)
    return False
def mine(): return os.path.exists(LOCKD + "/owner") and open(LOCKD + "/owner").read().strip() == WHO
def release():
    if mine(): open("/tmp/qc-global-changes.log", "a").write(f"{time.strftime('%H:%M:%S')} {WHO}: UNLOCK\n"); subprocess.run(["rm", "-rf", LOCKD])
def routes_(): return call("GET", "/api/v1/admin/llm/routes", tok("admin"))[1]["items"]
def route_names(task="GRADING"): return [e["provider_name"] + "/" + e["model"] for e in [r for r in routes_() if r["task"] == task][0]["chain"]]
def setroute(model_id, task="GRADING"):
    cur = [i for i in routes_() if i["task"] == task][0]
    return call("PUT", "/api/v1/admin/llm/routes", tok("admin"), {"task": task, "chain": [model_id], "params": {}, "version": cur["version"]}, hdr={"If-Match": 'W/"v%d"' % cur["version"]})[0]
def providers(): return {p["name"]: p for p in call("GET", "/api/v1/admin/llm/providers", tok("admin"))[1]["items"]}
def model_of(provname, model): return [m for m in providers()[provname]["models"] if m["model"] == model][0]["id"]
ORIG_ROUTE = None
def route_to(provname, model):
    """Đổi tuyến GRADING (cần đang giữ khoá). Lưu tuyến cũ để route_back."""
    global ORIG_ROUTE
    cur = [i for i in routes_() if i["task"] == "GRADING"][0]
    if ORIG_ROUTE is None: ORIG_ROUTE = [e["model_id"] if "model_id" in e else None for e in cur["chain"]]; json.dump(cur["chain"], open(f"{TMP}/orig_route.json", "w"))
    return setroute(model_of(provname, model))
def route_back():
    try: ch = json.load(open(f"{TMP}/orig_route.json"))
    except Exception: return None
    cur = [i for i in routes_() if i["task"] == "GRADING"][0]
    ids = [e.get("model_id") or e.get("id") for e in ch]
    return call("PUT", "/api/v1/admin/llm/routes", tok("admin"), {"task": "GRADING", "chain": ids, "params": cur["params"], "version": cur["version"]}, hdr={"If-Match": 'W/"v%d"' % cur["version"]})[0]
# ---- stub
def stub_up():
    try: urllib.request.urlopen("http://127.0.0.1:9751/v1/models", timeout=2); return True
    except Exception: return False
def stub_start():
    if not stub_up(): subprocess.Popen(f"nohup python3 {os.path.dirname(__file__)}/q9stub.py 9751 > /tmp/q9s/stub.log 2>&1 &", shell=True); time.sleep(2)
def stub_provider():
    if "Q9-stub" not in providers():
        s, j, _ = call("POST", "/api/v1/admin/llm/providers", tok("admin"), {"type": "openai_compatible", "name": "Q9-stub", "base_url": "http://host.docker.internal:9751/v1", "api_key": "good-key", "models": [{"model": m, "kind": "chat", "price_in": "0", "price_out": "0"} for m in ("g-main", "g-timeout")]}); assert s == 201, (s, j)
    return {m["model"]: m["id"] for m in providers()["Q9-stub"]["models"]}
def mode(m): open(f"{TMP}/mode", "w").write(m)
def reqs(since=0.0, model=None):
    out = []
    try:
        for l in open(f"{TMP}/req.jsonl"):
            r = json.loads(l)
            if r["t"] >= since and (model is None or r["model"] == model): out.append(r)
    except FileNotFoundError: pass
    return out
def sysuser(r):
    m = r["body"]["messages"]; return " ".join(x["content"] for x in m if x["role"] == "system"), " ".join(x["content"] for x in m if x["role"] == "user")
# ---- dữ liệu: lớp riêng QD9*, bài tập, bài nộp
def course(code): return psql(f"select id from courses where class_code='{code}'")
def mkcourse(code, students=("sv.gioi", "sv.kha", "sv.nguyco")):
    cid = course(code)
    A = tok("admin")
    if not cid:
        tid = psql("select id from users where email='teacher@edupilot.local'"); taid = psql("select id from users where email='ta@edupilot.local'")
        s, j, _ = call("POST", "/api/v1/admin/courses", A, {"subject_code": "QD9", "class_code": code, "name": f"QC smoke9 {code}", "semester": "2026-2027-HK1", "capacity": 60, "teacher_id": tid}); assert s in (200, 201), (s, j)
        cid = j["id"]; call("POST", f"/api/v1/admin/courses/{cid}/assign", A, {"ta_ids": [taid]})
    jc = psql(f"select join_code from courses where id='{cid}'")
    for u in students:
        e = u if "@" in u else u + "@edupilot.local"
        if psql(f"select count(*) from enrollments en join users x on x.id=en.user_id where en.course_id='{cid}' and x.email='{e}'") == "0":
            s, j, _ = call("POST", "/api/v1/courses/join", tok(u), {"code": jc}); assert s in (200, 201), (u, s, j)
        psql(f"update enrollments set joined_via='ROSTER', student_code_snapshot=(select student_code from users where id=enrollments.user_id) where course_id='{cid}' and user_id=(select id from users where email='{e}')")
    return cid
def mkassign(Q, T, code, due_days=3, publish=True, **ov):
    b = json.load(open(f"{FX}/create-bt03.json")); b["due_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + due_days * 86400)); b["submission_code"] = code; b.update(ov)
    s, a, _ = call("POST", f"{Q}/assignments", T, b, idem()); assert s in (200, 201), (s, a); aid = a.get("assignment", a)["id"]
    if publish: call("POST", f"{Q}/assignments/{aid}/publish", T, {}, idem())
    return aid
def submit(Q, aid, tk, name, data):
    sha = hashlib.sha256(data).hexdigest()
    s, pre, _ = call("POST", f"{Q}/assignments/{aid}/submit/presign", tk, {"filename": name, "size_bytes": len(data), "sha256": sha}); assert s == 200, (s, pre)
    raw(pre["method"], pre["url"], pre["headers"], data)
    return call("POST", f"{Q}/assignments/{aid}/submit", tk, {"upload_id": pre["upload_id"]}, idem())
def sub_id(aid, email, ver=None):
    q = f"select s.id from submissions s join users u on u.id=s.student_id where s.assignment_id='{aid}' and u.email='{email}@edupilot.local'" + (f" and s.version={ver}" if ver else " order by s.version desc") + " limit 1"; return psql(q)
def pending(aid): return int(psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{aid}' and gr.run_state in ('QUEUED','RUNNING')"))
def wait_done(aid, t=180):
    for _ in range(t // 2):
        if pending(aid) == 0: return True
        time.sleep(2)
    return False
def result(aid, email):
    row = psql(f"select row_to_json(g) from (select gr.* from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{aid}' and u.email='{email}@edupilot.local' order by gr.round desc, s.version desc limit 1) g"); return json.loads(row) if row else None
def mailpit(): return json.loads(urllib.request.urlopen(MP + "/api/v1/messages?limit=200").read())
def stub_provider_paid():
    if "Q9-stub-paid" not in providers():
        s, j, _ = call("POST", "/api/v1/admin/llm/providers", tok("admin"), {"type": "openai_compatible", "name": "Q9-stub-paid", "base_url": "http://host.docker.internal:9751/v1", "api_key": "good-key", "models": [{"model": "g-main", "kind": "chat", "price_in": "1000000", "price_out": "1000000"}]}); assert s == 201, (s, j)
    return {m["model"]: m["id"] for m in providers()["Q9-stub-paid"]["models"]}
def stub_on(note="stub", paid=False):
    assert acquire(), "không lấy được khoá global"; stub_start(); ids = stub_provider_paid() if paid else stub_provider()
    cur = routes_(); json.dump([i for i in cur if i["task"] == "GRADING"][0]["chain"], open(f"{TMP}/orig_route.json", "w"))
    open("/tmp/qc-global-changes.log", "a").write(f"{time.strftime('%H:%M:%S')} {WHO}: GRADING -> Q9-stub (:9751) [{note}]; hoàn lại ≤15 phút\n")
    assert setroute(ids["g-main"]) == 200; return ids
def stub_off():
    if mine():
        route_back(); open("/tmp/qc-global-changes.log", "a").write(f"{time.strftime('%H:%M:%S')} {WHO}: GRADING -> hoàn lại QC 9router\n"); release()
def stub_ctx(paid=False):
    class C:
        def __enter__(s): return stub_on(paid=paid)
        def __exit__(s, *a): stub_off()
    return C()
def today(user, course=None):
    s, j, _ = call("GET", f"/api/v1/courses/{course}/today" if course else "/api/v1/me/today", tok(user))
    items = ((j.get("actions") or []) + (j.get("attention") or []) + ([j["recommended"]] if j.get("recommended") else [])) if isinstance(j, dict) else []
    return s, items, j
import csv
def seed_bt03(Q, T, code, accounts=None, grade=True, due_days=2, wait=180):
    """Dựng Bài tập 03 trong lớp QD9* như seed-plan.csv: mỗi sinh viên nộp mẫu S1–S7 (IN_APP), hạn đã qua, submitted_at = hạn + giờ; chấm bằng stub (replay mẫu). Trả {aid, sub:{acc:id}, res:{acc:id}}."""
    plan = [r for r in csv.DictReader(open(f"{FX}/seed-plan.csv")) if r["sample_v1"] and r["initial_match"] == "MATCHED" and r["channel"] in ("IN_APP", "UPLOAD", "IMAP")]
    if accounts: plan = [r for r in plan if r["account"] in accounts]
    smp = {"S1": "s1-tot", "S2": "s2-kha", "S3": "s3-kem", "S4": "s4-trong", "S5": "s5-lac-de", "S6": "s6-injection", "S7": "s7-lech-hai-luot"}
    aid = mkassign(Q, T, code, due_days=3)
    out = {"aid": aid, "sub": {}, "res": {}}
    for r in plan:
        sv = r["sample_v2"] or r["sample_v1"]
        if r["sample_v2"]: submit(Q, aid, tok(r["account"]), "v1.md", open(f"{FXS}/{smp[r['sample_v1']]}.md", "rb").read())
        s, x, _ = submit(Q, aid, tok(r["account"]), smp[sv] + ".md", open(f"{FXS}/{smp[sv]}.md", "rb").read()); assert s in (200, 201), (r["account"], s, x)
    psql(f"update assignments set due_at = now() - interval '{due_days} days' where id='{aid}'")
    for r in plan:
        h = float(r["hours_vs_due"] or 0)
        psql(f"update submissions s set submitted_at = (select due_at from assignments where id='{aid}') + interval '{h} hours' where s.assignment_id='{aid}' and s.student_id=(select id from users where email='{r['account']}@edupilot.local')")
    if grade:
        s, g, _ = call("POST", f"{Q}/assignments/{aid}/grade-all", T, {}); assert s == 202, (s, g); wait_done(aid, wait); time.sleep(2)
    for r in plan:
        e = f"{r['account']}@edupilot.local"
        out["sub"][r["account"]] = sub_id(aid, r["account"]); res = result(aid, r["account"]); out["res"][r["account"]] = res["id"] if res else None
    return out
FXS = FX + "/submissions"
