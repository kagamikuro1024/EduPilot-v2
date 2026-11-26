# SEC-14 / SEC-15 / SEC-11 (API + SQL) cho lô Android-only. Mật khẩu chỉ đọc từ file bí mật, không in.
import os, sys, json, uuid, urllib.request, urllib.error
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import m, vps
H = vps.HOST
def req(method, path, body=None, tok=None):
    h = {"Content-Type": "application/json"}
    if method != "GET": h["Idempotency-Key"] = str(uuid.uuid4())
    if tok: h["Authorization"] = "Bearer " + tok
    r = urllib.request.Request(H + "/api/v1" + path, data=(json.dumps(body).encode() if body is not None else None), headers=h, method=method)
    try:
        with urllib.request.urlopen(r, timeout=60) as x: raw = x.read(); return x.status, (json.loads(raw) if raw else None)
    except urllib.error.HTTPError as e:
        raw = e.read(); return e.code, (json.loads(raw) if raw else None)
def login(email, pw):
    c, j = req("POST", "/auth/login", {"email": email, "password": pw}); return (j or {}).get("access_token") if c == 200 else None
if __name__ == "__main__":
    pw_sv = m.acct("sv.qc2")
    sv = login("sv.qc2@edupilot.local", pw_sv); print("sv.qc2 login:", bool(sv))
    toks = {"SV sv.qc2": sv}
    for u in ("teacher", "ta", "admin"):
        t = login(f"{u}@edupilot.local", vps.PW); toks[u] = t; print(u, "login:", bool(t))
    toks["khong token"] = None
    print("--- SEC-14: POST /me/devices/challenge")
    for k, t in toks.items():
        if k != "khong token" and k != "SV sv.qc2" and not t: print(k, "-> bo qua (khong dang nhap duoc)"); continue
        c, j = req("POST", "/me/devices/challenge", {}, t); print(f"{k:12s} -> {c} {(j or {}).get('code','')}")
