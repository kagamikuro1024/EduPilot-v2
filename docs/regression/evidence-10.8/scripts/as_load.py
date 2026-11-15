# smoke r10.8: TC-AS14-06/07 thu nhỏ: ≤ 100 SV (tài khoản seed ACTIVE của ep-r) thay cho k6 200 SV/T1 (ep-r không có seed T1). spread 30 s + burst 5 s.
import os, sys, json, time, uuid, random, statistics, subprocess, threading, re
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from asl import *
import concurrent.futures as cf
em = [x[0].split("@")[0] for x in sql("select email from users where role='STUDENT' and status='ACTIVE' and password_hash is not null and (email like 'sv%@edupilot.local' or email like 'g7s%@edupilot.local' or email like 'r5s%@edupilot.local') order by email limit 100")]
print("SV:", len(em))
code = "QCKXDRZ"
ex = (sql("select id from courses where class_code='QC-K6r108'") or [[None]])[0][0]
if not ex:
    c, j, _ = api(ADMIN, "POST", "/admin/courses", {"subject_code": "QCAS", "class_code": "QC-K6r108", "name": "QC tải r108", "semester": "2026-2027-HK1", "teacher_id": uid(TEACHER), "ta_ids": [uid(TA)], "join_code": code}); assert c == 201, (c, j); ex = j["id"]
    gen_sessions(ex, 1, 40)
K = ex; policy(K, rotate_seconds=60)
def joiner(u):
    for _ in range(4):
        c = api(u, "POST", "/courses/join", {"code": code})[0]
        if c == 200: return True
        time.sleep(3)
    return False
with cf.ThreadPoolExecutor(8) as ex_: jr = list(ex_.map(joiner, em))
em = [u for u, ok_ in zip(em, jr) if ok_]; print("joined", len(em))
ss = sessions(TEACHER, K)
def run(name, sid, spread):
    c, j, _ = openw(K, sid); assert c == 201; w = j["id"]; t = tokenof(display(K, sid, w)[1])
    toks = {u: tok(u) for u in em}   # đăng nhập trước, ngoài đồng hồ đo
    lat, codes = [], []; lock = threading.Lock(); t0 = time.time()
    def one(u):
        time.sleep(random.uniform(0, spread))
        r = raw("POST", f"/courses/{K}/checkins", {"token": t}, {"Idempotency-Key": str(uuid.uuid4())}, toks[u])
        with lock: lat.append(r[2]); codes.append(r[0])
    # nền: poll display của GV 1 req / 10 s + đọc sessions 5/s
    stop = threading.Event(); dl = []
    def bg():
        while not stop.is_set():
            r = raw("GET", f"/courses/{K}/sessions/{sid}/checkin-windows/{w}/display", None, {}, tok(TEACHER)); dl.append((r[0], r[2])); stop.wait(10)
    th = threading.Thread(target=bg); th.start()
    with cf.ThreadPoolExecutor(len(em)) as p: list(p.map(one, em))
    stop.set(); th.join(); close(K, w)
    rows = int(sql(f"select count(*) from attendance_checkins where window_id='{w}'")[0][0]); lat.sort()
    p95 = lat[int(len(lat) * 0.95) - 1]; p99 = lat[min(len(lat) - 1, int(len(lat) * 0.99))]
    return dict(name=name, n=len(em), ok=codes.count(201), non201=[c for c in codes if c != 201][:5], rows=rows, p95=round(p95, 1), p99=round(p99, 1), mx=round(lat[-1], 1), disp=dl[:3], secs=round(time.time() - t0, 1))
T0 = time.strftime('%Y-%m-%dT%H:%M:%S', time.gmtime())
# docker stats lúc tải
def stats(): return subprocess.run("docker stats --no-stream --format '{{.Name}} {{.CPUPerc}} {{.MemUsage}}' ep-r-gateway-1 ep-r-postgres-1 ep-r-redis-1", shell=True, capture_output=True, text=True).stdout.strip().replace("\n", " | ")
res = []
mon = subprocess.Popen("docker exec ep-r-redis-1 sh -c 'timeout 40 redis-cli monitor'", shell=True, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
for name, spread, k in (("spread", 30, 33), ("burst", 5, 34)):
    r = run(name, ss[k]["id"], spread); res.append(r); r["stats"] = stats(); print(json.dumps(r, ensure_ascii=False))
try: mon.terminate()
except Exception: pass
mout = mon.stdout.read() if mon.stdout else ""
att = [l for l in mout.splitlines() if "ep:att:" in l and "codefail" not in l]
okk = all(r["ok"] >= r["n"] - 1 and r["rows"] == r["ok"] and r["p95"] <= 500 and set(r["non201"]) <= {403} for r in res)
import subprocess as sp
lg = sp.run("docker logs --since 10m ep-r-gateway-1 2>&1 | grep '/checkins\"' | grep '\"method\":\"POST\"'", shell=True, capture_output=True, text=True).stdout.splitlines()
dur = sorted(json.loads(l)["duration_ms"] for l in lg if l.startswith("{") and '"status":201' in l)
srv = f"server n={len(dur)} p50={dur[len(dur)//2] if dur else None} p95={dur[int(len(dur)*.95)-1] if dur else None} max={dur[-1] if dur else None} ms"
print(srv)
ok("TC-AS14-06 (thu nhỏ ≤ 100 SV) spread 30 s và burst 5 s: p95 POST ≤ 500 ms, lỗi 0 %, đúng N hàng, display ≤ 300 ms", okk and all(d[1] < 300 for r in res for d in r["disp"] if d[0] == 200), srv + " | " + json.dumps([{k: v for k, v in r.items() if k in ("name", "n", "ok", "rows", "p95", "p99", "mx", "secs", "non201")} for r in res]))
ok("TC-AS14-07 trong tải: không 5xx; Redis không dùng khoá ep:att:* ngoài codefail trên đường nóng (MONITOR mẫu); docker stats ghi nhận", okk and not att, f"ep:att ngoài codefail={len(att)}; {res[0]['stats']} || {res[1]['stats']}")
