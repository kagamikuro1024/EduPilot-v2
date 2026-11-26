# D01-07 (lặp có nhịp ≤ 1,6 lời gọi/s để không chạm RPM 150): p50/p95 theo cấu hình + mặc định 426/682
import sys; from q105 import *
from concurrent.futures import ThreadPoolExecutor
A = cid_of("QD105A")
USERS = ["sv.kha", "sv.gioi", "sv.nguyco", "sv31", "sv32", "sv33"] + [f"sv{n:02d}" for n in range(4, 14)]
for u in USERS: tok(u)
def paced(n, tag, rate=1.6):
    T0 = time.time()
    def one(i):
        time.sleep(max(0, T0 + i / rate - time.time())); u = USERS[i % len(USERS)]
        c, j, ms = post_thread(u, A, f"Nhịp {tag}{i} {u} {int(time.time()*1000)%100000}", f"Cho mình hỏi cách ôn tập chương {i} của môn an ninh mạng sao cho hiệu quả nhất, bạn nào có kinh nghiệm chỉ giúp mình với ạ ({u}).")
        return c
    with ThreadPoolExecutor(8) as ex: return list(ex.map(one, range(n)))
def summarize(rows):
    lat = sorted(int(r[5]) for r in rows if r[2] == "ok"); n = len(lat)
    return n, lat[n // 2], lat[min(n - 1, int(n * 0.95))]
set_gates(A, on=("THREADS_PRECHECK",))
fake_reset(); fake(decision_p50_ms=100, decision_p95_ms=150, decision_mode="always_no", decision_error_rate=0)
s = now_db(); cs = paced(160, "A", 2.0); rows = decide_rows(s, A, wait=8); n, p50, p95 = summarize(rows)
stt = {}; [stt.__setitem__(r[2], stt.get(r[2], 0) + 1) for r in rows]
rec("TC-D01-07", "P" if len(rows) == 160 and stt.get("ok") == 160 and 80 <= p50 <= 120 and 120 <= p95 <= 180 else "F", f"cấu hình P50=100/P95=150, 160 lời gọi (nhịp 2/s): llm_audit {len(rows)} dòng {stt}; latency_ms p50={p50} p95={p95} (±20 %: 80–120 / 120–180); HTTP {sorted(set(cs))}")
fake_reset(); fake(decision_mode="always_no")
s = now_db(); cs = paced(100, "B", 2.0); rows = decide_rows(s, A, wait=8); n, p50, p95 = summarize(rows)
stt = {}; [stt.__setitem__(r[2], stt.get(r[2], 0) + 1) for r in rows]
rec("TC-D01-07m", "P" if stt.get("ok") == 100 and 340 <= p50 <= 520 and 545 <= p95 <= 830 else "F", f"mặc định (p50/p95 = 426/682, lỗi 0): 100 lời gọi → {stt}; latency_ms p50={p50} p95={p95} (kỳ vọng ≈426 / ≈682 ±20 %)")
fake_reset(); all_off(A)
