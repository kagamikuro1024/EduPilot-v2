# Thêm provider «OpenCode Go» qua admin API ep-r, đổi tuyến sang deepseek-v4.1-flash. Khoá đọc từ tệp, KHÔNG in/ghi.
import os, sys, json
sys.path.insert(0, "../../evidence-10.5/s10x2")
os.environ.setdefault("SEED_DEFAULT_PASSWORD", "Edupilot#Seed-2026")
import qlib
qlib.B = "https://localhost:1143/api/v1"
qlib.raw.__defaults__ = (None, None, None, qlib.B, 60)
api = qlib.api
KEY = open(os.path.expanduser("~/.config/edupilot-team/secrets/opencode-go.key")).read().strip()
c, j, _ = api("admin", "GET", "/admin/llm/providers"); assert c == 200, (c, j)
items = j.get("items") or j.get("providers") or []
ex = next((p for p in items if p["name"] == "OpenCode Go"), None)
if not ex:
    c, ex, ms = api("admin", "POST", "/admin/llm/providers", {"type": "openai_compatible", "name": "OpenCode Go", "base_url": "https://opencode.ai/inference/go/v1",
        "api_key": KEY, "enabled": True, "models": [{"model": "deepseek-v4.1-flash", "kind": "chat", "price_in": "0", "price_out": "0", "enabled": True}]})
    print("create", c, json.dumps(ex, ensure_ascii=False)[:300].replace(KEY, "***")); assert c == 201
pid = ex["id"]
c, t, _ = api("admin", "POST", f"/admin/llm/providers/{pid}/test", {}); print("test", c, t)
c, p, _ = api("admin", "GET", "/admin/llm/providers"); pr = next(p_ for p_ in (p.get("items") or p.get("providers")) if p_["id"] == pid)
mid = next(m["id"] for m in pr["models"] if m["model"] == "deepseek-v4.1-flash"); print("model", mid)
c, r, _ = api("admin", "GET", "/admin/llm/routes"); assert c == 200
print(json.dumps(r, ensure_ascii=False)[:900])
json.dump({"provider": pid, "model": mid}, open("ids.json", "w"))
