# Smoke S10 lượt 11-18 — lô A: dựng đề / rò đề thi / DTO / chặn truy cập (API+DB) trên lớp QD10A (không đụng 761987/761988)
from pm import *
from datetime import datetime, timezone, timedelta
T = "QC-r10-Y"
def exam_with(qids, status, title):
    now = datetime.now(timezone.utc)
    c, e, _ = api("teacher", "POST", f"{P}/exams", {"title": title, "opens_at": (now + timedelta(days=3)).isoformat(), "closes_at": (now + timedelta(days=3, hours=2)).isoformat(), "duration_minutes": 30}); assert c == 201, (c, e)
    c, j, _ = api("teacher", "PUT", f"{P}/exams/{e['id']}/items", {"items": [{"question_id": q, "points": "1"} for q in qids], "version": e["version"]}); assert c == 200, (c, j)
    if status == "SCHEDULED":
        sql(f"update exams set status='SCHEDULED' where id='{e['id']}'")  # opens_at = +3 ngày: worker không mở
    elif status != "DRAFT":
        pub = ", published_at=now()" if status == "PUBLISHED" else ", publish_hold=true"  # giữ CLOSED chưa công bố (worker tự công bố nếu không giữ)
        sql(f"update exams set status='{status}', opens_at=now()-interval '2 hours', closes_at=now()+interval '2 hours', duration_minutes=30{pub} where id='{e['id']}'")
    return e["id"]
def set_status(eid, status):
    pub = ", published_at=now(), publish_hold=false" if status == "PUBLISHED" else ", published_at=null, publish_hold=true"
    sql(f"update exams set status='{status}'{pub} where id='{eid}'")
def pool_n(u): return next(t["available"] for t in api(u, "GET", f"{P}/practice/overview")[1]["topics"] if t["topic"] == T)
def full(u):
    """đề chứa TOÀN BỘ câu đủ điều kiện của chủ đề T → tập stem"""
    finish_open(u); n = min(50, pool_n(u)); c, a, _ = start(u, {"mode": "TOPIC", "topic": T, "count": n}); assert c in (200, 201), (c, a)
    st = [it["question"]["stem"] for it in a["items"]]; submit(u, a["id"]); return set(st)
def drain(u):
    finish_open(u)
# ---- dữ liệu: 8 MCQ chủ đề T (X1..X6 + 2 tự do), mỗi câu đủ để dựng đề
have = [q for q in bank() if q["topic"] == T and q["review_status"] == "APPROVED"]
if len(have) < 20:
    free = mk_mcq(T, 14, "QCr10Yfree"); X = [q[0] for q in mk_mcq(T, 6, "QCr10Y")]
    xs = [q["stem"] for q in (detail(i) for i in X)]
else:
    raise SystemExit("đã dựng: xoá chủ đề QC-r10-X hoặc đặt tên mới")
# E1 DRAFT, E2 SCHEDULED, E3 OPEN, E4 CLOSED chưa công bố, E5 PUBLISHED, X6 vừa PUBLISHED vừa DRAFT (còn ở bài thi chưa công bố)
E = [exam_with([X[0]], "DRAFT", "QC S10 E1"), exam_with([X[1]], "SCHEDULED", "QC S10 E2"), exam_with([X[2]], "OPEN", "QC S10 E3"),
     exam_with([X[3]], "CLOSED", "QC S10 E4"), exam_with([X[4]], "PUBLISHED", "QC S10 E5"), exam_with([X[5]], "PUBLISHED", "QC S10 E6a"), exam_with([X[5]], "DRAFT", "QC S10 E6b")]
SV = ["sv.kha", "sv.gioi", "sv.nguyco", "sv.moi", "sv31", "sv32", "sv33", "sv34"]
seen = {i: 0 for i in range(6)}; leaks = []; nreq = 0
for rep in range(5):
    for u in SV:
        drain(u)
        mode = {"mode": "TOPIC", "topic": T, "count": 10} if rep % 2 == 0 else {"mode": "MOCK", "count": 20, "duration_minutes": 20}
        c, a, _ = start(u, mode); assert c in (200, 201), (c, a); nreq += 1
        for it in a["items"]:
            for i, x in enumerate(xs):
                if x == it["question"]["stem"]: seen[i] += 1
        # phản hồi sau khi trả lời
        for it in a["items"][:3]:
            c, r, _ = put(u, a["id"], it["id"], {"option_ids": [it["question"]["options"][0]["id"]]} if it["question"].get("options") else {"text": "x"})
            if rep == 0:
                rr = json.dumps(r, ensure_ascii=False)
                for i in range(4): 
                    if xs[i] in rr: leaks.append(("put", i))
        submit(u, a["id"])
