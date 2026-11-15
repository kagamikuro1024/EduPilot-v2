# QC sprint 11 pha 2 — thư viện gọi API thật ep-qc (HTTPS 943) + SQL qua docker exec; kết quả ghi results-p10.jsonl
import os, sys, json, time, uuid, subprocess, ssl, urllib.request, urllib.error

B = os.environ.get("QC_API", "https://localhost:943/api/v1")
PG = os.environ.get("QC_PG", "ep-qc-postgres-1")
RES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results-p10.jsonl")
PW = os.environ.get("SEED_DEFAULT_PASSWORD") or [l.split("=", 1)[1].strip() for l in open(os.path.expanduser("~/Documents/TA_Agent_qc5/.env.local")) if l.startswith("SEED_DEFAULT_PASSWORD=")][0]
CTX = ssl.create_default_context(); CTX.check_hostname = False; CTX.verify_mode = ssl.CERT_NONE
_tok = {}


def raw(method, path, body=None, headers=None, token=None, timeout=60):
    h = {"content-type": "application/json"}
    if token: h["authorization"] = "Bearer " + token
    h.update(headers or {})
    r = urllib.request.Request(B + path, data=json.dumps(body).encode() if body is not None else None, method=method, headers=h)
    t0 = time.time()
    try:
        with urllib.request.urlopen(r, context=CTX, timeout=timeout) as f:
            b, code, hd = f.read(), f.status, dict(f.headers)
    except urllib.error.HTTPError as e:
        b, code, hd = e.read(), e.code, dict(e.headers)
    try: j = json.loads(b) if b else None
    except Exception: j = b.decode("utf8", "replace")
    return code, j, (time.time() - t0) * 1000, hd


def tok(user):
    if user not in _tok:
        c, j, _, _ = raw("POST", "/auth/login", {"email": f"{user}@edupilot.local", "password": PW}, {"Idempotency-Key": str(uuid.uuid4())})
        assert c == 200, (user, c, j)
        _tok[user] = j["access_token"]
    return _tok[user]


def api(user, method, path, body=None, headers=None):
    h = dict(headers or {})
    if method == "POST" and "Idempotency-Key" not in h: h["Idempotency-Key"] = str(uuid.uuid4())
    c, j, ms, hd = raw(method, path, body, h, tok(user) if user else None)
    return c, j, ms, hd


def sql(q):
    r = subprocess.run(["docker", "exec", "-i", PG, "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-F", "|", "-c", q], capture_output=True, text=True)
    if r.returncode: raise RuntimeError(r.stderr)
    return [l.split("|") for l in r.stdout.strip().splitlines() if l != "" and not l.startswith(("UPDATE ", "INSERT ", "DELETE "))]


def ok(tc, cond, note=""):
    print(f"{'PASS' if cond else 'FAIL'} {tc} {note}"[:300])
    with open(RES, "a") as f: f.write(json.dumps({"tc": tc, "st": "PASS" if cond else "FAIL", "note": note}, ensure_ascii=False) + "\n")
    return bool(cond)


def course(user, code="761987"):
    c, j, _, _ = api(user, "GET", "/me/courses")
    for it in j["items"]:
        if it["course"]["class_code"] == code: return it["course"]["id"]


def uid(email):
    return sql(f"select id from users where email='{email}@edupilot.local'")[0][0]
