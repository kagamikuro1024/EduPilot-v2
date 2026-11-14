# L2: cổng THREADS_PRECHECK (US-DM-03). Dùng: python dm_b_threads.py <ca>  (ca = off|p90|p01|p20|unk|err|fc|rules|precheck|crisis|roles|comment)
import sys
from dmx import *
C1 = course("sv.kha", "761987")
SV2 = None
r = sql("select u.email from users u join enrollments e on e.user_id=u.id join courses c on c.id=e.course_id where c.class_code='761988' and u.email like 'sv%' and u.id not in (select e2.user_id from enrollments e2 join courses c2 on c2.id=e2.course_id where c2.class_code='761987') limit 1")
if r: SV2 = r[0][0].split("@")[0]
print("C1", C1, "SV lớp 2:", SV2)
SUBTLE = ("Có bạn hoàn cảnh khó khăn bị trừ điểm chuyên cần", "Mình nghe nói có một bạn trong lớp hoàn cảnh khó khăn nên bị trừ điểm chuyên cần, không biết bạn đó có bị ảnh hưởng điểm cuối kỳ không?")
CLEAN = ("Hỏi về công thức tính điểm", "Công thức tính điểm quá trình của môn này gồm những thành phần nào và trọng số ra sao vậy ạ?")

def post(user, title, body, course=C1, **extra):
    c, j, ms = api(user, "POST", f"/courses/{course}/threads", dict({"title": title, "body": body}, **extra))
    return c, j, ms

def summarize(tag, c, j, ms, since):
    d = decide_rows(since); cl = llm_rows(since, "CLASSIFY")
    code = (j or {}).get("code") or (j or {}).get("error", {}).get("code") if isinstance(j, dict) else None
    print(f"{tag}: HTTP {c} code={code} {ms:.0f}ms | DECIDE={d} | CLASSIFY={[(x[1]) for x in cl]}")
    return d, cl

case = sys.argv[1]
if case == "off":
    gates_off(); s = now_db()
    c, j, ms = post("sv.kha", *CLEAN); d, _ = summarize("off clean", c, j, ms, s)
    ok("D02-06/D03-off cổng tắt: đăng được, 0 DECIDE", c in (200, 201) and not d, f"{c} {d}")
    c, j, ms = post("sv.kha", *SUBTLE); d, _ = summarize("off subtle", c, j, ms, s)
    ok("D03-off bài riêng tư tinh vi cổng tắt: như cũ, 0 DECIDE", not d, f"{c} {d}")
elif case in ("p90", "p01", "p20", "unk", "err"):
    s = now_db(); set_gate("THREADS_PRECHECK")
    for tag, t in (("subtle", SUBTLE), ("clean", CLEAN)):
        s = now_db(); c, j, ms = post("sv.kha", *t, **{}); d, cl = summarize(f"{case} {tag}", c, j, ms, s)
        print("   body:", json.dumps(j, ensure_ascii=False)[:200])
elif case == "rules":
    set_gate("THREADS_PRECHECK"); s = now_db()
    c, j, ms = post("sv.kha", "Hỏi bài", "MSSV của mình là 20201234 và mình muốn hỏi điểm của mình."); d, _ = summarize("rules MSSV", c, j, ms, s)
    ok("D03-01 luật chặn → 0 DECIDE", c == 422 and not d, f"{c} {d}")
    s = now_db(); c, j, ms = post("sv.kha", "Hỏi bài", "[[MSSV_1]]"); d, _ = summarize("placeholder only", c, j, ms, s)
    ok("D03-09 chỉ placeholder → chặn bởi luật, 0 DECIDE", c == 422 and not d, f"{c} {d}")
elif case == "precheck":
    set_gate("THREADS_PRECHECK"); s = now_db()
    for t in (SUBTLE, CLEAN, ("x", "Email mình là abc@gmail.com")):
        c, j, _ = api("sv.kha", "POST", f"/courses/{C1}/threads/precheck", {"title": t[0], "body": t[1]}); print("precheck", c, json.dumps(j, ensure_ascii=False)[:120])
    ok("D03-10 precheck (lúc gõ): 0 DECIDE", not decide_rows(s), str(decide_rows(s)))
elif case == "crisis":
    set_gate("THREADS_PRECHECK"); s = now_db()
    c, j, ms = post("sv.kha", "Mình không ổn", "Mình muốn chết, không còn muốn sống nữa."); d, cl = summarize("crisis", c, j, ms, s)
    ok("D03-14 khủng hoảng → 422 CRISIS_SUPPORT, 0 DECIDE, 0 CLASSIFY", c == 422 and "CRISIS" in json.dumps(j) and not d and not cl, f"{c} {d} {cl}")
elif case == "roles":
    set_gate("THREADS_PRECHECK")
    for u in ("teacher", "ta"):
        s = now_db(); c, j, ms = post(u, *SUBTLE); summarize(f"role {u}", c, j, ms, s)
    if SV2:
        s = now_db(); c, j, ms = api(SV2, "POST", f"/courses/{course(SV2, '761988')}/threads", {"title": SUBTLE[0], "body": SUBTLE[1]}); d = decide_rows(s)
        print("lớp 2:", c, d); ok("D03-17 lớp 2 (cổng chỉ lớp 1) không DECIDE", not d, f"{c} {d}")
elif case == "comment":
    set_gate("THREADS_PRECHECK")
    c, j, _ = post("sv.gioi", *CLEAN); tid = (j.get("thread") or j).get("id"); print("thread", c, tid)
    s = now_db(); c, j, ms = api("sv.kha", "POST", f"/courses/{C1}/threads/{tid}/comments", {"body": SUBTLE[1]}); d, _ = summarize("comment subtle", c, j, ms, s)
