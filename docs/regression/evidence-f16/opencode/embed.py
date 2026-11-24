# Thêm model embedding text-embedding-3-small (dims 1536 = cột vector hiện có) vào OpenCode Go, đặt tuyến EMBEDDING.
import os, sys, json
sys.path.insert(0, "../../evidence-10.5/s10x2"); os.environ.setdefault("SEED_DEFAULT_PASSWORD", "Edupilot#Seed-2026")
import qlib; qlib.raw.__defaults__ = (None, None, None, os.environ.get("QB", "https://localhost:1143/api/v1"), 60)
api = qlib.api
c, j, _ = api("admin", "GET", "/admin/llm/providers"); pr = next(p for p in (j.get("items") or j.get("providers")) if p["name"] == "OpenCode Go")
models = [{"model": m["model"], "kind": m["kind"], "price_in": str(m.get("price_in", "0")), "price_out": str(m.get("price_out", "0")), "enabled": m.get("enabled", True),
           **({"dims": m["dims"]} if m.get("dims") else {})} for m in pr["models"]]
if not any(m["model"] == "text-embedding-3-small" for m in models):
    models.append({"model": "text-embedding-3-small", "kind": "embedding", "dims": 1536, "price_in": "0", "price_out": "0", "enabled": True})
    c, o, _ = api("admin", "PUT", f"/admin/llm/providers/{pr['id']}", {"type": pr["type"], "name": pr["name"], "base_url": pr["base_url"], "enabled": True, "models": models, "version": pr["version"]})
    print("put provider", c, "" if c == 200 else json.dumps(o, ensure_ascii=False)[:300])
c, j, _ = api("admin", "GET", "/admin/llm/providers"); pr = next(p for p in (j.get("items") or j.get("providers")) if p["name"] == "OpenCode Go")
em = next(m for m in pr["models"] if m["model"] == "text-embedding-3-small"); print("embed model", em["id"], em.get("dims"))
c, t, _ = api("admin", "POST", f"/admin/llm/providers/{pr['id']}/test", {"model": "text-embedding-3-small"}); print("test embed", c, t)
c, r, _ = api("admin", "GET", "/admin/llm/routes"); e = next(i for i in r["items"] if i["task"] == "EMBEDDING"); print("embedding route trước", e["chain"], e["version"], r["embedding"])
if "--apply" in sys.argv:
    c, o, _ = api("admin", "PUT", "/admin/llm/routes", {"task": "EMBEDDING", "chain": [em["id"]], "params": {}, "version": e["version"]}); print("PUT EMBEDDING", c, json.dumps(o, ensure_ascii=False)[:400])
    c, r, _ = api("admin", "GET", "/admin/llm/routes"); print("sau", r["embedding"])
