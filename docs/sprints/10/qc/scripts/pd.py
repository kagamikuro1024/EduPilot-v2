from pa import *
# --- US-P9-09
c, h, _ = api("sv.kha", "GET", P + "/practice/history?limit=20")
ok("TC-09-01 history 200, giảm dần theo thời gian", c == 200 and [i["started_at"] for i in h["items"]] == sorted([i["started_at"] for i in h["items"]], reverse=True), f"{c} n={len(h['items'])} keys={list(h['items'][0]) if h['items'] else ''}")
# phân trang: sv.kha có >45 lượt sau khi chạy các batch trước
seen = []; cur = None; pages = []
while True:
    c, j, _ = api("sv.kha", "GET", P + "/practice/history?limit=20" + (f"&cursor={cur}" if cur else "")); pages.append(len(j["items"])); seen += [i["id"] for i in j["items"]]; cur = j["next_cursor"]
    if not cur: break
ok("TC-09-03 duyệt cursor đến hết: không trùng", len(seen) == len(set(seen)), f"{len(seen)} lượt, trang {pages}")
c, j, _ = api("sv.kha", "GET", P + "/practice/history?limit=101"); ok("TC-09-03b limit=101 → cắt ≤100 hoặc 422", (c == 200 and len(j["items"]) <= 100) or c == 422, f"{c} {len(j['items']) if c==200 else ''}")
lat = []
for _ in range(60): lat.append(api("sv.kha", "GET", P + "/practice/history?limit=20")[2])
ok("TC-09-21 history p95 ≤ 300 ms", pct(lat, .95) <= 300, f"n=60 p50={pct(lat,.5):.0f} p95={pct(lat,.95):.0f} ms; {len(seen)} lượt của sv.kha (không đạt 200 lượt)")
c, w, _ = api("sv.kha", "GET", P + "/practice/weakness"); print("WEAK", json.dumps(w, ensure_ascii=False)[:400])
ok("TC-09-04 weakness ≤3 chủ đề, mỗi chủ đề ≥3 câu, xếp theo tỉ lệ sai", c == 200 and len(w["topics"]) <= 3 and all(t["answered"] >= 3 for t in w["topics"]) and [t["wrong"] / t["answered"] for t in w["topics"]] == sorted([t["wrong"] / t["answered"] for t in w["topics"]], reverse=True), f"{[(t['topic'], t['wrong'], t['answered']) for t in w['topics']]}")
c, w2, _ = api("sv.moi", "GET", P + "/practice/weakness"); ok("TC-09-sv.moi weakness → 403/404", c in (403, 404), str(c))
# 17: lớp khác
c, h2, _ = api("sv.gioi", "GET", f"/courses/{C2}/practice/history?limit=100"); c1, h1, _ = api("sv.gioi", "GET", P + "/practice/history?limit=100")
ids1 = {i["id"] for i in h1["items"]}; ids2 = {i["id"] for i in h2["items"]} if c == 200 else set()
ok("TC-09-17 lịch sử lớp 2 không chứa lượt lớp 1", c == 200 and not (ids1 & ids2), f"{c} lớp1={len(ids1)} lớp2={len(ids2)}")
# Hôm nay
c, t, _ = api("sv.kha", "GET", P + "/today"); print("TODAY", json.dumps(t, ensure_ascii=False)[:600])
items = t.get("items") or t.get("tasks") or []
wk = [i for i in items if "PRACTICE" in json.dumps(i)]
print("PRACTICE items", json.dumps(wk, ensure_ascii=False)[:300])
for u in ["teacher", "ta", "admin"]:
    cc, tt, _ = api(u, "GET", P + "/today"); ok(f"TC-09-09 today {u} không có PRACTICE_WEAK_TOPIC", "PRACTICE_WEAK_TOPIC" not in json.dumps(tt), f"{cc}")
# 06: có việc ôn chủ đề yếu? (seed hide 3 ngày)
ok("TC-09-06 today SV có PRACTICE_WEAK_TOPIC", len(wk) > 0, f"{len(wk)} việc: {[w.get('title') or w.get('label') for w in wk]}")
# Hồ sơ 360: GV không thấy dữ liệu luyện đề
sid = sql("select id from users where email='sv.kha@edupilot.local'")[0][0]
c, s360, _ = api("teacher", "GET", f"{P}/students/{sid}/360"); sj = json.dumps(s360, ensure_ascii=False)
ok("TC-09-15 Hồ sơ 360 không chứa dữ liệu luyện đề", c == 200 and not re.search(r"practice_attempt|weak(ness)?_topic|chủ đề yếu|luyện đề", sj, re.I), f"{c} khớp: {re.findall(r'(practice[a-z_]*|chủ đề yếu|luyện đề)', sj, re.I)[:4]}")