fullset = full("sv34")
print("đề trọn ngân hàng: X1..X6 có mặt", [xs[i] in fullset for i in range(6)])
print("lần X xuất hiện", seen, "lần dựng", nreq)
chk("TC-05-07 X1,X2,X3,X4,X6 không xuất hiện trong %d đề (5 vòng × 8 SV) + đề trọn ngân hàng; X5 có (PUBLISHED)" % nreq, all(seen[i] == 0 for i in (0, 1, 2, 3, 5)) and not leaks and not any(xs[i] in fullset for i in (0, 1, 2, 3, 5)) and xs[4] in fullset, f"đếm theo X1..X6 = {[seen[i] for i in range(6)]}; rò phản hồi {leaks}")
chk("TC-10-02 ≥ 50 lượt TOPIC/MOCK chủ đề T (40 lượt): X1-X4,X6 0 lần ở đề/phản hồi; X5 {}".format(seen[4]), nreq >= 40 and all(seen[i] == 0 for i in (0, 1, 2, 3, 5)), f"{nreq} lượt")
# nhiều lượt thêm để ≥ 50
for rep in range(2):
    for u in SV[:2]:
        drain(u); c, a, _ = start(u, {"mode": "TOPIC", "topic": T, "count": 10}); nreq += 1
        for it in a["items"]:
            for i, x in enumerate(xs): seen[i] += x == it["question"]["stem"]
        submit(u, a["id"])
chk("TC-10-02 ≥ 50 lượt (tổng %d): X1,X2,X3,X4,X6 = 0 lần; X5 = %d" % (nreq, seen[4]), nreq >= 50 and all(seen[i] == 0 for i in (0, 1, 2, 3, 5)), f"{[seen[i] for i in range(6)]}")
# explain trên 6 câu: không nhắc X1..X4
ex_txt = ""
drain("sv.kha"); c, a, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10})
for it in a["items"][:4]:
    put("sv.kha", a["id"], it["id"], {"option_ids": [it["question"]["options"][0]["id"]]})
    c, r, _ = api("sv.kha", "POST", f"{P}/practice/attempts/{a['id']}/items/{it['id']}/explain", {})
    ex_txt += json.dumps(r, ensure_ascii=False)
submit("sv.kha", a["id"])
chk("TC-10-02 explain: không nhắc nội dung X1..X4,X6", not any(xs[i] in ex_txt for i in (0, 1, 2, 3, 5)), f"{len(ex_txt)} ký tự explain")
# ---- TC-05-09: E3 OPEN → CLOSED → PUBLISHED (đề trọn ngân hàng đủ điều kiện)
pres = {}
for st in ("OPEN", "CLOSED", "PUBLISHED"):
    set_status(E[2], st); pres[st] = xs[2] in full("sv33")
chk("TC-05-09 X3: vắng khi OPEN và CLOSED, hiện sau PUBLISHED (đề trọn ngân hàng)", not pres["OPEN"] and not pres["CLOSED"] and pres["PUBLISHED"], f"{pres}")
# ---- TC-10-17: PE OPEN, SV đang thi tạo lượt luyện cùng chủ đề
set_status(E[2], "OPEN")
c, at, _ = api("sv.kha", "POST", f"{P}/exams/{E[2]}/attempts", {}); print("exam attempt", c, str(at)[:100])
drain("sv.kha"); c, a, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10}); n3 = sum(it["question"]["stem"] == xs[2] for it in a["items"])
chk("TC-10-17 SV đang có lượt thi OPEN dựng luyện cùng chủ đề: không có câu của bài thi OPEN", c in (200, 201) and n3 == 0, f"exam_attempt={c} start={c} X3={n3}")
submit("sv.kha", a["id"])
# ---- TC-05-15 ghi lại ở pm_a1 (P). TC-05-18 DIFF
# ---- TC-06-02: SV chưa có lượt nào → khuyến nghị = chủ đề nhiều câu APPROVED nhất
sql(f"delete from practice_attempts where student_id=(select id from users where email='sv34@edupilot.local') and course_id='{C1}'")
c, ov, _ = api("sv34", "GET", f"{P}/practice/overview")
tops = ov["topics"]; mx = max(t["available"] for t in tops)
rec = ov.get("recommended") or ov.get("recommendation")
print("overview keys", list(ov.keys()), str(rec)[:200])
rt = (rec or {}).get("topic") if isinstance(rec, dict) else rec
chk("TC-06-02 SV chưa có lượt: khuyến nghị = chủ đề có nhiều câu APPROVED nhất", rt is not None and next(t["available"] for t in tops if t["topic"] == rt) == mx, f"khuyến nghị={rt} có {next((t['available'] for t in tops if t['topic'] == rt), None)}/{mx} (max={[t['topic'] for t in tops if t['available'] == mx]})")
# ---- TC-07-03
bad = []
for body in ({"mode": "MOCK", "count": 20, "duration_minutes": 9}, {"mode": "MOCK", "count": 20, "duration_minutes": 181}, {"mode": "MOCK", "count": 4, "duration_minutes": 20}, {"mode": "MOCK", "count": 51, "duration_minutes": 20}):
    drain("sv.kha"); c, j, _ = start("sv.kha", body)
    bad.append((body["count"], body["duration_minutes"], c, (j or {}).get("code")))
