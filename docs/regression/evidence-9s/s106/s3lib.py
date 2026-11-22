# Gọi API qc2seedt3 (https://localhost:4743) + SQL qua docker exec qc2seedt3-postgres-1. Không dùng biến môi trường ep-qc.
import json, os, ssl, subprocess, time, urllib.request, urllib.error
A = "https://localhost:4743/api/v1"; CTX = ssl._create_unverified_context()
PW = [l.split("=", 1)[1].strip() for l in open(os.path.expanduser("~/Documents/TA_Agent_qc5/.env.local")) if l.startswith("SEED_DEFAULT_PASSWORD=")][0]
_tok = {}
def raw(method, path, body=None, tok=None, hdr=None):
    h = {"content-type": "application/json", **(hdr or {})}
    if tok: h["authorization"] = "Bearer " + tok
    if method == "POST" and "idempotency-key" not in h: import uuid; h["idempotency-key"] = str(uuid.uuid4())
    rq = urllib.request.Request(A + path, data=json.dumps(body).encode() if body is not None else None, method=method, headers=h)
    try:
        with urllib.request.urlopen(rq, context=CTX, timeout=60) as r: t = r.read().decode(); return r.status, (json.loads(t) if t else None), t
    except urllib.error.HTTPError as e: t = e.read().decode(); return e.code, (json.loads(t) if t.startswith(("{", "[")) else None), t
def tok(u):
    if u not in _tok: _tok[u] = raw("POST", "/auth/login", {"email": f"{u}@edupilot.local", "password": PW})[1]["access_token"]
    return _tok[u]
def api(u, m, p, b=None, **kw): return raw(m, p, b, tok(u), kw.get("hdr"))
def sql(q): 
    r = subprocess.run(["docker", "exec", "qc2seedt3-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True)
    if r.returncode: raise RuntimeError(r.stderr)
    return [l.split("|") for l in r.stdout.strip().splitlines()]
def cid(code): return sql(f"select id from courses where class_code='{code}'")[0][0]
