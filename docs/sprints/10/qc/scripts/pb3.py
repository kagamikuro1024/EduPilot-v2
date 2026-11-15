from pb import *
def bulk(user, body, C=C1): return api(user, "POST", f"/courses/{C}/questions/bulk", body)
pend = [q for q in bank() if q["review_status"] == "PENDING" and q["origin"] != "EXTRACTED"][:6]
ids = [q["id"] for q in pend]; print("pending", len(ids), {q["origin"] for q in pend})
# TC-04-05: 2 PENDING hợp lệ + id lớp 2 + id lạ
c2q = [q["id"] for q in bank("teacher", C2)][:1] if C2 else []
c, j, _ = bulk("teacher", {"action": "APPROVE", "ids": ids[:2] + c2q + [str(uuid.uuid4())]})
print(c, json.dumps(j, ensure_ascii=False)[:500])
res = j.get("results", []) if isinstance(j, dict) else []
ok("TC-04-05 bulk APPROVE: 2 ok, lớp 2/id lạ ok:false (không huỷ lô)", c == 200 and sum(1 for r in res if r["ok"]) == 2 and all(not r["ok"] for r in res[2:]), f"{c} {[(r.get('ok'), r.get('code')) for r in res]}")
# CITATION_REQUIRED: câu GENERATED/EXTRACTED không trích dẫn — dựa vào AI_DRAFT PENDING seed
aid_ = [q for q in bank() if q["origin"] in ("AI_DRAFT", "EXTRACTED", "GENERATED") and q["review_status"] == "PENDING"]
print("AI-origin pending", [(q["origin"], q["title"][:20]) for q in aid_][:5])
if aid_:
    q = aid_[0]; d = detail(q["id"]); v = d["version"]
    c, j, _ = api("teacher", "PUT", f"{P}/questions/{q['id']}/review", {"decision": "APPROVE", "version": v})
    ok("TC-04-13 duyệt câu AI-origin không trích dẫn → 422 CITATION_REQUIRED", c == 422 and "CITATION_REQUIRED" in json.dumps(j), f"origin={q['origin']} citations={len(d.get('citations') or [])} → {c} {json.dumps(j, ensure_ascii=False)[:120]}")
    c, j, _ = bulk("teacher", {"action": "APPROVE", "ids": [aid_[0]["id"]]}); res = j.get("results", [])
    ok("TC-04-05b bulk APPROVE câu AI-origin không trích dẫn → ok:false CITATION_REQUIRED", res and not res[0]["ok"] and res[0].get("code") == "CITATION_REQUIRED", f"{c} {res}")
c, j, _ = bulk("teacher", {"action": "SET_TAGS", "ids": ids[2:5], "topic": "QC-Tag", "difficulty": "HARD"}); res = j.get("results", [])
chk = [detail(i) for i in ids[2:5]]
ok("TC-04-07a SET_TAGS 3 câu đúng topic/difficulty", c == 200 and all(r["ok"] for r in res) and all(d["topic"] == "QC-Tag" and d["difficulty"] == "HARD" for d in chk), f"{c} {[(d['topic'], d['difficulty']) for d in chk]}")
for lab, body in [("ids rỗng", {"action": "REJECT", "ids": []}), ("51 ids", {"action": "REJECT", "ids": [str(uuid.uuid4()) for _ in range(51)]}), ("action lạ", {"action": "NUKE", "ids": ids[:1]}), ("SET_TAGS thiếu topic/difficulty", {"action": "SET_TAGS", "ids": ids[:1]})]:
    c, j, _ = bulk("teacher", body); ok(f"TC-04-07 {lab} → 422", c == 422, f"{c}")
c, j, _ = bulk("teacher", {"action": "REJECT", "ids": ids[2:5]}); ok("TC-04-06 bulk REJECT 3 câu", c == 200 and all(r["ok"] for r in j["results"]), f"{c}")
for u, exp in [("sv.kha", 403), ("admin", 403), ("ta", 200)]:
    c, j, _ = bulk(u, {"action": "SET_TAGS", "ids": ids[:1], "topic": "QC-Tag2", "difficulty": "EASY"}); ok(f"TC-04-P bulk {u} → {exp}", c == exp, f"{c}")
if C2:
    t2 = api("teacher", "GET", "/me/courses")
    c, j, _ = bulk("teacher", {"action": "SET_TAGS", "ids": ids[:1], "topic": "x", "difficulty": "EASY"}, C=C2); res = j.get("results", []) if isinstance(j, dict) else []
    ok("TC-04-P4 bulk với id lớp 1 trên đường dẫn lớp 2 → ok:false/403/404", c in (403, 404) or (c == 200 and all(not r["ok"] for r in res)), f"{c} {[(r.get('ok'), r.get('code')) for r in res]}")
# TC-04-11 xung đột version
q = ids[0]; d = detail(q); v = d["version"]
c1, j1, _ = api("teacher", "PUT", f"{P}/questions/{q}", {**{k: d[k] for k in ("type", "title", "topic", "difficulty", "stem")}, "stem": d["stem"] + " (GV)", "options": [{"body": o["body"]} for o in d["options"]], "correct": [i for i, o in enumerate(d["options"]) if o["id"] in ((d.get("answer_key") or {}).get("option_ids") or [])], "version": v})
c2, j2, _ = api("ta", "PUT", f"{P}/questions/{q}", {**{k: d[k] for k in ("type", "title", "topic", "difficulty", "stem")}, "stem": d["stem"] + " (TA)", "options": [{"body": o["body"]} for o in d["options"]], "correct": [i for i, o in enumerate(d["options"]) if o["id"] in ((d.get("answer_key") or {}).get("option_ids") or [])], "version": v})
ok("TC-04-11 version cũ → 409 VERSION_CONFLICT", c1 == 200 and c2 == 409 and "VERSION_CONFLICT" in json.dumps(j2), f"{c1}/{c2} {json.dumps(j2, ensure_ascii=False)[:160]}")
# phân trang
seen = []; cur = None; pages = 0
while True:
    c, j, _ = api("teacher", "GET", f"{P}/questions?limit=100" + (f"&cursor={cur}" if cur else "")); seen += [i["id"] for i in j["items"]]; cur = j.get("next_cursor"); pages += 1
    if not cur: break
ok("TC-04-17a theo cursor đến hết, không trùng", len(seen) == len(set(seen)) and pages >= 2, f"{len(seen)} câu / {pages} trang")
c, j, _ = api("teacher", "GET", f"{P}/questions?limit=101"); ok("TC-04-17b limit=101 cắt ≤100 hoặc 422", (c == 200 and len(j["items"]) <= 100) or c == 422, f"{c} n={len(j['items']) if c==200 else '-'}")
for u in ["sv.kha"]:
    codes = [api(u, "GET", f"{P}/questions")[0], api(u, "POST", f"{P}/questions", {})[0], api(u, "PUT", f"{P}/questions/{ids[0]}", {})[0], api(u, "GET", f"{P}/questions/gen-stats?job_id={uuid.uuid4()}")[0]]
    ok("TC-04-P1 STUDENT /questions* → 403", all(x == 403 for x in codes), str(codes))
ok("TC-04-P5 ADMIN /questions → 403", api("admin", "GET", f"{P}/questions")[0] == 403, "")
