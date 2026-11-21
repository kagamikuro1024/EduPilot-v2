import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
def run():
    a1, _, _ = guided("g0512", 14, 6, 2)   # 8 nháp stale (G05-12)
    a2, _, _ = guided("g0517", 14, 6, 2)   # + 2 bài duyệt nguyên nháp (G05-17)
    flag("GRADING_GUIDANCE", True)
    rs = [r for r in done_results(a2) ][:0]
    dr = [r for r in results(a2) if r[2] == "DRAFT" and r[5] == "DONE"]
    for r in dr[:2]:
        c, j, _ = api("teacher", "GET", f"/courses/{CG0}/grading-results/{r[0]}"); api("teacher", "POST", f"/courses/{CG0}/grading-results/{r[0]}/review", {"version": j["result"]["version"]})
    print("a2 reviewed", sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a2}' and r.status='REVIEWED'")[0][0], "assignment v", sql(f"select guidance_version from assignments where id='{a2}'")[0][0])
    json.dump({"a1": a1, "a2": a2}, open(W + "g05fx.json", "w"))
with window(): run()
