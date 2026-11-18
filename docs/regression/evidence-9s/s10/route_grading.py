#!/usr/bin/env python3
"""route_grading.py stub|real — đổi tuyến GRADING giữa Q10-stub và QC 9router (giữ /tmp/qc-global.lock)."""
import sys, os; sys.path.insert(0, os.path.expanduser("~/Documents/TA_Agent_qc106/docs/sprints/11/qc/scripts"))
from p10lib import *
cmd = sys.argv[1]
c, j, _, _ = api("admin", "GET", "/admin/llm/providers"); ps = {p["name"]: p for p in j["items"]}
p = ps["Q10-stub" if cmd == "stub" else "QC 9router"]; mid = [m["id"] for m in p["models"] if m["model"] == ("stub10" if cmd == "stub" else "ag/gemini-3.8-flash")][0]
c, rt, _, _ = api("admin", "GET", "/admin/llm/routes")
for r in rt["items"]:
    if r["task"] == "GRADING": print(api("admin", "PUT", "/admin/llm/routes", {"task": "GRADING", "chain": [mid], "params": r["params"], "version": r["version"]})[0])
