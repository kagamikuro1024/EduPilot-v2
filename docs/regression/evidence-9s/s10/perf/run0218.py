#!/usr/bin/env python3
# TC-02-18: chat 1 QPS (stub CHAT, nhanh) — nền 60 s rồi 60 s có việc extract chạy nền (stub chậm cho QUESTION_GEN). CẦN /tmp/qc-global.lock.
import sys, os, subprocess, time, json, re
sys.path.insert(0, os.path.expanduser("~/Documents/TA_Agent_qc106/docs/sprints/11/qc/scripts")); sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from p10lib import *
from pm import chk
D = os.path.dirname(os.path.abspath(__file__)); S = os.path.dirname(D)
def prov(): return {p["name"]: p for p in api("admin", "GET", "/admin/llm/providers")[1]["items"]}
def setroute(task, name, model):
    p = prov()[name]; mid = [m["id"] for m in p["models"] if m["model"] == model][0]
    rt = api("admin", "GET", "/admin/llm/routes")[1]["items"]; r = [x for x in rt if x["task"] == task][0]
    return api("admin", "PUT", "/admin/llm/routes", {"task": task, "chain": [mid], "params": r["params"], "version": r["version"]})[0]
env = dict(os.environ, STUB_MODE_FILE="/tmp/stub10b_mode", STUB_REQ_FILE="/tmp/stub10b_req.jsonl"); open("/tmp/stub10b_mode", "w").write("slow8"); open("/tmp/stub10_mode", "w").write("ok")
st2 = subprocess.Popen(["python3", os.path.join(S, "stub_s10.py"), "9762"], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL); time.sleep(1)
if "Q10-stub2" not in prov(): print("tạo stub2", api("admin", "POST", "/admin/llm/providers", {"name": "Q10-stub2", "type": "openai_compatible", "base_url": "http://host.docker.internal:9762/v1", "api_key": "stub", "enabled": True, "models": [{"model": "stub10", "kind": "chat", "price_in": "0", "price_out": "0", "enabled": True}]})[0])
CID = sql("select id from courses where class_code='QD10A'")[0][0]; toks = ",".join(tok(f"qcperf{i:03d}") for i in range(1, 21))
try:
    print("CHAT→stub", setroute("CHAT", "Q10-stub", "stub10"), "QUESTION_GEN→stub2", setroute("QUESTION_GEN", "Q10-stub2", "stub10"))
    def k6(tag): 
        r = subprocess.run(["k6", "run", "--insecure-skip-tls-verify", "-e", f"BASE=https://localhost:943", "-e", f"COURSE={CID}", "-e", f"TOKENS={toks}", "-e", "DUR=60s", "-e", f"TAG={tag}", os.path.join(D, "chat-ttft.js")], capture_output=True, text=True)
        o = r.stdout + r.stderr; m = re.search(r"ttft_ms.*?p\(95\)=([\d.]+)(ms|s)", o); f = re.search(r"first_event_ms.*?p\(95\)=([\d.]+)(ms|s)", o); n = re.search(r"ttft_samples\.*:\s*(\d+)", o); fail = re.search(r"checks_failed\.*:\s*([\d.]+%)", o)
        g = lambda x: (float(x.group(1)) * (1000 if x.group(2) == "s" else 1)) if x else None
        return g(m), g(f), int(n.group(1)) if n else 0, fail.group(1) if fail else "?", o[-600:]
    base = k6("base"); print("NỀN", base[:4])
    CD = sql("select id from courses where class_code='QD10D'")[0][0]; DD = sql(f"select id from documents where course_id='{CD}' and status='READY' order by created_at desc limit 1")[0][0]
    rj = api("teacher", "POST", f"/courses/{CD}/questions/extract", {"document_id": DD}); print("extract nền", rj[0], str(rj[1])[:80]); time.sleep(8)
    load = k6("load"); print("TẢI", load[:4]); jid = rj[1].get("job_id"); jst = api("teacher", "GET", f"/jobs/{jid}")[1]; print("job nền sau k6:", jst.get("status"), jst.get("progress"))
    t0 = sql("select count(*) from jobs where kind='question.extract' and created_at > now() - interval '5 minutes'")[0][0]
    ratio = (load[0] / base[0]) if base[0] and load[0] else None
    chk("TC-02-18", ratio is not None and ratio <= 1.2 and base[2] >= 50 and load[2] >= 50, f"k6 chat 1 QPS (20 SV qcperf, stub CHAT nhanh): nền 60 s TTFT p95 {base[0]} ms ({base[2]} mẫu, lỗi {base[3]}); trong lúc 1 việc extract đề 60 trang chạy nền (stub chậm 8 s/lời gọi; trạng thái việc khi k6 xong: {jst.get('status')} {jst.get('progress')}%): TTFT p95 {load[0]} ms ({load[2]} mẫu, lỗi {load[3]}); tỉ lệ {ratio and round(ratio, 2)} (≤1,20)")
finally:
    setroute("CHAT", "QC 9router", "ag/gemini-3.8-flash"); setroute("QUESTION_GEN", "QC 9router", "ag/gemini-3.8-flash"); st2.terminate()
