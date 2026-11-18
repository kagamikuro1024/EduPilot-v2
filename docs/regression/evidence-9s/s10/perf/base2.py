import sys, os, subprocess, time, re
sys.path.insert(0, os.path.expanduser("~/Documents/TA_Agent_qc106/docs/sprints/11/qc/scripts")); sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from p10lib import *
D = os.path.dirname(os.path.abspath(__file__))
def prov(): return {p["name"]: p for p in api("admin", "GET", "/admin/llm/providers")[1]["items"]}
def setroute(task, name, model):
    p = prov()[name]; mid = [m["id"] for m in p["models"] if m["model"] == model][0]
    r = [x for x in api("admin", "GET", "/admin/llm/routes")[1]["items"] if x["task"] == task][0]
    return api("admin", "PUT", "/admin/llm/routes", {"task": task, "chain": [mid], "params": r["params"], "version": r["version"]})[0]
CID = sql("select id from courses where class_code='QD10A'")[0][0]
try:
    for _ in range(60):
        n = sql("select count(*) from jobs where kind='question.extract' and status in ('RUNNING','QUEUED')")[0][0]
        if n == "0": break
        time.sleep(5)
    print("việc extract còn chạy:", n); setroute("CHAT", "Q10-stub", "stub10"); open("/tmp/stub10_mode", "w").write("ok")
    toks = ",".join(tok(f"qcperf{i:03d}") for i in range(1, 21)); outs = []
    for tag in ("baseA", "baseB"):
        r = subprocess.run(["k6", "run", "--insecure-skip-tls-verify", "-e", "BASE=https://localhost:943", "-e", f"COURSE={CID}", "-e", f"TOKENS={toks}", "-e", "DUR=60s", "-e", f"TAG={tag}", os.path.join(D, "chat-ttft.js")], capture_output=True, text=True); o = r.stdout + r.stderr
        m = re.search(r"ttft_ms.*?p\(95\)=([\d.]+)(ms|s)", o); print(tag, "TTFT p95", m.group(1) + m.group(2) if m else None)
finally:
    setroute("CHAT", "QC 9router", "ag/gemini-3.8-flash")
