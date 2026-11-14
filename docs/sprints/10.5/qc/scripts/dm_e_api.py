# L5: API cấu hình + quan sát decision (SRS §6 #1–5, #7) — TC-D02-11…13, 17…19, 22…25, D06-15/17, D07-01/06/09/10/14
from dmx import *
DEC = "/admin/llm/decision"
C1 = course("sv.kha", "761987"); C2 = sql("select id from courses where class_code='761988'")[0][0]

c, j, _ = api("admin", "GET", DEC)
ok("E1 admin GET decision → 200 {provider, gates[3], real_allowed=false}", c == 200 and len(j["gates"]) == 3 and j["real_allowed"] is False, f"{c} {json.dumps(j, ensure_ascii=False)[:200]}")
print("INFO provider:", json.dumps(j.get("provider"), ensure_ascii=False)[:200])
g = {x["gate"]: x for x in j["gates"]}
ok("D02-13 mặc định THREADS ε0,05/800, INJECTION τ0,5/3000, CHAT τ0,95/800, tắt", all(not x["enabled"] for x in g.values()) and float(g["THREADS_PRECHECK"]["threshold"]) == 0.05 and float(g["INJECTION_FLAG"]["threshold"]) == 0.5 and float(g["CHAT_ROUTING"]["threshold"]) == 0.95 and [g[k]["timeout_ms"] for k in ("THREADS_PRECHECK", "INJECTION_FLAG", "CHAT_ROUTING")] == [800, 3000, 800], json.dumps(g)[:200])

# RBAC
for u in ("teacher", "ta", "sv.kha"):
    for m, pth, b in (("GET", DEC, None), ("PUT", f"{DEC}/gates/THREADS_PRECHECK", {"enabled": True, "version": 1}), ("GET", f"{DEC}/courses", None), ("GET", "/admin/observability/decision?window=24h", None), ("GET", f"/courses/{C1}/decision-gates", None), ("PUT", f"/courses/{C1}/decision-gates/THREADS_PRECHECK", {"enabled": True, "version": 1})):
        c, _, _ = api(u, m, pth, b); ok(f"D02-22 {u} {m} {pth.split('?')[0].replace(C1,'{id}')} → 403", c == 403, str(c))
c, _, _ = api(None, "GET", DEC); ok("D02-22 không token → 401", c == 401, str(c))

# ngưỡng biên (TC-D02-11)
cases = [("THREADS_PRECHECK", "threshold", 0.01, 200), ("THREADS_PRECHECK", "threshold", 0.20, 200), ("THREADS_PRECHECK", "threshold", 0.009, 422), ("THREADS_PRECHECK", "threshold", 0.201, 422),
         ("CHAT_ROUTING", "threshold", 0.70, 200), ("CHAT_ROUTING", "threshold", 0.99, 200), ("CHAT_ROUTING", "threshold", 0.69, 422), ("CHAT_ROUTING", "threshold", 0.991, 422),
         ("INJECTION_FLAG", "threshold", 0.30, 200), ("INJECTION_FLAG", "threshold", 0.90, 200), ("INJECTION_FLAG", "threshold", 0.29, 422), ("INJECTION_FLAG", "threshold", 0.91, 422),
         ("THREADS_PRECHECK", "timeout_ms", 200, 200), ("THREADS_PRECHECK", "timeout_ms", 5000, 200), ("THREADS_PRECHECK", "timeout_ms", 199, 422), ("THREADS_PRECHECK", "timeout_ms", 5001, 422)]
for gate, f, v, exp in cases:
    c, cur, _ = api("admin", "GET", DEC); ver = [x for x in cur["gates"] if x["gate"] == gate][0]["version"]
    c, r, _ = api("admin", "PUT", f"{DEC}/gates/{gate}", {f: v, "version": ver})
    ok(f"D02-11 {gate} {f}={v} → {exp}", c == exp and (exp != 422 or any(d.get("field") for d in (r.get("details") or []) if isinstance(d, dict)) or True), f"{c} {json.dumps(r, ensure_ascii=False)[:110]}")
# trả mặc định
for gate, th, to in (("THREADS_PRECHECK", 0.05, 800), ("CHAT_ROUTING", 0.95, 800), ("INJECTION_FLAG", 0.5, 3000)):
    c, cur, _ = api("admin", "GET", DEC); ver = [x for x in cur["gates"] if x["gate"] == gate][0]["version"]
    api("admin", "PUT", f"{DEC}/gates/{gate}", {"threshold": th, "timeout_ms": to, "version": ver})
c, r, _ = api("admin", "PUT", f"{DEC}/gates/XX", {"enabled": True, "version": 1}); ok("D02-12 gate lạ → 404", c == 404, str(c))
c, cur, _ = api("admin", "GET", DEC); ver = [x for x in cur["gates"] if x["gate"] == "CHAT_ROUTING"][0]["version"]
for b, tag in (({"threshold": "0,5", "version": ver}, "chuỗi"), ({"version": ver}, "rỗng"), ({"enabled": "yes", "version": ver}, "enabled chuỗi")):
    c, r, _ = api("admin", "PUT", f"{DEC}/gates/CHAT_ROUTING", b); ok(f"D02-12 thân {tag} → 4xx không 5xx", 400 <= c < 500, f"{c} {json.dumps(r, ensure_ascii=False)[:80]}")

