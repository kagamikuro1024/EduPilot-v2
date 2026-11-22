import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
from gdlib import window, CG0
def run():
    for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC fake", "fake-chat")
    gate("THREADS_PRECHECK", True, [C]); redis_del("ep:llm:cb:*"); fake(decision_mode="", decision_p50_ms=1500, decision_p95_ms=1500, decision_error_rate=0.0)
    try:
        os.system(f'cd {os.path.dirname(os.path.abspath(__file__))} && QC_RES= ./runpw.sh qc-real-105-ui.spec.ts "D03-13" 150 2>&1 | grep -E "✓|✘|Error" | cut -c1-200')
        print(open("/Users/kuro/.config/edupilot-team/qc3-work/d0313.txt").read())
        print(sql(f"select cascade_path, status, latency_ms from llm_audit where task='DECIDE' and cascade_gate='THREADS_PRECHECK' order by created_at desc limit 2"))
    finally:
        fake(decision_mode="", decision_p50_ms=0, decision_p95_ms=0); off()
        for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
with window(): run()
