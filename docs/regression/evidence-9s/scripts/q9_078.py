#!/usr/bin/env python3
"""TC-P7-07-08: k6 grading-vs-chat thu nhỏ (4 luồng chat, 22 bài chấm bằng stub) + làn trong llm_audit."""
import sys, os, json, subprocess; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
K = json.load(open("/tmp/q9s/k6.json")); base = json.load(open("/tmp/q9s/k6-base.json"))["metrics"]["ttft_ms"]; b95 = base.get("p(95)", base.get("values", {}).get("p(95)"))
env = dict(os.environ, K6_INSECURE_SKIP_TLS_VERIFY="true")
t0 = psql("select now()")
with stub_ctx():
    r = subprocess.run(["k6", "run", "--quiet", os.path.expanduser("~/Documents/TA_Agent_qc106/benchmarks/load/grading-vs-chat.js"), "-e", "BASE=https://localhost:943", "-e", f"COURSE_ID={K['course']}", "-e", f"ASSIGNMENT_ID={K['aid']}", "-e", "PHASE=grading", "-e", "VUS=4", "-e", "CHAT_CLASS=QD9V", "-e", f"SEED_DEFAULT_PASSWORD={os.environ['SEED_DEFAULT_PASSWORD']}", "-e", f"BASELINE_TTFT_MS={b95}", "-e", "OUT=/tmp/q9s/k6-grading.json", "-e", 'META={"by":"Smoke9Runner","stack":"ep-qc"}'], capture_output=True, text=True, env=env)
    wait_done(K["aid"], 150)
print("k6 rc", r.returncode, r.stderr[-300:])
d = json.load(open("/tmp/q9s/k6-grading.json")); M = d["metrics"]
g = lambda k, f: (M.get(k, {}).get("values", M.get(k, {})) or {}).get(f)
print("baseline ttft p95", b95, "| grading: first_event p95", g("first_event_ms", "p(95)"), "ttft p95", g("ttft_ms", "p(95)"), "queued", g("grading_queued", "count"), "http_req_failed", g("http_req_failed", "rate"))
print("ngưỡng:", json.dumps(d.get("thresholds"), ensure_ascii=False)[:400])
print(psql(f"select coalesce(task,'-')||' / '||coalesce(lane,'-')||' = '||count(*) from llm_audit where created_at > '{t0}' group by task, lane order by 1"))
