# Đổi tuyến đang trỏ «QC 9router» sang OpenCode Go / deepseek-v4.1-flash (giữ provider 9router).
import os, sys, json
sys.path.insert(0, "../../evidence-10.5/s10x2")
os.environ.setdefault("SEED_DEFAULT_PASSWORD", "Edupilot#Seed-2026")
import qlib
qlib.raw.__defaults__ = (None, None, None, os.environ.get("QB", "https://localhost:1143/api/v1"), 60)
api = qlib.api
ids = json.load(open(os.environ.get("IDS", "ids.json")))
c, r, _ = api("admin", "GET", "/admin/llm/routes"); assert c == 200
for it in r["items"]:
    names = [x["provider_name"] for x in it["chain"]]
    print("trước", it["task"], names, it["version"])
    if "QC 9router" in names:
        c, o, _ = api("admin", "PUT", "/admin/llm/routes", {"task": it["task"], "chain": [ids["model"]], "params": it.get("params") or {}, "version": it["version"]})
        print("  PUT", c, "" if c == 200 else json.dumps(o, ensure_ascii=False)[:200])
c, r, _ = api("admin", "GET", "/admin/llm/routes")
for it in r["items"]: print("sau", it["task"], [(x["provider_name"], x["model"]) for x in it["chain"]])
print({k: v for k, v in r.items() if k != "items"})
