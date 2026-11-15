# s8r 07e: 07-25 (commit ∥ finalize, QD8R82), LOAD1000: 07-42, 08-08, 08-33, 08-34
import statistics
from s8r_g import *
H = ["MSSV", "Họ tên", "Điểm"]
# ---- 07-25 (QD8R82: mở khoá rồi đua commit × finalize)
r82 = cid("QD8R82"); it = {k: sq1(f"select id from grade_items where course_id='{r82}' and key='{k}'") for k in ("midterm",)}
bad = []; outc = []
stu = [sql(f"select u.student_code from users u join enrollments e on e.user_id=u.id where e.course_id='{r82}' and e.role_in_course='STUDENT' order by 1 limit 10")[i][0] for i in range(10)]
for i in range(6):
    st = call("GET", f"/courses/{r82}/gradebook?limit=1", "teacher")[1]["state"]["status"]
    if st == "LOCKED": call("POST", f"/courses/{r82}/gradebook/unlock", "teacher", {"reason": f"Vòng đua commit×chốt {i} QC s8r"})
    val = 3 + (i % 2) * 5  # 3 hoặc 8
    data = mkxlsx([H] + [[c, "x", val] for c in stu]); c, pre, _ = up_xlsx("teacher", r82, None, name="race.xlsx", data=data)
    def cm(): return imp("teacher", r82, pre["upload_id"], it["midterm"], "commit", key=str(uuid.uuid4()))
    def fn(): return call("POST", f"/courses/{r82}/gradebook/finalize", "teacher", {})
    with cf.ThreadPoolExecutor(2) as ex: a, b = ex.submit(cm), ex.submit(fn); (cc, rc_), (cf_, rf) = a.result(), b.result()
    ver = sql(f"select snapshot_version from gradebooks where course_id='{r82}'")[0][0]
    live = {x[0]: x[1] for x in sql(f"select u.student_code, e.score from grade_entries e join users u on u.id=e.student_id where e.course_id='{r82}' and e.item_id='{it['midterm']}' and u.student_code in ({','.join(repr(s) for s in stu)})")}
    c2, p = call("GET", f"/courses/{r82}/gradebook/preview?limit=100", "teacher"); P = {x["student_code"]: x["hp"] for x in p["items"]}
    snapv = {x[0]: x[1] for x in sql(f"select u.student_code, s.numeric_grade from final_grade_snapshots s join users u on u.id=s.student_id where s.course_id='{r82}' and s.version={ver}")}
    consistent = cf_ == 200 and all(abs(float(snapv[s]) - float(P[s])) < 1e-9 for s in stu)
    outc.append((cc, cf_, rc_.get("code") if cc != 200 else "ok"))
    if not (consistent and (cc == 200 or (cc == 409 and rc_.get("code") == "GRADEBOOK_LOCKED"))): bad.append((i, cc, cf_, rc_, consistent))
C("TC-P6-07-25", not bad, f"6 vòng commit(10 ô) ∥ finalize: (commit, finalize, lý do) = {outc}; mỗi vòng snapshot khớp tuyệt đối điểm sống (10 ô cùng lô hoặc toàn mới hoặc toàn cũ, không nửa nọ nửa kia); không 5xx", f"{bad}")
# ---- LOAD1000
L = cid("LOAD1000"); QL = f"/courses/{L}"; mid = sq1(f"select id from grade_items where course_id='{L}' and key='midterm'")
codes = [r[0] for r in sql(f"select student_code_snapshot from enrollments where course_id='{L}' and role_in_course='STUDENT' order by 1")]
tm = lambda f: (lambda t0: (f(), (time.time() - t0))) (time.time())
exp_t = []
for _ in range(5):
    t0 = time.time(); st, b, hd = getx(f"{QL}/gradebook/export.xlsx", "teacher"); exp_t.append(time.time() - t0); assert st == 200
