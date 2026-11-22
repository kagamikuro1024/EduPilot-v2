from q105 import *
C1 = cid_of("761987")
setenv(FAKE_LLM_LATENCY="500-500", LLM_DECISION_FAKE_P50_MS="600", LLM_DECISION_FAKE_P95_MS="650"); set_gates(C1, on=("CHAT_ROUTING",))
print([ (g["gate"], g["enabled"]) for g in gates()["gates"]])
for q in ["Giải thích giúp mình khái niệm hàm băm mật mã là gì và dùng để làm gì?", "Chào bạn"]:
    s = now_db(); st, t1, t2, body, sess = chat("sv.gioi", C1, q); time.sleep(5)
    print(q[:30], st, round(t1), round(t2)); [print("  ", r) for r in sql(f"select task,status,latency_ms,coalesce(cascade_gate,''),coalesce(cascade_path,''),created_at from llm_audit where created_at>='{s}' order by created_at")]
setenv(); all_off(C1)
