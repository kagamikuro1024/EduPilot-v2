import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
def run():
    a, ids, info = guided("g0606", 14, 6, 0, do_apply=False); flag("GRADING_GUIDANCE", True)
    its = items_of(a, "DRAFT"); by = {}
    for x in its: by.setdefault(x["criterion_id"], []).append(x)
    print({k: len(v) for k, v in by.items()})
    c2 = by.get("c2", [])[:3]
    for x in c2: decide(a, x["id"])
    print("apply", apply_v(a, 0)[0])
    c, j, _ = api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/regrade-stale", {}); print("regrade", c, lastj(j, 120))
    for _ in range(100):
        if sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.run_state in ('QUEUED','RUNNING')")[0][0] == "0": break
        time.sleep(2)
    v1 = sql(f"select r.id, s.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.guidance_version=1 and r.status='DRAFT' limit 1")[0]
    v0 = sql(f"select r.id, s.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.guidance_version=0 and r.status='REVIEWED' limit 1")[0]
    print("v1 sub", v1, "v0 sub", v0, "c2 applied", len(c2))
    sql(f"update assignments set due_at=now()+interval '40 days', created_at=now() where id='{a}'")
    json.dump({"a": a, "v1sub": v1[1], "v1res": v1[0], "v0sub": v0[1], "v0res": v0[0], "n_c2": len(c2)}, open(W + "g06fx.json", "w"))
with window(): run()
