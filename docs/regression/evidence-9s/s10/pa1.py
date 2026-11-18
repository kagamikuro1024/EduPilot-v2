# US-P9-01 (API) + US-P9-05
from pa import *
import unicodedata

def mkq(user, body, C=C1): return api(user, "POST", f"/courses/{C}/questions", body)
def review(qid, dec, C=C1, user="teacher"):
    v = api(user, "GET", f"/courses/{C}/questions/{qid}")[1]["version"]
    return api(user, "PUT", f"/courses/{C}/questions/{qid}/review", {"decision": dec, "version": v})
def approve(qid, C=C1, user="teacher"):
    c, j, _ = review(qid, "REQUEST", C, user); c2, j2, _ = review(qid, "APPROVE", C, user)
    return c, c2, j2
def mk_short(topic, n, accepted, tag, user="teacher"):
    ids = []
    for i in range(n):
        c, j, _ = mkq(user, {"type": "SHORT", "title": f"{tag} {i}", "topic": topic, "difficulty": "EASY", "stem": f"{tag} câu {i}: nhập đáp án?", "accepted": accepted})
        assert c == 201, (c, j); ids.append(j["id"]); assert approve(j["id"])[1] == 200
    return ids

# --- TC-01-12 / 16 / P1 / P2
c, j, _ = mkq("teacher", {"type": "SHORT", "title": "QC short 1", "topic": "QC-tmp", "difficulty": "EASY", "stem": "Thủ đô của Việt Nam?", "accepted": ["Hà Nội"]})
ok("TC-01-12a", c == 201 and j.get("review_status") == "DRAFT", f"SHORT {c} {j.get('review_status') if isinstance(j,dict) else j}")
c, j, _ = mkq("teacher", {"type": "ESSAY", "title": "QC essay 1", "topic": "QC-tmp", "difficulty": "MEDIUM", "stem": "Trình bày TCP.", "model_answer": "TCP tin cậy, hướng kết nối.", "rubric": "Nêu 3 đặc điểm"})
ok("TC-01-12b", c == 201 and j.get("review_status") == "DRAFT", f"ESSAY {c}")
essay_id = j.get("id")
for lab, acc in [("[]", []), ('[""]', [""]), ('["  "]', ["  "]), ('["...","?!"]', ["...", "?!"])]:
    c, j, _ = mkq("teacher", {"type": "SHORT", "title": "QC bad", "topic": "QC-tmp", "stem": "x?", "accepted": acc})
    codes = [d.get("code") for d in (j.get("details") or [])] if isinstance(j, dict) else []
    ok(f"TC-01-16 {lab}", c == 422 and "ANSWER_EMPTY" in codes, f"{c} {codes}")
c, j, _ = mkq("sv.kha", {"type": "SHORT", "title": "x", "topic": "t", "stem": "x?", "accepted": ["a"]})
ok("TC-01-P1", c == 403, str(c))
c, j, _ = mkq("ta", {"type": "SHORT", "title": "QC ta", "topic": "QC-tmp", "stem": "TA tạo?", "accepted": ["ok"]})
ok("TC-01-P2", c == 201, str(c))
c, j, _ = mkq("teacher", {"type": "ESSAY", "title": "e", "topic": "QC-tmp", "stem": "e?"})
ok("TC-S1 ESSAY thiếu model_answer/rubric → 422", c == 422, f"{c} {[d.get('code') for d in (j.get('details') or [])]}")

# --- TC-01-08/09: chấm SHORT qua lượt TOPIC (sv.gioi, tránh lẫn lượt sv.kha)
A = mk_short("QC-ShortA", 5, ["Hà Nội"], "QCA")
Bq = mk_short("QC-ShortB", 5, ["TCP/IP", "TCP IP"], "QCB")
nfd = unicodedata.normalize("NFD", "Hà Nội")
VA = [("Hà Nội", 1), ("  hà   nội  ", 1), ("HÀ NỘI.", 1), ("Hà Nội!", 1), ("Hà Nội...", 1), ("Hà Nội?!", 1), ("Hà Nội…", 1), (nfd, 1), ("Ha Noi", 0), ("ha noi", 0), ("Hà  Nội", 1), ("", 0), ("Hà Nội Hà Nội", 0), ("Hanoi", 0), ("hà nội.", 1)]
def run_short(user, topic, variants):
    res = []; i = 0
    while i < len(variants):
        c, a, _ = start(user, {"mode": "TOPIC", "topic": topic, "count": 5}); assert c in (200, 201), (c, a)
        for it in a["items"]:
            if i >= len(variants): break
            txt, exp = variants[i]; i += 1
            c, j, _ = put(user, a["id"], it["id"], {"text": txt})
            got = (j.get("feedback") or {}).get("correct") if c == 200 else f"HTTP{c}"
            res.append((txt, exp, got, c))
        submit(user, a["id"])
    return res
