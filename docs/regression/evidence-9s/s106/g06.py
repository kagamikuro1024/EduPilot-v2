import sys, os, statistics; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def stats(a): return api("teacher", "GET", G(a) + "/stats")
def run():
    only = sys.argv[1:]; go = lambda k: not only or k in only
    flag("GRADING_GUIDANCE", True)
    if go("02") or go("08"):
        # G06-02: n=4 → null
        a = fxa("g0602", 8); review_with(a, [(i, {"c2": "0.00"}) for i in range(4)])
        c, j, _ = stats(a); it = j.get("items", [])
        chk("TC-G06-02", c == 200 and len(it) == 1 and it[0]["n"] == 4 and it[0]["mae_criterion"] is None and it[0]["mae_total"] is None and it[0]["bias_total"] is None and it[0]["edited_rate"] is None, f"phiên bản v0 có n=4 bài duyệt: GET stats → {c} {lastj(j, 300)} (các chỉ số null khi n<5)")
        # G06-08: bài chấm lại v0 → v1; chỉ kết quả hiện hành
        a, ids, info = guided("g0608", 16, 6, 2); flag("GRADING_GUIDANCE", True)
        c, j, _ = api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/regrade-stale", {}); 
        for _ in range(100):
            if sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.run_state in ('QUEUED','RUNNING')")[0][0] == "0": break
            time.sleep(2)
        dr = [r for r in sql(f"select r.id, r.version from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.round=2 and r.status='DRAFT'")]
        for rid, v in dr[:5]: api("teacher", "POST", f"/courses/{CG0}/grading-results/{rid}/review", {"version": int(v)})
        c2, j2, _ = stats(a); items = j2.get("items", []); tot = sum(i["n"] for i in items)
        cur = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.status in ('REVIEWED','PUBLISHED') and r.round=(select max(x.round) from grading_results x where x.submission_id=r.submission_id)")[0][0]
        allr = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.status in ('REVIEWED','PUBLISHED')")[0][0]
        chk("TC-G06-08", tot == int(cur) and [i["version"] for i in items] == sorted(i["version"] for i in items) and {i["version"] for i in items} == {0, 1}, f"v0→v1, 16 bài: 6 duyệt (v0, round 1) + {len(dr[:5])} bài chấm lại round 2 rồi duyệt (v1): stats {[(i['version'], i['n']) for i in items]} tổng n={tot}; số kết quả HIỆN HÀNH REVIEWED/PUBLISHED trong DB={cur} (tất cả hàng REVIEWED/PUBLISHED={allr}); round cũ không tính")
    if go("09"):
        # nạp 1000 kết quả REVIEWED: thêm SV phụ tới ≥ 1000 SV trong QD9G
        sql(f"""with src as (select password_hash from users where email='sv.kha@edupilot.local'),
ins as (insert into users (email, password_hash, full_name, role, student_code, email_verified_at, status)
 select format('qcgd%s@edupilot.local', lpad(g::text,4,'0')), (select password_hash from src), format('QC GD %s', g), 'STUDENT', format('QCG%s', lpad(g::text,5,'0')), now(), 'ACTIVE' from generate_series(221,1000) g on conflict (email) do nothing returning id, student_code)
insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) select '{CG0}', id, 'STUDENT', 'ACTIVE', 'ROSTER', student_code from ins;""")
        tpl = fxa("g0609tpl", 6); review_with(tpl, [(i, {"c2": "0.00"}) for i in range(5)]); trid = [r for r in results(tpl) if r[2] == "REVIEWED"][0][0]; new = str(uuid.uuid4())
        GC = ", ".join(r[0] for r in sql("select column_name from information_schema.columns where table_name='grading_results' and is_generated='NEVER' order by ordinal_position"))
        sql(f"""create temp table ta as select * from assignments where id='{tpl}';
update ta set created_at=now(), id='{new}', title='QC106 g0609', submission_code='QC'||upper(substr(md5(random()::text),1,8));
insert into assignments select * from ta;
create temp table ns as select s.*, e.user_id as nsid from (select * from submissions where assignment_id='{tpl}' limit 1) s cross join (select user_id from enrollments where course_id='{CG0}' and role_in_course='STUDENT' and status='ACTIVE' limit 1000) e;
update ns set id=uuidv7(), assignment_id='{new}', student_id=nsid, candidate_student_id=null, sha256=md5(random()::text)||md5(random()::text), blob_key='qc106/'||md5(random()::text), external_ref=null, version=1;
alter table ns drop column nsid; insert into submissions select * from ns;
create temp table tr as select r.*, s2.id as nsub from (select * from grading_results where id='{trid}') r cross join (select id from submissions where assignment_id='{new}') s2;
update tr set id=uuidv7(), submission_id=nsub, round=1, guidance_version=(random()*2)::int; alter table tr drop column nsub; insert into grading_results ({GC}) select {GC} from tr;""")
        n = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{new}' and r.status='REVIEWED'")[0][0]
        sql("analyze grading_results; analyze submissions")
        ts = []
        for _ in range(30):
            t0 = time.time(); c, j, _ = stats(new); ts.append((time.time() - t0) * 1000)
        ts.sort(); p95 = ts[int(len(ts) * 0.95) - 1]; print("n", n, "p95", p95, "median", statistics.median(ts), c, lastj(j, 200))
        ex = subprocess.run(["docker", "exec", "ep-qc-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", f"explain (analyze, buffers) select r.guidance_version, r.criteria, r.ai_scores, r.ai_total_before_late, r.total, r.late_days, r.late_penalty from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id where r.course_id = '{CG0}' and s.assignment_id = '{new}'::uuid and r.status in ('REVIEWED', 'PUBLISHED') and r.ai_scores is not null and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id)"], capture_output=True, text=True).stdout
        seq = [l.strip() for l in ex.splitlines() if "Seq Scan" in l]; idx = [l.strip()[:110] for l in ex.splitlines() if "Index" in l][:4]; print("PLAN", seq, idx, ex.splitlines()[-1] if ex else "")
        open("/Users/kuro/.config/edupilot-team/qc3-work/g0609-explain.txt", "w").write(ex)
        chk("TC-G06-09", c == 200 and p95 <= 500, f"{n} kết quả REVIEWED (nạp SQL, 1000 SV): 30 lần GET stats: p95 {p95:.0f} ms, trung vị {statistics.median(ts):.0f} ms, max {max(ts):.0f} ms; EXPLAIN ANALYZE truy vấn gom GuidanceStatsRows: Seq Scan {seq or 'không'}; chỉ mục: {idx}; {ex.splitlines()[-1] if ex else ''}")
with window(): run()
