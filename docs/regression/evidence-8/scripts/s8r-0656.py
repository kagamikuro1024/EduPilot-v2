# TC-P6-06-56 (sửa TC theo P-11-54): (a) version kiểu chữ → 422; (b) thiếu/null version trên ô CÓ điểm → 409 và điểm KHÔNG đổi; (c) thiếu version trên ô MỚI → 200
import csv
from s8r_lib import *
CODE = "QD8R6W"; C0 = cid(CODE); Q = f"/courses/{C0}"
ITEM = sq1(f"select id from grade_items where course_id='{C0}' and key='midterm'")
CSV = {r["account"]: r for r in csv.DictReader(open(REPO + "/seed/expected_final_grades.csv"))}
SV = lambda a: sq1("select u.id from users u join enrollments e on e.user_id=u.id where e.course_id='%s' and e.role_in_course='STUDENT' and u.student_code='%s'" % (C0, CSV[a]["student_code"]))
cell = lambda s: sql(f"select score,version from grade_entries where course_id='{C0}' and item_id='{ITEM}' and student_id='{s}'")
put = lambda s, body: call("PUT", f"{Q}/grade-entries", "teacher", {"item_id": ITEM, "student_id": s, **body})
s30 = SV("sv30"); before = cell(s30)
res = {}
# (a) version chữ — ô có điểm và ô mới
ra = put(s30, {"score": "7", "version": "1"}); res["a_exist"] = (ra[0], ra[1].get("code"))
# (b) thiếu version / version null trên ô có điểm
rb1 = put(s30, {"score": "7"}); rb2 = put(s30, {"score": "7", "version": None}); after = cell(s30)
res["b_missing"] = (rb1[0], rb1[1].get("code")); res["b_null"] = (rb2[0], rb2[1].get("code")); res["b_db"] = (before, after)
okb = rb1[0] == 409 and rb2[0] == 409 and before == after
# (c) ô mới: tìm SV chưa có entry cho item (xoá entry của một SV trong lớp riêng chỉ để tạo ô mới)
cand = sql(f"select u.id from users u join enrollments e on e.user_id=u.id where e.course_id='{C0}' and e.role_in_course='STUDENT' and not exists (select 1 from grade_entries g where g.course_id='{C0}' and g.item_id='{ITEM}' and g.student_id=u.id) limit 1")
if cand:
    s_new = cand[0][0]; rc = put(s_new, {"score": "6"}); res["c_new"] = (rc[0], rc[1].get("code") if isinstance(rc[1], dict) else None, cell(s_new))
else:
    res["c_new"] = "không có SV chưa có điểm giữa kỳ trong lớp riêng"
print(res)
T("TC-P6-06-56a", "P" if res["a_exist"][0] == 422 else "F", f"version chữ → {res['a_exist']}")
T("TC-P6-06-56b", "P" if okb else "HIGH-F", f"thiếu version → {res['b_missing']}, null → {res['b_null']}; DB trước/sau {before}/{after}")
T("TC-P6-06-56c", "P" if (isinstance(res['c_new'], tuple) and res['c_new'][0] == 200) else "N", f"ô mới thiếu version → {res['c_new']}")
