# US-P9-07 (MOCK) + vài TC US-P9-10
from pa import *
from datetime import datetime, timedelta, timezone
load_keys()
for q in BANK:
    if q["review_status"] == "APPROVED" and q["type"] == "SHORT" and q["title"] and True:
        pass
def short_keys():
    for q in BANK:
        if q["review_status"] == "APPROVED" and q["type"] == "SHORT":
            d = detail(q["id"]); KEY[d["stem"]] = d
short_keys()
def good(item):
    d = KEY.get(item["question"]["stem"]); ak = (d or {}).get("answer_key") or {}; t = item["question"]["type"]
    if t == "TRUE_FALSE": return {"value": ak.get("value")}
    if t == "SHORT": return {"text": (ak.get("accepted") or ["?"])[0]}
    return {"option_ids": ak.get("option_ids")}
def bad(item):
    t = item["question"]["type"]; g = good(item)
    if t == "TRUE_FALSE": return {"value": not g["value"]}
    if t == "SHORT": return {"text": "đáp án sai chắc chắn zz"}
    w = [o for o in [x["id"] for x in item["question"]["options"]] if o not in (g["option_ids"] or [])]
    return {"option_ids": w[:1]}
pd = lambda s: datetime.fromisoformat(s.replace("Z", "+00:00"))
def fresh_mock(user="sv.kha", **kw):
    # nộp lượt MOCK dở (nếu có) rồi dựng lượt mới
    c, a, _ = start(user, {"mode": "MOCK", **kw}); 
    if c == 200:
        submit(user, a["id"]); c, a, _ = start(user, {"mode": "MOCK", **kw})
    return c, a
def answer_n(user, a, n_correct, tab, rest="wrong"):
    done = 0; exp = 0
    for i, it in enumerate(a["items"]):
        cor = i < n_correct
        c, j, _ = put(user, a["id"], it["id"], good(it) if cor else bad(it), version=0, tab=tab)
        assert c == 200, (c, j)
        exp += 1 if cor else 0
    return exp

# --- TC-07-07/08/14, tab
tab1, tab2 = str(uuid.uuid4()), str(uuid.uuid4())
c, a = fresh_mock(count=20, duration_minutes=30); aid = a["id"]; it0 = a["items"][0]
c, j, _ = put("sv.kha", aid, it0["id"], good(it0), version=0, tab=tab1); v1 = j["version"]
c, j, _ = put("sv.kha", aid, it0["id"], bad(it0), version=0, tab=tab1)
ok("TC-07-07 version cũ → 409 VERSION_CONFLICT", c == 409 and "VERSION_CONFLICT" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:120]}")
c, j, _ = put("sv.kha", aid, it0["id"], {"option_ids": ["00000000-0000-0000-0000-000000000000"]}, version=v1, tab=tab1)
ok("TC-07-08 option_id lạ → 422 INVALID_OPTION_ID", c == 422 and "INVALID_OPTION" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:150]}")
c, j, _ = put("sv.kha", str(uuid.uuid4()), it0["id"], good(it0), tab=tab1); ok("TC-07-08b lượt không tồn tại → 404", c == 404, str(c))
c, j, _ = put("sv.kha", aid, str(uuid.uuid4()), good(it0), version=0, tab=tab1); ok("TC-07-08c itemId lạ → 404/422", c in (404, 422), f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
for cur in (-1, 20, 99):
    c, j, _ = api("sv.kha", "PATCH", f"{P}/practice/attempts/{aid}", {"current_index": cur}, headers={"X-Practice-Tab": tab1}); ok(f"TC-07-14 PATCH current_index={cur}", c in (409, 422), f"{c}")
c, j, _ = api("sv.kha", "PATCH", f"{P}/practice/attempts/{aid}", {"current_index": 7}, headers={"X-Practice-Tab": tab1}); ok("TC-07-13 PATCH current_index=7 lưu", c == 200 and j["current_index"] == 7, f"{c}")
ok("TC-07-13b GET lại current_index=7, deadline không đổi", att("sv.kha", aid, headers={"X-Practice-Tab": tab1})[1]["current_index"] == 7 and att("sv.kha", aid)[1]["deadline_at"] == a["deadline_at"], "")
# TAB_TAKEN_OVER: mở tab2 (GET với tab2) rồi ghi bằng tab1
att("sv.kha", aid, headers={"X-Practice-Tab": tab2})
c, j, _ = put("sv.kha", aid, a["items"][1]["id"], good(a["items"][1]), version=0, tab=tab1)
ok("TC-07-06 tab cũ ghi sau khi tab mới mở → 409 TAB_TAKEN_OVER", c == 409 and "TAB_TAKEN_OVER" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:120]}")
c, j, _ = put("sv.kha", aid, a["items"][1]["id"], good(a["items"][1]), version=0, tab=tab2); ok("TC-07-06b tab mới ghi được; câu đã lưu còn", c == 200, str(c))
ok("TC-07-06c thiếu X-Practice-Tab khi ghi MOCK → 4xx", put("sv.kha", aid, a["items"][2]["id"], good(a["items"][2]), version=0)[0] in (400, 409, 422), "")
# --- chấm điểm 13/20 (+ idempotent submit) ; trước/sau gradebook + exam_events + exam-lock
ge0 = sql("select count(*) from grade_entries")[0][0]; ee0 = sql("select count(*) from exam_events")[0][0]; lock0 = api("sv.kha", "GET", "/me/exam-lock")[1]
c, a = fresh_mock(count=20, duration_minutes=30); aid = a["id"]
exp = answer_n("sv.kha", a, 13, tab2); 
lock1 = api("sv.kha", "GET", "/me/exam-lock")[1]
ok("TC-10-13 MOCK đang làm không đổi exam-lock", lock0 == lock1, f"{json.dumps(lock0)[:80]} → {json.dumps(lock1)[:80]}")
k = str(uuid.uuid4()); c, r1, _ = submit("sv.kha", aid, idem=k, tab=tab2); c2, r2, _ = submit("sv.kha", aid, tab=tab2)
R = r1.get("result") or r1
ok("TC-07-16 13/20 → correct=13, score=6.5", c == 200 and R["correct"] == 13 and R["total"] == 20 and R["score"] in ("6.5", "6.50"), f"{c} {json.dumps(R, ensure_ascii=False)[:260]}")
ok("TC-07-17 submit lần 2 cùng kết quả", c2 == 200 and (r2.get("result") or r2) == R, "")
d = att("sv.kha", aid)[1]
ok("TC-07-16c sau nộp lộ đáp án + giải thích từng câu", all((i.get("feedback") or {}).get("key") is not None for i in d["items"]), f"{sum(1 for i in d['items'] if i.get('feedback'))}/20 câu có phản hồi")
ok("TC-07-18 điểm luyện không vào sổ điểm (grade_entries không đổi)", sql("select count(*) from grade_entries")[0][0] == ge0, f"{ge0}")
# 5/8 = 6.3 (nửa lên); 7/20=3.5 ; 1/3 → 3.3 ; 3/7 → 4.3 (count nhỏ nhất 5 → dùng ma trận)
c, tpv, _ = api("sv.kha", "GET", P + "/practice/overview"); tpv = sorted(tpv["topics"], key=lambda t: -t["available"])
for n, tot, want in [(5, 8, "6.3"), (7, 20, "3.5"), (3, 7, "4.3"), (1, 3, "3.3")]:
    if tot < 5: continue
    c, a = fresh_mock(count=tot, duration_minutes=10, matrix=[{"topic": tpv[0]["topic"], "count": tot}] if tot != 20 else None) if tot != 20 else fresh_mock(count=20, duration_minutes=30)
    if c not in (200, 201): ok(f"TC-07-16b {n}/{tot}", False, f"start {c} {a}"); continue
    tb = str(uuid.uuid4()); expc = answer_n("sv.kha", a, n, tb); r = submit("sv.kha", a["id"], tab=tb)[1]; R = r.get("result") or r
    ok(f"TC-07-16b {n}/{tot} → {want}", R["correct"] == expc and R["score"] in (want, want + "0"), f"correct={R['correct']} (mong {expc}) total={R['total']} score={R['score']}")
