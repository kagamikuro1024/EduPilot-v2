# TC-01-13 lượt 3 (P-11-44 bước 2): 1 triệu hàng llm_audit rải ~11,6 ngày → worker cộng gộp theo giờ → đo summary 7d/24h + đối chiếu thô.
import statistics, subprocess, sys, time, os
from p10lib import *
LOAD = open("/proc/loadavg").read().split()[0] if os.path.exists("/proc/loadavg") else subprocess.run("uptime", capture_output=True, text=True).stdout.split("load averages:")[-1].split()[0]
BAD = "('ok','degraded','cancelled')"
c1, c2 = course("teacher", "761987"), course("teacher", "761988")
step = sys.argv[1] if len(sys.argv) > 1 else "all"
if step in ("all", "load"):
    sql(f"""insert into llm_audit(task,lane,provider,model,latency_ms,status,trace_id,created_at,course_id,cost_est,pii_masked_count,fallback_index,tokens_in,tokens_out)
select (array['CHAT','CLASSIFY','UTILITY','GRADING','INSIGHT'])[1+i%5],'INTERACTIVE','fake','fake-chat',100+(i*37)%4000,(array['ok','ok','ok','ok','ok','ok','ok','ok','error','degraded'])[1+i%10],'qc-1m-'||i,
 now()-((i%1000000)||' seconds')::interval, case when i%3=0 then '{c1}'::uuid when i%3=1 then '{c2}'::uuid else null end, (i%50)/10.0, (i%7=0)::int, (i%11=0)::int, 100, 50 from generate_series(1,1000000) i""")
    sql("analyze llm_audit"); print("tổng hàng", sql("select count(*) from llm_audit")[0][0])
if step in ("all", "measure"):
    print("watermark:", sql("select watermark, updated_at from llm_audit_rollup_state"), "giờ đã cộng gộp:", sql("select count(distinct hour) from llm_audit_hourly")[0][0], "/ giờ có dữ liệu thô:", sql("select count(distinct date_trunc('hour', created_at)) from llm_audit")[0][0])
    def lat(path, n=30):
        ms = []
        for _ in range(n):
            c, _, t, _ = api("admin", "GET", path); assert c == 200, (c, path); ms.append(t)
        ms.sort(); return ms[int(.95 * n) - 1], statistics.median(ms), ms[-1]
    for w, interval in (("7d", "7 days"), ("24h", "24 hours")):
        p95, p50, mx = lat(f"/admin/observability/summary?window={w}")
        ok(f"TC-01-13 (rollup) 1 triệu hàng «summary {w}»: p95 ≤ 500 ms", p95 <= 500, f"p50={p50:.0f} p95={p95:.0f} max={mx:.0f} ms, load máy={LOAD}")
        c, s, _, _ = api("admin", "GET", f"/admin/observability/summary?window={w}"); t = s["totals"]
        n, e, fb, cost = sql(f"select count(*), count(*) filter (where status not in {BAD}), count(*) filter (where fallback_index>0), coalesce(sum(cost_est),0) from llm_audit where created_at >= now() - interval '{interval}'")[0]
        q50, q95 = [float(x) for x in sql(f"select percentile_cont(0.5) within group (order by latency_ms), percentile_cont(0.95) within group (order by latency_ms) from llm_audit where created_at >= now() - interval '{interval}'")[0]]
        d_n = abs(t["requests"] - int(n)); d_e = abs(t["errors"] - int(e)); d_fb = abs(t["fallbacks"] - int(fb)); d_c = abs(float(t["cost"]) - float(cost))
        # khoảng trống ±10 phút quanh mốc 24 giờ và 7 ngày (fixture) để cửa sổ trôi không đổi số hàng → so khớp tuyệt đối
        ok(f"TC-01-13 (rollup) «summary {w}» đếm / lỗi / dự phòng / chi phí khớp thô", d_n == 0 and d_e == 0 and d_fb == 0 and d_c < 0.01, f"api={t['requests']}/{t['errors']}/{t['fallbacks']}/{t['cost']} sql={n}/{e}/{fb}/{cost}")
        a50, a95 = float(t.get("p50_ms", 0)), float(t.get("p95_ms", 0))
        ok(f"TC-01-13 (rollup) «summary {w}» p50/p95 lệch ≤ 5 %", abs(a50 - q50) <= .05 * q50 and abs(a95 - q95) <= .05 * q95, f"api p50={a50:.0f} p95={a95:.0f} sql p50={q50:.0f} p95={q95:.0f}")
    for name, path, lim in (("requests limit=50", "/admin/observability/requests?limit=50", 300),):
        p95, p50, mx = lat(path); ok(f"TC-01-13 (rollup) «{name}»: p95 ≤ {lim} ms", p95 <= lim, f"p50={p50:.0f} p95={p95:.0f} max={mx:.0f} ms, load={LOAD}")
if step in ("all", "clean"):
    sql("delete from llm_audit where trace_id like 'qc-1m-%'"); print("dọn xong", sql("select count(*) from llm_audit")[0][0])