r = run_short("sv.gioi", "QC-ShortA", VA)
bad = [x for x in r if not ((x[2] is True) == bool(x[1]) and x[3] == 200)]
ok("TC-01-08", not bad, f"{len(r)} ca; lệch: {[(x[0], x[1], x[2]) for x in bad]}")
r = run_short("sv.gioi", "QC-ShortB", [("tcp/ip.", 1), ("TCP-IP", 0), ("tcp ip", 1), ("", 0)])
bad = [x for x in r if not ((x[2] is True) == bool(x[1]) and x[3] == 200)]
ok("TC-01-09", not bad, f"{[(x[0], x[1], x[2], x[3]) for x in r]}")

# --- TC-05: dựng đề
c, ov, _ = api("sv.kha", "GET", P + "/practice/overview"); ok("TC-06-03 overview", c == 200 and ov["topics"], f"{c} {len(ov['topics'])} chủ đề")
topics = sorted(ov["topics"], key=lambda t: -t["available"]); print(topics)
T = topics[0]["topic"]; TA_ = topics[0]["available"]
c, a, ms = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10})
stems = [i["question"]["stem"] for i in a["items"]]
ok("TC-05-02 TOPIC 10 câu không trùng", c in (200, 201) and len(a["items"]) == 10 and len(set(stems)) == 10, f"{c} {len(stems)} câu")
submit("sv.kha", a["id"])
# TC-05-13 thứ tự ổn định
c, a, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10}); aid = a["id"]
seq = [tuple(i["question"]["stem"] for i in att("sv.kha", aid)[1]["items"]) for _ in range(3)]
opt = [tuple(tuple(o["id"] for o in i["question"]["options"]) for i in att("sv.kha", aid)[1]["items"]) for _ in range(3)]
ok("TC-05-13", len(set(seq)) == 1 and len(set(opt)) == 1, "thứ tự câu + phương án ổn định qua 3 lần GET")
# TC-05-12 khác nhau giữa hai SV
c, b, _ = start("sv.gioi", {"mode": "TOPIC", "topic": T, "count": 10})
sa = [i["question"]["stem"] for i in a["items"]]; sb = [i["question"]["stem"] for i in b["items"]]
oa = [tuple(o["id"] for o in i["question"]["options"]) for i in a["items"]]; ob = [tuple(o["id"] for o in i["question"]["options"]) for i in b["items"]]
ok("TC-05-12", (sa != sb or oa != ob), f"cùng tập câu={set(sa)==set(sb)}, thứ tự câu khác={sa!=sb}, thứ tự phương án khác={oa!=ob}")
submit("sv.gioi", b["id"]); submit("sv.kha", aid)
# TC-05-16 biên
for lab, body, exp in [("count=4", {"mode": "TOPIC", "topic": T, "count": 4}, 422), ("count=51", {"mode": "TOPIC", "topic": T, "count": 51}, 422), ("count=0", {"mode": "TOPIC", "topic": T, "count": 0}, 422), ('count="abc"', {"mode": "TOPIC", "topic": T, "count": "abc"}, 422), ("topic lạ", {"mode": "TOPIC", "topic": "ZZ-không-có", "count": 10}, 409), ("mode lạ", {"mode": "X", "topic": T, "count": 10}, 422), ("TOPIC thiếu topic", {"mode": "TOPIC", "count": 10}, 422)]:
    c, j, _ = start("sv.kha", body); ok(f"TC-05-16 {lab}", c == exp, f"{c} {(j or {}).get('error', j) if isinstance(j, dict) else j}")
