# D05-04: decision (P50=600) chạy SONG SONG với nhúng trong request chat. Nhúng giả ép 500 ms (FAKE_LLM_LATENCY=500-500) để cộng ≠ max phân biệt được.
from q105 import *
C1 = cid_of("761987")
Q = "Cho mình hỏi cách tính điểm chuyên cần của môn này như thế nào ạ?"
def run(tag):
    s = now_db(); st, t1, t2, body, sess = chat("sv.gioi", C1, Q); time.sleep(5)
    rows = sql(f"select task,status,latency_ms,created_at from llm_audit where created_at>='{s}' and course_id='{C1}' and task in ('EMBEDDING','DECIDE','CLASSIFY','CHAT') order by created_at")
    return st, round(t1), round(t2), rows, body[:80].replace("\n", " ")
setenv(); all_off(C1)
off = run("off"); print("cổng tắt, nhúng 0 ms:", off[:3], [(r[0], r[2]) for r in off[3]])
setenv(FAKE_LLM_LATENCY="500-500"); all_off(C1)
off5 = run("off5"); print("cổng tắt, nhúng 500 ms:", off5[:3], [(r[0], r[1], r[2]) for r in off5[3]])
setenv(FAKE_LLM_LATENCY="500-500", LLM_DECISION_FAKE_P50_MS="600", LLM_DECISION_FAKE_P95_MS="650"); set_gates(C1, on=("CHAT_ROUTING",))
on = run("on"); print("CHAT_ROUTING bật, decision 600 + nhúng 500:", on[:3]); [print("  ", r) for r in on[3]]
import datetime
def ts(x): return datetime.datetime.fromisoformat(x.replace("+00", "+00:00")).timestamp()
em = [r for r in on[3] if r[0] == "EMBEDDING"]; de = [r for r in on[3] if r[0] == "DECIDE"]
if em and de:
    e_end, d_end = ts(em[0][3]), ts(de[0][3]); e_st, d_st = e_end - int(em[0][2]) / 1000, d_end - int(de[0][2]) / 1000
    print("embed start", round(e_st % 100, 2), "end", round(e_end % 100, 2), "| decide start", round(d_st % 100, 2), "end", round(d_end % 100, 2))
    overlap = min(e_end, d_end) - max(e_st, d_st)
    par = overlap > 0.3
    rec("TC-D05-04", "P" if par else "F", f"qc2seedt2, CHAT_ROUTING bật lớp 761987, decision P50=600 ms, nhúng giả ép 500 ms (FAKE_LLM_LATENCY=500-500): llm_audit EMBEDDING {em[0][2]} ms và DECIDE {de[0][2]} ms; khoảng chạy [bắt đầu→xong] nhúng {round(e_st % 100, 2)}→{round(e_end % 100, 2)}, decision {round(d_st % 100, 2)}→{round(d_end % 100, 2)}: chồng lên nhau {round(overlap, 2)} s (song song nếu > 0,3 s; nối tiếp thì ≈ 0); tổng thời gian tới byte đầu SSE {on[1]} ms, hoàn tất {on[2]} ms vs cổng tắt cùng nhúng 500 ms: {off5[1]} / {off5[2]} ms. TestOneEmbedPerMessage: PASS (Go, chạy riêng)")
else: rec("TC-D05-04", "N", f"không đủ dòng EMBEDDING/DECIDE: {on[3]}")
setenv(); all_off(C1)
