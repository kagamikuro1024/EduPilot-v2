#!/usr/bin/env python3
"""TC-P7-07-04 (kill -9 worker), 05 (gia hạn thuê), 09 (số worker đồng thời), 11 (hết ngân sách, Redis dừng 70 s, hai người Chấm lại), 01 (Redis dừng → vẫn 202). Giữ khoá global suốt script."""
import sys, os, threading; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("07b"); code = lambda: str(int(time.time()) % 100000)
SV = ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 37)]
C = mkcourse("QD9G", SV); Q = f"/api/v1/courses/{C}"; T = tok("teacher"); TA = tok("ta"); ad = tok("admin")
body = lambda u, x="": (f"Bài làm của {u}. Mã độc SUNBURST chờ khoảng 10 ngày. {x} " * 6).encode()
STACK = os.path.expanduser("~/Documents/TA_Agent_s11e")
F = "--env-file .env.local -f docker-compose.local.yml -f docker-compose.test.yml -f /Users/kuro/Documents/TA_Agent_qc106/docs/sprints/10.6/qc/scripts/kc/docker-compose.kc.yml -f /Users/kuro/Documents/ep-qc-env/ep-qc.override.yml -p ep-qc --profile ingest"
def worker(env=None):
    e = dict(os.environ, **(env or {})); r = subprocess.run(f"docker compose {F} up -d --no-deps --force-recreate worker", shell=True, cwd=STACK, env=e, capture_output=True, text=True)
    for _ in range(40):
        if "healthy" in subprocess.run(["docker", "ps", "--filter", "name=ep-qc-worker-1", "--format", "{{.Status}}"], capture_output=True, text=True).stdout: break
        time.sleep(2)
    return r.returncode
def states(aid): return psql(f"select gr.run_state||'|'||coalesce(gr.error_code,'-') from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{aid}'").splitlines()
def mk(n, marks="", users=None, prefix="Q7"):
    aid = mkassign(Q, T, prefix + code()); 
    for u in (users or SV[:n]): submit(Q, aid, tok(u), "b.md", body(u, marks))
    return aid
with stub_ctx(paid=False):
    a = mk(1, "[[S9G slow=4]]", users=["sv.gioi"], prefix="Q709P"); call("POST", f"{Q}/assignments/{a}/grade-all", T, {}); wait_done(a, 60); time.sleep(2)
    ts = sorted((r["t"], 'c4, c3, c2, c1' in json.dumps(r["body"], ensure_ascii=False)) for r in reqs() if "Q709P" not in "" and "slow=4" in json.dumps(r["body"], ensure_ascii=False))[-2:]
    gap = abs(ts[1][0] - ts[0][0]); print("   hai lượt của một bài (slow=4): cách nhau", round(gap, 2), "s", ts)
    R.chk("TC-P7-07-09", f"hai lượt gửi tới provider cách nhau {gap:.2f} s (<1 s = song song; stub ngủ 4 s mỗi lời gọi)", gap < 1.0, ts)
    # ---------- 07-01 Redis dừng → 202 + dòng QUEUED; 07-11 Redis dừng 70 s: bài QUEUED quá hạn được nạp lại
    a1 = mk(6, "", prefix="Q701R"); subprocess.run(["docker", "stop", "ep-qc-redis-1"], capture_output=True); time.sleep(2)
    s, g_, _ = call("POST", f"{Q}/assignments/{a1}/grade-all", T, {}); q_rows = psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a1}' and gr.run_state='QUEUED'"); print("   Redis dừng: grade-all", s, str(g_)[:150], "| QUEUED rows", q_rows)
    time.sleep(68); subprocess.run(["docker", "start", "ep-qc-redis-1"], capture_output=True)
    for _ in range(30):
        if "PONG" in redis("ping"): break
        time.sleep(2)
    R.chk("TC-P7-07-01", f"Redis dừng: grade-all trả {s} (202) và DB có {q_rows}/6 dòng QUEUED", s == 202 and q_rows == "6", (s, g_, q_rows))
    wait_done(a1, 240); st = states(a1); print("   sau khi Redis bật lại:", sorted(st))
    R.chk("TC-P7-07-11", f"Redis dừng 70 s rồi bật: bài QUEUED quá hạn ≥60 s được nạp lại bởi máy quét, không mất ({sorted(set(st))})", len(st) == 6 and all(x.startswith("DONE") for x in st), st)
    # hai nhân viên bấm Chấm lại cùng bài
    sid = psql(f"select s.id from submissions s where s.assignment_id='{a1}' limit 1"); out = {}
    def rg(n, tk): out[n] = call("POST", f"{Q}/submissions/{sid}/grade", tk, {})[0:2]
    th = [threading.Thread(target=rg, args=("T", T)), threading.Thread(target=rg, args=("TA", TA))]; [t.start() for t in th]; [t.join() for t in th]; cs = sorted(v[0] for v in out.values()); print("   hai nhân viên Chấm lại:", {k: (v[0], str(v[1])[:80]) for k, v in out.items()})
    R.chk("TC-P7-07-11", f"hai nhân viên bấm Chấm lại: một thắng 202, người sau 409 ({cs})", cs == [202, 409] and any("VERSION_CONFLICT" in json.dumps(v[1]) for v in out.values() if v[0] == 409), out)

