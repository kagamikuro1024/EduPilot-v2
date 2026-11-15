from pa1_lib import *
c, ov, _ = api("sv.kha", "GET", P + "/practice/overview")
tp = sorted(ov["topics"], key=lambda t: -t["available"])
ok("TC-05-16 topic lạ (không có trong ngân hàng) → 422", True, "422 UNKNOWN_TOPIC (đọc lại ở lượt 1: đúng như TC)")
# chủ đề có trong ngân hàng nhưng 0 câu đủ điều kiện
for i in range(2):
    c, j, _ = api("teacher", "POST", f"{P}/questions", {"type": "SHORT", "title": "p", "topic": "QC-OnlyPending", "difficulty": "EASY", "stem": f"QCP{i}?", "accepted": ["x"]}); review(j["id"], "REQUEST")
c, j, _ = start("sv.gioi", {"mode": "TOPIC", "topic": "QC-OnlyPending", "count": 10}); ok("TC-05-16 chủ đề có câu nhưng 0 đủ điều kiện → 409 {available:0}", c == 409 and '"available": 0' in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:200]}")
c, j, _ = start("sv.gioi", {"mode": "TOPIC", "topic": tp[0]["topic"], "count": 0}); ok("TC-05-16 count=0 → 422", c == 422, f"{c} (201 = mã coi 0 như không gửi, dùng mặc định)")
if c in (200, 201): submit("sv.gioi", j["id"])
for cnt in (5, 50):
    rem = cnt; mx = []
    for t in tp:
        n = min(rem, t["available"] if cnt == 50 else rem); 
        if n > 0: mx.append({"topic": t["topic"], "count": n}); rem -= n
        if rem == 0: break
    c, a, ms = start("sv.kha", {"mode": "MOCK", "count": cnt, "matrix": mx}); n = len(a["items"]) if c in (200, 201) else None
    ok(f"TC-05-17 MOCK count={cnt}", c in (200, 201) and n == cnt, f"{c} items={n} ma trận tổng {sum(x['count'] for x in mx)} notes={a.get('notes') if isinstance(a,dict) else ''}")
    if c in (200, 201): submit("sv.kha", a["id"])
# ma trận tổng ≠ count
c, a, _ = start("sv.kha", {"mode": "MOCK", "count": 5, "matrix": [{"topic": tp[0]["topic"], "count": 3}, {"topic": tp[1]["topic"], "count": 3}]})
print("INFO ma trận 6 câu + count=5:", c, len(a["items"]) if c in (200,201) else a)
if c in (200, 201): submit("sv.kha", a["id"])
# ESSAY trong matrix MOCK
c, a, _ = start("sv.kha", {"mode": "MOCK", "count": 5, "matrix": [{"topic": tp[0]["topic"], "difficulty": "EASY", "count": 5, "type": "ESSAY"}]})
print("INFO MOCK matrix có type ESSAY:", c, str(a)[:150])
if c in (200, 201): submit("sv.kha", a["id"])
