from pa import *
ov = api("sv.nguyco", "GET", P + "/practice/overview")[1]
nc = 0; code429 = None; calls = 0
for t in sorted(ov["topics"], key=lambda t: -t["available"]):
    if code429: break
    c, a, _ = start("sv.nguyco", {"mode": "TOPIC", "topic": t["topic"], "count": 20})
    if c not in (200, 201): continue
    for it in a["items"]:
        q = it["question"]
        if q["type"] == "ESSAY": continue
        ans = {"option_ids": [q["options"][0]["id"]]} if q["type"] in ("MCQ_SINGLE", "MCQ_MULTI") else ({"value": True} if q["type"] == "TRUE_FALSE" else {"text": "x"})
        c, j, _ = put("sv.nguyco", a["id"], it["id"], ans)
        if c != 200: continue
        st, t1, tt, txt = sse("sv.nguyco", f"{P}/practice/attempts/{a['id']}/items/{it['id']}/explain"); calls += 1
        if st == 429: code429 = (calls, nc, txt[:100]); break
        if '"cached":false' in txt: nc += 1
    submit("sv.nguyco", a["id"])
print("calls", calls, "LLM-needing", nc, "429 at", code429)
ok("TC-S4 explain 429 sau 30 lượt cần LLM/giờ/SV (P-10-8 F4)", code429 is not None and 30 <= nc <= 31, f"calls={calls} cần LLM trước 429={nc}; {code429}")
