# L5: cảnh báo DECISION_FAIL_CLOSED + số liệu /observability. python dm_f_alert_obs.py <step>
import sys, time
from dmx import *
step = sys.argv[1]
def ins(gate, path, status, n, tag, minutes_ago=5):
    sql(f"insert into llm_audit(task,lane,provider,model,status,latency_ms,trace_id,cascade_gate,cascade_path,tokens_in,cost_est,created_at) select 'DECIDE','INTERACTIVE','Quyết định','decision-fake','{status}',{300},'qc-{tag}-'||g,'{gate}','{path}',330,0,now()-interval '{minutes_ago} minutes' from generate_series(1,{n}) g")
if step == "seed":
    sql("delete from llm_audit where trace_id like 'qc-obs-%'")
    sql("delete from notifications where type in ('DECISION_FAIL_CLOSED') ")
    # THREADS: 30 lời gọi trong giờ, 3 FAIL_CLOSED (10 %); INJECTION: 19 lời gọi, 100 % FAIL_CLOSED (dưới ngưỡng 20); CHAT: 40 lời gọi, 2 FAIL_CLOSED (5,0 % đúng)
    ins("THREADS_PRECHECK", "FAST", "ok", 20, "obs-t1"); ins("THREADS_PRECHECK", "FALLBACK", "ok", 7, "obs-t2"); ins("THREADS_PRECHECK", "FAIL_CLOSED", "error", 3, "obs-t3")
    ins("INJECTION_FLAG", "FAIL_CLOSED", "error", 19, "obs-i1")
    ins("CHAT_ROUTING", "FAST", "ok", 30, "obs-c1"); ins("CHAT_ROUTING", "FALLBACK", "ok", 8, "obs-c2"); ins("CHAT_ROUTING", "FAIL_CLOSED", "error", 2, "obs-c3")
    print("seeded")
if step == "obs":
    c, r, _ = api("admin", "GET", "/admin/observability/decision?window=24h")
    for g in r["gates"]:
        q = sql(f"select count(*), count(*) filter (where cascade_path='FAST'), count(*) filter (where cascade_path='FALLBACK'), count(*) filter (where cascade_path='FAIL_CLOSED'), count(*) filter (where status not in ('ok','degraded')), coalesce(sum(cost_est),0) from llm_audit where task='DECIDE' and cascade_gate='{g['gate']}' and created_at > now()-interval '24 hours'")[0]
        n = int(q[0]); 
        exp = (n, round(100*int(q[1])/n, 1) if n else 0, round(100*int(q[2])/n, 1) if n else 0, round(100*int(q[3])/n, 1) if n else 0)
        got = (g["calls"], round(g["fast_pct"], 1), round(g["fallback_pct"], 1), round(g["fail_closed_pct"], 1))
        ok(f"D07-01 {g['gate']} số khớp SQL (calls, FAST%, FALLBACK%, FAIL_CLOSED%)", exp == got, f"api={got} sql={exp} p50={g.get('p50_ms')} p95={g.get('p95_ms')} err%={g.get('error_pct')} cost={g.get('cost_vnd')} state={g.get('state')}")
if step == "alerts":
    n = sql("select count(*) from notifications where type='DECISION_FAIL_CLOSED'")[0][0]
    rows = sql("select u.email, n.type, n.created_at::time(0), left(coalesce(n.payload::text,''),100) from notifications n join users u on u.id=n.user_id where n.type='DECISION_FAIL_CLOSED' order by 3")
    print("notifications:", n); [print(" ", r) for r in rows]
    for u in ("teacher", "ta", "sv.kha"):
        c, j, _ = api(u, "GET", "/notifications"); items = j.get("items", []) if isinstance(j, dict) else []
        ok(f"D07-10 {u} không nhận DECISION_FAIL_CLOSED", c == 200 and not any("DECISION" in json.dumps(i) for i in items), f"{c}")
    c, j, _ = api("admin", "GET", "/notifications"); items = j.get("items", []) if isinstance(j, dict) else []
    print("admin DECISION items:", [(i.get("type"), i.get("title") if isinstance(i, dict) else "") for i in items if "DECISION" in json.dumps(i)][:5])
