# Gỡ embedding thật: route EMBEDDING → provider fake (DB), DELETE provider Gemini (xoá khoá), reindex lại bằng embedding giả.
from rlib import *
import json, time
st = json.load(open("/tmp/r_embed_state.json")); pid = st["pid"]
print("Gemini rows/attempts/ok:", sql("select count(*), coalesce(sum(attempts),0), count(*) filter (where status='ok') from llm_audit where provider='gemini-embed-qcR' and created_at > now() - interval '%d hours'" % 6))
c, r, _ = api("admin", "GET", "/admin/llm/providers")
fk = [x for x in r["items"] if x["name"] == "fake-embed-qcR"]
if not fk:
    c, pv, _ = api("admin", "POST", "/admin/llm/providers", {"type": "fake", "name": "fake-embed-qcR", "enabled": True, "models": [{"model": "fake-embed", "kind": "embedding", "dims": 1536, "price_in": "0", "price_out": "0", "enabled": True}]})
    c, r, _ = api("admin", "GET", "/admin/llm/providers"); fk = [x for x in r["items"] if x["name"] == "fake-embed-qcR"]
mid = [m["id"] for m in fk[0]["models"]][0]
c, rt, _ = api("admin", "GET", "/admin/llm/routes"); v = [x for x in rt["items"] if x["task"] == "EMBEDDING"][0]["version"]
c, pr, _ = api("admin", "PUT", "/admin/llm/routes", {"task": "EMBEDDING", "chain": [mid], "params": {}, "version": v}); print("route → fake:", c)
c, d, _ = api("admin", "DELETE", f"/admin/llm/providers/{pid}"); print("delete gemini provider:", c)
c, r, _ = api("admin", "GET", "/admin/llm/providers"); print("provider còn:", [p["name"] for p in r["items"]])
c1 = C("teacher"); c, dls, _ = api("teacher", "GET", f"/courses/{c1}/documents?limit=100")
for d in [d for d in dls["items"] if d.get("status") == "READY" and d.get("use_for_rag")]:
    api("teacher", "POST", f"/courses/{c1}/documents/{d['id']}/reindex", {}); time.sleep(3)
time.sleep(5)
print("audit EMBEDDING theo provider:", sql("select provider, count(*), sum(attempts) from llm_audit where task='EMBEDDING' group by 1 order by 2 desc"))
