# dùng: python3 setroute.py TASK "QC fake"|"QC 9router"  (chỉ đổi 1 tuyến; gọi dưới khoá /tmp/qc2-lock-global)
import sys
from qlib import *
task, prov = sys.argv[1], sys.argv[2]
c, j, _ = api("admin", "GET", "/admin/llm/providers"); p = [x for x in j["items"] if x["name"] == prov][0]
mid = p["models"][0]["id"]
c, rt, _ = api("admin", "GET", "/admin/llm/routes"); r = [x for x in rt["items"] if x["task"] == task][0]
c, j, _ = api("admin", "PUT", "/admin/llm/routes", {"task": task, "chain": [mid], "params": r["params"], "version": r["version"]}); print(task, prov, c)
