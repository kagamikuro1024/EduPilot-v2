import sys, os, re, time; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
import gdlib
P = "/tmp/qc9s-pw.lock"
C = cid("QD105A"); C2 = cid("QD105B"); DEC = "/admin/llm/decision"
def fake(**k): return api("admin", "POST", "/_test/llm/fake", k)
def gate(g, system, courses):
    sql(f"UPDATE decision_gates SET enabled={'true' if system else 'false'}, version=version+1 WHERE gate='{g}'")
    for cc in (C, C2):
        sql(f"INSERT INTO course_decision_gates(course_id,gate,enabled) VALUES ('{cc}','{g}',{'true' if cc in courses else 'false'}) ON CONFLICT (course_id,gate) DO UPDATE SET enabled=EXCLUDED.enabled, version=course_decision_gates.version+1")
    os.system("docker exec ep-qc-redis-1 redis-cli publish ep:decision:cfg x >/dev/null"); time.sleep(1.5)
def off(): sql("UPDATE decision_gates SET enabled=false"); sql("UPDATE course_decision_gates SET enabled=false"); os.system("docker exec ep-qc-redis-1 redis-cli publish ep:decision:cfg x >/dev/null"); time.sleep(1.2)
def now(): return sql("select now()")[0][0]
def rows(since, extra=""): return sql(f"select cascade_gate,cascade_path,status,coalesce(error_kind,''),coalesce(latency_ms,0) from llm_audit where task='DECIDE' and created_at>='{since}' {extra} order by created_at")
SUBTLE = ("Có bạn hoàn cảnh khó khăn bị trừ điểm", "Mình nghe nói có một bạn trong lớp hoàn cảnh khó khăn nên bị trừ điểm chuyên cần, không biết có đúng không và nên xử lý thế nào ạ.")
def post(user, course, t=None):
    t = t or ("QC105 " + str(time.time())[-6:] + " " + SUBTLE[0], SUBTLE[1])
    return api(user, "POST", f"/courses/{course}/threads", {"title": t[0], "body": t[1]})
def run():
    flag_ok = fake(decision_mode="", decision_error_rate=0.0, decision_p50_ms=0, decision_p95_ms=0); print("fake reset", flag_ok[0])
    off(); redis_del("ep:llm:cb:*")
    # baseline độ trễ khi cổng tắt
    lat0 = []
    for _ in range(4): c, j, ms = post("sv.kha", C); lat0.append(round(ms)); time.sleep(3.4)
    # D01-16 / D02-20: ép lỗi decision → cầu dao mở
    gate("THREADS_PRECHECK", True, [C]); s = now(); fake(decision_mode="error")
    st = []; lat = []
    for _ in range(12): c, j, ms = post("sv.kha", C); st.append(c); lat.append(round(ms)); time.sleep(3.4)
    d = rows(s); circ = [x for x in d if x[2] == "circuit_open"]
    print("circuit rows", len(circ), "of", len(d), set((x[1], x[2]) for x in d)); print("post status", st, "lat", lat, "base", lat0)
    fake(decision_mode="")
    ok1 = len(circ) >= 1 and all(x[1] == "FALLBACK" for x in circ) and max(int(float(x[4])) for x in circ) <= 5 and all(c in (200, 201, 422) for c in st)
    chk("TC-D01-16", ok1, f"decision_mode=error (ép lỗi) + cổng THREADS bật ở QD105A: 14 bài đăng → {len(d)} dòng DECIDE, {len(circ)} dòng status=circuit_open path=FALLBACK latency_ms tối đa {max([int(float(x[4])) for x in circ] or [0])} (không gọi mạng); mọi POST 200/201/422 {sorted(set(st))}")
    lc = sorted(lat)[len(lat)//2]; lb = sorted(lat0)[len(lat0)//2]
    chk("TC-D02-20", ok1 and abs(lc - lb) < 400, f"cầu dao decision mở + cổng bật: đường FALLBACK, không 5xx; độ trễ đăng bài trung vị {lc} ms so với cổng tắt {lb} ms (cầu dao mở không chờ)")
    # D02-10 / D01-25: provider tắt
    redis_del("ep:llm:cb:*"); gate("THREADS_PRECHECK", True, [C])
    c, pj, _ = api("admin", "GET", DEC); pid = pj["provider"]["id"]; c, pr, _ = api("admin", "GET", "/admin/llm/providers"); cur = [x for x in pr["items"] if x["id"] == pid][0]
    c, r, _ = api("admin", "PUT", f"/admin/llm/providers/{pid}", {**{k: cur.get(k) for k in ("name", "type", "base_url", "rpm_limit", "tpm_limit") if cur.get(k) is not None}, "enabled": False, "version": cur["version"]}); print("provider off", c, lastj(r, 300))
    time.sleep(2); s = now(); res = [post("sv.kha", C)[0] for _ in range(3)]; d = rows(s)
    cur2 = [x for x in api("admin", "GET", "/admin/llm/providers")[1]["items"] if x["id"] == pid][0]
    api("admin", "PUT", f"/admin/llm/providers/{pid}", {**{k: cur2.get(k) for k in ("name", "type", "base_url", "rpm_limit", "tpm_limit") if cur2.get(k) is not None}, "enabled": True, "version": cur2["version"]}); time.sleep(2)
    chk("TC-D02-10", c == 200 and len(d) == 0 and all(x in (200, 201, 422) for x in res), f"cổng THREADS bật (hệ thống + QD105A) nhưng provider «Quyết định» tắt (PUT enabled=false → {c}): 3 bài đăng → DECIDE {len(d)} dòng (kỳ vọng 0), POST {res}; đã bật lại provider")
    off(); fake(decision_mode="")
import shutil
t0 = time.time()
while True:
    try: os.mkdir("/tmp/qc-global.lock"); break
    except FileExistsError:
        if time.time() - t0 > 600: raise SystemExit("LOCKED")
        time.sleep(5)
open("/tmp/qc-global.lock/owner", "w").write("QC3 105 decision")
try:
        run()
finally:
    off(); fake(decision_mode=""); redis_del("ep:llm:cb:*"); shutil.rmtree("/tmp/qc-global.lock", ignore_errors=True)
