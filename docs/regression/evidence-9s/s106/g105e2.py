import sys, os, re; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
import gdlib
from gdlib import mk_assignment, grade_all_wait, window, CG0
from g105x import write_override, envswap, snap
def gate_cg(on):
    sql(f"UPDATE decision_gates SET enabled={'true' if on else 'false'}, version=version+1 WHERE gate='INJECTION_FLAG'")
    sql(f"INSERT INTO course_decision_gates(course_id,gate,enabled) VALUES ('{CG0}','INJECTION_FLAG',{'true' if on else 'false'}) ON CONFLICT (course_id,gate) DO UPDATE SET enabled=EXCLUDED.enabled, version=course_decision_gates.version+1")
    os.system("docker exec ep-qc-redis-1 redis-cli publish ep:decision:cfg x >/dev/null"); time.sleep(1.5)
def drows(since): return sql(f"select cascade_path,status,coalesce(error_kind,'') from llm_audit where task='DECIDE' and cascade_gate='INJECTION_FLAG' and created_at>='{since}' order by created_at")
def run():
    ov = write_override("ep-qc.g2w.yml", {"LLM_DECISION_FAKE_P50_MS": "5000", "LLM_DECISION_FAKE_P95_MS": "6500"}); print("swap", envswap(ov)[:50]); snap("g2w")
    gate_cg(True); redis_del("ep:llm:cb:*")
    # --- D04-13: decision chậm 5 s → hết hạn ~3 s
    redis_del("ep:llm:cb:*"); a3 = mk_assignment("g105d13", 1); s = now(); t0 = time.time(); grade_all_wait(a3); dt = time.time() - t0; time.sleep(2); d = drows(s)
    chk5("TC-D04-13", any(x[1] == "timeout" for x in d) and dt < 12, f"env worker+gateway LLM_DECISION_FAKE_P50_MS=5000/P95=6500 (recreate), timeout cổng 3000 ms: chấm 1 bài xong sau {dt:.1f} s; DECIDE {[(x[0], x[1]) for x in d]}")
def main():
    try: run()
    finally:
        gate_cg(False); print("restore", envswap(None)[:50]); off(); redis_del("ep:llm:cb:*")
with window(): main()
