import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def addsub(a):
    sql(f"""create temp table tn as select * from submissions where assignment_id='{a}' order by created_at limit 1;
update tn set id=uuidv7(), sha256=md5(random()::text)||md5(random()::text), blob_key='qc106/'||md5(random()::text), external_ref=null, student_id=(select user_id from enrollments e where e.course_id=tn.course_id and e.role_in_course='STUDENT' and e.status='ACTIVE' and not exists (select 1 from submissions x where x.assignment_id=tn.assignment_id and x.student_id=e.user_id) limit 1);
insert into submissions select * from tn;""")
def run():
    # ===== G02-08: kill worker giữa chừng =====
    a, ids, info = guided("g0208", 12, 8, 2); flag("GRADING_GUIDANCE", True); addsub(a)
    [decide(a, i) for i in [x["id"] for x in items_of(a, "DRAFT")][:2]]
    stub("slow"); c, j, _ = api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/grade-all", {})
    for _ in range(40):
        if sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.run_state='RUNNING'")[0][0] == "1": break
        time.sleep(0.5)
    time.sleep(2)
    pre = sql(f"select r.id, r.run_state, r.attempts, r.guidance_version from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' order by r.created_at desc limit 1")[0]
    os.system("docker kill ep-qc-worker-1 >/dev/null 2>&1"); stub("allow"); cA, jA, _ = apply_v(a, 1); print("apply khi worker chết", cA); os.system("docker start ep-qc-worker-1 >/dev/null 2>&1")
    for _ in range(170):
        r = sql(f"select r.run_state from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' order by r.created_at desc limit 1")[0][0]
        if r == "DONE": break
        time.sleep(2)
    post = sql(f"select r.run_state, r.attempts, r.guidance_version, r.round from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' order by r.created_at desc limit 1")[0]
    dup = sql(f"select count(*) from (select r.submission_id, r.round from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' group by 1,2 having count(*)>1) x")[0][0]
    gva = sql(f"select guidance_version from assignments where id='{a}'")[0][0]
    chk("TC-G02-08", post[0] == "DONE" and dup == "0" and post[2] == gva and int(post[1]) >= 2, f"lượt chấm bài mới đang chạy ({pre[1]}, attempts {pre[2]}, ghim v{pre[3]}) → docker kill worker (RUNNING) rồi start + apply v1→v2 lúc worker chết ({cA}); sau khi chạy lại: run_state={post[0]} attempts={post[1]} guidance_version ghim={post[2]} (assignments={gva}); cặp (submission, round) trùng: {dup}")
with window(): run()
