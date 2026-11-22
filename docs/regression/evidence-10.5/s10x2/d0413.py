# D04-13: LLM_DECISION_FAKE_P50_MS=5000 (worker) → cổng INJECTION_FLAG hết hạn ở ~3 s (timeout_ms=3000); chấm không bị giữ quá hạn; kết quả hoàn tất
import json; from q105 import *
C1 = cid_of("761987"); aid = json.load(open("state.json"))["aid"]
sub = {em: sid for sid, em in sql(f"select s.id,u.email from submissions s join users u on u.id=s.student_id where s.assignment_id='{aid}'")}
sid = sub["sv.gioi@edupilot.local"]
def one():
    s = now_db(); t0 = time.time(); c, rnd, st, tot, fl = regrade(C1, sid, wait=120); wall = time.time() - t0
    return s, c, rnd, st, round(wall, 1), fl
setenv(); set_gates(C1, on=("INJECTION_FLAG",))
sb, cb, rb, stb, wb, flb = one(); db = decide_rows(sb, C1, wait=5)
setenv(LLM_DECISION_FAKE_P50_MS="5000", LLM_DECISION_FAKE_P95_MS="6000"); set_gates(C1, on=("INJECTION_FLAG",))
s1, c1, r1, st1, w1, fl1 = one(); d1 = decide_rows(s1, C1, wait=6)
print("baseline (p50 426):", cb, stb, wb, flb, [(r[0], r[1], r[2], r[5]) for r in db])
print("p50=5000:", c1, st1, w1, fl1, [(r[0], r[1], r[2], r[5]) for r in d1])
tmo = [r for r in d1 if r[0] == "INJECTION_FLAG" and r[2] in ("timeout", "error")]
lat = [int(r[5]) for r in tmo]
ok_ = st1 == "DONE" and tmo and all(2700 <= l <= 3400 for l in lat) and w1 < wb + 5 + 3.4
rec("TC-D04-13", "P" if ok_ else "F", f"qc2seedt2, worker LLM_DECISION_FAKE_P50_MS=5000/P95=6000, cổng INJECTION_FLAG (timeout_ms=3000) lớp 761987, chấm lại bài sv.gioi (chèn lệnh): run_state={st1}, tổng chờ {w1} s (baseline p50=426: {wb} s, {stb}); DECIDE: {[(r[0], r[1], r[2], r[3], r[5]) for r in d1]} → hết hạn ở {lat} ms; cờ INJECTION_SUSPECTED cuối: {'INJECTION_SUSPECTED' in (fl1 or '')} (đường FALLBACK)")
setenv(); all_off(C1)
