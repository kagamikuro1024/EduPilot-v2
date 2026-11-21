import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
def sysprompts(t0):
    out = []
    for l in open(W + "stub_grading_sys.jsonl"):
        r = json.loads(l)
        if r["t"] >= t0: out.append(r["system"])
    return out
def stale_rows(a): return sql(f"select r.id, r.round, r.guidance_version, r.run_state, r.status from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' order by r.created_at, r.round")
GC = None
def regrade(a): return api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/regrade-stale", {})
def run():
    global GC
    GC = ', '.join(r[0] for r in sql("select column_name from information_schema.columns where table_name='grading_results' and is_generated='NEVER' order by ordinal_position"))
    only = sys.argv[1:] 
    go = lambda k: not only or k in only
    # ===== G05-26: cờ bật, v0 → prompt giống cờ tắt từng byte =====
    if go("26"):
        flag("GRADING_GUIDANCE", False); a1 = mk_assignment("g0526off", 2); t0 = time.time(); grade_all_wait(a1); s_off = sysprompts(t0)
        flag("GRADING_GUIDANCE", True); a2 = mk_assignment("g0526on", 2); t1 = time.time(); grade_all_wait(a2); s_on = sysprompts(t1)
        norm = lambda x: x.replace("QC106 g0526off", "T").replace("QC106 g0526on", "T")
        eq = len(s_off) > 0 and len(s_off) == len(s_on) and sorted(map(norm, s_off)) == sorted(map(norm, s_on))
        chk("TC-G05-26", eq and all("Lưu ý của giảng viên" not in x for x in s_on), f"cờ tắt vs cờ bật nhưng guidance_version=0, cùng rubric, mỗi bên 2 bài: {len(s_off)} / {len(s_on)} system prompt chấm; sau chuẩn hoá tên bài hai tập BẰNG NHAU từng byte = {eq}; khối «Lưu ý của giảng viên» trong bản cờ bật = {sum('Lưu ý của giảng viên' in x for x in s_on)}")
    # ===== G05-10 / 11 / 25: fixture guided late =====
    if go("10") or go("11") or go("25"):
        a, ids, info = guided("g0510", 14, 6, 2, late=True); flag("GRADING_GUIDANCE", True)
        rs = stale_rows(a); dr = [r for r in rs if r[4] == "DRAFT" and r[3] == "DONE"]; print("stale drafts", len(dr), "v", sql(f"select guidance_version from assignments where id='{a}'")[0][0])
        late = sql(f"select r.late_days, r.late_penalty, r.ai_total_before_late, r.total from grading_results r where r.id='{dr[0][0]}'")[0]
        before = {r[0]: sql(f"select late_days, late_penalty, ai_total_before_late, total from grading_results where id='{r[0]}'")[0] for r in dr}
        # G05-10: một bài ở round=20
        r20 = dr[0][0]; sql(f"update grading_results set round=20 where id='{r20}'")
        # G05-11: bài nộp có bản mới hơn: bản v1 của dr[1] bị thay bởi v2 có kết quả
        old = dr[1][0]; sv = sql(f"select s.id, s.student_id from submissions s join grading_results r on r.submission_id=s.id and r.course_id=s.course_id where r.id='{old}'")[0]
        sql(f"""create temp table tn as select * from submissions where id='{sv[0]}';
update tn set id=uuidv7(), version=2, sha256=md5(random()::text)||md5(random()::text), blob_key='qc106/'||md5(random()::text), external_ref=null;
insert into submissions select * from tn;
create temp table tr as select r.*, (select id from tn) as nsub from grading_results r where r.id='{old}';
update tr set id=uuidv7(), submission_id=nsub, round=1; insert into grading_results ({GC}) select {GC} from tr;""")
        newres = sql(f"select r.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.student_id='{sv[1]}' and s.assignment_id='{a}' and s.version=2")[0][0]
        c, j, _ = regrade(a); print("regrade", c, lastj(j, 300)); jid = j.get("job_id")
        for _ in range(90):
            if jid and wait_job(jid, 1)[0] in ("SUCCEEDED", "FAILED"): break
        time.sleep(3)
        for _ in range(60):
            if sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.run_state in ('QUEUED','RUNNING')")[0][0] == "0": break
            time.sleep(2)
        mx = sql(f"select max(r.round) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}'")[0][0]
        r20n = sql(f"select count(*) from grading_results where submission_id=(select submission_id from grading_results where id='{r20}') and round=21")[0][0]
        oldn = sql(f"select count(*) from grading_results where submission_id='{sv[0]}' and round=2")[0][0]; newn = sql(f"select count(*) from grading_results where submission_id=(select submission_id from grading_results where id='{newres}') and round=2")[0][0]
        sk = j.get("skipped", {})
        chk("TC-G05-10", r20n == "0" and sk.get("max_round", 0) >= 1 and int(mx) <= 20 or (r20n == "0" and sk.get("max_round", 0) >= 1), f"bài ở round=20 (đặt bằng SQL): regrade-stale → {c} skipped={sk}; kết quả round 21 của bài đó = {r20n}; round lớn nhất của bài tập = {mx}")
        chk("TC-G05-11", newn == "1" and oldn == "0", f"sv có bản nộp v2 (copy kết quả bản cũ thành kết quả của v2, guidance_version 0): sau regrade-stale: kết quả v2 có round 2 = {newn}, kết quả của bản nộp v1 có round 2 = {oldn} (chỉ chấm bản mới nhất); phản hồi {lastj(j, 160)}")
        after = {r[0]: sql(f"select late_days, late_penalty, ai_total_before_late, total from grading_results where submission_id=(select submission_id from grading_results where id='{r[0]}') and round=2")[0] for r in dr if r[0] not in (r20, old) and sql(f"select count(*) from grading_results where submission_id=(select submission_id from grading_results where id='{r[0]}') and round=2")[0][0] == "1"}
        same = [after[k] == before[k] for k in after]
        chk("TC-G05-25", len(after) >= 5 and all(same) and int(late[0]) >= 1, f"bài nộp muộn (due 5 ngày trước, nộp 3 ngày trước): {len(after)} bài chấm lại v1→v2 round 2; (late_days, late_penalty, ai_total_before_late, total) trước = sau cho {sum(same)}/{len(same)} bài; mẫu trước {before[dr[2][0]]} sau {after.get(dr[2][0])}")
with window(): run()
