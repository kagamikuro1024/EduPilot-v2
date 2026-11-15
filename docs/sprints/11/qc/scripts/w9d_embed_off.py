from rlib import *
import json, time
st = json.load(open("/tmp/r_embed_state.json")); pid = st["pid"]
print("gemini rows/attempts/ok:", sql("select count(*), coalesce(sum(attempts),0), count(*) filter (where status='ok'), coalesce(sum(tokens_in),0) from llm_audit where provider='gemini-embed-qcR'"))
c, rt, _ = api("admin", "GET", "/admin/llm/routes"); v = [x for x in rt["items"] if x["task"] == "EMBEDDING"][0]["version"]
c, pr, _ = api("admin", "PUT", "/admin/llm/routes", {"task": "EMBEDDING", "chain": [], "params": {}, "version": v}); print("route chain []:", c)
c, dl, _ = api("admin", "DELETE", f"/admin/llm/providers/{pid}"); print("delete provider:", c)
c, r, _ = api("admin", "GET", "/admin/llm/providers"); print("còn provider gemini:", [p["name"] for p in r["items"] if "gemini" in p["name"].lower()])
c1 = C("teacher")
c, dls, _ = api("teacher", "GET", f"/courses/{c1}/documents?limit=100")
for d in [d for d in dls["items"] if d.get("status") == "READY" and d.get("use_for_rag")]:
    c, j, _ = api("teacher", "POST", f"/courses/{c1}/documents/{d['id']}/reindex", {}); time.sleep(3)
print("audit sau reindex giả:", sql("select provider, count(*) from llm_audit where task='EMBEDDING' group by 1 order by 2 desc"))
