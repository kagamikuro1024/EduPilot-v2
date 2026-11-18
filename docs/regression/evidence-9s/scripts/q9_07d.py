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
    a = mk(12, "[[S9G slow=8]]", prefix="Q711R"); call("POST", f"{Q}/assignments/{a}/grade-all", T, {}); time.sleep(4)
    st0 = sorted(states(a)); print("   trước khi dừng Redis:", st0)
    subprocess.run(["docker", "stop", "ep-qc-redis-1"], capture_output=True); t0 = time.time(); time.sleep(70); subprocess.run(["docker", "start", "ep-qc-redis-1"], capture_output=True)
    for _ in range(30):
        if "PONG" in redis("ping"): break
        time.sleep(2)
    print("   Redis dừng", round(time.time() - t0), "s rồi bật;", sorted(states(a)))
    wait_done(a, 360); time.sleep(3); st = states(a); print("   cuối:", sorted(set(st)), len(st))
    R.chk("TC-P7-07-11", f"Redis dừng 70 s giữa lúc chấm 12 bài rồi bật: không bài nào mất, {len(st)} kết quả, trạng thái {sorted(set(st))}", len(st) == 12 and all(x.startswith("DONE") for x in st), st)
    dup = psql(f"select count(*) from (select submission_id from grading_results gr join submissions s on s.id=gr.submission_id where s.assignment_id='{a}' group by submission_id having count(*)>1) x"); R.chk("TC-P7-07-11", f"mỗi bài đúng một kết quả sau khi Redis hồi phục (trùng {dup})", dup == "0", dup)
sys.exit(R.end())
