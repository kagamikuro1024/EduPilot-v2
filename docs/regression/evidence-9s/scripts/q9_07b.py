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
    # ---------- 07-04 kill -9 worker
    a = mk(8, "[[S9G slow=10]]", prefix="Q704"); t0 = time.time(); n_req0 = len(reqs())
    call("POST", f"{Q}/assignments/{a}/grade-all", T, {}); time.sleep(9)
    run_before = states(a); subprocess.run(["docker", "kill", "ep-qc-worker-1"], capture_output=True); tk = time.time(); print("   docker kill (SIGKILL) worker; trạng thái:", sorted(run_before))
    time.sleep(3); subprocess.run(["docker", "start", "ep-qc-worker-1"], capture_output=True)
    lease_seen = psql(f"select count(*) from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a}' and gr.run_state='RUNNING'")
    wait_done(a, 240); time.sleep(3); st = states(a); dur = time.time() - tk
    dup = psql(f"select count(*) from (select submission_id from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a}' group by submission_id having count(*)>1) x")
    req_per = {u: len([r for r in reqs(t0) if f"Bài làm của {u}." in json.dumps(r["body"], ensure_ascii=False)]) for u in SV[:8]}
    print("   sau khi worker sống lại: ", sorted(st), f"{dur:.0f}s", "| lời gọi LLM/bài:", req_per)
    R.chk("TC-P7-07-04", f"sau kill -9 + chờ thuê 60 s: đủ 8 DONE, không bài mất ({sorted(set(st))})", len(st) == 8 and all(x.startswith("DONE") for x in st), st)
    R.chk("TC-P7-07-04", f"đúng một kết quả mỗi bài (having count>1 = {dup})", dup == "0", dup)
    R.chk("TC-P7-07-04", f"số lời gọi LLM mỗi bài ≤ 4 (2 lượt × tối đa 2 lần chạy do bị kill), bài không bị kill đúng 2: {req_per}", all(v <= 4 for v in req_per.values()) and sum(1 for v in req_per.values() if v == 2) >= 1, req_per)
    # worker chậm quay lại sau khi bài xong → ghi bị bỏ: cướp thuê giữa chừng
    a2 = mk(1, "[[S9G slow=15]]", prefix="Q704B"); call("POST", f"{Q}/assignments/{a2}/grade-all", T, {}); time.sleep(5)
    psql(f"update grading_results set lease_owner='thief-qc9s', lease_until=now()+interval '1 second' where id in (select gr.id from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a2}')"); wait_done(a2, 120); time.sleep(3)
    wl = subprocess.run(["docker", "logs", "ep-qc-worker-1", "--since", "3m"], capture_output=True, text=True); lost = (wl.stdout + wl.stderr).count("mất thuê")
    s2 = states(a2); print("   cướp thuê:", s2, "| 'mất thuê — bỏ kết quả' trong log:", lost)
    R.chk("TC-P7-07-04", f"worker mất thuê (lease_owner bị đổi) → ghi bị bỏ (log 'mất thuê' {lost} lần), bài vẫn có đúng 1 kết quả {s2}", lost >= 1 and len(s2) == 1, (lost, s2))
    # ---------- 07-05 / 07-09 cấu hình worker
    print("   worker: GRADING_LEASE=10 HEARTBEAT=3 REQUEUE_AFTER=10"); assert worker({"GRADING_LEASE": "10", "GRADING_HEARTBEAT": "3", "GRADING_REQUEUE_AFTER": "10"}) == 0
    a5 = mk(1, "[[S9G slow=25]]", prefix="Q705"); call("POST", f"{Q}/assignments/{a5}/grade-all", T, {}); leases = []; t0 = time.time()
    while time.time() - t0 < 60:
        v = psql(f"select gr.run_state||'|'||coalesce(to_char(gr.lease_until,'SS.MS'),'-')||'|'||gr.attempts from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a5}'"); leases.append((round(time.time() - t0, 1), v))
        if v.startswith("DONE"): break
        time.sleep(1.5)
    for x in leases[::2]: print("   ", x)
    distinct = len({v.split("|")[1] for _, v in leases if v.startswith("RUNNING")}); fin = leases[-1][1]
    req5 = len([r for r in reqs(t0 - 1) if "Bài làm của sv.gioi" in json.dumps(r["body"], ensure_ascii=False) and "slow=25" in json.dumps(r["body"], ensure_ascii=False)])
    R.chk("TC-P7-07-05", f"bài chậm 25 s với thuê 10 s: lease_until được gia hạn ({distinct} giá trị khác nhau), kết thúc {fin} (attempts=1, không nhận lại oan)", distinct >= 3 and fin.startswith("DONE") and fin.endswith("|1"), leases[-3:])
    R.chk("TC-P7-07-05", f"chỉ một lần chạy: {req5} lời gọi stub (2 lượt)", req5 == 2, req5)
    # 07-09: 3 bài tập × 36 = 108 bài, slow=1
    print("   worker mặc định GRADING_WORKERS=2"); assert worker() == 0
    asgs = [mk(36, "[[S9G slow=1]]", users=SV[:36], prefix=f"Q709{i}") for i in range(3)]
    mx = 0; samples = []; stop = False
    def sampler():
        global mx
        while not stop:
            n = int(psql("select count(*) from grading_results where run_state='RUNNING'")); mx = max(mx, n); samples.append(n); time.sleep(0.2)
    th = threading.Thread(target=sampler); th.start(); t0 = time.time()
    for a_ in asgs: call("POST", f"{Q}/assignments/{a_}/grade-all", T, {})
    for _ in range(120):
        if sum(pending(a_) for a_ in asgs) == 0: break
        time.sleep(3)
    stop = True; th.join(); dur = time.time() - t0
    tot = sum(1 for a_ in asgs for x in states(a_) if x.startswith("DONE")); print(f"   108 bài: {tot} DONE, {dur:.0f}s, RUNNING đồng thời tối đa {mx} ({len(samples)} mẫu)")
    R.chk("TC-P7-07-09", f"108 bài xong ({tot} DONE); RUNNING đồng thời tối đa {mx} ≤ GRADING_WORKERS×1 worker = 2", tot == 108 and mx <= 2 and mx >= 1, (tot, mx))
    ts = sorted(r["t"] for r in reqs(t0 - 1) if "slow=1" in json.dumps(r["body"], ensure_ascii=False) and "Q709" not in ""); 
    pairs = {}
    for r in reqs(t0 - 1):
        k = json.dumps(r["body"], ensure_ascii=False)[:0]
    byuser = {}
    for r in reqs(t0 - 1):
        m = re.search(r"Bài làm của (sv[.\w]*)\.", json.dumps(r["body"], ensure_ascii=False)); 
        if m: byuser.setdefault((m.group(1), r["body"]["messages"][-1]["content"][:80] if False else m.group(1)), []).append(r["t"])
    par = sum(1 for v in byuser.values() if len(v) >= 2 and max(v) - min(v) < 0.8); print("   hai lượt chạy song song (cách nhau <0,8 s):", par, "/", len(byuser))
    R.chk("TC-P7-07-09", f"hai lượt (pass1/pass2) của một bài gọi gần như đồng thời: {par} cặp song song", par >= 20, (par, len(byuser)))
    g = subprocess.run("grep -n 'AttemptTimeout' internal/grading/essay.go cmd/worker/gradingq.go; grep -n 'gradingTask\\|RegisterBatch' cmd/worker/*.go | head", shell=True, capture_output=True, text=True, cwd=os.path.expanduser("~/Documents/qc9s-mut/w3/backend-go")).stdout; print(g)
    R.chk("TC-P7-07-09", "GRADING_ATTEMPT_TIMEOUT 120 s đi xuống lời gọi (context.WithTimeout trong passes) và consumer ep:grading là task riêng (gradingTask) không nằm trong outbox consumer", "context.WithTimeout(ctx, g.Cfg.AttemptTimeout)" in g and "gradingTask" in g and "AttemptTimeout = envSec" in g.replace("ec.AttemptTimeout = envSec", "AttemptTimeout = envSec"), g)
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
worker()  # trả worker về cấu hình mặc định (đã mặc định)
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
