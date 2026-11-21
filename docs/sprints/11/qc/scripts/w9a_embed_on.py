# Đợt R · lượt embedding THẬT (P-11 PM duyệt: trần 2.000 lời gọi; chỉ tài liệu lớp 761987; decision vẫn giả)
# Khoá chỉ đọc từ file của PM, đẩy qua API admin (lưu mã hoá trong DB ep-qc), KHÔNG in, KHÔNG ghi log.
from rlib import *
import re, os, json, time
key = re.search(r"^GEMINI_API_KEY=(.+)$", open(os.path.expanduser("~/Documents/TA_Agent_v2-s95/.env.local")).read(), re.M).group(1).strip().strip('"').strip("'")
n0 = int(sql("select count(*) from llm_audit where task='EMBEDDING'")[0][0])
c, pv, _ = api("admin", "POST", "/admin/llm/providers", {"type": "gemini", "name": "gemini-embed-qcR", "api_key": key, "enabled": True, "rpm_limit": 60, "tpm_limit": 100000,
    "models": [{"model": "gemini-embedding-001", "kind": "embedding", "dims": 1536, "price_in": "0", "price_out": "0", "enabled": True}]})
del key
print("create provider:", c, {k: v for k, v in (pv or {}).items() if k in ("id", "name", "has_key", "key_status", "code", "message")})
pid = (pv or {}).get("id")
open("/tmp/r_embed_state.json", "w").write(json.dumps({"pid": pid, "n0": n0}))
if c in (200, 201):
    c, r, _ = api("admin", "GET", "/admin/llm/providers"); p = [x for x in r["items"] if x["id"] == pid][0]
    mid = [m["id"] for m in p["models"] if m["kind"] == "embedding"][0]
    c, rt, _ = api("admin", "GET", "/admin/llm/routes"); v = [x for x in rt["items"] if x["task"] == "EMBEDDING"][0]["version"]
    c, pr, _ = api("admin", "PUT", "/admin/llm/routes", {"task": "EMBEDDING", "chain": [mid], "params": {}, "version": v})
    print("put route:", c, {k: pr.get(k) for k in ("reindex_required", "code", "message")} if isinstance(pr, dict) else pr)
    print("llm_audit EMBEDDING trước:", n0)