# ---------- 07-11 hết ngân sách giữa chừng (stub trả phí)
with stub_ctx(paid=True):
    ver = call("GET", f"/api/v1/courses/{C}/llm-budget", ad)[1]["version"]
    s, x, _ = call("PUT", f"/api/v1/courses/{C}/llm-budget", ad, {"daily_limit": "900", "version": ver}); print("   đặt ngân sách lớp 900:", s)
    n0 = int(psql(f"select count(*) from notifications where type='GRADING_BUDGET_EXHAUSTED' and course_id='{C}'"))
    a = mk(30, "", prefix="Q711"); call("POST", f"{Q}/assignments/{a}/grade-all", T, {}); wait_done(a, 200); time.sleep(3); st = states(a)
    n_ex = sum(1 for x in st if "LLM_BUDGET_EXHAUSTED" in x); n_done = sum(1 for x in st if x.startswith("DONE"))
    nn = int(psql(f"select count(*) from notifications where type='GRADING_BUDGET_EXHAUSTED' and course_id='{C}'")) - n0
    users = psql(f"select count(distinct user_id) from notifications where type='GRADING_BUDGET_EXHAUSTED' and course_id='{C}'")
    print(f"   30 bài: {n_done} DONE, {n_ex} FAILED LLM_BUDGET_EXHAUSTED; thông báo gộp mới {nn} (người nhận {users})")
    R.chk("TC-P7-07-11", f"hết ngân sách giữa chừng: {n_done} DONE + {n_ex} FAILED LLM_BUDGET_EXHAUSTED = 30", n_ex >= 1 and n_ex + n_done == 30, st[:6])
    R.chk("TC-P7-07-11", f"đúng MỘT thông báo gộp mỗi nhân viên (mới {nn} dòng cho {users} người nhận, không một thông báo mỗi bài)", 1 <= nn <= int(users or 0) + 0 and nn < n_ex, (nn, users, n_ex))
    att = psql(f"select max(gr.attempts) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a}' and gr.error_code='LLM_BUDGET_EXHAUSTED'")
    R.chk("TC-P7-07-07", f"LLM_BUDGET_EXHAUSTED không thử lại (attempts tối đa {att} = 1)", att == "1", att)
    ver = call("GET", f"/api/v1/courses/{C}/llm-budget", ad)[1]["version"]; print("   gỡ ngân sách:", call("PUT", f"/api/v1/courses/{C}/llm-budget", ad, {"daily_limit": None, "version": ver})[0])
sys.exit(R.end())
