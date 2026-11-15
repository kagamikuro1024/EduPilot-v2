from pb import *
def gen(body, user="teacher", C=C1, idem=None): return api(user, "POST", f"/courses/{C}/questions/generate", body, idem=idem)
c, lst, _ = api("teacher", "GET", f"{P}/documents?limit=100"); L = [d for d in lst["items"] if d["type"] == "LECTURE" and d["status"] == "READY"]
L1 = L[0]; print("LECTURE", L1["title"], len(L))
c, j, _ = gen({"document_ids": [L1["id"]], "topic": "Mẫu sinh câu", "difficulty": "MEDIUM", "count": 5, "types": ["MCQ_SINGLE"]})
ok("TC-03-01 generate → 202", c == 202, f"{c} {j}")
J = wait_job(j["job_id"]); r = J.get("result") or {}; print("GENJOB", json.dumps(J, ensure_ascii=False)[:400])
gq = [q for q in bank() if q["origin"] == "GENERATED"]
ok("TC-03-01b câu GENERATED PENDING", len(gq) > 0 and all(q["review_status"] == "PENDING" for q in gq), f"{len(gq)} câu; created={len(r.get('created',[]))} skipped={[(s.get('number'), s.get('reason')) for s in r.get('skipped',[])]}")
bad = 0
for q in gq:
    d = detail(q["id"]); cs = d.get("citations") or []
    if not cs or not all(c.get("document_id") and c.get("chunk_id") and isinstance(c.get("page"), int) for c in cs): bad += 1
ok("TC-03-02 mỗi câu sinh có trích dẫn {document_id,chunk_id,page}", bad == 0 and gq, f"thiếu: {bad}/{len(gq)}")
# 04 ANSWER_KEY
dk = docid("QC Đáp án mẫu")["id"]
c, j, _ = gen({"document_ids": [dk], "topic": "Mẫu sinh câu", "difficulty": "EASY", "count": 3}); ok("TC-03-04 sinh từ ANSWER_KEY → 422", c == 422, f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
for lab, body, exp in [("count=11", {"document_ids": [L1["id"]], "topic": "x", "difficulty": "EASY", "count": 11}, 422), ("types=[]", {"document_ids": [L1["id"]], "topic": "x", "difficulty": "EASY", "count": 3, "types": []}, 422)]:
    c, j, _ = gen(body); ok(f"TC-03-07 {lab} → 422", c == exp, f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
if C2:
    l2 = api("teacher", "GET", f"/courses/{C2}/documents?limit=10")[1]["items"]
    if l2: c, j, _ = gen({"document_ids": [l2[0]["id"]], "topic": "x", "difficulty": "EASY", "count": 3}); ok("TC-03-07 document_ids lớp 2 → 422", c == 422, f"{c} {json.dumps(j, ensure_ascii=False)[:100]}")
c, j, _ = gen({"document_ids": [L1["id"]], "topic": "x", "difficulty": "EASY", "count": 3}, user="sv.kha"); ok("TC-03-10 STUDENT generate → 403", c == 403, str(c))
# gen-stats
if gq:
    jid = None
c, st, _ = api("teacher", "GET", f"{P}/questions/gen-stats?job_id={uuid.uuid4()}"); ok("TC-03-12 gen-stats job lạ → 404", c == 404, str(c))
# TC-03-13: các việc generate: đã có 1 ở trên (+ các 422 không xếp việc). Gửi thêm cho tới khi chạm trần
codes = []
for i in range(6):
    c, j, _ = gen({"document_ids": [L1["id"]], "topic": "Mẫu sinh câu", "difficulty": "MEDIUM", "count": 5, "types": ["MCQ_SINGLE"]}); codes.append(c)
ok("TC-03-13 việc generate thứ 6 trong giờ → 429", 429 in codes and codes.index(429) == 4, f"mã lần lượt: {codes} (việc #1 ở trên + 4 → 429 ở lần gửi thứ 5 của vòng này)")
