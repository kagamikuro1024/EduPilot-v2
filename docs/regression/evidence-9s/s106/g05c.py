import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def regrade(a): return api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/regrade-stale", {})
def run():
    flag("GRADING_GUIDANCE", True)
    sql(f"""with src as (select password_hash from users where email='sv.kha@edupilot.local'),
ins as (insert into users (email, password_hash, full_name, role, student_code, email_verified_at, status)
 select format('qcgd%s@edupilot.local', lpad(g::text,3,'0')), (select password_hash from src), format('QC GD %s', g), 'STUDENT', format('QCG%s', lpad(g::text,5,'0')), now(), 'ACTIVE' from generate_series(1,220) g on conflict (email) do nothing returning id, student_code)
insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) select '{CG0}', id, 'STUDENT', 'ACTIVE', 'ROSTER', student_code from ins;""")
    print("SV QD9G:", sql(f"select count(*) from enrollments where course_id='{CG0}' and role_in_course='STUDENT' and status='ACTIVE'")[0][0])
    tpl = fxa("g0509tpl", 2); trid = done_results(tpl)[0][0]; new = str(uuid.uuid4())
    GC = ", ".join(r[0] for r in sql("select column_name from information_schema.columns where table_name='grading_results' and is_generated='NEVER' order by ordinal_position"))
    sql(f"""create temp table ta as select * from assignments where id='{tpl}';
update ta set id='{new}', title='QC106 g0509', submission_code='QC'||upper(substr(md5(random()::text),1,8)), guidance_version=1;
insert into assignments select * from ta;
create temp table ns as select s.*, e.user_id as nsid from (select * from submissions where assignment_id='{tpl}' limit 1) s cross join (select user_id from enrollments where course_id='{CG0}' and role_in_course='STUDENT' and status='ACTIVE' limit 250) e;
update ns set id=uuidv7(), assignment_id='{new}', student_id=nsid, candidate_student_id=null, sha256=md5(random()::text)||md5(random()::text), blob_key='qc106/'||md5(random()::text), external_ref=null, version=1;
alter table ns drop column nsid; insert into submissions select * from ns;
create temp table tr as select r.*, s2.id as nsub from (select * from grading_results where id='{trid}') r cross join (select id from submissions where assignment_id='{new}') s2;
update tr set id=uuidv7(), submission_id=nsub, round=1, guidance_version=0; insert into grading_results ({GC}) select {GC} from tr;""")
    n = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{new}'")[0][0]; print("nạp", n, "kết quả stale")
    c1, j1, _ = regrade(new); print("lần 1", c1, lastj(j1, 200)); 
    for _ in range(200):
        q = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{new}' and r.run_state in ('QUEUED','RUNNING')")[0][0]
        if q == "0": break
        time.sleep(3)
    r2a = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{new}' and r.round=2")[0][0]
    c2, j2, _ = regrade(new); print("lần 2", c2, lastj(j2, 200))
    for _ in range(100):
        q = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{new}' and r.run_state in ('QUEUED','RUNNING')")[0][0]
        if q == "0": break
        time.sleep(3)
    r2b = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{new}' and r.round=2")[0][0]
    st = sql(f"select r.run_state, count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{new}' and r.round=2 group by 1")
    chk("TC-G05-09", n == "250" and j1.get("queued") == 200 and j2.get("queued") == 50 and r2a == "200" and r2b == "250", f"{n} nháp stale chưa sửa (nạp SQL, 250 SV): regrade-stale lần 1 → queued {j1.get('queued')} (round 2 trong DB {r2a}); lần 2 → queued {j2.get('queued')} (tổng round 2 = {r2b}); trạng thái round 2: {st}")
with window(): run()
