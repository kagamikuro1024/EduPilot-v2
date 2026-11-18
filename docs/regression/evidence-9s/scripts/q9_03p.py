#!/usr/bin/env python3
"""TC-P7-03-02 pha giữa: chấm bằng stub, rồi (--publish) nhập điểm tay, duyệt, công bố bài của sv30."""
import sys, os, json; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
st = json.load(open("/tmp/q9s/uic-state.json")); U = json.load(open("/tmp/q9s/ui.json")); Q = f"/api/v1/courses/{U['course']}"; T = tok("teacher"); aid = st["id"]
if "--publish" not in sys.argv:
    with stub_ctx():
        s, g, _ = call("POST", f"{Q}/assignments/{aid}/grade-all", T, {}); print("grade-all", s); wait_done(aid, 120); time.sleep(2)
    r = result(aid, "sv30"); print("result", r["status"], r.get("run_state"), r.get("total"))
else:
    r = result(aid, "sv30"); rid = r["id"]; ver = lambda: int(psql(f"select version from grading_results where id='{rid}'"))
    crit = [{"id": c["id"], "score": "2.00"} for c in r["criteria"]]
    print("PUT", call("PUT", f"{Q}/grading-results/{rid}", T, {"version": ver(), "criteria": crit, "overall_feedback": "Nhận xét QC."})[0])
    print("review", call("POST", f"{Q}/grading-results/{rid}/review", T, {"version": ver()})[0])
    print("publish", call("POST", f"{Q}/grading-results/publish", T, {"result_ids": [rid]}, idem())[:2])
