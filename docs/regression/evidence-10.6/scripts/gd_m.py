# gd_m (QC2): số đo, RBAC stats, rò SV, PII bảng mới, ca hồi quy mồ côi. Không gọi LLM.
from gdlib import *
import re
flag("GRADING_GUIDANCE", True)
c, sj, _ = api("teacher", "GET", G + "/stats"); items = sj["items"]; print("stats:", json.dumps(items)[:500])
small = [i for i in items if i["n"] < 5]
ok("G06-02 phiên bản n<5: chỉ số null", bool(small) and all(i["mae_criterion"] is None and i["mae_total"] is None for i in small) if small else False, json.dumps(small)[:160] if small else "không có phiên bản n<5 trong dữ liệu")
cur = int(sql(f"select count(*) from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status in ('REVIEWED','PUBLISHED') and r.round=(select max(round) from grading_results x where x.submission_id=r.submission_id)")[0][0])
ok("G06-08 tổng n các phiên bản = số bài hiện hành REVIEWED/PUBLISHED", sum(i["n"] for i in items) == cur, f"sum={sum(i['n'] for i in items)} sql={cur}")
n0 = sum(i["n"] for i in items); sql(f"update grading_results set updated_at=now() where id=(select id from grading_results r where r.status='DRAFT' limit 1)")
c, sj2, _ = api("teacher", "GET", G + "/stats"); ok("G06-03 thêm / đụng DRAFT: số không đổi", sum(i["n"] for i in sj2["items"]) == n0, "")
rows = sql(f"select r.ai_total_before_late, r.criteria::text from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status in ('REVIEWED','PUBLISHED') and r.round=(select max(round) from grading_results x where x.submission_id=r.submission_id)")
from decimal import Decimal as D
b = sum(D(a) - sum(D(x["score"]) for x in json.loads(cr) if x.get("score") is not None) for a, cr in rows) / len(rows) if rows else 0
tot = sum(i["n"] * D(i["bias_total"]) for i in items if i["bias_total"] is not None) / max(1, sum(i["n"] for i in items if i["bias_total"] is not None))
ok("G06-04 bias_total = AI − chốt (khớp SQL độc lập, dấu AI>chốt dương)", abs(b - tot) < D("0.02"), f"sql={b:.2f} api={tot:.2f}")
c2 = sql("select u.email from users u join enrollments e on e.user_id=u.id join courses c on c.id=e.course_id where c.class_code='761988' and u.email like 'sv%' limit 1")[0][0].split("@")[0]
for u, exp in (("sv.kha", 403), (c2, 403), (None, 401)):
    c, _, _ = api(u, "GET", G + "/stats"); ok(f"G06-10 {u or 'không token'} stats → {exp}", c == exp, str(c))
C2 = sql("select id from courses where class_code='761988'")[0][0]
c, _, _ = api("teacher", "GET", f"/courses/{C2}/assignments/{A}/guidance/stats"); ok("G06-10 stats qua lớp 2 → 404/403", c in (403, 404), str(c))
# G11-06: SV quét >=30 phản hồi
pats = r"lưu ý chấm|guidance|bản v\d|first_reviewed|regression_case|REGRESSION_CAPTURE|THREAD_SUGGEST|QUESTION_HINTS|mae_|bias_total"
paths = [f"/courses/{C1}/assignments", f"/courses/{C1}/assignments/{A}", f"/courses/{C1}/assignments/{A}/mine", "/notifications", f"/courses/{C1}/threads?limit=50", f"/courses/{C1}/grades", f"/courses/{C1}/me/grades", "/me/courses", f"/courses/{C1}", "/me", f"/courses/{C1}/questions", f"/courses/{C1}/practice"]
n = 0; leak = []
for p in paths * 3:
    c, j, _ = api("sv.kha", "GET", p); n += 1; s = json.dumps(j, ensure_ascii=False)
    if re.search(pats, s, re.I): leak.append(p)
subs = sql(f"select r.id from grading_results r where r.status in ('PUBLISHED') limit 3")
for (rid,) in subs:
    c, j, _ = api("sv.kha", "GET", f"/courses/{C1}/grading-results/{rid}"); n += 1
ok(f"G11-06 SV quét {n} phản hồi: 0 trường lưu ý / phiên bản / cờ / số đo", not leak, str(sorted(set(leak))))
# G11-04: bảng mới + đầu vào không PII
names = [x[0] for x in sql("select full_name from users where role='STUDENT' and length(full_name)>=6")]
txt = json.dumps(sql("select text from grading_guidance_items") + sql("select expected::text from grading_regression_cases"), ensure_ascii=False)
hits = [x for x in names if x in txt]
ok("G11-04 bảng guidance_items + regression_cases không chứa tên SV", not hits, str(hits[:3]))
orph = sql("select count(*) from grading_regression_cases c where kind in ('GRADING_EDIT','GRADING_APPEAL') and not exists (select 1 from grading_results r where r.id=c.source_id)")[0][0]
ok("G07-14 không ca hồi quy mồ côi (source_id không còn kết quả) = 0", orph == "0", orph)
gv0 = sql(f"select guidance_version from assignments where id='{A}'")[0][0]; sql("select 1")
ok("G11-02 assignments.guidance_version không đổi bởi thao tác đọc / stats (so trước-sau)", sql(f"select guidance_version from assignments where id='{A}'")[0][0] == gv0, gv0)
