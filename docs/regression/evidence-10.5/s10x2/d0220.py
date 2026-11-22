# D02-20 (nhịp 2,5 s/bài để không chạm RPM 60 của provider LLM giả; D01-16 đã chạy lượt nhanh) + D01-16 trên qc2seedt2: cầu dao decision mở (LLM_DECISION_FAKE_MODE=error) → circuit_open ngay, đường FALLBACK, không chờ
import sys, statistics; from q105 import *
A = cid_of("761987")
U = ["sv.kha", "sv.gioi", "sv.nguyco", "sv04", "sv05", "sv06", "sv07", "sv08"]
def posts(n, tag, gap=0):
    out = []
    for i in range(n):
        u = U[i % len(U)]
        c, j, ms = post_thread(u, A, f"Cau dao {tag}{i} {int(time.time()*1000)%100000}", f"Mình muốn hỏi về cách ôn tập chương {i} của môn an ninh mạng, nhờ cả lớp góp ý giúp mình với nhé ({tag}{i}).")
        out.append((c, round(ms)))
        if gap: time.sleep(gap)
    return out
setenv(); all_off(A)
base = posts(8, "B", 2.5); bms = [m for c, m in base if c in (200, 201)]
print("baseline (cổng tắt)", base)
set_gates(A, on=("THREADS_PRECHECK",))
print("env:", setenv(LLM_DECISION_FAKE_MODE="error")); set_gates(A, on=("THREADS_PRECHECK",))
s = now_db(); on = posts(5, "E", 2.5) + posts(8, "F", 2.5)
rows = decide_rows(s, A, wait=6)
print("posts bật cổng+error", on)
for r in rows: print(r)
co = [r for r in rows if r[2] == "circuit_open"]; er = [r for r in rows if r[2] == "error"]
print("n DECIDE", len(rows), "error", len(er), "circuit_open", len(co))
lat_co = [int(r[5]) for r in co]; paths = sorted(set(r[1] for r in co))
ms_on = [m for c, m in on[5:] if c in (200, 201)]
mb = statistics.median(bms) if bms else None; mo = statistics.median(ms_on) if ms_on else None
print("median tắt", mb, "median bật+circuit_open", mo, "paths", paths, "latency_ms circuit_open", lat_co)
rec("TC-D01-16b", "P" if len(er) >= 5 and len(co) >= 3 and max(lat_co) <= 5 else "F", f"qc2seedt2 (LLM_PROVIDER=fake, worker+gateway LLM_DECISION_FAKE_MODE=error, cổng THREADS_PRECHECK lớp 761987): DECIDE {len(rows)} dòng: error {len(er)} → circuit_open {len(co)} (sau 5 lỗi liên tiếp, BREAKER_FAILS=5); latency_ms của dòng circuit_open {lat_co} (≤ 5 = không gọi mạng); đường {paths}")
rec("TC-D02-20", "P" if co and paths == ["FALLBACK"] and mo is not None and mb is not None and mo <= mb * 1.5 + 30 else "F", f"cầu dao mở, cổng THREADS_PRECHECK bật: dòng circuit_open đường {paths}; độ trễ đăng bài trung vị {mo} ms (cầu dao mở) vs {mb} ms (cổng tắt); POST bật {on}; POST tắt {base}")
fake_reset(); all_off(A)
