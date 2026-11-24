# PM 24/11: OpenCode Go có embeddings ở /inference/go/openai/v1. Đổi base_url, thêm text-embedding-3-small (1536), tuyến EMBEDDING, Test chat + embed.
import os, sys, json
sys.path.insert(0, "../../evidence-10.5/s10x2"); os.environ.setdefault("SEED_DEFAULT_PASSWORD", "Edupilot#Seed-2026")
import qlib; qlib.raw.__defaults__ = (None, None, None, "https://localhost:1143/api/v1", 60)
api = qlib.api
NEW = "https://opencode.ai/inference/go/openai/v1"
c, j, _ = api("admin", "GET", "/admin/llm/providers"); pr = next(p for p in (j.get("items") or j.get("providers")) if p["name"] == "OpenCode Go")
ms = [{"model": m["model"], "kind": m["kind"], "price_in": str(m.get("price_in", "0")), "price_out": str(m.get("price_out", "0")), "enabled": m.get("enabled", True), **({"dims": m["dims"]} if m.get("dims") else {})} for m in pr["models"]]
if not any(m["model"] == "text-embedding-3-small" for m in ms):
    ms.append({"model": "text-embedding-3-small", "kind": "embedding", "dims": 1536, "price_in": "0", "price_out": "0", "enabled": True})
for t in range(5):  # lưu kèm kiểm tra kết nối (probe cố định 10 s; chat model 8–20 s nên có lượt quá hạn)
    c, o, _ = api("admin", "PUT", f"/admin/llm/providers/{pr['id']}", {"type": pr["type"], "name": pr["name"], "base_url": NEW, "enabled": True, "models": ms, "version": pr["version"], **({"skip_verify": True} if t >= 3 else {})})
    print("PUT provider try", t + 1, c, "" if c == 200 else json.dumps(o, ensure_ascii=False)[:250])
    if c == 200: break
    c2, j2, _ = api("admin", "GET", "/admin/llm/providers"); pr = next(p for p in (j2.get("items") or j2.get("providers")) if p["name"] == "OpenCode Go")
assert c == 200
c, j, _ = api("admin", "GET", "/admin/llm/providers"); pr = next(p for p in (j.get("items") or j.get("providers")) if p["name"] == "OpenCode Go"); print("base_url", pr["base_url"])
em = next(m for m in pr["models"] if m["model"] == "text-embedding-3-small"); print("embed model", em["id"], em.get("dims"))
c, t, _ = api("admin", "POST", f"/admin/llm/providers/{pr['id']}/test", {}); print("test chat", c, t)
c, t, _ = api("admin", "POST", f"/admin/llm/providers/{pr['id']}/test", {"model": "text-embedding-3-small"}); print("test embed", c, t)
c, r, _ = api("admin", "GET", "/admin/llm/routes"); e = next(i for i in r["items"] if i["task"] == "EMBEDDING"); print("EMBEDDING trước", e["chain"], e["version"], r["embedding"])
if t.get("ok") and "--apply" in sys.argv:
    c, o, _ = api("admin", "PUT", "/admin/llm/routes", {"task": "EMBEDDING", "chain": [em["id"]], "params": {}, "version": e["version"]}); print("PUT EMBEDDING", c, json.dumps(o, ensure_ascii=False)[:300])
    c, r, _ = api("admin", "GET", "/admin/llm/routes"); print("sau", r["embedding"])
