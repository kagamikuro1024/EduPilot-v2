#!/usr/bin/env python3
"""QC2 r95 — TC-N03-07, TC-N04-07/08, TC-N07-06 trên gateway mini T1 (http://127.0.0.1:18095)."""
import json, statistics, sys, time, urllib.request, urllib.error, uuid, subprocess
B = "http://127.0.0.1:18095/api/v1"; PW = "Edupilot#Seed-2026"

def req(m, p, tok=None, body=None, idem=False):
    h = {"Content-Type": "application/json"}
    if tok: h["Authorization"] = "Bearer " + tok
    if idem: h["Idempotency-Key"] = str(uuid.uuid4())
    r = urllib.request.Request(B + p, method=m, headers=h, data=None if body is None else json.dumps(body).encode())
    t = time.perf_counter()
    try:
        with urllib.request.urlopen(r, timeout=30) as x: c, b = x.status, x.read()
    except urllib.error.HTTPError as e: c, b = e.code, e.read()
    return c, (time.perf_counter() - t) * 1000, b

def login(u, pw=PW):
    c, _, b = req("POST", "/auth/login", body={"email": f"{u}@edupilot.local", "password": pw}, idem=False)
    return c, (json.loads(b).get("access_token") if c == 200 else None)

def psql(sql):
    return subprocess.run(["docker", "exec", "-i", "ep-r95-pg", "psql", "-U", "edupilot", "-d", "edupilot_t1_r95", "-qAt"], input=sql, capture_output=True, text=True).stdout.strip()

out = {}
# N03-07
res = {}
for u in ["t1sv0007", "t1sv0123", "t1sv0456", "t1sv0789", "t1sv0999", "t1ta1", "t1gv1"]:
    res[u] = login(u)[0]
res["sai_mat_khau"] = login("t1sv0007", "sai-r95")[0]
out["N03-07"] = res
# N04-07/08
c1 = psql("select c.id from courses c where class_code='T1-03'")
sv = "t1sv0001"  # lớp T1-01
c_other = c1  # SV lớp 1 gọi lớp 3
_, tok = login(sv)
codes = {}
for _ in range(100):
    c, ms, _b = req("GET", f"/courses/{c_other}/gradebook", tok); codes[c] = codes.get(c, 0) + 1
out["N04-07_gradebook_x100_SV_lop_khac"] = codes
own = psql("select c.id from courses c join enrollments e on e.course_id=c.id join users u on u.id=e.user_id where u.email='t1sv0001@edupilot.local' limit 1")
codes = {}
for _ in range(100):
    c, ms, _b = req("GET", f"/courses/{own}/gradebook", tok); codes[c] = codes.get(c, 0) + 1
out["N04-07_gradebook_x100_SV_lop_minh"] = codes
r8 = {}
for name, m, p, body in [("grade-items", "GET", f"/courses/{own}/grade-items", None), ("POST questions", "POST", f"/courses/{own}/questions", {"type": "MCQ_SINGLE", "title": "x", "topic": "x", "stem": "x"})]:
    cs = {}
    for _ in range(20):
        c, _ms, _b = req(m, p, tok, body, idem=(m == "POST")); cs[c] = cs.get(c, 0) + 1
    r8[name] = cs
out["N04-08"] = r8
# N07-06: 200 lần tuần tự
gv_tok = login("t1gv1")[1]
gv_course = psql("select e.course_id from enrollments e join users u on u.id=e.user_id where u.email='t1gv1@edupilot.local' limit 1")
def lat(m, p, t, n=200, body=None, idem=False):
    xs = []
    for _ in range(n):
        c, ms, _b = req(m, p, t, body, idem); xs.append(ms)
        if c >= 500: xs[-1] = float("inf")
    xs.sort()
    return {"p50": round(statistics.median(xs), 1), "p95": round(xs[int(.95 * n) - 1], 1), "n": n}
out["N07-06"] = {
    "sv_me_today (đọc)": lat("GET", "/me/today", tok),
    "gv_gradebook (đọc nặng)": lat("GET", f"/courses/{gv_course}/gradebook", gv_tok),
    "w_heartbeat (ghi)": lat("POST", "/activity/heartbeat", tok, body={"course_id": own, "area": "CHAT", "tab_id": "00000000-0000-4000-8000-000000000001"}),
}
print(json.dumps(out, ensure_ascii=False, indent=1))
