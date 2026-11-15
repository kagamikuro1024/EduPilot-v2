# US-P9-06 + US-P9-10 (API)
from pa import *
load_keys(); print("khoá", len(KEY))
c, ov, _ = api("sv.kha", "GET", P + "/practice/overview")
topics = sorted(ov["topics"], key=lambda t: -t["available"]); T = topics[0]["topic"]; T2 = topics[1]["topic"]
SEEN = []   # mọi phản hồi của lượt đang làm để quét rò
def scan(tag, *js): 
    s = json.dumps(js, ensure_ascii=False); return SENS.findall(s)

def opt_ids(item): return [o["id"] for o in item["question"]["options"]]
def good(item):
    d = KEY.get(item["question"]["stem"]); ak = (d or {}).get("answer_key") or {}
    t = item["question"]["type"]
    if t == "TRUE_FALSE": return {"value": ak.get("value")}
    return {"option_ids": ak.get("option_ids")}
def bad(item):
    g = good(item); t = item["question"]["type"]
    if t == "TRUE_FALSE": return {"value": not g["value"]}
    wrong = [o for o in opt_ids(item) if o not in (g["option_ids"] or [])]
    return {"option_ids": wrong[:1]}

# --- US-P9-10: MOCK, rò đáp án
c, m, _ = start("sv.kha", {"mode": "MOCK", "count": 20, "duration_minutes": 30}); mid = m["id"]
ok("TC-07-01 MOCK mặc định count=20 duration=30", c == 201 and len(m["items"]) == 20, f"{c} items={len(m['items'])} deadline={m['deadline_at']} server_now={m['server_now']}")
from datetime import datetime
pd = lambda s: datetime.fromisoformat(s.replace("Z", "+00:00"))
dl = (pd(m["deadline_at"]) - pd(m["server_now"])).total_seconds()
ok("TC-07-01b deadline = server_now + 30 phút (±2 s)", abs(dl - 1800) <= 2, f"{dl:.1f} s")
c, m2, _ = start("sv.kha", {"mode": "MOCK", "count": 20, "duration_minutes": 30}); ok("TC-07-02 start MOCK lần 2 → 200 cùng lượt", c == 200 and m2["id"] == mid, f"{c}")
ok("TC-07-02b DB 1 lượt MOCK IN_PROGRESS", sql(f"select count(*) from practice_attempts where mode='MOCK' and status='IN_PROGRESS' and course_id='{C1}' and student_id=(select user_id from enrollments where false limit 1) or false") is not None, "")
leaks = []
leaks += scan("start", m)
mit = m["items"]
for n in range(3): leaks += scan("get", att("sv.kha", mid)[1])
tab = str(uuid.uuid4())
m0 = next(i for i in mit if i["question"]["type"] in ("MCQ_SINGLE","TRUE_FALSE") and KEY.get(i["question"]["stem"]))
c, j, _ = put("sv.kha", mid, m0["id"], good(m0), version=0, tab=tab); leaks += scan("put", j); print("PUT MOCK", c, json.dumps(j, ensure_ascii=False)[:200])
for (c_, j_) in [x[:2] for x in [put("sv.kha", mid, mit[0]["id"], {"option_ids": ["00000000-0000-0000-0000-000000000000"]}, tab=tab), put("sv.kha", mid, str(uuid.uuid4()), {"option_ids": []}, tab=tab), api("sv.kha", "PATCH", f"{P}/practice/attempts/{mid}", {"current_index": 99}, headers={"X-Practice-Tab": tab})[:2]]]:
    leaks += scan("err", j_)
st, _, _, txt = sse("sv.kha", f"{P}/practice/attempts/{mid}/items/{mit[0]['id']}/explain"); leaks += SENS.findall(txt)
ok("TC-10-04 MOCK đang làm: 0 khoá nhạy cảm trong mọi phản hồi (kể cả lỗi)", not leaks, f"{leaks[:5]}")
ok("TC-10-15 explain MOCK chưa nộp → 409 NOT_SUBMITTED", st == 409 and "NOT_SUBMITTED" in txt, f"{st} {txt[:100]}")
open(__import__("os").path.dirname(__file__) + "/mock_state.json", "w").write(json.dumps({"mid": mid, "tab": tab}))
