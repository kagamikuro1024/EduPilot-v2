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
    with ThreadPoolExecutor(6) as ex: return list(ex.map(one, range(n)))
def summarize(rows):
    lat = sorted(int(r[5]) for r in rows if r[2] == "ok"); n = len(lat)
    return n, lat[n // 2], lat[min(n - 1, int(n * 0.95))]
PH = sys.argv[1]

if PH == "a":
    set_gates(A, on=("THREADS_PRECHECK",))
    fake_reset(); fake(decision_p50_ms=100, decision_p95_ms=150, decision_mode="always_no", decision_error_rate=0)
    s = now_db(); cs = paced(200, "A"); rows = decide_rows(s, A, wait=8)
    k = 0
    while len(rows) < 200 and k < 5: k += 1; cs += paced(210 - len(rows), f"A{k}x"); rows = decide_rows(s, A, wait=8)  # bù bài đăng bị 409/422 trước khi tới decision
    n, p50, p95 = summarize(rows)
    stt = {}; [stt.__setitem__(r[2], stt.get(r[2], 0) + 1) for r in rows]
    rec("TC-D01-07a", "P" if len(rows) >= 200 and stt.get("ok") == len(rows) and 80 <= p50 <= 120 and 120 <= p95 <= 180 else "F", f"cấu hình P50=100/P95=150, 200 lời gọi (nhịp 1,6/s): llm_audit {len(rows)} dòng {stt}; latency_ms p50={p50} p95={p95} (±20 %: 80–120 / 120–180); HTTP {sorted(set(cs))}")
    fake_reset(); all_off(A)
if PH == "c":
    set_gates(A, on=("THREADS_PRECHECK",))
    fake_reset(); fake(decision_mode="always_no")
    s = now_db(); cs = paced(120, "B"); rows = decide_rows(s, A, wait=8); n, p50, p95 = summarize(rows)
    stt = {}; [stt.__setitem__(r[2], stt.get(r[2], 0) + 1) for r in rows]
    rec("TC-D01-07c", "P" if stt.get("ok") == 120 and 340 <= p50 <= 520 and 545 <= p95 <= 830 else "F", f"mặc định (p50/p95 = 426/682, lỗi 0): 120 lời gọi → {stt}; latency_ms p50={p50} p95={p95} (kỳ vọng ≈426 / ≈682 ±20 %)")
    fake_reset(); all_off(A)
if PH == "b":
    set_gates(A, on=("THREADS_PRECHECK",))
    # phần b: lỗi 0,5. Cầu dao (5 lỗi liên tiếp → mở ~30 s) làm dòng circuit_open; chạy đến khi đủ 200 lời gọi THỰC (ok + error) rồi tính tỉ lệ lỗi.
    fake_reset(); fake(decision_p50_ms=100, decision_p95_ms=150, decision_mode="always_no", decision_error_rate=0.5)
    s = now_db(); posts = 0; real = 0; fd0 = stats()["fake_decisions"]
    while real < 200 and posts < 1200:
        fake(decision_p50_ms=100, decision_p95_ms=150, decision_mode="always_no", decision_error_rate=0.5)  # đặt lại mỗi lô phòng nhóm khác đổi bản giả
        paced(20, f"C{posts}"); posts += 20
        rows = decide_rows(s, A, wait=6)
        real = sum(1 for r in rows if r[2] in ("ok", "error"))
        if sum(1 for r in rows[-20:] if r[2] == "circuit_open"): time.sleep(35)
    rows = decide_rows(s, A, wait=8); stt = {}
    for r in rows: stt[(r[2], r[3])] = stt.get((r[2], r[3]), 0) + 1
    ok_n = sum(v for (k, _), v in stt.items() if k == "ok"); er_n = sum(v for (k, _), v in stt.items() if k == "error"); rate = er_n / max(1, ok_n + er_n)
    fd1 = stats()["fake_decisions"]; cfg = fake()
    rec("TC-D01-07b", "P" if ok_n + er_n >= 200 and 0.4 <= rate <= 0.6 else "F", f"error_rate=0,5, {posts} bài đăng nhịp 1,6/s: llm_audit {len(rows)} dòng {stt}; lời gọi thực {ok_n + er_n} (ok {ok_n}, error {er_n}) tỉ lệ lỗi {rate:.3f} (kỳ vọng 0,4–0,6); lời gọi tới provider giả (stats.fake_decisions) {fd1 - fd0}; cấu hình cuối {cfg}")
    fake_reset(); all_off(A)
