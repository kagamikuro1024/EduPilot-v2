"""QC kiểm chứng giaovien: route 6 tác vụ sinh chữ → 9router (ag/gemini-3.8-flash). Embedding giữ `fake` (không đụng). Khoá 9router đọc từ sqlite cục bộ, không in."""
import json, os, ssl, subprocess, urllib.request, re
API = "https://localhost:943/api/v1"; MODEL = "ag/gemini-3.8-flash"
pw = re.search(r"^SEED_DEFAULT_PASSWORD=(.*)$", open(os.path.expanduser("~/Documents/TA_Agent_qc5/.env.local")).read(), re.M).group(1)
ctx = ssl._create_unverified_context()
def call(m, p, tok=None, body=None, idem=None):
    h = {"Content-Type": "application/json", "Origin": "https://localhost:943"}
    if tok: h["Authorization"] = "Bearer " + tok
    if idem: h["Idempotency-Key"] = idem
    r = urllib.request.Request(API + p, method=m, headers=h, data=None if body is None else json.dumps(body).encode())
    try:
        with urllib.request.urlopen(r, context=ctx, timeout=60) as x: t = x.read(); return x.status, json.loads(t) if t else None
    except urllib.error.HTTPError as e: return e.code, json.loads(e.read() or b"null")
c, j = call("POST", "/auth/login", body={"email": "admin@edupilot.local", "password": pw}); tok = j["access_token"]
key = subprocess.run(["sqlite3", "-readonly", os.path.expanduser("~/.9router/db/data.sqlite"), "select key from apiKeys where isActive=1 limit 1;"], capture_output=True, text=True).stdout.strip()
assert key, "9router không có khoá"
c, j = call("GET", "/admin/llm/providers", tok); have = {p["name"]: p for p in j["items"]}
if "9router" not in have:
    c, p = call("POST", "/admin/llm/providers", tok, {"type": "openai_compatible", "name": "9router", "base_url": "http://host.docker.internal:20128/v1", "api_key": key, "enabled": True, "rpm_limit": 20,
        "models": [{"model": MODEL, "kind": "chat", "price_in": "0", "price_out": "0", "enabled": True}]}, "kc-9router"); print("tạo provider", c, p.get("key_status"), p.get("code"))
else: p = have["9router"]; print("provider có sẵn")
mid = [m["id"] for m in p["models"] if m["model"] == MODEL][0]
c, j = call("GET", "/admin/llm/routes", tok); routes = j.get("items") or j.get("routes") or j
vers = {r["task"]: r.get("version", 1) for r in routes} if isinstance(routes, list) else {}
for t in ("CHAT", "CLASSIFY", "UTILITY", "GRADING", "QUESTION_GEN", "INSIGHT"):
    c, j = call("PUT", "/admin/llm/routes", tok, {"task": t, "chain": [mid], "version": vers.get(t, 1)}); print(t, c, j.get("code") if isinstance(j, dict) and c != 200 else "")
c, j = call("GET", "/admin/llm/routes", tok); print(json.dumps(j, ensure_ascii=False)[:700])
