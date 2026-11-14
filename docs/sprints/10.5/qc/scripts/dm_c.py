# L3: cổng INJECTION_FLAG. python dm_c.py <nhãn>   (gateway đã khởi động lại với FAKE_DECISION_PROB / MODE phù hợp)
import sys, time
from dmx import *
C1 = course("sv.kha", "761987")
label = sys.argv[1]
base = {r[0]: (r[1], r[2]) for r in sql("select u.email,g.total,g.flags from grading_results g join submissions s on s.id=g.submission_id join users u on u.id=s.student_id where g.round=1")}
subs = sql("select s.id,u.email from submissions s join users u on u.id=s.student_id join grading_results g on g.submission_id=s.id where g.round=1")
if label == "off":
    gates_off()
else:
    set_gate("INJECTION_FLAG")
open("/tmp/stub_mode", "w").write("allow")
s0 = now_db()
for sid, em in subs:
    c, j, _ = api("teacher", "POST", f"/courses/{C1}/submissions/{sid}/grade", {})
    if c != 202: print("regrade", em, c, json.dumps(j, ensure_ascii=False)[:120])
for i in range(30):
    time.sleep(6)
    r = sql(f"select run_state,count(*) from grading_results where round>1 and created_at>='{s0}' group by 1")
    if r and all(x[0] in ("DONE", "FAILED") for x in r) and sum(int(x[1]) for x in r) >= len(subs): break
time.sleep(4)
res = sql(f"select u.email,g.round,g.run_state,g.total,g.flags from grading_results g join submissions s on s.id=g.submission_id join users u on u.id=s.student_id where g.round>1 and g.created_at>='{s0}' order by 1")
chg_tot = [(r[0], base[r[0]][0], r[3]) for r in res if r[0] in base and str(base[r[0]][0]) != r[3]]
new_flags = [(r[0], r[4]) for r in res if "INJECTION_SUSPECTED" in r[4] and "INJECTION_SUSPECTED" not in base[r[0]][1]]
lost = [(r[0]) for r in res if "INJECTION_SUSPECTED" in base[r[0]][1] and "INJECTION_SUSPECTED" not in r[4]]
dec = sql(f"select cascade_gate,cascade_path,status,lane,count(*) from llm_audit where task='DECIDE' and created_at>='{s0}' group by 1,2,3,4")
print(f"[{label}] regraded={len(res)} states={sorted(set(r[2] for r in res))}")
print("  tổng điểm đổi so với round 1:", chg_tot)
print("  cờ INJECTION mới:", [e for e, _ in new_flags])
print("  mất cờ regex:", lost)
print("  DECIDE:", dec)
print("  s4(sv24) cờ:", [r[4] for r in res if r[0] == "sv24@edupilot.local"], " s6(sv14):", [r[4] for r in res if r[0] == "sv14@edupilot.local"])
