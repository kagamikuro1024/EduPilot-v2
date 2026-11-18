# TC-P104-46 (kịch bản scenario-P1.md, QC chạy lại độc lập trên ep-r dưới khoá global). Tạo nhà QC2S-Chinh/Phu (fake) + QC2S-Q (openai_compatible → stub :9764).
import sys, os, subprocess, time, json
sys.path.insert(0, '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-8/scripts')
os.chdir('/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-8/scripts')
exec(open('s8r-03i.py').read().split('os.makedirs("/tmp/qc2-s8r"')[0])
import s8r_chat
OUTF = '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-3/logs/qc2-s3-degrade.txt'; LOGL = []
def rec(s): LOGL.append(s); print(s, flush=True)
if not acquire(): sys.exit('no lock')
stub = subprocess.Popen([sys.executable, 's8r-stub.py', '9764']); time.sleep(1.5)
logl("LOCK — P105-55 degrade: CHAT → QC2S-*, stub :9764; ≤ 8 phút"); old = routes()["CHAT"]; ids = []
try:
    pv = lambda: call("GET", "/admin/llm/providers", "admin")[1]
    setroute_c = lambda chain: call("PUT", "/admin/llm/routes", "admin", {"task": "CHAT", "chain": chain, "params": {}, "version": routes()["CHAT"]["version"]}, headers={"If-Match": 'W/"v%d"' % routes()["CHAT"]["version"]})
    sp = call("POST", "/admin/llm/providers", "admin", {"type": "openai_compatible", "name": "QC2S-Q", "base_url": "http://host.docker.internal:9764/v1", "api_key": "good-key", "models": [{"model": "m-ext", "kind": "chat", "price_in": "0", "price_out": "0"}]}); rec(f"POST Q {sp[0]}")
    P = {p["name"]: p for p in pv()["items"]}; ids.append(P["QC2S-Q"]["id"]); mQ = P["QC2S-Q"]["models"][0]["id"]
    rec(f"CHAT=[Q] chỉ một nhà {setroute_c([mQ])[0]}"); setmode("fail503"); time.sleep(2)
    t0 = s8r_chat.now_db(); sid, txt, ev = s8r_chat.ask("sv.gioi", s8r_chat.cid("QD8R6U"), "Cho mình hỏi lịch học tuần này, lần cuối?")
    done = [e[1] for e in ev if e[0] == "done"]; rec(f"tắt hết (nhà duy nhất lỗi 503): câu trả lời «{txt[:260]}»; done={json.dumps(done, ensure_ascii=False)[:200]}")
    rec("audit: " + str(sql(f"select provider, model, fallback_index, status, error_kind from llm_audit where created_at >= '{t0}' and task='CHAT' order by created_at desc limit 3")))
finally:
    try: print("restore", call("PUT", "/admin/llm/routes", "admin", {"task": "CHAT", "chain": [c["model_id"] for c in old["chain"]], "params": old.get("params") or {}, "version": routes()["CHAT"]["version"]}, headers={"If-Match": 'W/"v%d"' % routes()["CHAT"]["version"]})[0])
    except Exception as e: print("restore err", e)
    for i in ids: print("del", call("DELETE", f"/admin/llm/providers/{i}", "admin")[0])
    stub.terminate(); setmode("lop1"); open(OUTF, "w").write("\n".join(LOGL))
    logl("UNLOCK — P105-55 degrade xong; route CHAT hoàn lại, 3 nhà QC2S-* đã xoá")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
