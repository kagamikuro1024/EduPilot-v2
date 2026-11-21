import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
S = json.load(open("/tmp/q9s/uic-state.json")); U = json.load(open("/tmp/q9s/ui.json")); C = U["course"]; A = S["id"]
def run():
    stub("allow"); c, j, _ = api("teacher", "POST", f"/courses/{C}/assignments/{A}/grade-all", {}); print("grade-all", c, lastj(j, 80))
    # trong lúc chạy chậm: chụp trạng thái «Đang chấm»
    for _ in range(60):
        r = sql(f"select r.run_state, r.status from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{A}'")
        if r and r[0][0] == "DONE": break
        time.sleep(1)
    print("kết quả", r)
    os.system("cd ~/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106 && PHASE=2 ./runpw.sh qc-real-9s-uic.spec.ts UIC 150 | head -1")
    print("S3", json.load(open("/tmp/q9s/uic-state.json")).get("s3"))
    rid = sql(f"select r.id, r.version from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{A}'")[0]
    c, j, _ = api("teacher", "POST", f"/courses/{C}/grading-results/{rid[0]}/review", {"version": int(rid[1])}); print("review", c, lastj(j, 80))
    c, j, _ = api("teacher", "POST", f"/courses/{C}/assignments/{A}/publish-results", {}) ; print("publish-results", c, lastj(j, 100))
with window(): run()
