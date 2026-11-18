#!/usr/bin/env python3
"""TC-P7-08-03: lớp QD9P (1.000 bài): p95 trang đầu, duyệt con trỏ không lặp/sót, limit ≤ 100, EXPLAIN (ANALYZE) truy vấn thật AssignmentQueue."""
import sys, os, json, statistics, subprocess; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
R = Rep("083"); C = psql("select id from courses where class_code='QD9P'"); Q = f"/api/v1/courses/{C}"; T = tok("teacher")
tt = []
for i in range(40):
    t0 = time.time(); s, x, _ = call("GET", f"{Q}/submissions?limit=50", T); tt.append((time.time() - t0) * 1000)
tt.sort(); p95 = tt[int(len(tt) * .95) - 1]; print("   trang đầu: n=40, p50 %.0f ms, p95 %.0f ms, max %.0f ms" % (statistics.median(tt), p95, tt[-1]), "| keys", list(x)[:6])
R.chk("TC-P7-08-03", f"p95 trang đầu {p95:.0f} ms (≤ 300)", s == 200 and p95 <= 300, tt[-5:])
ids = []; cur = None; pages = 0
while True:
    s, x, _ = call("GET", f"{Q}/submissions?limit=100" + (f"&cursor={cur}" if cur else ""), T); assert s == 200, (s, x)
    items = x.get("items", []); ids += [i["id"] for i in items]; pages += 1; cur = x.get("next_cursor")
    if not cur or pages > 30: break
exp = int(psql(f"select count(*) from submissions s where s.course_id='{C}' and s.match_status='MATCHED' and not exists (select 1 from submissions n where n.course_id=s.course_id and n.assignment_id=s.assignment_id and n.student_id=s.student_id and n.match_status='MATCHED' and n.version>s.version)"))
R.chk("TC-P7-08-03", f"con trỏ liên tiếp: {pages} trang × ≤100 → {len(ids)} bài, {len(set(ids))} khác nhau; kỳ vọng {exp} (không lặp / sót)", len(ids) == len(set(ids)) == exp, (len(ids), len(set(ids)), exp))
s, x, _ = call("GET", f"{Q}/submissions?limit=500", T); n = len(x.get("items", [])) if isinstance(x, dict) else -1; R.chk("TC-P7-08-03", f"limit=500 → {s}, {n} dòng (≤ 100 hoặc 422)", (s == 200 and n <= 100) or s == 422, (s, n))
s, x, _ = call("GET", f"{Q}/submissions?limit=100&offset=100", T); R.chk("TC-P7-08-03", f"tham số offset không có tác dụng (trang đầu giống không offset): {s}", s in (200, 422), s)
sql = open(os.path.expanduser("~/Documents/qc9s-mut/w4/backend-go/internal/store/queries/review.sql")).read().split("-- name: AssignmentQueue :many")[1].split("-- name:")[0]
sql = "\n".join(l for l in sql.split("\n") if not l.startswith("--")).replace("sqlc.arg(course_id)", f"'{C}'").replace("sqlc.arg(page_limit)", "50")
import re
sql = re.sub(r"sqlc\.narg\((\w+)\)::(\w+)", lambda m: "NULL::" + m.group(2), sql)
def explain(pre=""):
    r = subprocess.run(["docker", "exec", "-i", "ep-qc-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", pre + "explain (analyze, buffers) " + sql], capture_output=True, text=True); return r.stdout + r.stderr
e1 = explain(); open("/tmp/q9s/explain083.txt", "w").write(e1); print(e1[:3000])
print("tổng dòng submissions:", psql("select count(*) from submissions"), "| index:", psql("select string_agg(indexname,', ') from pg_indexes where tablename='submissions'"))
e2 = explain("set enable_seqscan=off; "); print("--- enable_seqscan=off ---\n", e2[:2500])
open("../logs/08-03-explain.txt", "w").write(e1 + "\n--- enable_seqscan=off ---\n" + e2)
sys.exit(R.end())