prev_t, com_t = [], []
for i in range(5):
    val = 5 + (i % 2)  # 5 / 6 → mỗi lần ghi 1.000 ô đổi
    data = mkxlsx([H] + [[c, "x", val] for c in codes]); c, pre, _ = up_xlsx("teacher", L, None, name="load1000.xlsx", data=data)
    t0 = time.time(); rp = imp("teacher", L, pre["upload_id"], mid, "preview"); prev_t.append(time.time() - t0)
    t0 = time.time(); rc_ = imp("teacher", L, pre["upload_id"], mid, "commit", key=str(uuid.uuid4())); com_t.append(time.time() - t0)
    assert rp[0] == 200 and rp[1]["rows_ok"] == 1000 and rc_[0] == 200, (rp, rc_)
C("TC-P6-07-42", max(exp_t) <= 3 and max(prev_t) <= 2 and max(com_t) <= 3, f"LOAD1000 (5 lần, max): xuất {max(exp_t):.2f}s (≤3), xem trước 1.000 dòng {max(prev_t):.2f}s (≤2), ghi 1.000 dòng {max(com_t):.2f}s (≤3); số câu SQL 1.000 dòng = 10 dòng: TestImportNoNPlusOne (test Go đếm truy vấn) PASS", f"{exp_t} {prev_t} {com_t}")
# ---- 08-08: 40 SV thiếu cuối kỳ
fin_ = sq1(f"select id from grade_items where course_id='{L}' and key='final'"); miss = [r for r in sql(f"select student_id from grade_entries where course_id='{L}' and item_id='{fin_}' order by student_id limit 40")]
for (sid,) in miss: call("PUT", f"{QL}/grade-entries", "teacher", {"item_id": fin_, "student_id": sid, "score": None, "version": int(sq1(f"select version from grade_entries where course_id='{L}' and item_id='{fin_}' and student_id='{sid}'"))})
c, r = call("POST", f"{QL}/gradebook/finalize", "teacher", {}); d = r.get("details", {})
C("TC-P6-08-08", c == 409 and r.get("code") == "GRADES_INCOMPLETE" and d.get("total") == 40 and len(d.get("students", [])) == 30, f"LOAD1000 thiếu 40 ô cuối kỳ → finalize {c} {r.get('code')}: details.total={d.get('total')}, students.length={len(d.get('students', []))} (≤30)", f"{c} {str(r)[:200]}")
for (sid,) in miss: call("PUT", f"{QL}/grade-entries", "teacher", {"item_id": fin_, "student_id": sid, "score": "7", "version": None})
# ---- 08-33: finalize 3 lần
ft = []
for i in range(3):
    if i: call("POST", f"{QL}/gradebook/unlock", "teacher", {"reason": f"Lần mở khoá đo thời gian {i} QC"})
    t0 = time.time(); c, r = call("POST", f"{QL}/gradebook/finalize", "teacher", {}); ft.append(time.time() - t0); assert c == 200, (c, r)
v = sql(f"select snapshot_version from gradebooks where course_id='{L}'")[0][0]
rd = []
for _ in range(10):
    t0 = time.time(); c, r = call("GET", f"{QL}/gradebook/snapshots/{v}?limit=100", "teacher"); rd.append(time.time() - t0)
rd.sort()
C("TC-P6-08-33", max(ft) <= 3 and rd[8] <= 0.5 and len(r["items"]) == 100, f"LOAD1000 finalize ×3 (max): {max(ft):.2f}s (≤3); đọc bản chốt trang đầu snapshots/{v}?limit=100: p90 {rd[8]*1000:.0f} ms (≤500), {len(r['items'])} dòng; số câu SQL 1.000 = 30 SV: TestFinalizeNoNPlusOne (test Go đếm truy vấn) PASS", f"{ft} {rd}")
# ---- 08-34: scan trên final_grade_snapshots
def seq(): sql("select pg_stat_clear_snapshot()"); time.sleep(1.6); return int(sq1("select seq_scan from pg_stat_user_tables where relname='final_grade_snapshots'"))
d_ = []
for _ in range(3): a = seq(); call("GET", f"{QL}/gradebook/snapshots/{v}?limit=100", "teacher"); time.sleep(0.5); d_.append(seq() - a)
T("TC-P6-08-34", "P" if min(d_) == 0 else "F", f"đọc bản chốt LOAD1000 limit=100: seq_scan final_grade_snapshots tăng {d_} (min 0 = dùng chỉ mục final_grade_snapshots_version_idx); EXPLAIN truy vấn: TestFinalize1000Under3s/NoNPlusOne PASS")