chk("TC-07-03 duration 9/181, count 4/51 → 422 VALIDATION_FAILED", all(b[2] == 422 and b[3] == "VALIDATION_FAILED" for b in bad), str(bad))
# ---- TC-07-24
drain("sv.kha"); c, a, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 5}); it = a["items"][0]
r1 = [att("sv.gioi", a["id"])[0], put("sv.gioi", a["id"], it["id"], {"text": "x"})[0], submit("sv.gioi", a["id"])[0]]
r2 = [att("sv35", a["id"])[0], put("sv35", a["id"], it["id"], {"text": "x"})[0], submit("sv35", a["id"])[0]]
r3 = [put("teacher", a["id"], it["id"], {"text": "x"})[0], put("ta", a["id"], it["id"], {"text": "x"})[0]]
chk("TC-07-24 SV khác lớp/cùng lớp: 404 mọi API lượt; staff PUT answers: 403", set(r1) == {404} and set(r2) <= {404, 403} and set(r3) == {403}, f"sv.gioi {r1}; sv lớp khác {r2}; GV/TA PUT {r3}")
submit("sv.kha", a["id"])
# ---- TC-08-07 MOCK mặc định: 0 ESSAY
ess = 0; cnt = 0
for u in SV:
    drain(u); c, a, _ = start(u, {"mode": "MOCK", "count": 20, "duration_minutes": 20})
    if c in (200, 201): cnt += 1; ess += sum(it["question"]["type"] == "ESSAY" for it in a["items"]); submit(u, a["id"])
chk("TC-08-07 %d lượt MOCK ma trận mặc định: 0 câu ESSAY" % cnt, cnt >= 8 and ess == 0, f"ESSAY={ess}")
# ---- TC-08-13 Hồ sơ 360 (GV)
u = sql("select id from users where email='sv.kha@edupilot.local'")[0][0]
c, j, _ = api("teacher", "GET", f"{P}/students/{u}/profile"); body = json.dumps(j, ensure_ascii=False).lower()
chk("TC-08-13 hồ sơ SV của GV (profile) không có bài viết / nhận xét luyện", c == 200 and not any(k in body for k in ("practice", "luyện", "essay", "feedback")), f"{c} {len(body)} ký tự; khoá: {list(j.keys())[:12]}")
# ---- TC-10-16 chéo lớp
c, q2, _ = api("teacher", "GET", f"/courses/{C2}/questions?limit=5"); q2id = q2["items"][0]["id"]
drain("sv.kha"); c, a, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 5}); it = a["items"][0]
c1, j1, _ = api("sv.kha", "PUT", f"{P}/practice/attempts/{a['id']}/answers/{it['id']}", {"answer": {"text": "x"}, "question_id": q2id})
c2, j2, _ = api("sv.kha", "GET", f"/courses/{C2}/questions/{q2id}")
c3, j3, _ = api("sv.kha", "GET", f"/courses/{C2}/questions")
chk("TC-10-16 question_id lớp 2 trong PUT answers → 422 INVALID_QUESTION; đọc câu lớp 2 → 403/404", c1 == 422 and (j1 or {}).get("code") == "INVALID_QUESTION" and c2 in (403, 404) and c3 in (403, 404), f"PUT {c1} {(j1 or {}).get('code')}; GET câu {c2}; GET ds {c3}")
submit("sv.kha", a["id"])
print("done s10a")
