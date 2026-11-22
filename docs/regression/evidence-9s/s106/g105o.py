import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
from gdlib import window, CG0, mk_assignment
import datetime
def dec_cfg(): return api("admin", "GET", "/admin/llm/decision")[1]
def gv(): return [g for g in dec_cfg()["gates"] if g["gate"] == "INJECTION_FLAG"][0]
def put(en): g = gv(); return api("admin", "PUT", "/admin/llm/decision/gates/INJECTION_FLAG", {"enabled": en, "version": g["version"]})[0]
def probe(a): 
    t = time.time(); api("teacher", "POST", f"/courses/{CG0}/assignments/{a}/grade-all", {}); 
    for _ in range(40):
        time.sleep(0.5)
        n = int(sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.run_state='DONE'")[0][0])
        if n >= 1: break
    time.sleep(1.5); return t
def decide_at(a): 
    r = sql(f"select extract(epoch from min(created_at)) from llm_audit where task='DECIDE' and cascade_gate='INJECTION_FLAG' and created_at >= now() - interval '10 minutes' and trace_id in (select trace_id from llm_audit where task='GRADING' and created_at >= now() - interval '10 minutes')")
    return r
def run():
    sql(f"INSERT INTO course_decision_gates(course_id,gate,enabled) VALUES ('{CG0}','INJECTION_FLAG',true) ON CONFLICT (course_id,gate) DO UPDATE SET enabled=true, version=course_decision_gates.version+1"); put(False); time.sleep(3); fake(decision_mode="")
    A1 = [mk_assignment(f"g105o1_{i}", 1) for i in range(8)]; A2 = [mk_assignment(f"g105o2_{i}", 1) for i in range(22)]
    # --- (1) qua pub/sub (PUT API)
    t0 = time.time(); pc = put(True); first = None; used = 0
    for a in A1:
        used += 1; s = now(); probe(a); n = sql(f"select count(*), extract(epoch from min(created_at)) from llm_audit where task='DECIDE' and cascade_gate='INJECTION_FLAG' and created_at >= '{s}'")[0]
        if int(n[0]) > 0: first = float(n[1]) - t0; break
    chk5("TC-D02-16", first is not None and first <= 5, f"hai tiến trình (gateway PUT cổng INJECTION_FLAG qua API → publish ep:decision:cfg; tiến trình worker đánh giá cổng khi chấm): DECIDE đầu tiên của worker sau {first and round(first, 1)} s (≤ 5 s) ở lần chấm thử #{used}") if False else None
    print("pubsub: PUT", pc, "→ worker có hiệu lực sau", first and round(first, 1), "s (lần thử", used, ")", flush=True)
    put(False); time.sleep(3)
    # --- (2) chặn pub/sub: sửa thẳng DB (không publish), chờ làm mới mỗi 60 s của worker
    sql("UPDATE decision_gates SET enabled=true, version=version+1 WHERE gate='INJECTION_FLAG'"); t1 = time.time(); first2 = None; used2 = 0
    for a in A2:
        used2 += 1; s = now(); probe(a); n = sql(f"select count(*), extract(epoch from min(created_at)) from llm_audit where task='DECIDE' and cascade_gate='INJECTION_FLAG' and created_at >= '{s}'")[0]
        if int(n[0]) > 0: first2 = float(n[1]) - t1; break
        time.sleep(1.5)
    print("không publish (SQL): worker có hiệu lực sau", first2 and round(first2, 1), "s (lần thử", used2, ")", flush=True)
    json.dump({"pub": first, "pubUsed": used, "ttl": first2, "ttlUsed": used2}, open("/Users/kuro/.config/edupilot-team/qc3-work/d0216.json", "w"))
def main():
    try: run()
    finally:
        sql(f"UPDATE course_decision_gates SET enabled=false WHERE course_id='{CG0}'"); sql("UPDATE decision_gates SET enabled=false, version=version+1 WHERE gate='INJECTION_FLAG'"); os.system("docker exec ep-qc-redis-1 redis-cli publish ep:decision:cfg x >/dev/null"); off()
with window(): main()
