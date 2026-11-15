# s8r 08b: tiếp chuỗi trên lớp đã LOCKED v1 (QD8R8): 08-13, 07-35..40, 08-16, 08-18, 08-21..28, 08-47, 09-04/38
from s8r_g import *
S1 = snap(1); H1 = h(1); sv4 = "sv04"; sv8 = "sv08"
T("TC-P6-08-27", "P" if notif("GRADE_FINALIZED") == 30 else "F", f"GRADE_FINALIZED = {notif('GRADE_FINALIZED')} cho cả lớp 30 sau lần chốt đầu; gửi lại cùng khoá không thêm (08-04: vẫn 30 snapshot)")
# ---- 08-13 + 07-37: nguồn sống đổi sau chốt
ex = sql(f"select id from exam_attempts where course_id='{C0}' and student_id='{SV(sv4)}' and status='GRADED'")[0][0]
sql(f"update exam_attempts set adjusted_score=0, adjusted_reason='QC s8r', adjusted_at=now(), adjusted_by='{TCH}' where id='{ex}'")
c, g = call("GET", f"{Q}/gradebook?limit=100", "teacher"); rows = {x["student"]["id"]: x for x in g["rows"]}
drift = [k for k, x in rows.items() if x.get("drift")]
c, p = call("GET", f"{Q}/gradebook/preview?limit=100", "teacher"); live_hp = [r for r in p["items"] if r["student_id"] == SV(sv4)][0]["hp"]; snap_hp = [x for x in S1 if x[0] == SV(sv4)][0][1]
c2, _ = call("GET", f"{Q}/gradebook?limit=2", "sv.gioi")
C("TC-P6-08-13", h(1) == H1 and drift == [SV(sv4)] and c2 == 403 and float(live_hp) != float(snap_hp), f"sửa điểm bài thi sv04 (nguồn sống 6→0): băm SNAP(1) không đổi; hàng sv04 drift=true & locked.version=1 (xem trước sống HP {live_hp} ≠ snapshot {float(snap_hp):.1f}); chỉ 1 hàng drift; STUDENT GET gradebook → {c2}; UI nhãn 'Khác bản chốt': s8r-ui-08", f"{h(1)==H1} {drift} {c2} {live_hp} {snap_hp}")
st, b, hd = getx(f"{Q}/gradebook/export.xlsx", "teacher"); rx, nf, xml = xlsx_rows(b)
hr = rx[1]; ihp = hr.index("Điểm học phần"); xr = [r for r in rx[2:] if r[0] == CSV[sv4]["student_code"]][0]
C("TC-P6-07-37", st == 200 and "Bản chốt v1" in " ".join(rx[0]) and float(xr[ihp]) == float(snap_hp) != float(live_hp), f"export khi LOCKED v1 sau khi nguồn sống đổi: hàng đầu '{' '.join(rx[0])[-60:]}'; HP sv04 trong tệp {xr[ihp]} = snapshot {float(snap_hp)} (sống {live_hp})", f"{st} {rx[0]} {xr}")
# 07-35/36/38/39/40 xuất
sql(f"update exam_attempts set adjusted_score=null, adjusted_reason=null, adjusted_at=null, adjusted_by=null where id='{ex}'")