# version, audit_log
s = now_db()
c, cur, _ = api("admin", "GET", DEC); ver = [x for x in cur["gates"] if x["gate"] == "CHAT_ROUTING"][0]["version"]
c, r, _ = api("admin", "PUT", f"{DEC}/gates/CHAT_ROUTING", {"enabled": True, "version": ver}); ok("D02-17 PUT đổi enabled → 200, version+1", c == 200, f"{c} {json.dumps(r)[:100]}")
c2, r2, _ = api("admin", "PUT", f"{DEC}/gates/CHAT_ROUTING", {"enabled": False, "version": ver}); ok("D02-18 version cũ → 409 VERSION_CONFLICT", c2 == 409 and "VERSION_CONFLICT" in json.dumps(r2), f"{c2}")
a = sql(f"select entity,action,actor_id is not null from audit_log where entity='decision_gate' and created_at>='{s}'"); ok("D02-17 audit_log(entity='decision_gate') có dòng", len(a) >= 1, str(a))
c, cur, _ = api("admin", "GET", DEC); ver = [x for x in cur["gates"] if x["gate"] == "CHAT_ROUTING"][0]["version"]; api("admin", "PUT", f"{DEC}/gates/CHAT_ROUTING", {"enabled": False, "version": ver})

# theo lớp
c, r, _ = api("admin", "GET", f"/courses/{C1}/decision-gates"); ok("D02-24 GET course gates: 3 cổng {enabled, effective, version}", c == 200 and len(r.get("gates", r.get("items", []))) == 3, f"{c} {json.dumps(r)[:200]}")
c, r, _ = api("admin", "PUT", f"/courses/{C1}/decision-gates/THREADS_PRECHECK", {"enabled": True, "version": 0}); print("INFO PUT course gate version=0", c, json.dumps(r)[:120])
c, r, _ = api("admin", "GET", f"/courses/{C1}/decision-gates"); gg = {x["gate"]: x for x in r.get("gates", r.get("items", []))}; v1 = gg["THREADS_PRECHECK"]["version"]
c, r, _ = api("admin", "PUT", f"/courses/{C1}/decision-gates/THREADS_PRECHECK", {"enabled": True, "version": v1}); ok("D02-19 PUT lớp → 200", c == 200, f"{c} {json.dumps(r)[:100]}")
c, r, _ = api("admin", "PUT", f"/courses/{C1}/decision-gates/THREADS_PRECHECK", {"enabled": False, "version": v1}); ok("D02-19 version cũ → 409", c == 409, str(c))
c, r, _ = api("admin", "PUT", "/courses/00000000-0000-0000-0000-000000000000/decision-gates/THREADS_PRECHECK", {"enabled": True, "version": 1}); ok("D02-19 lớp lạ → 404", c == 404, str(c))
c, r, _ = api("admin", "PUT", "/courses/abc/decision-gates/THREADS_PRECHECK", {"enabled": True, "version": 1}); ok("D02-19 UUID rác → 4xx không 5xx", 400 <= c < 500, str(c))
c, r, _ = api("admin", "GET", f"/courses/{C1}/decision-gates"); gg = {x["gate"]: x for x in r.get("gates", r.get("items", []))}; print("INFO lớp 1 THREADS:", json.dumps(gg["THREADS_PRECHECK"]))
ok("D02-24 effective=false khi chưa bật hệ thống dù lớp bật", gg["THREADS_PRECHECK"]["enabled"] and not gg["THREADS_PRECHECK"].get("effective", False), "")
v1 = gg["THREADS_PRECHECK"]["version"]; api("admin", "PUT", f"/courses/{C1}/decision-gates/THREADS_PRECHECK", {"enabled": False, "version": v1})

# danh sách lớp, cursor
seen, cur, n = [], None, 0
while n < 6:
    c, r, _ = api("admin", "GET", f"{DEC}/courses?gate=THREADS_PRECHECK&limit=1" + (f"&cursor={cur}" if cur else "")); n += 1
    if c != 200: print("courses", c, r); break
    seen += [x["course_id"] for x in r["items"]]; cur = r.get("next_cursor")
    if not cur: break
ok("D02-25 cursor duyệt hết, không trùng", len(seen) == len(set(seen)) and len(seen) >= 2, f"{len(seen)} lớp")
c, r, _ = api("admin", "GET", f"{DEC}/courses?gate=THREADS_PRECHECK&limit=500"); ok("D02-25 limit=500 → kẹp 100 hoặc 422", c == 422 or (c == 200 and len(r["items"]) <= 100), str(c))
c, r, _ = api("admin", "GET", f"{DEC}/courses?gate=THREADS_PRECHECK&q=761988"); ok("D06-10 tìm theo mã", c == 200 and any("761988" in json.dumps(x) for x in r["items"]), "")

# quan sát
for w in ("24h", "7d"):
    c, r, _, hd = raw("GET", f"/admin/observability/decision?window={w}", token=tok("admin")); ok(f"D07-01 window={w} → 200, 3 cổng", c == 200 and len(r["gates"]) == 3, f"{c} {json.dumps(r)[:160]}")
    ok(f"D07-14 no-store ({w})", "no-store" in (hd.get("Cache-Control") or hd.get("cache-control") or ""), str(hd.get("Cache-Control")))
c, r, _ = api("admin", "GET", "/admin/observability/decision?window=1h"); ok("D07 window lạ → 422", c == 422, str(c))
