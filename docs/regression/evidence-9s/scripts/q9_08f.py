#!/usr/bin/env python3
"""TC-P7-08-11 chuẩn bị: bài mới trong QD9V, sv33 (nộp lại sau) + sv34 (đặt RUNNING) đã nộp và được stub chấm."""
import sys, os, json; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
U = json.load(open("/tmp/q9s/ui.json")); Q = f"/api/v1/courses/{U['course']}"; T = tok("teacher")
with stub_ctx():
    aid = mkassign(Q, T, "Q8F" + str(int(time.time()) % 100000), due_days=3)
    for u in ("sv33", "sv34"):
        s, x, _ = submit(Q, aid, tok(u), "bai.md", ("Bài làm của " + u + ". SUNBURST. " * 12).encode()); assert s in (200, 201), (s, x)
    s, g, _ = call("POST", f"{Q}/assignments/{aid}/grade-all", T, {}); assert s == 202; wait_done(aid, 120); time.sleep(2)
S = {u: sub_id(aid, u) for u in ("sv33", "sv34")}; R = {u: result(aid, u)["id"] for u in S}
psql(f"update grading_results set run_state='RUNNING' where id='{R['sv34']}'")
json.dump({"aid": aid, "sub": S, "res": R}, open("/tmp/q9s/ui08f.json", "w")); print(aid, S, R)
