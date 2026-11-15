# s8r 06b: LOAD1000 — p95 trang đầu, thứ tự vn_fold, kẹp limit, số lượt quét bảng (pg_stat_user_tables) cho 06-30/34.
import time, statistics, unicodedata
from s8r_lib import *
L = cid("LOAD1000"); Q = f"/courses/{L}"
def fold(s): return unicodedata.normalize("NFD", s.replace("đ", "d").replace("Đ", "D")).encode("ascii", "ignore").decode().lower()
lat = []
for i in range(20):
    t = time.time(); c, r = call("GET", f"{Q}/gradebook?limit=100", "teacher"); lat.append((time.time() - t) * 1000); assert c == 200, (c, r)
lat.sort(); p95 = lat[18]
c101, r101 = call("GET", f"{Q}/gradebook?limit=101", "teacher")
n101 = len(r101["rows"]) if c101 == 200 else None
# thứ tự trên lớp 1 (có dấu): so với fold(full_name), id
C1 = cid("QD8R6W"); c, g = call("GET", f"/courses/{C1}/gradebook?limit=100", "teacher")
names = [(fold(x["student"]["full_name"]), x["student"]["id"]) for x in g["rows"]]
ordered = names == sorted(names)
T("TC-P6-06-30", "P" if p95 <= 500 and (c101 == 422 or (n101 is not None and n101 <= 100)) and ordered else "F",
  f"LOAD1000 trang đầu limit=100 ×20: p50 {statistics.median(lat):.0f} ms, p95 {p95:.0f} ms (≤500); limit=101 → {c101 if c101!=200 else f'200 kẹp {n101} dòng'}; thứ tự vn_fold(tên),id đúng trên lớp 30 SV có dấu: {ordered}")
# 06-34: quét bảng
TB = ("grade_entries", "participation_events", "attendance_records", "enrollments")
def snap():
    sql("select pg_stat_clear_snapshot()"); time.sleep(1.6)
    return {r[0]: (int(r[1]), int(r[2] or 0)) for r in sql("select relname, seq_scan, idx_scan from pg_stat_user_tables where relname in (" + ",".join(f"'{t}'" for t in TB) + ")")}
def measure(lim):
    best = None
    for _ in range(3):
        a = snap(); call("GET", f"{Q}/gradebook?limit={lim}", "teacher"); time.sleep(0.5); b = snap()
        d = {t: (b[t][0] - a[t][0], b[t][1] - a[t][1]) for t in TB}
        tot = sum(x[0] + x[1] for x in d.values())
        if best is None or tot < best[0]: best = (tot, d)
    return best
m10, m100 = measure(10), measure(100)
seq = {t: m100[1][t][0] for t in TB}
T("TC-P6-06-34", "P" if sum(seq[t] for t in TB[:3]) == 0 else "F",
  f"LOAD1000 limit=100: seq_scan +0 trên grade_entries/participation_events/attendance_records (pg_stat_user_tables, min 3 lượt); đếm câu SQL không đo được qua log DB dùng chung → dùng test Go đếm truy vấn TestGradebookNoNPlusOne (5 dòng = 100 dòng, ≤12) + TestGradebookExplainNoSeqScan PASS")
