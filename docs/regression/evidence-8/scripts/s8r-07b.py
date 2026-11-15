# s8r 07b: QD8R7 — xuất OPEN (07-35/36/39/40), commit (07-18..22, 54), MSSV #20 (07-14..16, 55..58), xuất inject (07-38)
from s8r_g import *
XD = "/tmp/qc2-s8r-xlsx"; MID = ITEM["midterm"]
N_E = lambda: int(sq1(f"select count(*) from grade_entries where course_id='{C0}'")); N_A = lambda: int(sq1(f"select count(*) from audit_log where course_id='{C0}'"))
def vers(item=MID): return {r[0]: (r[1], int(r[2]), r[3]) for r in sql(f"select student_id,score,version,source from grade_entries where course_id='{C0}' and item_id='{item}'")}
def commit(path, item=MID, key=None, skip=False, name=None, data=None):
    c, pre, sha = up_xlsx("teacher", C0, path, name=name, data=data); assert c == 200, (c, pre)
    return imp("teacher", C0, pre["upload_id"], item, "commit", key=key, skip_errors=skip), pre
# ---- xuất khi OPEN
a0 = N_A(); st, b, hd = getx(f"{Q}/gradebook/export.xlsx", "teacher"); rx, nf, xml = xlsx_rows(b)
C("TC-P6-07-35", st == 200 and hd.get("Cache-Control") == "no-store" and "attachment" in hd.get("Content-Disposition", "") and len(b) > 1000, f"export OPEN → {st}; Cache-Control {hd.get('Cache-Control')}; Content-Disposition {hd.get('Content-Disposition')}; {len(b)} B mở được", f"{st} {hd}")
cols = ["MSSV", "Họ tên", "Giữa kỳ", "Kiểm tra hằng tuần", "Thi cuối kỳ", "Điểm cộng", "Điểm trừ", "Điểm quá trình", "Điểm thi cuối kỳ", "Điểm học phần", "Điểm chữ", "Ghi chú"]
C("TC-P6-07-36", "Bản xem trước — chưa chốt" in " ".join(rx[0]) and CODE in rx[0] and rx[1] == cols and len(rx) - 2 == 30, f"hàng đầu {rx[0]}; tiêu đề cột đúng thứ tự; {len(rx)-2} dòng sinh viên", f"{rx[0]} {rx[1]} {len(rx)}")
ex_n = sq1(f"select count(*) from audit_log where course_id='{C0}' and action='export'"); last = sql(f"select after::text from audit_log where course_id='{C0}' and action='export' order by created_at desc limit 1")
C("TC-P6-07-39", int(ex_n) >= 1 and last and "30" in last[0][0], f"audit_log action='export': {ex_n} dòng; mới nhất {last[0][0][:100]}", f"{ex_n} {last}")
rr = [getx(f"{Q}/gradebook/export.xlsx", u)[0] for u in ("ta", "sv.gioi", "admin")]
C("TC-P6-07-40", rr == [403, 403, 403], f"export bởi TA/STUDENT/ADMIN → {rr}", str(rr))
# ---- 07-18 commit midterm.xlsx sau khi 3 ô bị sửa tay
for a, v in (("sv04", "5"), ("sv05", "6"), ("sv06", "7")): setc(a, "midterm", v)
V0 = vers(); a0 = N_A(); act0 = {r[0]: int(r[1]) for r in sql(f"select action,count(*) from audit_log where course_id='{C0}' group by 1")}
K = str(uuid.uuid4()); (c, r), pre = commit(f"{XD}/midterm.xlsx", key=K); V1 = vers(); act1 = {r_[0]: int(r_[1]) for r_ in sql(f"select action,count(*) from audit_log where course_id='{C0}' group by 1")}
chg = [k for k in V0 if V0[k][:2] != V1[k][:2]]; same = [k for k in V0 if V0[k][:2] == V1[k][:2]]
ok = c == 200 and r["rows_ok"] == 30 and len(chg) == 3 and all(V1[k][1] == V0[k][1] + 1 and V1[k][2] == "IMPORT" for k in chg) and all(V1[k] == V0[k] for k in same if V0[k][2] == "IMPORT")
dd = {k: act1.get(k, 0) - act0.get(k, 0) for k in act1 if act1.get(k, 0) != act0.get(k, 0)}
C("TC-P6-07-18", ok, f"3 ô bị sửa tay → commit midterm.xlsx: 200 (ok {r['rows_ok']}, overwrites {r.get('overwrites')}, unchanged {r.get('unchanged')}); 3 ô đổi version+1 source IMPORT; 27 ô giống cũ không version mới; audit mới {dd} (3 dòng điểm + 1 dòng lô)", f"{c} {r} chg {len(chg)} {dd}")
# 07-20 cùng key
c2, r2 = imp("teacher", C0, pre["upload_id"], MID, "commit", key=K, skip_errors=False)
C("TC-P6-07-20", c2 == 200 and r2 == r and vers() == V1 and N_A() == a0 + sum(dd.values()), f"commit lại cùng Idempotency-Key → {c2}, phản hồi y hệt lần đầu; version/audit không đổi", f"{c2} {r2} vs {r}")
# 07-19 cùng tệp khoá mới
a1 = N_A(); (c3, r3), _ = commit(f"{XD}/midterm.xlsx")
C("TC-P6-07-19", c3 == 200 and r3.get("unchanged") == r3["rows_ok"] == 30 and vers() == V1 and N_A() == a1, f"commit lần hai cùng tệp, khoá mới: unchanged {r3.get('unchanged')} = rows_ok {r3['rows_ok']}; 0 version mới; audit {a1}→{N_A()} (0 dòng mới)", f"{c3} {r3} {a1} {N_A()}")
# 07-21 errors không skip
a2, e2, Vb = N_A(), N_E(), vers(); (c4, r4), _ = commit(f"{XD}/errors.xlsx")
C("TC-P6-07-21", c4 == 409 and r4.get("code") == "IMPORT_HAS_ERRORS" and vers() == Vb and N_E() == e2 and N_A() == a2, f"errors.xlsx commit không skip_errors → {c4} {r4.get('code')}; không ghi dòng nào (kể cả dòng hợp lệ): version/entries/audit giữ nguyên", f"{c4} {r4}")
# 07-22 skip_errors
(c5, r5), _ = commit(f"{XD}/errors.xlsx", skip=True); V2 = vers()
b_, s6 = SV("sv.kha"), SV("sv06")
ch = [k for k in Vb if Vb[k][:2] != V2[k][:2]]
C("TC-P6-07-22", c5 == 200 and sorted(ch) == sorted([b_, s6]) and V2[b_][0] in ("7.00", "7") and V2[s6][0] in ("8.50", "8.5") and r5.get("skipped_errors") == 6, f"skip_errors:true → {c5}: ghi {r5.get('written')} dòng hợp lệ (B=7, sv06=8,5) trong một lô, bỏ {r5.get('skipped_errors')} dòng lỗi; ô khác không đổi (đổi: {len(ch)}); UI nút 'Bỏ qua N dòng lỗi và ghi M dòng': dev e2e 'import errors listed' xanh + s8r-ui-07", f"{c5} {r5} {ch}")
commit(f"{XD}/midterm.xlsx")  # khôi phục
# 07-54 hai commit song song
for a, v in (("sv04", "5"), ("sv05", "6")): setc(a, "midterm", v)
Vp = vers(); c, pre = up_xlsx("teacher", C0, f"{XD}/midterm.xlsx")[:2]; c_, pre_ = up_xlsx("teacher", C0, f"{XD}/midterm.xlsx")[:2]
def go(p): return imp("teacher", C0, p["upload_id"], MID, "commit", key=str(uuid.uuid4()), skip_errors=False)
with cf.ThreadPoolExecutor(2) as ex: rs = list(ex.map(go, [pre, pre_]))
Vq = vers(); chg2 = [k for k in Vp if Vp[k][:2] != Vq[k][:2]]
C("TC-P6-07-54", all(x[0] == 200 for x in rs) and all(Vq[k][1] == Vp[k][1] + 1 for k in chg2) and len(chg2) == 2, f"2 commit song song (khoá khác nhau) → {[x[0] for x in rs]}; sau cùng bằng một lần commit: 2 ô đổi, mỗi ô version+1 (không đôi); không 5xx/deadlock", f"{rs} {chg2}")
