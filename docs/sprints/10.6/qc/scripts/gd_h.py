"""QC 10.6 — kiểm lại sau P-106-10 (975f9541): GD-BUG-9 PUBLISH, GD-BUG-10 UNSPECIFIED, regression_cases ở stats."""
import subprocess, sys, os
sys.path.insert(0, os.path.expanduser('~/Documents/TA_Agent_qc106/docs/sprints/10.5/qc/scripts'))
import dmx
from gdlib import *
QB = f"/courses/{C1}/questions"
def ok(n, c, d=""): print(("PASS " if c else "FAIL ") + n, d)

# ---- GD-BUG-9: seed-guidance PUBLISH ----
flag("GRADING_GUIDANCE", False); time.sleep(1)
ok("B9-0 tắt qua API → overview.enabled=false", api("teacher", "GET", G)[1]["enabled"] is False)
r = subprocess.run("cd ~/Documents/TA_Agent_qc5 && docker compose --env-file .env.local -f docker-compose.local.yml -p ep-qc exec -T gateway /gateway admin seed-guidance --course 761987 --assignment BT03 2>&1 | tail -2", shell=True, capture_output=True, text=True, env=dmx.BASE_ENV)
print(r.stdout.strip()[:160])
t0 = time.time(); en = api("teacher", "GET", G)[1]["enabled"]; dt = time.time() - t0
ok("B9-1 ngay sau seed-guidance: overview.enabled=true (không chờ 60 s)", en is True, f"{en} sau {dt:.2f}s")
ok("B9-2 hàng cờ DB", sql("select enabled from course_guidance_flags where feature='GRADING_GUIDANCE'")[0][0] == 't')

# ---- GD-BUG-10: REJECT không lý do ----
pend = sql(f"select id, version from question_bank where course_id='{C1}' and review_status='PENDING' order by id limit 4")
ok("B10-0 có ≥ 4 câu PENDING", len(pend) >= 4, str(len(pend)))
got = lambda i: sql(f"select review_status, reject_reason from question_bank where id='{i}'")[0]
c, j, _ = api("teacher", "PUT", f"{QB}/{pend[0][0]}/review", {"decision": "REJECT", "version": int(pend[0][1])})
ok("B10-1 REJECT đơn không lý do → UNSPECIFIED", c == 200 and got(pend[0][0]) == ["REJECTED", "UNSPECIFIED"], f"{c} {got(pend[0][0])}")
c, j, _ = api("teacher", "POST", f"{QB}/bulk", {"action": "REJECT", "ids": [str(pend[1][0]), str(pend[2][0])]})
ok("B10-2 bulk REJECT không reason → UNSPECIFIED cả lô", c == 200 and all(got(p[0]) == ["REJECTED", "UNSPECIFIED"] for p in pend[1:3]), f"{c} {[got(p[0]) for p in pend[1:3]]}")
c, j, _ = api("teacher", "PUT", f"{QB}/{pend[3][0]}/review", {"decision": "REJECT", "version": int(pend[3][1]), "reject_reason": "MO_HO"})
ok("B10-3 có lý do → giữ nguyên MO_HO", c == 200 and got(pend[3][0]) == ["REJECTED", "MO_HO"], f"{c} {got(pend[3][0])}")
aud = sql(f"select after::text from audit_log where entity_id in ('{pend[0][0]}','{pend[3][0]}') and action='question.review' order by id")
import json as J
vals = [J.loads(a[0]).get("reject_reason") for a in aud]
ok("B10-4 audit chỉ có reject_reason là enum (không chữ tự do)", all(v in (None, "UNSPECIFIED", "MO_HO") for v in vals) and "UNSPECIFIED" in vals, str(vals))
c, j, _ = api("sv.kha", "GET", f"{QB}"); ok("B10-5 SV không thấy reject_reason", "reject_reason" not in json.dumps(j), str(c))

# ---- regression_cases (P-106-10, US-GD-07 AC6) ----
flag("GRADING_GUIDANCE", True)
sqlc = f"select count(*) from grading_regression_cases c where c.course_id='{C1}' and c.kind in ('GRADING_EDIT','GRADING_APPEAL') and exists (select 1 from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where r.id=c.source_id and s.assignment_id='{A}')"
n_sql = int(sql(sqlc)[0][0])
for u in ("teacher", "ta"):
    c, j, _ = api(u, "GET", G + "/stats")
    ok(f"RC-1 {u}: stats.regression_cases = SQL ({n_sql})", c == 200 and j.get("regression_cases") == n_sql, f"{c} {j.get('regression_cases')}")
c, j, _ = api("sv.kha", "GET", G + "/stats"); ok("RC-2 SV 403", c == 403, str(c))
c, j, _ = api("admin", "GET", G + "/stats"); ok("RC-2 ADMIN 403", c == 403, str(c))
print("tổng ca theo loại:", sql(f"select kind, count(*) from grading_regression_cases where course_id='{C1}' group by 1"))
print("stats:", json.dumps(api("teacher", "GET", G + "/stats")[1])[:200])
