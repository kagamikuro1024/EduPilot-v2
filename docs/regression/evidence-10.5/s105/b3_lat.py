# D01-07: độ trễ / tỉ lệ lỗi decision giả theo cấu hình (route thử, không restart). 200 lời gọi thật qua cổng THREADS_PRECHECK ở QD105A.
import sys, os, statistics; from q105 import *
from concurrent.futures import ThreadPoolExecutor
A = cid_of("QD105A")
USERS = ["sv.kha", "sv.gioi", "sv.nguyco", "sv31", "sv32", "sv33"] + [f"sv{n:02d}" for n in range(4, 14)]
BODY = "Cho mình hỏi cách ôn tập chương {i} của môn an ninh mạng sao cho hiệu quả nhất, bạn nào có kinh nghiệm chỉ giúp mình với ạ ({u})."
def burst(n, tag):
    for u in USERS: tok(u)
    def one(i):
        u = USERS[i % len(USERS)]; c, j, ms = post_thread(u, A, f"Ôn tập {tag}{i} {u} {int(time.time()*1000)%100000}", BODY.format(i=i, u=u)); return c
    with ThreadPoolExecutor(8) as ex: return list(ex.map(one, range(n)))
def stat(s, n_expect):
    rows = decide_rows(s, A, wait=6, extra="")
    return rows
if sys.argv[1] == "lat":
    fake_reset(); set_gates(A, on=("THREADS_PRECHECK",))
    fake(decision_p50_ms=100, decision_p95_ms=150, decision_mode="always_no", decision_error_rate=0)
    s = now_db(); cs = burst(200, "L"); rows = decide_rows(s, A, wait=8)
    lat = sorted(int(r[5]) for r in rows if r[2] == "ok"); n = len(lat)
    p50, p95 = lat[n // 2], lat[min(n - 1, int(n * 0.95))]
    stt = {}; [stt.__setitem__(r[2], stt.get(r[2], 0) + 1) for r in rows]
    okA = len(rows) == 200 and 80 <= p50 <= 120 and 120 <= p95 <= 180 and set(cs) <= {201} and stt.get("ok") == 200
    rec("TC-D01-07a", "P" if okA else "F", f"cấu hình p50=100/p95=150: 200 lời gọi → llm_audit {len(rows)} dòng {stt}; latency_ms p50={p50} p95={p95} (±20 %: 80–120 / 120–180); HTTP {sorted(set(cs))}")
    print("lat", p50, p95)
    fake(decision_error_rate=0.5)
    s = now_db(); cs2 = burst(200, "E"); rows = decide_rows(s, A, wait=10)
    stt = {}; [stt.__setitem__((r[2], r[3]), stt.get((r[2], r[3]), 0) + 1) for r in rows]
    att = [r for r in rows if r[2] in ("ok", "error")]; er = sum(r[2] == "error" for r in att) / max(1, len(att))
    print("err", len(rows), stt, er, sorted(set(cs2)))
    rec("TC-D01-07b", "P" if 0.4 <= er <= 0.6 else "F", f"error_rate=0,5: 200 lời gọi → {len(rows)} dòng {stt}; tỉ lệ lỗi trên lời gọi thực tế {er:.3f} (0,4–0,6); HTTP {sorted(set(cs2))}")
    fake_reset()
    s = now_db(); cs3 = burst(40, "D"); rows = decide_rows(s, A, wait=10)
    # mặc định 426/682 (không đặt mode): 'always_no' để khỏi Structured
    all_off(A)
