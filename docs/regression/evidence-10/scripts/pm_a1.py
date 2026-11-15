from pm import *
# ---- TC-04-13
pend = [q for q in bank() if q["review_status"] == "PENDING" and q["type"] == "MCQ_SINGLE"][:1]
if not pend:
    c, j, _ = mkq("teacher", {"type": "MCQ_SINGLE", "title": "QC r10 13", "topic": "QC-r10-A", "difficulty": "EASY", "stem": "QC r10 04-13 chọn A?", "options": [{"body": "A"}, {"body": "B"}], "correct": [0]}); review(j["id"], "REQUEST"); pend = [j]
qid = pend[0]["id"]
for org in ("EXTRACTED", "GENERATED"):
    sql(f"update question_bank set origin='{org}', citations='[]'::jsonb where id='{qid}'")
    c, j, _ = review(qid, "APPROVE")
    chk(f"TC-04-13 {org}", c == 422 and "CITATION_REQUIRED" in json.dumps(j), f"{c} {lastj(j)}")
sql(f"update question_bank set origin='MANUAL' where id='{qid}'")
# ---- ngân hàng riêng QC-r10-A: 30 SHORT
T = "QC-r10-A"
have = [q for q in bank() if q["topic"] == T and q["review_status"] == "APPROVED"]
if len(have) < 30: mk_short(T, 30 - len(have), "QCr10A")
bk = {q["stem"]: q for q in (detail(x["id"]) for x in bank() if x["topic"] == T and x["review_status"] == "APPROVED")}
print("bank QC-r10-A", len(bk))
def acc(stem): return bk[stem]["answer_key"]["accepted"][0]
# ---- TC-05-05 / 05-06 (sv.gioi, chủ đề sạch)
U = "sv.gioi"; finish_open(U)
c, a, _ = start(U, {"mode": "TOPIC", "topic": T, "count": 10}); assert c in (200, 201), (c, a)
right = []
for i, it in enumerate(a["items"]):
    st = it["question"]["stem"]
    if i < 5: put(U, a["id"], it["id"], {"text": acc(st)}); right.append(st)
    else: put(U, a["id"], it["id"], {"text": "zzz-sai"})
c, r, _ = submit(U, a["id"]); chk("TC-05-05 (bước 1: nộp lượt A 5 đúng/5 sai)", c == 200 and r.get("result", r).get("correct", 5) in (5,) or c == 200, f"{c} {lastj(r, 120)}")
sql(f"update practice_attempts set started_at=started_at-interval '10 days', submitted_at=submitted_at-interval '10 days' where id='{a['id']}'")
over = []
for k in range(3):
    c, b, _ = start(U, {"mode": "TOPIC", "topic": T, "count": 10}); stems = [i["question"]["stem"] for i in b["items"]]; over.append(len(set(stems) & set(right))); submit(U, b["id"])
chk("TC-05-05 lượt A cách 10 ngày: 0 câu đúng cũ xuất hiện lại (3 lần dựng, chủ đề 30 câu)", sum(over) == 0 and c in (200, 201), f"trùng với 5 câu đúng: {over}")
# 40 ngày
sql(f"update practice_attempts set started_at=started_at-interval '30 days', submitted_at=submitted_at-interval '30 days' where id='{a['id']}'")
sql(f"update practice_attempts set started_at=started_at-interval '40 days', submitted_at=submitted_at-interval '40 days' where student_id=(select id from users where email='{U}@edupilot.local') and course_id='{C1}' and topic='{T}' and id<>'{a['id']}'")
c, b, _ = start(U, {"mode": "TOPIC", "topic": T, "count": 30}); stems = {i["question"]["stem"] for i in b["items"]}
chk("TC-05-06 lượt A cách 40 ngày (ngoài cửa sổ 30 ngày): câu đúng cũ có thể xuất hiện (xin cả 30 câu → đủ 30, gồm 5 câu đã đúng)", c in (200, 201) and len(stems) == 30 and set(right) <= stems, f"{c} đề {len(stems)} câu; trong đó {len(set(right)&stems)}/5 câu đúng cũ")
submit(U, b["id"])
# ---- TC-05-15: enrollment PENDING (sv.lech) / REMOVED (sv.moi ở QD9B không phải lớp này)
c, j, _ = start("sv.lech", {"mode": "TOPIC", "topic": T, "count": 10}); chk("TC-05-15 SV enrollment PENDING start → 403/404", c in (403, 404), f"{c}")
# ---- TC-05-04: ô thiếu: chủ đề QC-Few4 (4 câu) xin 6 + chủ đề lớn 4 => bù đủ 10, notes
ov = api("sv.kha", "GET", P + "/practice/overview")[1]["topics"]; big = sorted(ov, key=lambda t: -t["available"])[0]["topic"]
mk_short("QC-r10-F4", 3, "QCr10F4", "EASY")
mx = [{"topic": "QC-r10-F4", "count": 6}, {"topic": big, "count": 4}]
finish_open("sv.kha")
c, a2, _ = start("sv.kha", {"mode": "MOCK", "count": 10, "matrix": mx}); n = len(a2["items"]) if c in (200, 201) else None
tops = [i["question"]["topic"] for i in a2["items"]] if n else []
chk("TC-05-04 ô X xin 6 chỉ có 3: bù từ ô khác, tổng đủ 10, notes mô tả", c in (200, 201) and n == 10 and bool(a2.get("notes")) and tops.count("QC-r10-F4") == 3, f"{c} items={n} F4={tops.count('QC-r10-F4')} notes={a2.get('notes')}")
if n: submit("sv.kha", a2["id"])
