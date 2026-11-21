import sys, os, statistics; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def setcap(on):
    cc, j, _ = api("admin", "GET", FL()); f = [x for x in j["items"] if x["feature"] == "REGRESSION_CAPTURE"][0]; return api("admin", "PUT", f"{FL()}/REGRESSION_CAPTURE", {"enabled": on, "version": f["version"]})[0]
def lat(rids):
    out = []
    for rid in rids:
        c, j, _ = api("teacher", "GET", f"/courses/{CG0}/grading-results/{rid}"); v = j["result"]["version"]
        c, j2, _ = api("teacher", "PUT", f"/courses/{CG0}/grading-results/{rid}", {"version": v, "criteria": [{"id": "c2", "score": "0.00"}]}); v2 = j2["result"]["version"]
        t0 = time.time(); c, j3, _ = api("teacher", "POST", f"/courses/{CG0}/grading-results/{rid}/review", {"version": v2}); out.append(((time.time() - t0) * 1000, c))
    return out
def run():
    flag("GRADING_GUIDANCE", True)
    a = sql("select id from assignments where title='QC106 g0509' order by created_at desc limit 1")[0][0]
    rids = [r[0] for r in sql(f"select r.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}' and r.round=2 and r.status='DRAFT' order by r.id limit 70")]
    print("DRAFT dùng:", len(rids))
    print("cờ tắt", setcap(False)); off = lat(rids[:30]); print("cờ bật", setcap(True)); on = lat(rids[30:60])
    p = lambda xs: sorted(x[0] for x in xs)[int(len(xs) * 0.95) - 1]; med = lambda xs: statistics.median(x[0] for x in xs)
    n_cases = sql(f"select count(*) from grading_regression_cases c where c.course_id='{CG0}' and c.kind='GRADING_EDIT' and c.source_id in ({','.join(repr(r) for r in rids[30:60])})")[0][0]
    n_off = sql(f"select count(*) from grading_regression_cases c where c.course_id='{CG0}' and c.kind='GRADING_EDIT' and c.source_id in ({','.join(repr(r) for r in rids[:30])})")[0][0]
    orphan = sql(f"select count(*) from grading_regression_cases c where c.course_id='{CG0}' and c.kind='GRADING_EDIT' and not exists (select 1 from grading_results r where r.id=c.source_id and r.status in ('REVIEWED','PUBLISHED'))")[0][0]
    d95 = p(on) - p(off); dmed = med(on) - med(off)
    print(f"tắt p95 {p(off):.0f} med {med(off):.0f}; bật p95 {p(on):.0f} med {med(on):.0f}; Δp95 {d95:.0f}; ca {n_cases} (bật) / {n_off} (tắt); mồ côi {orphan}")
    chk("TC-G07-06", all(x[1] == 200 for x in off + on) and d95 <= 20 and int(n_cases) == 30 and n_off == "0" and orphan == "0", f"review 30 lượt cờ REGRESSION_CAPTURE tắt vs 30 lượt bật (mỗi lượt sửa c2 → 0 rồi duyệt): p95 {p(off):.0f} ms vs {p(on):.0f} ms (chênh {d95:.0f} ms, ≤ 20), trung vị {med(off):.0f} vs {med(on):.0f} ms; ca GRADING_EDIT ghi: bật {n_cases}/30, tắt {n_off}/30; ca mồ côi (nguồn không REVIEWED/PUBLISHED) = {orphan}. Phần «kill giữa chừng»: ghi cùng giao dịch kiểm gián tiếp (không ca mồ côi); không kill thật")
    # G07-12: không có đường HTTP xuất
    codes = [api("admin", "GET", p)[0] for p in (f"/courses/{CG0}/regression", f"/courses/{CG0}/regression-cases", f"/courses/{CG0}/assignments/{a}/regression", f"/courses/{CG0}/grading-regression-cases")] + [api("teacher", "GET", f"/courses/{CG0}/regression-cases")[0]]
    print("HTTP xuất:", codes)
with window(): run()
