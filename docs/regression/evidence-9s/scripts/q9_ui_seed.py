#!/usr/bin/env python3
"""Dựng dữ liệu UI cho Playwright: lớp QD9V; BT03 (22 SV theo seed-plan) đã chấm nháp; BT03b công bố cho B (sv.kha) để xem phía sinh viên. Ghi /tmp/q9s/ui.json."""
import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
ALL = ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 41)]
C = mkcourse("QD9V", ALL); Q = f"/api/v1/courses/{C}"; T = tok("teacher")
ver = lambda rid: int(psql(f"select version from grading_results where id='{rid}'"))
with stub_ctx():
    S = seed_bt03(Q, T, "QUI" + str(int(time.time()) % 100000), None); aid = S["aid"]
    # bài công bố cho B + A: dùng bài riêng
    S2 = seed_bt03(Q, T, "QUP" + str(int(time.time()) % 100000), ["sv.gioi", "sv.kha"]); aid2 = S2["aid"]
    for acc, r in S2["res"].items():
        rr = result(aid2, acc); ai = rr["ai_scores"]; sc = [ai["c1"], "2.00" if acc == "sv.kha" else ai["c2"], ai["c3"], ai["c4"]]
        call("PUT", f"{Q}/grading-results/{r}", T, {"version": ver(r), "criteria": [{"id": f"c{i+1}", "score": s} for i, s in enumerate(sc)], "overall_feedback": "Nhận xét giảng viên cho bài làm."}); call("POST", f"{Q}/grading-results/{r}/review", T, {"version": ver(r)})
    print(call("POST", f"{Q}/grading-results/publish", T, {"result_ids": list(S2["res"].values())})[:2])
json.dump({"course": C, "aid": aid, "res": S["res"], "sub": S["sub"], "aid2": aid2, "res2": S2["res"], "sub2": S2["sub"]}, open(f"{TMP}/ui.json", "w")); print("ok", C, aid, aid2)
