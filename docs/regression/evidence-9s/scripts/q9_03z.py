#!/usr/bin/env python3
"""TC-P7-03-19: B thấy bài đã công bố, không rubric/đáp án; lịch DEADLINE; Hôm nay ASSIGNMENT_DUE (≤24 giờ, chưa nộp) mất ≤5 s sau nộp. Lớp mới QD9Z."""
import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("03z"); ALL = ["sv.gioi", "sv.kha", "sv.nguyco"]
C = mkcourse("QD9Z" + str(int(time.time()) % 1000), ALL); Q = f"/api/v1/courses/{C}"; T = tok("teacher"); SB = tok("sv.kha"); SC = tok("sv.nguyco")
a_pub = mkassign(Q, T, "Q3Z" + str(int(time.time()) % 100000), due_days=0.5)   # hạn +12 giờ
a_draft = mkassign(Q, T, "Q3D" + str(int(time.time()) % 100000), due_days=5, publish=False)
s, L, _ = call("GET", f"{Q}/assignments", SB); items = L.get("items", L.get("assignments", L)) if isinstance(L, dict) else L; js = json.dumps(L, ensure_ascii=False)
ids = [i["id"] for i in items]
R.chk("TC-P7-03-19", f"GET assignments (B): 200, thấy bài đã công bố, KHÔNG thấy bài nháp ({len(ids)} bài)", s == 200 and a_pub in ids and a_draft not in ids, (s, ids))
mine = [i for i in items if i["id"] == a_pub][0]
R.chk("TC-P7-03-19", f"mỗi bài có my:{{state}} = {mine.get('my')}; không rubric/guidance/reference_answer/ai_scores", "my" in mine and not any(w in js for w in ("guidance", "reference_answer", "ai_scores", "rubric")), js[:300])
s, d, _ = call("GET", f"{Q}/assignments/{a_pub}", SB); jd = json.dumps(d, ensure_ascii=False)
R.chk("TC-P7-03-19", "GET chi tiết bài (B): không guidance/reference_answer", s == 200 and "guidance" not in jd and "reference_answer" not in jd, jd[:300])
s, d, _ = call("GET", f"{Q}/assignments/{a_draft}", SB); R.chk("TC-P7-03-19", f"bài nháp (B) → {s} (không lộ)", s in (403, 404), (s, d))
fr = time.strftime("%Y-%m-%dT00:00:00Z", time.gmtime(time.time() - 86400)); to = time.strftime("%Y-%m-%dT00:00:00Z", time.gmtime(time.time() + 10 * 86400))
s, cal, _ = call("GET", f"{Q}/calendar?from={fr}&to={to}&limit=100", SB); jc = json.dumps(cal, ensure_ascii=False); R.chk("TC-P7-03-19", f"lịch (B) có DEADLINE của bài đã công bố, không có bài nháp: {s}", s == 200 and a_pub in jc and a_draft not in jc and "DEADLINE" in jc and '"assignment"' in jc, jc[:300])
def due(tk):
    s, it, _ = today("sv.kha" if tk is SB else "sv.nguyco", C); return [i for i in it if i["kind"] == "ASSIGNMENT_DUE" and a_pub in i["href"]]
d1 = due(SB); R.chk("TC-P7-03-19", f"Hôm nay (B, hạn +12 giờ, chưa nộp) có ASSIGNMENT_DUE: {[(i['reason']) for i in d1]}", len(d1) == 1, d1)
s, x = submit(Q, a_pub, SB, "bai.md", ("Bài làm. SUNBURST. " * 12).encode())[:2]; t0 = time.time(); gone = None
while time.time() - t0 < 12:
    if not due(SB): gone = time.time() - t0; break
    time.sleep(0.5)
R.chk("TC-P7-03-19", f"nộp xong ({s}) → ASSIGNMENT_DUE mất sau {gone if gone is None else round(gone, 1)} s (≤ 5 s); C (chưa nộp) vẫn còn: {len(due(SC))}", s == 201 and gone is not None and gone <= 5 and len(due(SC)) == 1, (s, gone))
sys.exit(R.end())
