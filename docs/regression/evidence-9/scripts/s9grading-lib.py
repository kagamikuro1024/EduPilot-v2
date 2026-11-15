"""Thư viện dùng chung của QC s9grading (US-P7-06/07/08) trên ep-r: gọi API, token cache, SQL, khoá global + route GRADING→stub, log yêu cầu stub, tạo bài tập / bài nộp."""
import copy, hashlib, json, os, ssl, subprocess, sys, time, urllib.request, uuid
GW = os.environ.get("GW", "https://localhost:1143"); PW = "Edupilot#Seed-2026"; ctx = ssl._create_unverified_context()
FX = "/Users/kuro/Documents/TA_Agent_qc2/docs/specs/FEAT-assignments/fixtures"; TMP = "/tmp/s9g"; os.makedirs(TMP, exist_ok=True)
STUB_MODEL_ID = None
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
def tok(user):
    e = user if "@" in user else user + "@edupilot.local"; c = _tc(); n = time.time()
    if e in c and n - c[e][1] < 600: return c[e][0]
    for i in range(4):
        s, j, _ = call("POST", "/api/v1/auth/login", body={"email": e, "password": PW})
        if s == 200: break
        if s == 429 or (isinstance(j, dict) and j.get("code") == "LOGIN_THROTTLED"):
            subprocess.run(["docker", "exec", "ep-r-redis-1", "sh", "-c", "for p in 'ep:auth:blockip:*' 'ep:auth:ipfail:*' 'ep:auth:fail:*' 'ep:rl:auth*'; do redis-cli --scan --pattern \"$p\" | xargs -r redis-cli del; done"], capture_output=True); time.sleep(2)
    c = _tc(); c[e] = [j["access_token"], n]; json.dump(c, open(f"{TMP}/tok.json", "w")); return j["access_token"]
def psql(q):
    r = subprocess.run(["docker", "exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True)
    if r.returncode: raise RuntimeError(r.stderr)
    return r.stdout.strip()
idem = lambda: str(uuid.uuid4())
class Rep:
    def __init__(s, name): s.name = name; s.fails = 0; s.res = {}
    def chk(s, tc, n, ok, x=""):
        s.fails += (not ok); print("PASS" if ok else "FAIL", tc, n, "" if ok else str(x)[:300], flush=True); s.res.setdefault(tc, []).append(bool(ok))
    def end(s):
        print("SUMMARY", json.dumps({k: all(v) for k, v in s.res.items()}), flush=True); print("RESULT:", "FAIL" if s.fails else "PASS", f"({s.fails} lỗi)"); return 1 if s.fails else 0
# ---- khoá global + route
LOCKD = "/tmp/qc2-lock-global"; WHO = "s9grading"
def log_global(msg):
    open("/tmp/qc2-global-changes.log", "a").write(f"{time.strftime('%H:%M:%S')} {WHO}: {msg}\n")
def acquire(wait=1200):
    t0 = time.time()
    while time.time() - t0 < wait:
        try: os.mkdir(LOCKD); open(LOCKD + "/owner", "w").write(WHO); return True
        except FileExistsError: time.sleep(10)
    return False
def mine(): return os.path.exists(LOCKD + "/owner") and open(LOCKD + "/owner").read().strip() == WHO
def release():
    if mine(): subprocess.run(["rm", "-rf", LOCKD])
REAL = "01a1df2d-a8d7-76a5-9cd8-5dbdb8646c78"
def stub_model():
    A = tok("admin"); ex = [p for p in call("GET", "/api/v1/admin/llm/providers", A)[1]["items"] if p["name"] == "S9G-stub"]
    if not ex:
        s, j, _ = call("POST", "/api/v1/admin/llm/providers", A, {"type": "openai_compatible", "name": "S9G-stub", "base_url": "http://host.docker.internal:9751/v1", "api_key": "good-key", "models": [{"model": m, "kind": "chat", "price_in": "1000000", "price_out": "1000000"} for m in ("g-main", "g-timeout")]}); assert s == 201, (s, j); ex = [j]
    return {m["model"]: m["id"] for m in ex[0]["models"]}
def setroute(model_id, task="GRADING"):
    A = tok("admin"); cur = [i for i in call("GET", "/api/v1/admin/llm/routes", A)[1]["items"] if i["task"] == task][0]
    return call("PUT", "/api/v1/admin/llm/routes", A, {"task": task, "chain": [model_id], "params": {}, "version": cur["version"]}, hdr={"If-Match": 'W/"v%d"' % cur["version"]})[0]
def route_on(note):
    assert acquire(), "không lấy được khoá global"; log_global(f"LOCK {note}: GRADING→S9G-stub (:9751); hoàn lại ≤10 phút"); assert setroute(stub_model()["g-main"]) == 200
def route_off(note="xong"):
    if mine():
        setroute(REAL); log_global(f"UNLOCK {note}: GRADING→QC 9router"); release()
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
# ---- dữ liệu
def course(code): return psql(f"select id from courses where class_code='{code}'")
def mkassign(Q, T, code, due_days=3, **ov):
    b = json.load(open(f"{FX}/create-bt03.json")); b["due_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + due_days * 86400)); b["submission_code"] = code; b.update(ov)
    s, a, _ = call("POST", f"{Q}/assignments", T, b, idem()); assert s in (200, 201), (s, a); aid = a.get("assignment", a)["id"]; call("POST", f"{Q}/assignments/{aid}/publish", T, {}, idem()); return aid
def submit(Q, aid, tk, name, data):
    sha = hashlib.sha256(data).hexdigest()
    s, pre, _ = call("POST", f"{Q}/assignments/{aid}/submit/presign", tk, {"filename": name, "size_bytes": len(data), "sha256": sha}); assert s == 200, (s, pre)
    raw(pre["method"], pre["url"], pre["headers"], data)
    return call("POST", f"{Q}/assignments/{aid}/submit", tk, {"upload_id": pre["upload_id"]}, idem())
def sub_id(aid, email, ver=None):
    q = f"select s.id from submissions s join users u on u.id=s.student_id where s.assignment_id='{aid}' and u.email='{email}@edupilot.local'" + (f" and s.version={ver}" if ver else " order by s.version desc") + " limit 1"; return psql(q)
def states(aid, rnd=1):
    return psql(f"select string_agg(x,',' order by x) from (select gr.run_state||':'||coalesce(gr.error_code,'-') x from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{aid}' and gr.round={rnd}) q")
def pending(aid): return int(psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{aid}' and gr.run_state in ('QUEUED','RUNNING')"))
def wait_done(aid, t=180):
    for _ in range(t // 2):
        if pending(aid) == 0: return True
        time.sleep(2)
    return False
def result(aid, email):
    row = psql(f"select row_to_json(g) from (select gr.* from grading_results gr join submissions s on s.id=gr.submission_id join users u on u.id=s.student_id where s.assignment_id='{aid}' and u.email='{email}@edupilot.local' order by gr.round desc, s.version desc limit 1) g"); return json.loads(row) if row else None
