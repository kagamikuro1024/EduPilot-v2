# TC-P104-46 (kịch bản scenario-P1.md, QC chạy lại độc lập trên ep-r dưới khoá global). Tạo nhà QC2S-Chinh/Phu (fake) + QC2S-Q (openai_compatible → stub :9764).
import sys, os, subprocess, time, json
sys.path.insert(0, '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-8/scripts')
os.chdir('/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-8/scripts')
exec(open('s8r-03i.py').read().split('os.makedirs("/tmp/qc2-s8r"')[0])
import s8r_chat
OUTF = '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-3/logs/qc2-s3-scenario.txt'; LOGL = []
def rec(s): LOGL.append(s); print(s, flush=True)
if not acquire(): sys.exit('no lock')
stub = subprocess.Popen([sys.executable, 's8r-stub.py', '9764']); time.sleep(1.5)
logl("LOCK — P104-46 kịch bản: CHAT → QC2S-*, stub :9764; ≤ 8 phút"); old = routes()["CHAT"]; ids = []
try:
    pv = lambda: call("GET", "/admin/llm/providers", "admin")[1]
    def chat4():
        out = []
        for i in range(4):
            t0 = s8r_chat.now_db(); s8r_chat.ask("sv.gioi", s8r_chat.cid("QD8R6U"), f"Cho mình hỏi lịch học tuần này, lần {i}?"); time.sleep(0.5)
            r = sql(f"select provider, model, fallback_index, status from llm_audit where created_at >= '{t0}' and task='CHAT' order by created_at desc limit 1"); out.append(tuple(r[0]) if r else None)
        return out
    setroute_c = lambda chain: call("PUT", "/admin/llm/routes", "admin", {"task": "CHAT", "chain": chain, "params": {}, "version": routes()["CHAT"]["version"]}, headers={"If-Match": 'W/"v%d"' % routes()["CHAT"]["version"]})
    c1, d1 = call("POST", "/admin/llm/providers", "admin", {"type": "fake", "name": "QC2S-Chinh", "api_key": "good-key", "models": [{"model": "fake-chat", "kind": "chat", "price_in": "0", "price_out": "0"}]}); rec(f"3 POST Chinh {c1} has_key={d1.get('has_key')} key_status={d1.get('key_status')} last_test.ok={(d1.get('last_test') or {}).get('ok')} api_key_in_body={'api_key' in json.dumps(d1)}")
    c2, d2 = call("POST", "/admin/llm/providers", "admin", {"type": "fake", "name": "QC2S-Phu", "api_key": "good-key", "models": [{"model": "fake-chat-2", "kind": "chat", "price_in": "0", "price_out": "0"}]}); rec(f"4 POST Phu {c2}")
    P = {p["name"]: p for p in pv()["items"]}; ids = [P["QC2S-Chinh"]["id"], P["QC2S-Phu"]["id"]]; mC = P["QC2S-Chinh"]["models"][0]["id"]; mP = P["QC2S-Phu"]["models"][0]["id"]
    rec(f"5 PUT CHAT=[Chinh] {setroute_c([mC])[0]}"); time.sleep(2)
    rec(f"6 CHAT=[Phu,Chinh] {setroute_c([mP, mC])[0]}; chat×4: {chat4()}"); 
    c, d = call("POST", "/admin/llm/providers/test", "admin", {"type": "openai_compatible", "base_url": "http://host.docker.internal:20128/v1", "api_key": "sai-khoa", "model": "ag/gemini-3.8-flash"}); rec(f"7 test khoá sai {c} {json.dumps(d, ensure_ascii=False)[:200]}")
    c, d = call("POST", "/admin/llm/providers", "admin", {"type": "openai_compatible", "name": "QC2S-Sai", "base_url": "http://host.docker.internal:20128/v1", "api_key": "sai-khoa", "models": [{"model": "ag/gemini-3.8-flash", "kind": "chat", "price_in": "0", "price_out": "0"}]}); rec(f"8 POST Sai {c} {json.dumps(d, ensure_ascii=False)[:230]}")
    body = json.dumps(pv(), ensure_ascii=False); rec(f"9 GET providers: có QC2S-Sai {'QC2S-Sai' in body}; chuỗi sai-khoa {'sai-khoa' in body}; good-key {'good-key' in body}")
    cur = [p for p in pv()["items"] if p["name"] == "QC2S-Phu"][0]
    def toggle(en): return call("PUT", f"/admin/llm/providers/{ids[1]}", "admin", {"type": "fake", "name": "QC2S-Phu", "enabled": en, "models": [{"model": "fake-chat-2", "kind": "chat", "price_in": "0", "price_out": "0"}], "version": [p for p in pv()["items"] if p["id"] == ids[1]][0]["version"]})
    rec(f"10 tắt Phu {toggle(False)[0]}; chat×4: {(time.sleep(2), chat4())[1]}")
    rec(f"11 bật Phu {toggle(True)[0]}; chat×4: {(time.sleep(2), chat4())[1]}")
    sp = call("POST", "/admin/llm/providers", "admin", {"type": "openai_compatible", "name": "QC2S-Q", "base_url": "http://host.docker.internal:9764/v1", "api_key": "good-key", "models": [{"model": "m-ext", "kind": "chat", "price_in": "0", "price_out": "0"}]}); rec(f"12a POST Q {sp[0]}")
    P = {p["name"]: p for p in pv()["items"]}; ids.append(P["QC2S-Q"]["id"]); mQ = P["QC2S-Q"]["models"][0]["id"]
    setmode("lop1"); rec(f"12 CHAT=[Q,Chinh] {setroute_c([mQ, mC])[0]}; chat×4: {(time.sleep(2), chat4())[1]}")
    setmode("fail503"); open(REQ, "w").close(); rec(f"13 stub trả lỗi: chat×4: {(time.sleep(1), chat4())[1]}; stub nhận {nreq()} yêu cầu")
    rec("13b audit: " + str(sql("select provider, model, fallback_index, status from llm_audit where provider like 'QC2S%' order by created_at desc limit 6")))
finally:
    try: print("restore", call("PUT", "/admin/llm/routes", "admin", {"task": "CHAT", "chain": [c["model_id"] for c in old["chain"]], "params": old.get("params") or {}, "version": routes()["CHAT"]["version"]}, headers={"If-Match": 'W/"v%d"' % routes()["CHAT"]["version"]})[0])
    except Exception as e: print("restore err", e)
    for i in ids: print("del", call("DELETE", f"/admin/llm/providers/{i}", "admin")[0])
    stub.terminate(); setmode("lop1"); open(OUTF, "w").write("\n".join(LOGL))
    logl("UNLOCK — P104-46 xong; route CHAT hoàn lại, 3 nhà QC2S-* đã xoá")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
