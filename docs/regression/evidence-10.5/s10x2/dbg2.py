from q105 import *
import statistics
C1 = cid_of("761987")
Qs = ["Giải thích giúp mình khái niệm hàm băm mật mã là gì và dùng để làm gì?", "Cho mình hỏi sự khác nhau giữa mã hoá đối xứng và bất đối xứng là gì vậy?", "Phân biệt tấn công chủ động và thụ động trong an ninh mạng giúp mình với", "Chữ ký số hoạt động như thế nào và vì sao cần chứng thư số?"]
import random
def runs(tag):
    out = []
    for q0 in Qs:
        q = q0 + " (" + tag + str(random.randint(1000, 9999)) + ")"
        s = now_db(); st, t1, t2, body, sess = chat("sv.gioi", C1, q); time.sleep(3)
        rows = sql(f"select task,latency_ms,created_at from llm_audit where created_at>='{s}' order by created_at")
        out.append((round(t2), [(r[0], r[1]) for r in rows]))
    return out
setenv(FAKE_LLM_LATENCY="500-500", LLM_DECISION_FAKE_P50_MS="600", LLM_DECISION_FAKE_P95_MS="650"); all_off(C1)
off = runs("off"); print("OFF", off)
set_gates(C1, on=("CHAT_ROUTING",))
on = runs("on"); print("ON", on)
mo = statistics.median(x[0] for x in off); mn = statistics.median(x[0] for x in on)
print("median OFF", mo, "median ON", mn, "delta", mn - mo)
setenv(); all_off(C1)
