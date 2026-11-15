# TC-01-13: 1 triệu hàng llm_audit (QC fixture trace_id 'qc-1m-*', xoá sau khi đo)
import statistics, subprocess, re
from p10lib import *
c1, c2 = course("teacher", "761987"), course("teacher", "761988")
b = int(sql("select count(*) from llm_audit")[0][0])
sql(f"""insert into llm_audit(task,lane,provider,model,latency_ms,status,trace_id,created_at,course_id,cost_est,pii_masked_count,fallback_index,tokens_in,tokens_out)
select (array['CHAT','CLASSIFY','UTILITY','GRADING','INSIGHT'])[1+i%5],'INTERACTIVE','fake','fake-chat',100+(i*37)%4000,(array['ok','ok','ok','ok','ok','ok','ok','ok','error','degraded'])[1+i%10],'qc-1m-'||i,
 now()-((i%604800)||' seconds')::interval, case when i%3=0 then '{c1}'::uuid when i%3=1 then '{c2}'::uuid else null end, (i%50)/10.0, (i%7=0)::int, (i%11=0)::int, 100, 50 from generate_series(1,1000000) i""")
sql("analyze llm_audit"); print("tổng hàng", sql("select count(*) from llm_audit")[0][0], "(trước:", b, ")")
def lat(path, n=30):
    ms = []
    for _ in range(n):
        c, _, t, _ = api("admin", "GET", path); assert c == 200, (c, path); ms.append(t)
    ms.sort(); return ms[int(.95 * n) - 1], statistics.median(ms), ms[-1]
for name, path, lim in (("summary 7d", "/admin/observability/summary?window=7d", 500), ("summary 24h", "/admin/observability/summary?window=24h", 500), ("requests limit=50", "/admin/observability/requests?limit=50", 300), ("requests problem", "/admin/observability/requests?filter=problem&limit=50", 300), (f"course summary", f"/courses/{c1}/observability/summary", 500)):
    p95, p50, mx = lat(path); ok(f"TC-01-13 1 triệu hàng «{name}»: p95 ≤ {lim} ms", p95 <= lim, f"p50={p50:.0f} p95={p95:.0f} max={mx:.0f} ms")
for name, q in (("bảng yêu cầu", "select * from llm_audit order by created_at desc, id desc limit 51"), ("summary 7d", "select task, count(*), percentile_cont(0.95) within group (order by latency_ms) from llm_audit where created_at >= now()-interval '7 days' group by 1"), ("problem", "select * from llm_audit where status not in ('ok','degraded','cancelled') order by created_at desc limit 51")):
    r = subprocess.run(["docker", "exec", "-i", PG, "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", "explain (analyze) " + q], capture_output=True, text=True).stdout
    ok(f"TC-01-13 EXPLAIN «{name}»: không Seq Scan trừ tổng hợp 7 ngày, không OFFSET", "Seq Scan on llm_audit" not in r or name == "summary 7d", " ".join(re.findall(r"(?:Index[\w ]*Scan\w* (?:Backward )?using \w+|Seq Scan on \w+|Execution Time: [\d.]+ ms)", r))[:200])
sql("delete from llm_audit where trace_id like 'qc-1m-%'"); sql("analyze llm_audit"); print("dọn xong", sql("select count(*) from llm_audit")[0][0])
