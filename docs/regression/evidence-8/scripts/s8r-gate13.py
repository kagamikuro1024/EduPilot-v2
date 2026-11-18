# TC-GATEP1-13 (sprint 3, "2 provider thật"): 2 nhà thật qua 9router (model khác nhau), CHAT=[A,B]; dưới khoá global. Không in khoá.
import sys, os, time, json, sqlite3
sys.path.insert(0, '.')
exec(open('s8r-03i.py').read().split('os.makedirs("/tmp/qc2-s8r"')[0])
import s8r_lib
KEY = sqlite3.connect("file:" + os.path.expanduser("~/.9router/db/data.sqlite") + "?mode=ro", uri=True).execute("select key from apiKeys where isActive = 1 limit 1").fetchone()[0]
os.makedirs("/tmp/qc2-s8r", exist_ok=True)
if not acquire(): print("không lấy được khoá"); sys.exit(2)
logl("LOCK — GATEP1-13: 2 nhà thật qua 9router, CHAT=[A,B]; ≤ 5 phút")
ids = []; old = routes()["CHAT"]
try:
    mk = lambda n, m: call("POST", "/admin/llm/providers", "admin", {"type": "openai_compatible", "name": n, "base_url": "http://host.docker.internal:20128/v1", "api_key": KEY, "models": [{"model": m, "kind": "chat", "price_in": "0", "price_out": "0"}]})
    ca, a = mk("QC2-real-A", "ag/gemini-3.8-flash"); cb, b = mk("QC2-real-B", "ag/gemini-3.8-flash-medium"); print("create", ca, cb)
    pa = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "QC2-real-A"][0]; pb = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "QC2-real-B"][0]
    ma, mb = pa["models"][0]["id"], pb["models"][0]["id"]; ids = [pa["id"], pb["id"]]
    lt = lambda p: (p.get("last_test") or {})
    cur = routes()["CHAT"]; r = call("PUT", "/admin/llm/routes", "admin", {"task": "CHAT", "chain": [ma, mb], "params": {"max_tokens": 200}, "version": cur["version"]}, headers={"If-Match": 'W/"v%d"' % cur["version"]}); print("route", r[0])
    def chat(q):
        import s8r_chat; t0 = s8r_chat.now_db(); sid, txt, ev = s8r_chat.ask("sv.gioi", s8r_chat.cid("QD8R6U"), q)
        row = sql(f"select provider, model, fallback_index, status from llm_audit where created_at >= '{t0}' and task='CHAT' order by created_at desc limit 1"); return txt, row
    t1, r1 = chat("Xin chào, cho mình hỏi lịch học tuần này?"); time.sleep(5)
    pa2 = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["id"] == pa["id"]][0]
    cd = call("PUT", f"/admin/llm/providers/{pa['id']}", "admin", {"type": "openai_compatible", "name": "QC2-real-A", "base_url": "http://host.docker.internal:20128/v1", "enabled": False, "models": [{"model": "ag/gemini-3.8-flash", "kind": "chat", "price_in": "0", "price_out": "0"}], "version": pa2["version"]}); print("disable A", cd[0])
    time.sleep(3); t2, r2 = chat("Nhắc lại giúp mình lịch học tuần này?")
    T("TC-GATEP1-13", "P" if ca == 201 and cb == 201 and r1 and r1[0] and "QC2-real-A" in str(r1) and r2 and "QC2-real-B" in str(r2) else "F", f"2 nhà THẬT (9router, ag/gemini-3.8-flash và -medium; 9router chỉ phục vụ họ gemini-3.8-flash): tạo 201/201; CHAT=[A,B]: lượt 1 llm_audit {r1}; tắt A → lượt 2 {r2}; (bản 10/2026 với khoá OpenAI/Gemini thật đã PASS: report-GATE-P1 phần Bổ sung)")
finally:
    try:
        cur = routes()["CHAT"]; print("restore", call("PUT", "/admin/llm/routes", "admin", {"task": "CHAT", "chain": [c["model_id"] for c in old["chain"]], "params": old.get("params") or {}, "version": cur["version"]}, headers={"If-Match": 'W/"v%d"' % cur["version"]})[0])
    except Exception as e: print("restore err", e)
    for i in ids: print("del", call("DELETE", f"/admin/llm/providers/{i}", "admin")[0])
    logl("UNLOCK — GATEP1-13 xong, route CHAT hoàn lại")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
