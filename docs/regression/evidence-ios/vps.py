#!/usr/bin/env python3
"""VPS helper cho pha 3 iOS: đọc/ghi chính sách điểm danh, mở/đóng cửa sổ, lấy mã.
Mật khẩu đọc từ tệp bí mật; không in ra.
  python3 vps.py info <course>              # chính sách + version + sessions
  python3 vps.py policy <course> self=on|off
  python3 vps.py open <course> <session> [minutes]
  python3 vps.py display <course> <session> <window>
  python3 vps.py close <course> <window>
  python3 vps.py code <course> <session> <window>   # short_code hiện tại
"""
import json, os, ssl, sys, urllib.request, urllib.error, uuid

B = "https://v2202609419695521098.happysrv.de/api/v1"
CTX = ssl._create_unverified_context()
SEC = os.path.expanduser("~/.config/edupilot-team/secrets/")

def pw_for(email):
    acct = os.path.join(SEC, "vps-qc-accounts.txt")
    for line in open(acct):
        if "\t" in line and not line.startswith("#"):
            e, p = line.rstrip("\n").split("\t", 1)
            if e == email:
                return p
    return open(os.path.join(SEC, "vps-seed.txt")).read().strip()

def call(path, tk=None, method="GET", body=None, headers=None):
    h = {"content-type": "application/json"}
    if tk:
        h["authorization"] = "Bearer " + tk
    h.update(headers or {})
    req = urllib.request.Request(B + path, data=body, headers=h, method=method)
    try:
        r = urllib.request.urlopen(req, context=CTX, timeout=30)
        t = r.read().decode()
        return r.status, (json.loads(t) if t else None)
    except urllib.error.HTTPError as e:
        t = e.read().decode()
        try:
            return e.code, json.loads(t)
        except Exception:
            return e.code, t

def login(email="teacher@edupilot.local"):
    s, j = call("/auth/login", method="POST", body=json.dumps({"email": email, "password": pw_for(email)}).encode())
    if s != 200:
        raise SystemExit(f"login {s}")
    return j["access_token"]

def main(a):
    cmd, *x = a
    tk = login()
    if cmd == "info":
        cid = x[0]
        s, d = call(f"/courses/{cid}", tk)
        print("course", s, d.get("class_code"), "attendance", json.dumps(d.get("attendance")), "version", (d.get("settings") or {}).get("version"))
        s, cal = call(f"/courses/{cid}/calendar?limit=5", tk)
        for it in (cal.get("items") or [])[:5]:
            print("session", it.get("id") or it.get("session_id"), it.get("starts_at"), it.get("session_no"))
    elif cmd == "policy":
        cid, kv = x[0], x[1]
        s, d = call(f"/courses/{cid}", tk)
        ver = (d.get("settings") or {}).get("version")
        val = kv.split("=")[1] in ("on", "true", "1")
        body = json.dumps({"version": ver, "attendance": {"self_checkin": val}}).encode()
        s, j = call(f"/courses/{cid}/settings", tk, "PUT", body)
        print("policy", s, json.dumps(j)[:200])
    elif cmd == "open":
        cid, sid = x[0], x[1]
        mins = int(x[2]) if len(x) > 2 else 10
        s, j = call(f"/courses/{cid}/sessions/{sid}/checkin-windows", tk, "POST",
                    json.dumps({"minutes": mins}).encode(), {"Idempotency-Key": str(uuid.uuid4())})
        print("open", s, json.dumps(j)[:300])
    elif cmd == "display":
        cid, sid, wid = x
        s, j = call(f"/courses/{cid}/sessions/{sid}/checkin-windows/{wid}/display", tk)
        print("display", s, json.dumps(j)[:300])
    elif cmd == "close":
        cid, wid = x
        s, j = call(f"/courses/{cid}/checkin-windows/{wid}/close", tk, "POST", b"{}")
        print("close", s, json.dumps(j)[:200])
    elif cmd == "code":
        cid, sid, wid = x
        s, j = call(f"/courses/{cid}/sessions/{sid}/checkin-windows/{wid}/display", tk)
        print(j.get("short_code") if j else s, "|", j.get("qr_url") if j else "")
    else:
        raise SystemExit(__doc__)

main(sys.argv[1:])
