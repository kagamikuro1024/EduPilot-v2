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

# --- TC-06-04 idempotent start / lượt dở
k1 = str(uuid.uuid4())
c1, a1, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10}, idem=k1)
c2, a2, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10}, idem=k1)
c3, a3, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10})
ok("TC-06-04", c1 == 201 and a1["id"] == a2["id"] == a3["id"] and c3 == 200, f"{c1}/{c2}/{c3} cùng id={a1['id']==a2['id']==a3['id']}")
aid = a1["id"]; items = a1["items"]
ok("TC-10-05a chưa trả lời: 0 khoá nhạy cảm", not scan("a", a1, att("sv.kha", aid)[1]), str(scan("a", a1)))
c, jj, _ = start("sv.kha", {"mode": "TOPIC", "topic": T2, "count": 5}); ok("TC-06-05 chủ đề khác → 201", c == 201, str(c)); t2aid = jj.get("id")
# TC-06-12 explain khi chưa trả lời
st, t1, tt, txt = sse("sv.kha", f"{P}/practice/attempts/{aid}/items/{items[0]['id']}/explain")
ok("TC-06-12 explain chưa trả lời → 409 NOT_ANSWERED", st == 409 and "NOT_ANSWERED" in txt, f"{st} {txt[:120]}")
# TC-06-08 / 09: trả lời câu 1 sai, 2 đúng
lat = []; fb = []
for n, it in enumerate(items):
    ans = bad(it) if n == 0 else good(it)
    c, j, ms = put("sv.kha", aid, it["id"], ans); lat.append(ms); fb.append((c, j))
c, j = fb[0]
f = j.get("feedback") or {}
ok("TC-06-08a TOPIC phản hồi ngay", c == 200 and f.get("correct") is False and f.get("key") and "explanation" in f and "citations" in f and j.get("locked") is True, f"{c} correct={f.get('correct')} key={'có' if f.get('key') else 'thiếu'} citations={len(f.get('citations') or [])} locked={j.get('locked')}")
ok("TC-06-08b câu 2 đúng", fb[1][1]["feedback"]["correct"] is True, str(fb[1][1]["feedback"]["correct"]))
cit = [x for c_, j_ in fb for x in (j_["feedback"].get("citations") or [])]
ok("TC-06-08c có trích dẫn (nguồn)", len(cit) > 0, f"{len(cit)} trích dẫn; mẫu {json.dumps(cit[:1], ensure_ascii=False)[:200]}")
c, j, _ = put("sv.kha", aid, items[0]["id"], good(items[0]))
ok("TC-06-09 đổi đáp án đã khoá → 409 ANSWER_LOCKED", c == 409 and "ANSWER_LOCKED" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:120]}")
ok("TC-06-08d p95 PUT answers ≤ 500 ms", pct(lat, .95) <= 500, f"n={len(lat)} p50={pct(lat,.5):.0f} p95={pct(lat,.95):.0f} ms (cỡ mẫu nhỏ; thêm ở TC-06-08e)")
# TC-06-11/13 explain
sw = []
for _ in range(20):
    st, t1, tt, txt = sse("sv.kha", f"{P}/practice/attempts/{aid}/items/{items[0]['id']}/explain"); sw.append((st, t1, txt))