# --- hạn: dung sai 60 s
def force_deadline(aid, secs_ago): sql(f"update practice_attempts set deadline_at = now() - interval '{secs_ago} seconds' where id='{aid}'")
c, a = fresh_mock(count=20, duration_minutes=10); aid = a["id"]; tb = str(uuid.uuid4()); att("sv.kha", aid, headers={"X-Practice-Tab": tb})
its = a["items"]
put("sv.kha", aid, its[0]["id"], good(its[0]), version=0, tab=tb)
force_deadline(aid, 20)
dl = pd(att("sv.kha", aid)[1]["deadline_at"])
fmt = lambda d: d.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")
c, j, _ = put("sv.kha", aid, its[1]["id"], good(its[1]), version=0, tab=tb, ts=fmt(dl - timedelta(seconds=5)))
ok("TC-07-11/19b client_ts ≤ hạn, trong 60 s → nhận", c == 200, f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
c, j, _ = put("sv.kha", aid, its[2]["id"], good(its[2]), version=0, tab=tb, ts=fmt(dl + timedelta(seconds=5)))
d = att("sv.kha", aid)[1]
ok("TC-07-11b client_ts > hạn → bỏ (200), notes ghi lý do, không lưu câu", c == 200 and d["notes"] and d["items"][2]["answer"] is None, f"{c} notes={d['notes']} answer câu 3={d['items'][2]['answer']}")
c, j, _ = put("sv.kha", aid, its[3]["id"], good(its[3]), version=0, tab=tb)
ok("TC-07-11d thiếu client_ts quá hạn (trong 60 s) → 409 ATTEMPT_CLOSED", c == 409 and "ATTEMPT_CLOSED" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
force_deadline(aid, 90)
c, j, _ = put("sv.kha", aid, its[4]["id"], good(its[4]), version=0, tab=tb, ts=fmt(dl - timedelta(seconds=30)))
ok("TC-07-11c/19 sau hạn > 60 s → 409 ATTEMPT_CLOSED (kể cả client_ts ≤ hạn)", c == 409 and "ATTEMPT_CLOSED" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
ee1 = sql("select count(*) from exam_events")[0][0]
ok("TC-10-14/07-23 không có exam_events mới", ee1 == ee0, f"{ee0}→{ee1}")
open("expiry_state.json", "w").write(json.dumps({"aid": aid}))
