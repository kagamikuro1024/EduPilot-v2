#!/usr/bin/env python3
"""route10.py stub|real|status — đổi tuyến QUESTION_GEN + CHAT giữa Q10-stub (cổng 9761) và QC 9router. CẦN giữ /tmp/qc-global.lock khi đổi."""
import sys, os; sys.path.insert(0, os.path.expanduser("~/Documents/TA_Agent_qc106/docs/sprints/11/qc/scripts"))
from p10lib import *
TASKS = ["QUESTION_GEN", "CHAT"]
def provs():
    c, j, _, _ = api("admin", "GET", "/admin/llm/providers"); return {p["name"]: p for p in j["items"]}
cmd = sys.argv[1] if len(sys.argv) > 1 else "status"
ps = provs()
if cmd == "setup" and "Q10-stub" not in ps:
    print(api("admin", "POST", "/admin/llm/providers", {"name": "Q10-stub", "type": "openai_compatible", "base_url": "http://host.docker.internal:9761/v1", "api_key": "stub", "enabled": True, "models": [{"model": "stub10", "kind": "chat", "price_in": "0", "price_out": "0", "enabled": True}]})[0]); ps = provs()
if cmd in ("stub", "real"):
    p = ps["Q10-stub" if cmd == "stub" else "QC 9router"]; mid = [m["id"] for m in p["models"] if m["model"] == ("stub10" if cmd == "stub" else "ag/gemini-3.8-flash")][0]
    c, rt, _, _ = api("admin", "GET", "/admin/llm/routes")
    for r in rt["items"]:
        if r["task"] in TASKS: print(r["task"], api("admin", "PUT", "/admin/llm/routes", {"task": r["task"], "chain": [mid], "params": r["params"], "version": r["version"]})[0])
c, rt, _, _ = api("admin", "GET", "/admin/llm/routes")
print({r["task"]: [e["provider_name"] + "/" + e["model"] for e in r["chain"]] for r in rt["items"] if r["task"] in TASKS + ["GRADING"]})
