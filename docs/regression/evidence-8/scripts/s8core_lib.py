# s8core: thư viện dùng chung cho các script QC sprint 8 (P6-01..04) trên stack ep-r.
import json, os, ssl, subprocess, time, uuid, urllib.request, urllib.error
B = "https://localhost:1143/api/v1"
PW = "Edupilot#Seed-2026"
REPO = os.path.expanduser("~/Documents/TA_Agent_qc2")
RES = f"{REPO}/docs/regression/evidence-8/logs/s8core-results.jsonl"
CTX = ssl.create_default_context(); CTX.check_hostname = False; CTX.verify_mode = ssl.CERT_NONE
_tok = {}
def sh(cmd, timeout=120):
    return subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=timeout).stdout
def sql(q, db="edupilot"):
    r = subprocess.run(["docker", "exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", db, "-At", "-F", "|", "-c", q], capture_output=True, text=True, timeout=60)
    if r.returncode: raise RuntimeError(r.stderr)
    return [l.split("|") for l in r.stdout.strip().split("\n") if l]
def sq1(q, db="edupilot"):
    r = sql(q, db); return r[0][0] if r else None
def raw(method, path, headers=None, body=None):
    req = urllib.request.Request(B + path if path.startswith("/") else path, data=(json.dumps(body).encode() if body is not None and not isinstance(body, bytes) else body), method=method)
    for k, v in (headers or {}).items(): req.add_header(k, v)
    try:
        with urllib.request.urlopen(req, context=CTX, timeout=60) as r: return r.status, r.read(), dict(r.headers)
    except urllib.error.HTTPError as e: return e.code, e.read(), dict(e.headers)
def tok(user):
    if user is None: return None
    f = f"/tmp/s8core/tok-{user}.json"
    if os.path.exists(f):
        d = json.load(open(f))
        if time.time() - d["ts"] < 480: return d["t"]
    for i in range(4):
        c, b, _ = raw("POST", "/auth/login", {"content-type": "application/json", "Idempotency-Key": str(uuid.uuid4())}, {"email": f"{user}@edupilot.local", "password": PW})
        if c == 200:
            t = json.loads(b)["access_token"]; json.dump({"t": t, "ts": time.time()}, open(f, "w")); return t
        if c in (429, 502, 503):
            sh("docker exec ep-r-redis-1 sh -c \"for p in 'ep:auth:blockip:*' 'ep:auth:ipfail:*' 'ep:auth:fail:*' 'ep:rl:auth*'; do redis-cli --scan --pattern \\\"\\$p\\\" | xargs -r redis-cli del; done\""); time.sleep(5)
    raise RuntimeError(f"login {user}: {c} {b[:120]}")
def call(method, path, user="teacher", body=None, key=None, headers=None, text=False):
    h = {"content-type": "application/json"}
    if user: h["authorization"] = "Bearer " + tok(user)
    if method in ("POST", "PUT", "PATCH", "DELETE") and key != False: h["Idempotency-Key"] = key or str(uuid.uuid4())
    h.update(headers or {})
    for att in range(4):
        c, b, hd = raw(method, path, h, body)
        if c in (429, 502, 503, 504) and att < 3: time.sleep(8 * (att + 1)); continue
        break
    if text: return c, b
    try: return c, json.loads(b)
    except Exception: return c, b.decode("utf8", "replace")[:300]
def cid(code): return sq1(f"select id from courses where class_code='{code}'")
def uid(email): return sq1(f"select id from users where email='{email}@edupilot.local'" if "@" not in email else f"select id from users where email='{email}'")
def rec(tc, v, note):
    note = " ".join(str(note).split())[:155]
    with open(RES, "a") as f: f.write(json.dumps({"tc": tc, "v": v, "note": note}, ensure_ascii=False) + "\n")
    print(("PASS" if v == "P" else v), tc, note)
def chk(tc, cond, note_ok, note_bad=None, bad="F"):
    rec(tc, "P" if cond else bad, note_ok if cond else (note_bad or note_ok)); return cond
