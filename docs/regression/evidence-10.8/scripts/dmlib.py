# Thư viện QC sprint 10: gọi API thật của stack ep-qc (https://localhost:943), ghi kết quả TC vào results.jsonl
import json, ssl, time, uuid, urllib.request, urllib.error, os, sys
B = os.environ.get("QC_API","https://localhost:1143/api/v1")
PW = os.environ.get("SEED_DEFAULT_PASSWORD", "Edupilot#Seed-2026")
CTX = ssl.create_default_context(); CTX.check_hostname = False; CTX.verify_mode = ssl.CERT_NONE
_tok = {}
RES = os.path.join(os.path.dirname(__file__), "results.jsonl")

def raw(method, path, body=None, headers=None, token=None, base=B, timeout=60):
    # smoke r10.8: 502/503 thoáng qua (PM-QC/nhóm khác khởi động lại gateway) → thử lại tối đa 8 lần cách 5 s
    for _try in range(8):
        r = _raw(method, path, body, headers, token, base, timeout)
        if r[0] not in (502, 503, 504): return r
        time.sleep(5)
    return r

def _raw(method, path, body=None, headers=None, token=None, base=B, timeout=60):
    h = {"content-type": "application/json"}
    if token: h["authorization"] = "Bearer " + token
    h.update(headers or {})
    data = json.dumps(body).encode() if body is not None else None
    r = urllib.request.Request(base + path, data=data, method=method, headers=h)
    t0 = time.time()
    try:
        with urllib.request.urlopen(r, context=CTX, timeout=timeout) as f:
            b = f.read(); code = f.status; hd = dict(f.headers)
    except urllib.error.HTTPError as e:
        b = e.read(); code = e.code; hd = dict(e.headers)
    ms = (time.time() - t0) * 1000
    try: j = json.loads(b) if b else None
    except Exception: j = b.decode("utf8", "replace")
    return code, j, ms, hd

_TF = "/tmp/qc2-h108-tok.json"
def tok(user):
    if not _tok:
        try:
            d = json.load(open(_TF)); _tok.update({u: v[0] for u, v in d.items() if time.time() - v[1] < 600})
        except Exception: pass
    if user not in _tok:
        c, j, _, _ = raw("POST", "/auth/login", {"email": f"{user}@edupilot.local", "password": PW}, {"Idempotency-Key": str(uuid.uuid4())})
        assert c == 200, (user, c, j)
        _tok[user] = j["access_token"]
        try: d = json.load(open(_TF))
        except Exception: d = {}
        d[user] = [_tok[user], time.time()]; json.dump(d, open(_TF, "w"))
    return _tok[user]

def api(user, method, path, body=None, idem=None, **kw):
    h = dict(kw.pop("headers", {}) or {})
    if method in ("POST",) and "Idempotency-Key" not in h and idem is not False:
        h["Idempotency-Key"] = idem or str(uuid.uuid4())
    c, j, ms, hd = raw(method, path, body, h, tok(user) if user else None, **kw)
    return c, j, ms

_mem = {}
def ok(tc, cond, note=""):
    st = "PASS" if cond else "FAIL"
    print(f"{st} {tc} {note}"[:260])
    with open(RES, "a") as f: f.write(json.dumps({"tc": tc, "st": st, "note": note}, ensure_ascii=False) + "\n")
    return cond

def course(user, code="761987"):
    c, j, _ = api(user, "GET", "/me/courses")
    for it in j["items"]:
        if it["course"]["class_code"] == code: return it["course"]["id"]

import subprocess
def sql(q):
    r = subprocess.run(["docker", "exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-F", "|", "-c", q],
                       capture_output=True, text=True)
    if r.returncode: raise RuntimeError(r.stderr)
    return [l.split("|") for l in r.stdout.strip().splitlines()]

def sse(user, path, body=None, idem=None):
    """Trả (status, ms tới byte đầu, ms tổng, văn bản)."""
    h = {"content-type": "application/json", "authorization": "Bearer " + tok(user), "accept": "text/event-stream", "Idempotency-Key": idem or str(uuid.uuid4())}
    r = urllib.request.Request(B + path, data=json.dumps(body or {}).encode(), method="POST", headers=h)
    t0 = time.time()
    try:
        f = urllib.request.urlopen(r, context=CTX, timeout=60)
    except urllib.error.HTTPError as e:
        return e.code, (time.time() - t0) * 1000, (time.time() - t0) * 1000, e.read().decode("utf8", "replace")
    first = f.read1(1) if hasattr(f, "read1") else f.read(1)
    t1 = (time.time() - t0) * 1000
    rest = f.read()
    return f.status, t1, (time.time() - t0) * 1000, (first + rest).decode("utf8", "replace")

def pct(xs, p):
    xs = sorted(xs); return xs[min(len(xs) - 1, int(len(xs) * p))]
