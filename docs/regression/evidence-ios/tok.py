#!/usr/bin/env python3
"""In token GV dùng chung cho cả lô (cache .cache/teacher.json, không commit). 429 → chờ Retry-After rồi thử lại. Không in mật khẩu."""
import json, os, sys, time, urllib.request, urllib.error
HERE = os.path.dirname(os.path.abspath(__file__)); C = os.path.join(HERE, ".cache", "teacher.json")
BASE = os.environ.get("BASE", "https://v2202609419695521098.happysrv.de") + "/api/v1"
def cached():
    try:
        d = json.load(open(C))
        if d["exp"] - time.time() > 120: return d["tok"]
    except Exception: pass
def login():
    pw = open(os.path.expanduser("~/.config/edupilot-team/secrets/vps-seed.txt")).read().strip()
    for _ in range(6):
        r = urllib.request.Request(BASE + "/auth/login", data=json.dumps({"email": "teacher@edupilot.local", "password": pw}).encode(), method="POST", headers={"Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(r, timeout=30) as resp: return json.loads(resp.read())
        except urllib.error.HTTPError as e:
            if e.code == 429:
                w = int(e.headers.get("Retry-After") or 30); print(f"429, chờ {w}s", file=sys.stderr); time.sleep(min(w, 90)); continue
            raise
    sys.exit("login GV: vẫn 429 sau 6 lần — báo PM, không đổi env")
t = cached()
if not t:
    d = login(); t = d["access_token"]
    os.makedirs(os.path.dirname(C), exist_ok=True)
    json.dump({"tok": t, "exp": time.time() + int(d.get("expires_in", 900))}, open(C, "w")); os.chmod(C, 0o600)
print(t)
