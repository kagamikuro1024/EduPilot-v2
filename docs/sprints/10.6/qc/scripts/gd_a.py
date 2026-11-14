# 10.6 pha 2 — A: cờ theo lớp + RBAC; B: dựng ca duyệt có sửa (P-106-4). python gd_a.py
from gdlib import *
def feats(u):
    c, j, _ = api(u, "GET", FL); return c, {x["feature"]: x for x in (j.get("items") or [])} if isinstance(j, dict) else {}
# --- A. cờ
c, f = feats("teacher"); ok("G01-01 teacher: 4 cờ, mặc định tắt, can_edit đúng", c == 200 and len(f) == 4 and not any(x["enabled"] for x in f.values()) and f["REGRESSION_CAPTURE"]["can_edit"] is False and f["GRADING_GUIDANCE"]["can_edit"], json.dumps({k: (v["enabled"], v["can_edit"]) for k, v in f.items()}))
c, fa = feats("admin"); ok("G01-01 admin: can_edit cả 4", c == 200 and all(x["can_edit"] for x in fa.values()), str(c))
for u, tag in (("ta", "TA"), ("sv.kha", "SV")):
    c, _ = feats(u); ok(f"G01-05/06 {tag} GET cờ → 403", c == 403, str(c))
    c, _, _ = api(u, "PUT", f"{FL}/GRADING_GUIDANCE", {"enabled": True, "version": 1}); ok(f"G01-05/06 {tag} PUT cờ → 403", c == 403, str(c))
c = raw("GET", FL)[0]; ok("G01-05 không token → 401", c == 401, str(c))
c, _, _ = api("sv.nguyco", "GET", f"/courses/{C1}/guidance-flags") ; print("INFO sv.nguyco (SV lớp 1)", c)
c, j, _ = api("teacher", "GET", "/courses/00000000-0000-7000-8000-000000000000/guidance-flags"); ok("G01-05 lớp lạ → 404", c == 404, str(c))
# PUT + audit + version
v0 = f["GRADING_GUIDANCE"]["version"]
c, j, _ = api("teacher", "PUT", f"{FL}/GRADING_GUIDANCE", {"enabled": True, "version": v0}); ok("G01-02 teacher bật GRADING_GUIDANCE → 200, version+1", c == 200 and j["version"] == v0 + 1 and j["enabled"], f"{c} {json.dumps(j)[:120]}")
au = sql("select action, actor_id is not null, left(coalesce(before::text,''),60), left(coalesce(after::text,''),60) from audit_log where entity='course_guidance_flags' or action like 'grading_guidance.flag%' order by created_at desc limit 2") if False else sql("select action, left(coalesce(\"before\"::text,''),50), left(coalesce(\"after\"::text,''),50) from audit_log order by created_at desc limit 2")
print("INFO audit_log mới nhất:", au)
c, _, _ = api("teacher", "PUT", f"{FL}/GRADING_GUIDANCE", {"enabled": False, "version": v0}); ok("G01-16 version cũ → 409", c == 409, str(c))
c, j, _ = api("teacher", "PUT", f"{FL}/GRADING_GUIDANCE", {"version": v0 + 1}); ok("G01-16 thiếu enabled → 422", c == 422, f"{c}")
c, j, _ = api("teacher", "PUT", f"{FL}/GRADING_GUIDANCE", {"enabled": "yes", "version": v0 + 1}); ok("G01-16 enabled sai kiểu → 422", c == 422, f"{c}")
c, j, _ = api("teacher", "PUT", f"{FL}/KHONG_CO", {"enabled": True, "version": 1}); ok("G01-16 cờ lạ → 4xx", 400 <= c < 500, f"{c}")
c, j, _ = api("teacher", "PUT", f"{FL}/REGRESSION_CAPTURE", {"enabled": True, "version": 1}); ok("G01-04 teacher PUT REGRESSION_CAPTURE → 403", c == 403, f"{c}")
c, fa = feats("admin"); rv = fa["REGRESSION_CAPTURE"]["version"]
c, j, _ = api("admin", "PUT", f"{FL}/REGRESSION_CAPTURE", {"enabled": True, "version": rv}); ok("G01-04 admin PUT REGRESSION_CAPTURE → 200", c == 200, f"{c}")
api("admin", "PUT", f"{FL}/REGRESSION_CAPTURE", {"enabled": False, "version": rv + 1})
# --- tổng quan khi cờ bật, chưa đủ dữ liệu
c, j, _ = api("teacher", "GET", G); ok("G03-25 overview: enabled, eligible=false khi 0 bài duyệt", c == 200 and j["enabled"] and not j["reflect"]["eligible"], json.dumps(j["reflect"]) if c == 200 else str(c))
# --- B. dựng ca duyệt (8 bài; 6 tiêu chí c2 chấm cao hơn GV 1,0; 4 bài c3 thấp hơn 0,5)
rs = results(); print("kết quả hiện hành:", len(rs), {r[2] for r in rs})
done = 0; plan = []
for i, (rid, ver, st, rnd, gv, rs_) in enumerate(rs):
    if st != "DRAFT" or done >= 8: continue
    c, j, _ = api("teacher", "GET", f"/courses/{C1}/grading-results/{rid}")
    if c != 200: print("GET result", rid, c); continue
    crit = {x["id"]: x for x in j["criteria"]}; ver = j["result"]["version"]
    edits = []
    if i % 8 < 6 and "c2" in crit and crit["c2"].get("score") not in (None, "0", "0.00") and float(crit["c2"]["score"]) >= 1.0:
        edits.append({"id": "c2", "score": f"{float(crit['c2']['score']) - 1.0:.2f}"})
    if i % 4 == 1 and "c3" in crit and crit["c3"].get("score") is not None and float(crit["c3"]["score"]) + 0.5 <= float(crit["c3"].get("max_score", 99)):
        edits.append({"id": "c3", "score": f"{float(crit['c3']['score']) + 0.5:.2f}"})
    if edits:
        c, j2, _ = api("teacher", "PUT", f"/courses/{C1}/grading-results/{rid}", {"version": ver, "criteria": edits}); ver = (j2 or {}).get("result", {}).get("version", ver) if c == 200 else ver
        if c != 200: print("PUT edit", c, json.dumps(j2)[:150]); continue
    c, j3, _ = api("teacher", "POST", f"/courses/{C1}/grading-results/{rid}/review", {"version": ver})
    if c == 200: done += 1; plan.append((rid, [e["id"] for e in edits]))
    else: print("review", c, json.dumps(j3)[:150])
print("đã duyệt:", done, plan)
c, j, _ = api("teacher", "GET", G); print("overview:", json.dumps(j.get("reflect")), json.dumps(j.get("stale")), j.get("counts"))
