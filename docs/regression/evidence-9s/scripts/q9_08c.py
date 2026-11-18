import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("08c"); code = lambda: str(int(time.time()) % 100000)
C = mkcourse("QD9U", ["sv.gioi", "sv.kha"] + [f"sv{n:02d}" for n in range(4, 41)]); Q = f"/api/v1/courses/{C}"; T = tok("teacher")
with stub_ctx():
    S = seed_bt03(Q, T, "Q8C" + code(), ["sv.kha", "sv14", "sv19"]); aid = S["aid"]
    # dọn hàng cần xem kỹ cũ của lớp để chỉ còn 3 bài này
    ver = lambda rid: int(psql(f"select version from grading_results where id='{rid}'"))
    for rid in psql(f"select gr.id from grading_results gr where gr.course_id='{C}' and gr.status='DRAFT' and gr.needs_review and gr.id not in ({','.join(repr(x) for x in S['res'].values())})").splitlines():
        call("POST", f"{Q}/grading-results/{rid}/review", T, {"version": ver(rid)})
    time.sleep(70)  # cho cache Hôm nay (TTL 60 s) tự nguội: trạng thái tĩnh
    k = lambda: [i for i in today("teacher", C)[1] if i["kind"] == "GRADING_REVIEW"]
    print("   trước:", [i["title"][:40] for i in k()])
    R.chk("TC-P7-08-14", f"3 bài có cờ → GRADING_REVIEW hiện ({[i['title'][:30] for i in k()]})", len(k()) == 1, k())
    lat = []
    ids = list(S["res"].values())
    for rid in ids:
        t0 = time.time(); call("POST", f"{Q}/grading-results/{rid}/review", T, {"version": ver(rid)})
    t0 = time.time()
    while k() and time.time() - t0 < 90: time.sleep(0.2)
    dt = time.time() - t0; print("   biến mất sau", round(dt, 2), "s kể từ lần duyệt cuối")
    R.chk("TC-P7-08-14", f"duyệt hết bài cần xem kỹ ở trạng thái tĩnh → GRADING_REVIEW biến mất sau {dt:.1f} s (≤ 5 s)", dt <= 5.0, dt)
sys.exit(R.end())
