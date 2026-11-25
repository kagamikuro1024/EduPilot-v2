#!/usr/bin/env python3
"""qc2 25/11: bản thay cho phần (b) của qc-real-f1-account «dò mật khẩu» trên ep-r, đợi đúng Retry-After thay vì hỏi dồn mỗi 2,5 s
(hỏi dồn tự chạm RATE_LIMITED theo IP nên spec gốc không bao giờ tới lần sai thứ 10). Kiểm: 10 lần sai THẬT → locked_until ≈ 15 phút; đúng mật khẩu cũng bị từ chối; đúng MỘT thư cảnh báo."""
import json, re, ssl, subprocess, time, urllib.request, urllib.error, uuid
import os
T2 = bool(os.environ.get("T2"))
B = "https://localhost:4643/api/v1" if T2 else "https://localhost:1143/api/v1"; MP = "http://localhost:12225" if T2 else "http://localhost:8725"; PGC = "qc2seedt2-postgres-1" if T2 else "ep-r-postgres-1"; ctx = ssl._create_unverified_context()
def call(path, body):
    r = urllib.request.Request(B + path, data=json.dumps(body).encode(), headers={"Content-Type": "application/json", "Origin": B.split("/api")[0], "Accept": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(r, context=ctx, timeout=30) as x: return x.status, None, None
    except urllib.error.HTTPError as e:
        try: j = json.loads(e.read())
        except Exception: j = {}
        return e.code, j.get("code"), e.headers.get("Retry-After")
def sql(q): return subprocess.run(["docker", "exec", "-i", PGC, "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", q], capture_output=True, text=True).stdout.strip()
def mails(em):
    with urllib.request.urlopen(f"{MP}/api/v1/search?query=" + urllib.request.quote(f"to:{em}"), timeout=20) as x: return json.loads(x.read())["messages"]
tag = uuid.uuid4().hex[:6]; em = f"qcf1.lock.{tag}@edupilot.local"; pw = f"Lock!{tag}zA9"
print("register", call("/auth/register", {"email": em, "password": pw, "full_name": "QC Khoá Dò", "student_code": ""})[0])
t0 = time.time(); wrong = 0; seen = []
while wrong < 10 and time.time() - t0 < 300:
    s, c, ra = call("/auth/login", {"email": em, "password": f"Sai!{wrong}{tag}"}); seen.append((round(time.time() - t0, 1), s, c, ra)); print("  ", seen[-1], flush=True)
    if s == 401: wrong += 1
    else: time.sleep(float(ra or 2) + 0.5)
print(f"10 lần sai thật: {wrong} sau {time.time()-t0:.0f}s")
for x in seen: print("  ", x)
print("  số lần bị chờ:", sum(1 for x in seen if x[1] != 401), "codes:", sorted({x[2] for x in seen if x[1] != 401}))
lu = sql(f"select locked_until is not null and locked_until > now() + interval '10 minutes' and locked_until < now() + interval '20 minutes' from users where email='{em}'")
print("locked_until ≈ 15 phút:", lu)
s, c, ra = call("/auth/login", {"email": em, "password": pw}); print("đúng mật khẩu khi đang khoá:", s, c, "Retry-After", ra)
time.sleep(1); call("/auth/login", {"email": em, "password": f"Sai!x{tag}"}); time.sleep(5)
m = [x for x in mails(em) if re.search(r"khoá|bất thường|cảnh báo", x.get("Subject", ""), re.I)]
print("thư cảnh báo:", len(m), [x["Subject"] for x in m])
ok = wrong >= 10 and lu == "t" and s in (423, 429) and len(m) == 1
print("KẾT QUẢ:", "ĐẠT" if ok else "TRƯỢT")
