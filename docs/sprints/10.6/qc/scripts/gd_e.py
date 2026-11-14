# 10.6 pha 2 — E: SV không lộ; cờ tắt; GUIDANCE_ENABLED=false (restart). python gd_e.py
from gdlib import *
import re
# G01-18 SV không thấy lưu ý / phiên bản / cờ
leak = 0; n = 0
for p in (f"/courses/{C1}/assignments", f"/courses/{C1}/assignments/{A}", f"/courses/{C1}/assignments/{A}/mine", "/notifications", f"/courses/{C1}/threads?limit=20", f"/courses/{C1}/me/grades"):
    c, j, _ = api("sv.kha", "GET", p); s = json.dumps(j, ensure_ascii=False); n += 1
    if re.search(r"guidance|lưu ý|Lưu ý|ai_comments|first_reviewed|bản v\d", s): leak += 1; print("LEAK", p, re.findall(r"guidance\w*|ai_comments|first_reviewed\w*", s)[:3])
ok(f"G01-18 SV: {n} phản hồi không có guidance / ai_comments / first_reviewed", leak == 0, str(leak))
# cờ tắt: reflect / apply / regrade → 409
flag("GRADING_GUIDANCE", False)
c, j, _ = api("teacher", "POST", G + "/reflect", {}, idem=f"qc-off-r-{int(time.time())}"); ok("G01-09 cờ tắt: reflect → 409 GUIDANCE_DISABLED", c == 409 and (j or {}).get("code") == "GUIDANCE_DISABLED", f"{c} {(j or {}).get('code')}")
c, j, _ = api("teacher", "POST", G + "/apply", {"expected_version": 4}, idem=f"qc-off-a-{int(time.time())}"); ok("G01-09 cờ tắt: apply → 409 GUIDANCE_DISABLED", c == 409 and (j or {}).get("code") == "GUIDANCE_DISABLED", f"{c} {(j or {}).get('code')}")
c, j, _ = api("teacher", "POST", f"/courses/{C1}/assignments/{A}/regrade-stale", {}, idem=f"qc-off-g-{int(time.time())}"); ok("G01-09 cờ tắt: regrade-stale → 409 GUIDANCE_DISABLED", c == 409 and (j or {}).get("code") == "GUIDANCE_DISABLED", f"{c} {(j or {}).get('code')}")
c, j, _ = api("teacher", "GET", G); ok("G01-09 cờ tắt: overview enabled=false (dữ liệu cũ còn trong DB)", c == 200 and j["enabled"] is False, f"{c} {j.get('enabled')}")
n_it = sql(f"select count(*) from grading_guidance_items where assignment_id='{A}'")[0][0]; ok("G01-09 dữ liệu lưu ý cũ còn trong DB", int(n_it) > 0, n_it)
RID0 = sql(f"select r.id from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' order by r.id limit 1")[0][0]
c, j, _ = api("teacher", "GET", f"/courses/{C1}/grading-results/{RID0}")
ok("G01-09 cờ tắt: màn duyệt không có khối guidance", c == 200 and "guidance" not in (j or {}), str(list((j or {}).keys())[:8]))
# GUIDANCE_ENABLED=false
print(restart_gw(GUIDANCE_ENABLED="false")[-80:])
time.sleep(8)
c, f = None, None
c, j, _ = api("teacher", "GET", FL); print("flags khi GUIDANCE_ENABLED=false:", c, json.dumps(j)[:200])
c, j, _ = api("teacher", "PUT", f"{FL}/GRADING_GUIDANCE", {"enabled": True, "version": [x for x in j["items"] if x["feature"] == "GRADING_GUIDANCE"][0]["version"]}); ok("G01-07 GUIDANCE_ENABLED=false: ghi cờ → 409 GUIDANCE_DISABLED", c == 409 and (j or {}).get("code") == "GUIDANCE_DISABLED", f"{c} {(j or {}).get('code')}")
j0 = sql("select count(*) from jobs where kind='grading.reflect' and created_at > now() - interval '1 minute'")[0][0]
c, j, _ = api("teacher", "POST", G + "/reflect", {}, idem=f"qc-off2-r-{int(time.time())}"); ok("G01-07 reflect → 409, 0 job mới", c == 409 and sql("select count(*) from jobs where kind='grading.reflect' and created_at > now() - interval '1 minute'")[0][0] == j0, f"{c}")
# khôi phục
print(restart_gw()[-60:]); time.sleep(8)
flag("GRADING_GUIDANCE", True); print("đã khôi phục")
