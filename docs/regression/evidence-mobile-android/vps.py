# TC-AS09-34 trên VPS thật (ZeroSSL): API bằng urllib, mật khẩu đọc từ ~/.config/edupilot-team/secrets/vps-seed.txt (không in). Token chỉ giữ trong bộ nhớ.
import os, json, ssl, uuid, urllib.request, urllib.error
HOST = "https://v2202609419695521098.happysrv.de"
PW = open(os.path.expanduser("~/.config/edupilot-team/secrets/vps-seed.txt")).read().strip()
_tok = {}
def call(user, method, path, body=None):
    h = {"Content-Type": "application/json"}
    if method != "GET": h["Idempotency-Key"] = str(uuid.uuid4())
    if user:
        if user not in _tok:
            c, j = call(None, "POST", "/auth/login", {"email": f"{user}@edupilot.local", "password": PW}); assert c == 200, (user, c)
            _tok[user] = j["access_token"]
        h["Authorization"] = "Bearer " + _tok[user]
    r = urllib.request.Request(HOST + "/api/v1" + path, data=(json.dumps(body).encode() if body is not None else None), headers=h, method=method)
    try:
        with urllib.request.urlopen(r, timeout=60) as x: raw = x.read(); return x.status, (json.loads(raw) if raw else None)
    except urllib.error.HTTPError as e:
        raw = e.read(); return e.code, (json.loads(raw) if raw else None)
def course(code):
    c, j = call("teacher", "GET", "/me/courses?limit=100"); assert c == 200, c
    return next(x["course"] for x in j["items"] if x["course"]["class_code"] == code)
def today_session(cid):
    import datetime
    now = datetime.datetime.utcnow().isoformat() + "Z"
    c, j = call("teacher", "GET", f"/courses/{cid}/sessions?limit=100"); assert c == 200, (c, j)
    return next(s for s in j["items"] if s["starts_at"] <= now <= s["ends_at"])
def open_window(cid, sid, minutes=30):
    c, j = call("teacher", "POST", f"/courses/{cid}/sessions/{sid}/checkin-windows", {"minutes": minutes}); assert c == 201, (c, j)
    c, d = call("teacher", "GET", f"/courses/{cid}/sessions/{sid}/checkin-windows/{j['id']}/display"); assert c == 200, (c, d)
    return j["id"], d
def checkins(cid, sid):
    c, j = call("teacher", "GET", f"/courses/{cid}/sessions/{sid}/checkins?limit=100"); assert c == 200, (c, j); return j
if __name__ == "__main__":
    for code in ("QCVPSA", "QCVPSD"):
        try:
            k = course(code); print(code, k["id"]); s = today_session(k["id"]); print(" session", s["id"], s["starts_at"], s["ends_at"])
        except Exception as e: print(code, "ERR", repr(e)[:200])
