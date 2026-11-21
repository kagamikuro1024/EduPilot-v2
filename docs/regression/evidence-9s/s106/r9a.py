import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
RM9 = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "results_9rerun.jsonl")
def res9(tc, kq, note): print(kq, tc, note[:200]); open(RM9, "a", encoding="utf8").write(json.dumps({"tc": tc, "kq": kq, "note": note}, ensure_ascii=False) + "\n")
def run():
    a = mk_assignment("r9s04", 6); stub("allow")
    n0 = sql(f"select count(*) from jobs where created_at > now() - interval '1 hour' and kind ilike '%grad%'")[0][0]
    os.system("docker stop ep-qc-redis-1 >/dev/null 2>&1"); time.sleep(3)
    cc, jj, _, hd = raw("POST", f"/courses/{CG0}/assignments/{a}/grade-all", {}, {"Idempotency-Key": str(uuid.uuid4())}, tok("teacher"))
    rows = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}'")[0][0]
    n1 = sql(f"select count(*) from jobs where created_at > now() - interval '1 hour' and kind ilike '%grad%'")[0][0]
    ra = [f"{k}: {v}" for k, v in dict(hd).items() if k.lower() == "retry-after"]
    os.system("docker start ep-qc-redis-1 >/dev/null 2>&1")
    for _ in range(40):
        if subprocess.run(["docker", "exec", "ep-qc-redis-1", "redis-cli", "ping"], capture_output=True, text=True).stdout.startswith("PONG"): break
        time.sleep(1)
    time.sleep(8)
    c2, j2, _ = api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/grade-all", {})
    ok = cc == 503 and "SERVICE_UNAVAILABLE" in json.dumps(jj) and rows == "0" and n1 == n0 and bool(ra) and c2 == 202
    res9("TC-P7-07-01", "P" if ok else "F", f"Redis dừng: grade-all → {cc} {lastj(jj, 80)}; Retry-After: {ra}; grading_results của bài={rows}, jobs mới={int(n1)-int(n0)}; Redis hồi phục + bấm lại → {c2} {lastj(j2, 60)} (P-11-70: 503 fail-closed)")
    for _ in range(90):
        r = sql(f"select count(*) filter (where r.run_state='DONE'), count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}'")[0]
        if r[0] == "6": break
        time.sleep(2)
    print("sau chấm:", r)
with window(): run()
