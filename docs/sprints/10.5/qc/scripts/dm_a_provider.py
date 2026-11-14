# L1 còn lại: API provider (#6) — TC-D01-19, 20, 26, 27, D02-13 (baseline), D02-06 (mặc định tắt)
from dmx import *
since = now_db()
c, j, _ = api("admin", "GET", "/admin/llm/providers")
ok("A0 admin GET providers", c == 200, str(c))
dec = [p for p in (j.get("providers") or j.get("items") or []) if p.get("type") == "decision"]
ok("A1 có đúng 1 provider type=decision", len(dec) == 1, f"{len(dec)}")
p = dec[0]; pid = p["id"]; ver = p.get("version")
print("provider:", json.dumps({k: p.get(k) for k in ("id", "name", "type", "enabled", "base_url", "version", "key_hint", "has_key")}, ensure_ascii=False), [(m["model"], m.get("enabled")) for m in p["models"]])
ok("A1b decision-fake bật, gemini-decision-1 không bật", any(m["model"] == "decision-fake" and m.get("enabled") for m in p["models"]) and not any(m["model"] == "gemini-decision-1" and m.get("enabled") for m in p["models"]), "")

# D01-19: chọn bản thật khi ALLOW_REAL=false
models = [{"model": m["model"], "kind": m["kind"], "price_in": str(m["price_in"]), "price_out": str(m["price_out"]),
           "enabled": (m["model"] == "gemini-decision-1")} for m in p["models"]]
c, j2, _ = api("admin", "PUT", f"/admin/llm/providers/{pid}", {"type": "decision", "name": p["name"], "enabled": True, "models": models, "version": ver})
ok("D01-19 chọn gemini-decision-1 khi ALLOW_REAL=false → 409 DECISION_REAL_DISABLED", c == 409 and "DECISION_REAL_DISABLED" in json.dumps(j2), f"{c} {json.dumps(j2, ensure_ascii=False)[:160]}")
c, j3, _ = api("admin", "GET", "/admin/llm/providers")
p3 = [x for x in (j3.get("providers") or j3.get("items") or []) if x["id"] == pid][0]
ok("D01-19b mô hình KHÔNG đổi sau 409", not any(m["model"] == "gemini-decision-1" and m.get("enabled") for m in p3["models"]), "")
na = sql(f"select count(*) from audit_log where entity ilike '%provider%' and created_at>='{since}'")[0][0]
print("audit_log provider rows since:", na)

# thử tạo provider decision thứ hai (unique)
c, j4, _ = api("admin", "POST", "/admin/llm/providers", {"type": "decision", "name": "QC hai", "enabled": True, "models": [{"model": "decision-fake", "kind": "decision", "price_in": "0", "price_out": "0", "enabled": True}], "skip_verify": True})
ok("D01-28b tạo provider decision thứ hai bị từ chối (4xx, không 5xx)", 400 <= c < 500, f"{c} {json.dumps(j4, ensure_ascii=False)[:140]}")

# D01-26: vai khác
for u in ("teacher", "ta", "sv.kha"):
    c, j5, _ = api(u, "GET", "/admin/llm/providers")
    vis = None
    if c == 200: vis = any(x.get("type") == "decision" for x in (j5.get("providers") or j5.get("items") or []))
    print(f"INFO D01-26 {u}: GET providers → {c}; thấy provider decision: {vis}")
    ok(f"D01-26 {u} không thấy provider decision (403 hoặc ẩn)", c in (401, 403) or vis is False, f"{c} vis={vis}")
    c, _, _ = api(u, "PUT", f"/admin/llm/providers/{pid}", {"type": "decision", "name": "x", "version": ver})
    ok(f"D01-26b {u} PUT provider decision → 403", c == 403, str(c))
    c, _, _ = api(u, "POST", f"/admin/llm/providers/{pid}/test", {})
    ok(f"D01-26c {u} POST test → 403", c == 403, str(c))
c, _, _ = api(None, "GET", "/admin/llm/providers")
ok("D01-26d không token → 401", c == 401, str(c))

# D01-27: test kết nối (fake)
c, j6, ms = api("admin", "POST", f"/admin/llm/providers/{pid}/test", {})
ok("D01-27 test kết nối decision-fake → 200", c == 200, f"{c} {ms:.0f}ms {json.dumps(j6, ensure_ascii=False)[:140]}")

# baseline: mặc định tắt, 0 lời gọi DECIDE từ seed
n = sql("select count(*) from llm_audit where task='DECIDE'")[0][0]
print("INFO tổng dòng DECIDE sau test kết nối:", n)
g = sql("select gate,enabled,threshold,timeout_ms from decision_gates order by gate")
ok("D02-13 mặc định ba cổng tắt, ngưỡng đúng", g == [["CHAT_ROUTING", "f", "0.950", "800"], ["INJECTION_FLAG", "f", "0.500", "3000"], ["THREADS_PRECHECK", "f", "0.050", "800"]], str(g))
ok("D02-06 không hàng course_decision_gates bật", sql("select count(*) from course_decision_gates where enabled")[0][0] == "0", "")
