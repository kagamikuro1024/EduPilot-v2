import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def regrade(a): return api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/regrade-stale", {})
def rows(a): return sql(f"select r.run_state, r.round, r.guidance_version, r.attempts, r.error_code from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' order by r.created_at")
def settle(a, t=150):
    for _ in range(t):
        if sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.run_state in ('QUEUED','RUNNING')")[0][0] == "0": return
        time.sleep(2)
def run():
    only = sys.argv[1:]; go = lambda k: not only or k in only
    if go("22"):
        a, ids, info = guided("g0522", 16, 6, 2); flag("GRADING_GUIDANCE", True)
        sub = sql(f"select s.id from submissions s join grading_results r on r.submission_id=s.id and r.course_id=s.course_id where s.assignment_id='{a}' and r.status='DRAFT' and r.run_state='DONE' order by s.id limit 1")[0][0]
        sql(f"set session_replication_role=replica; update submissions set extracted_text = extracted_text || ' [[QCERR]]' where id='{sub}'")
        n0 = len(rows(a)); stub("slow"); c, j, _ = regrade(a); print("regrade", c, lastj(j, 200))
        time.sleep(4); os.system("docker kill ep-qc-worker-1 >/dev/null 2>&1"); stub("allow"); time.sleep(2); os.system("docker start ep-qc-worker-1 >/dev/null 2>&1")
        settle(a, 200); r = rows(a)
        dup = sql(f"select count(*) from (select r.submission_id, r.round from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' group by 1,2 having count(*)>1) x")[0][0]
        r2 = [x for x in r if x[1] == "2"]; done = sum(x[0] == "DONE" for x in r2); failed = [x for x in r2 if x[0] == "FAILED"]
        chk("TC-G05-22", dup == "0" and len(r2) == j.get("queued") and len(failed) == 1 and done == len(r2) - 1, f"regrade {j.get('queued')} bài (stub chậm 12 s, docker kill worker lúc đang chạy rồi start); 1 bài chứa [[QCERR]] (stub trả JSON hỏng riêng bài đó, không tính vào mạch hở); kết quả round 2: {len(r2)} dòng, DONE {done}, FAILED {[(x[0], x[4], x[3]) for x in failed]}; cặp (submission, round) trùng = {dup}")
    if go("23"):
        a, ids, info = guided("g0523", 14, 6, 2); flag("GRADING_GUIDANCE", True)
        stub("slow"); c, j, _ = regrade(a); print("regrade", c, lastj(j, 150)); time.sleep(3); print("tắt cờ", flag("GRADING_GUIDANCE", False)[0]); stub("allow")
        settle(a, 120); r = [x for x in rows(a) if x[1] == "2"]
        st = sql(f"select count(*) from jobs where status='FAILED' and updated_at > now() - interval '5 minutes' and kind ilike '%regrade%'")[0][0]
        gv = sorted({x[2] for x in r}); 
        chk("TC-G05-23", len(r) == j.get("queued") and all(x[0] == "DONE" for x in r) and "0" in gv, f"regrade {j.get('queued')} bài (stub chậm), tắt cờ GRADING_GUIDANCE sau 3 s: kết quả round 2 {len(r)} dòng, run_state {sorted({x[0] for x in r})}, guidance_version ghim {gv} (các bài nhận việc sau khi tắt cờ ghim 0, không crash; job regrade FAILED = {st})")
        flag("GRADING_GUIDANCE", True)
with window(): run()