c, j, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10}, idem=None, headers=None)
# TC-06-06 thiếu Idempotency-Key
c, j, _ = api("sv.kha", "POST", P + "/practice/start", {"mode": "TOPIC", "topic": T, "count": 10}, idem=False)
ok("TC-06-06 thiếu Idempotency-Key", c in (400, 422), str(c)); 
c, a, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10})
if a and a.get("id"): submit("sv.kha", a["id"])
# TC-05-14/15 phân quyền
c, j, _ = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10}, C=C2) if C2 else (0, 0, 0); ok("TC-05-14 sv lớp 1 start ở lớp 2", c in (403, 404), str(c))
for u in ["teacher", "ta", "admin"]:
    c, j, _ = start(u, {"mode": "TOPIC", "topic": T, "count": 10}); ok(f"TC-05-P staff {u} start", c == 403, str(c))
c, j, _ = start("sv.moi", {"mode": "TOPIC", "topic": T, "count": 10}); ok("TC-05-P sv.moi start", c in (403, 404), str(c))
# TC-05-11 / 22 chủ đề 4 câu; TC-05-10 chủ đề 6 câu
few = mk_short("QC-Few4", 4, ["x"], "QCF4"); c, j, _ = start("sv.gioi", {"mode": "TOPIC", "topic": "QC-Few4", "count": 10})
ok("TC-05-11/06-22 4 câu → 409 NOT_ENOUGH_QUESTIONS", c == 409 and "NOT_ENOUGH" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:150]}")
mk_short("QC-Few6", 6, ["x"], "QCF6"); c, j, _ = start("sv.gioi", {"mode": "TOPIC", "topic": "QC-Few6", "count": 10})
ok("TC-05-10 6 câu xin 10 → 6 câu + notes", c in (200, 201) and len(j["items"]) == 6 and j["notes"], f"{c} items={len(j['items']) if c in (200,201) else '-'} notes={j.get('notes') if isinstance(j,dict) else ''}")
if c in (200, 201): submit("sv.gioi", j["id"])
# TC-05-08 câu không đủ điều kiện không vào đề
mk_short("QC-Excl", 5, ["x"], "QCOK")
ids = {}
for st in ["DRAFT", "PENDING", "REJECTED", "ARCH"]:
    c, j, _ = mkq("teacher", {"type": "SHORT", "title": f"ZQ-NOAPPROVE-{st}", "topic": "QC-Excl", "difficulty": "EASY", "stem": f"ZQ-NOAPPROVE-{st} stem?", "accepted": ["x"]}); ids[st] = j["id"]
review(ids["PENDING"], "REQUEST")
review(ids["REJECTED"], "REQUEST"); review(ids["REJECTED"], "REJECT")
c1, j1 = approve(ids["ARCH"])[:2], None
c, j, _ = api("teacher", "POST", f"{P}/questions/{ids['ARCH']}/archive", {}); arch = c
seen = set()
for _ in range(30):
    c, a, _ = start("sv.gioi", {"mode": "TOPIC", "topic": "QC-Excl", "count": 5})
    if c not in (200, 201): break
    seen |= {i["question"]["stem"] for i in a["items"] if "NOAPPROVE" in i["question"]["stem"]}; submit("sv.gioi", a["id"])
ok("TC-05-08", not seen, f"30 lượt; câu DRAFT/PENDING/REJECTED/archived xuất hiện: {seen}; lệnh archive={arch}")
# TC-05-17 biên MOCK 5 và 50 (ma trận nhiều chủ đề)
mt = [{"topic": t["topic"], "count": 1} for t in topics[:6]]
for cnt in (5, 50):
    per = max(1, cnt // len(topics)); mx = [{"topic": t["topic"], "count": per} for t in topics]
    c, a, ms = start("sv.kha", {"mode": "MOCK", "count": cnt, "matrix": mx}); n = len(a["items"]) if c in (200, 201) else None
    ok(f"TC-05-17 MOCK count={cnt}", c in (200, 201) and n == cnt, f"{c} items={n} notes={a.get('notes') if isinstance(a,dict) else ''}")
    if c in (200, 201): submit("sv.kha", a["id"])
# TC-05-03 đo 60 lần dựng
ts = []
for i in range(60):
    c, a, ms = start("sv.kha", {"mode": "TOPIC", "topic": T, "count": 10}); ts.append(ms)
    if c in (200, 201): submit("sv.kha", a["id"])
ok("TC-05-03 dựng đề p95 ≤ 3 s", pct(ts, .95) <= 3000, f"n=60 p50={pct(ts,.5):.0f} p95={pct(ts,.95):.0f} max={max(ts):.0f} ms")