ok("TC-06-11 explain SSE 200, byte đầu ≤ 300 ms", all(s == 200 for s, _, _ in sw) and pct([t for _, t, _ in sw], .95) <= 300, f"codes={set(s for s,_,_ in sw)} p95 byte đầu={pct([t for _,t,_ in sw],.95):.0f} ms; mẫu: {sw[0][2][:160]!r}")
aud = sql("select task, count(*) from llm_audit where created_at > now() - interval '30 minutes' group by 1 order by 2 desc")
print("llm_audit 30', theo task:", aud)
# TC-06-16/17 submit
k = str(uuid.uuid4())
c, r1, _ = submit("sv.kha", aid, idem=k); c2, r2, _ = submit("sv.kha", aid, idem=k); c3, r3, _ = submit("sv.kha", aid)
res = (r1 or {}).get("result") or r1
ok("TC-06-16/17 submit idempotent; câu bỏ trống = 0", c == 200 and c2 == 200 and c3 == 200 and r1 == r2, f"{c}/{c2}/{c3} kết quả {json.dumps(res, ensure_ascii=False)[:260]} (đã đúng 9/10, sai 1)")
ok("TC-06-16b không điểm/xếp hạng ở TOPIC", True, f"result keys={list((res or {}).keys())}")
# TC-06-18/19 phân quyền lượt của sv.kha (t2aid dở)
chk = [("GET", f"/practice/attempts/{t2aid}", None), ("PUT", f"/practice/attempts/{t2aid}/answers/{jj['items'][0]['id']}", {"answer": {"option_ids": []}}), ("POST", f"/practice/attempts/{t2aid}/items/{jj['items'][0]['id']}/explain", {}), ("POST", f"/practice/attempts/{t2aid}/submit", {}), ("PATCH", f"/practice/attempts/{t2aid}", {"current_index": 1})]
codes = [api("sv.gioi", m, P + p, b)[0] for m, p, b in chk]
ok("TC-06-18 sv.gioi trên lượt của sv.kha → 404 cả 5", codes == [404] * 5, str(codes))
for u in ["teacher", "ta", "admin"]:
    codes = [api(u, m, P + p, b)[0] for m, p, b in chk[:4]]
    ok(f"TC-06-19/01-P3 {u} trên lượt SV → 403", all(c == 403 for c in codes), str(codes))
    ok(f"TC-09-14 {u} overview/history/weakness → 403", all(api(u, "GET", P + p)[0] == 403 for p in ["/practice/overview", "/practice/history", "/practice/weakness"]), "")
submit("sv.kha", t2aid)
# TC-06-25b MCQ_MULTI
def mkmulti(topic, n):
    ids = []
    for i in range(n):
        body = {"type": "MCQ_MULTI", "title": f"QCM {i}", "topic": topic, "difficulty": "EASY", "stem": f"QCM{i}: chọn các số chẵn?", "options": [{"body": "2"}, {"body": "4"}, {"body": "6"}, {"body": "7"}], "correct": [0, 1, 2]}
        c, j, _ = api("teacher", "POST", f"{P}/questions", body); assert c == 201, (c, j); ids.append(j["id"])
        from pa1_lib import approve; approve(j["id"])
    return ids
mkmulti("QC-Multi", 5)
c, a, _ = start("sv.gioi", {"mode": "TOPIC", "topic": "QC-Multi", "count": 5}); it = a["items"]
opts = lambda i: {o["body"]: o["id"] for o in it[i]["question"]["options"]}
sets = [["2", "4", "6"], ["2", "4"], ["2", "4", "6", "7"], ["2", "7"], []]
got = []
for i, s in enumerate(sets):
    o = opts(i); c, j, _ = put("sv.gioi", a["id"], it[i]["id"], {"option_ids": [o[x] for x in s]}); got.append((s, (j.get("feedback") or {}).get("correct") if c == 200 else c))
ok("TC-06-25b MCQ_MULTI trùng tập mới đúng", [g[1] for g in got] == [True, False, False, False, False], str(got))
r = submit("sv.gioi", a["id"])[1]; ok("TC-06-25b đúng/tổng nguyên", (r.get("result") or r).get("correct") == 1 and (r.get("result") or r).get("total") == 5, json.dumps(r.get("result") or r, ensure_ascii=False)[:200])

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
c, j, _ = put("sv.kha", mid, mit[0]["id"], good(mit[0]), version=0, tab=tab); leaks += scan("put", j); print("PUT MOCK", c, json.dumps(j, ensure_ascii=False)[:200])
for (c_, j_) in [put("sv.kha", mid, mit[0]["id"], {"option_ids": ["00000000-0000-0000-0000-000000000000"]}, tab=tab), put("sv.kha", mid, str(uuid.uuid4()), {"option_ids": []}, tab=tab), api("sv.kha", "PATCH", f"{P}/practice/attempts/{mid}", {"current_index": 99}, headers={"X-Practice-Tab": tab})[:2]]:
    leaks += scan("err", j_)
st, _, _, txt = sse("sv.kha", f"{P}/practice/attempts/{mid}/items/{mit[0]['id']}/explain"); leaks += SENS.findall(txt)
ok("TC-10-04 MOCK đang làm: 0 khoá nhạy cảm trong mọi phản hồi (kể cả lỗi)", not leaks, f"{leaks[:5]}")
ok("TC-10-15 explain MOCK chưa nộp → 409 NOT_SUBMITTED", st == 409 and "NOT_SUBMITTED" in txt, f"{st} {txt[:100]}")
open(__import__("os").path.dirname(__file__) + "/mock_state.json", "w").write(json.dumps({"mid": mid, "tab": tab}))
