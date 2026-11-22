import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def rows(s): time.sleep(4); return sql(f"select task, lane, count(*) from llm_audit where created_at>='{s}' group by 1,2 order by 1,2")
def run():
    flag("GRADING_GUIDANCE", True); s0 = sql("select now()")[0][0]; a = fxa("g1014", 10); print("chấm 10 bài (2 lượt mỗi bài):", rows(s0), flush=True)
    review_with(a, [(i, {"c2": "0.00", "c3": "0.00"}) for i in range(7)]); s1 = sql("select now()")[0][0]
    c, j, jb = reflect(a, "rf_many"); stub("allow"); print("reflect:", c, "→ lời gọi", rows(s1), flush=True)
    ids = [x["id"] for x in items_of(a, "DRAFT")]
    for iid in ids[:2]: decide(a, iid)
    s2 = sql("select now()")[0][0]; v = apply_v(a, 0); st = sql(f"select count(*) from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{a}'")[0][0]; print("áp dụng v1:", v, "kết quả hiện có", st, flush=True)
    s3 = sql("select now()")[0][0]; rc = api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/regrade-stale", {}, idem=str(uuid.uuid4())); print("regrade-stale:", rc[0], str(rc[1])[:120], flush=True)
    time.sleep(25); print("chấm lại:", rows(s3), flush=True)
with window(): run()
