# QC: tạo 2 nhà cung cấp (9router thật, fake) trong DB ep-qc và đổi tuyến chữ giữa chúng. Khoá 9router đọc từ ~/.9router (không in, không ghi file).
# Dùng: python3 llm_route.py setup | real | fake | status
import sys, sqlite3, shutil, os, json
from p10lib import *
MODEL = os.environ.get("QC_REAL_MODEL", "ag/gemini-3.8-flash")
TASKS = ["CHAT", "CLASSIFY", "UTILITY", "GRADING", "QUESTION_GEN", "INSIGHT"]

def key9():
    shutil.copy(os.path.expanduser("~/.9router/db/data.sqlite"), "/tmp/9r.sqlite")
    for ext in ("-wal", "-shm"):
        src = os.path.expanduser("~/.9router/db/data.sqlite" + ext)
        if os.path.exists(src): shutil.copy(src, "/tmp/9r.sqlite" + ext)
    return sqlite3.connect("/tmp/9r.sqlite").execute("select key from apiKeys where isActive=1 limit 1").fetchone()[0]

def provs():
    c, j, _, _ = api("admin", "GET", "/admin/llm/providers")
    return {p["name"]: p for p in j["items"]}

def mk(name, body):
    ps = provs()
    if name in ps: return ps[name]
    body = dict(body, name=name)
    c, j, _, _ = api("admin", "POST", "/admin/llm/providers", body)
    print("tạo", name, c, str(j)[:160] if c >= 300 else "")
    return provs()[name]

def model_id(p, model):
    c, j, _, _ = api("admin", "GET", f"/admin/llm/providers/{p['id']}")
    return [m for m in j["models"] if m["model"] == model][0]["id"]

def set_routes(prov, model):
    c, rt, _, _ = api("admin", "GET", "/admin/llm/routes")
    mid = model_id(prov, model)
    for r in rt["items"]:
        if r["task"] not in TASKS: continue
        c, j, _, _ = api("admin", "PUT", "/admin/llm/routes", {"task": r["task"], "chain": [mid], "params": r["params"], "version": r["version"]})
        print(r["task"], c, "" if c == 200 else str(j)[:200])

cmd = sys.argv[1]
if cmd == "setup":
    mk("QC 9router", {"type": "openai_compatible", "base_url": "http://host.docker.internal:20128/v1", "api_key": key9(), "enabled": True,
                      "models": [{"model": MODEL, "kind": "chat", "price_in": "0", "price_out": "0", "enabled": True}]})
    mk("QC fake", {"type": "fake", "api_key": "qc-fake", "enabled": True, "models": [{"model": "fake-chat", "kind": "chat", "price_in": "0", "price_out": "0", "enabled": True}]})
elif cmd in ("real", "fake"):
    p = provs()["QC 9router" if cmd == "real" else "QC fake"]
    set_routes(p, MODEL if cmd == "real" else "fake-chat")
c, rt, _, _ = api("admin", "GET", "/admin/llm/routes")
print({r["task"]: [e["provider_name"] + "/" + e["model"] for e in r["chain"]] for r in rt["items"]})
