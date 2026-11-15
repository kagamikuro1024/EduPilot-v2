# L2: ma trận P × chế độ stub Structured. Dùng: python dm_b2.py <nhãn> [FAKE_DECISION_PROB] [LLM_DECISION_FAKE_MODE]
import sys, time
from dmx import *
C1 = course("sv.kha", "761987")
label = sys.argv[1]
prob = sys.argv[2] if len(sys.argv) > 2 else ""
mode = sys.argv[3] if len(sys.argv) > 3 else ""
if label.startswith("restart"):
    print(restart_gw(FAKE_DECISION_PROB=prob, LLM_DECISION_FAKE_MODE=mode)); sys.exit()
CLEAN = ("Hỏi về công thức tính điểm", "Công thức tính điểm quá trình của môn này gồm những thành phần nào và trọng số ra sao vậy ạ?")
set_gate("THREADS_PRECHECK")
for sm in sys.argv[4:] or ["allow"]:
    open("/tmp/stub_mode", "w").write(sm)
    s = now_db(); t0 = time.time()
    c, j, ms = api("sv.gioi", "POST", f"/courses/{C1}/threads", {"title": CLEAN[0], "body": CLEAN[1]}, timeout=40)
    time.sleep(4)
    rows = [(r[1], r[2], r[3], r[4]) for r in sql(f"select created_at::time(2),task,status,coalesce(cascade_gate,''),coalesce(cascade_path,'') from llm_audit where created_at>='{s}' and task in ('DECIDE','CLASSIFY') order by created_at")]
    print(f"[P={prob or 'lạ'} mode={mode or '-'} stub={sm}] HTTP {c} {(j or {}).get('code')} {ms:.0f}ms rows={rows}")
