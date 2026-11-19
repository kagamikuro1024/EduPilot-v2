import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def run():
    flag("GRADING_GUIDANCE", True)
    # ===== G03-10: > 60 ca → đúng 60 ca mới nhất =====
    a = fxa("g0310", 21)
    rs = sql(f"select r.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' order by r.id")
    teacher = sql("select id from users where email='teacher@edupilot.local'")[0][0]
    V = ["0.00", "0.25", "0.50", "1.50", "1.75", "2.00", "2.25", "2.50"]
    def tv(i): return {f"c{k+1}": V[(i >> (3 * k)) & 7] for k in range(4)}
    for i, (rid,) in enumerate(rs, 1):
        tvv = tv(i); expr = "jsonb_build_array(" + ",".join(f"jsonb_set(jsonb_set((criteria->{k}),'{{score}}','\"{tvv['c'+str(k+1)]}\"'),'{{edited}}','true')" for k in range(4)) + ")"
        sql(f"""update grading_results set status='REVIEWED', reviewed_by='{teacher}', first_reviewed_by='{teacher}', reviewed_at=now() - interval '1 day' + {i} * interval '1 minute', first_reviewed_at=now() - interval '1 day' + {i} * interval '1 minute',
 criteria={expr} where id='{rid}'""")
    c, ov, _ = api("teacher", "GET", G(a)); print("overview", lastj(ov.get("reflect"), 100))
    cR, jR, job = reflect(a, "rf_ok"); d = json.load(open("/tmp/stub_reflect_in.json")); cs = d["cases"]
    from collections import Counter
    got = Counter((x["criterion_id"], x["teacher_score"]) for x in cs); exp_new = Counter((f"c{k+1}", tv(i)[f"c{k+1}"]) for i in range(7, 22) for k in range(4)); exp_old = Counter((f"c{k+1}", tv(i)[f"c{k+1}"]) for i in range(1, 16) for k in range(4))
    labels = [f"{len(got)} cặp"]
    chk("TC-G03-10", len(cs) == 60 and got == exp_new and exp_new != exp_old and cR == 202, f"21 bài duyệt (mỗi bài 4 tiêu chí lệch = 84 ca, reviewed_at tăng dần R01→R21); đầu vào reflect gửi tới stub: {len(cs)} ca, khớp đúng tập ca của 15 bài mới nhất R07–R21 (mã hoá điểm GV theo chỉ số bài; khác tập của 15 bài cũ nhất) = {got == exp_new}; job {job and job[0]}; overview reflect={lastj(ov.get('reflect'), 80)}")
with window(): run()
