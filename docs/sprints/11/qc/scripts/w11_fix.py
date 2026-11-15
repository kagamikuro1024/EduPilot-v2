# Đợt R · kiểm lại sau fix/dr-01 (P-11-16 D-R-01, P-11-17 D-R-03) + fix/dr-02 (D-R-05/06) ở main mới
from rlib import *
import json, time, hashlib, ssl, urllib.request, datetime as dt, uuid
exec(open("w4_assign.py").read().split("# 1 tạo + công bố")[0].split("from rlib import *")[1])  # mk(), submit(), iso(), rub, tag, c1
c1 = C("teacher")
def pre(t, b): c, j, _ = api("sv.nguyco", "POST", f"/courses/{c1}/threads/precheck", {"title": t, "body": b}); return c, j
OK = [("Worm", "Em nghĩ virus cần vật chủ."), ("Worm", "Mình nghĩ vậy"), ("Ý kiến", "Theo em nghĩ thì tường lửa nên đặt ở biên mạng."), ("Cho em hỏi", "Em được hỏi worm khác virus ở điểm nào?")]
for t, b in OK:
    c, j = pre(t, b); ok(f"R-D01-ok «{b[:34]}» không bị chặn", c == 200 and j.get("allowed") is True, f"{j.get('allowed')} {[r['type'] for r in j.get('reasons', [])]}")
BAD = [("Điểm", "Em nghỉ mấy buổi rồi, em có bị cấm thi không?"), ("Điểm", "Điểm giữa kỳ của em được bao nhiêu vậy ạ?"), ("Hỏi", "Em vắng 3 buổi có bị trừ điểm không?"), ("MSSV", "MSSV của em là 20229002 xem giúp em điểm."), ("Điểm", "Em nghi may buoi roi, con duoc thi khong?")]
for t, b in BAD:
    c, j = pre(t, b); ok(f"R-D01-bad «{b[:34]}» vẫn bị chặn (recall cá nhân giữ)", c == 200 and j.get("allowed") is False, f"{j.get('allowed')} {[r['type'] for r in j.get('reasons', [])]}")
# D-R-03: bài ngắn / bài trống
c, j, aid = mk(f"QC-R fix BT {tag}", iso(2)); ok("R-D03-00 tạo bài tập", aid is not None, str(c))
c, s1, sid1 = submit("sv.nguyco", aid, ["Tuong lua loc goi tin. Vi du chan IP la."])      # ~40 ký tự
c, s2, sid2 = submit("sv.kha", aid, [" "])                                                    # gần như rỗng
c, gj, _ = api("teacher", "POST", f"/courses/{c1}/assignments/{aid}/grade-all", {})
time.sleep(25)
rows = sql(f"select u.email, length(s.extracted_text), r.total, r.overall_feedback, r.flags::text from submissions s join users u on u.id=s.student_id left join grading_results r on r.submission_id=s.id where s.assignment_id='{aid}'") if True else []
for r in rows: print("  ", r)
fb = {r[0]: r for r in rows}
ng = fb.get("sv.nguyco@edupilot.local")
ok("R-D03-01 bài 40 ký tự: lời «quá ngắn … giảng viên chấm tay», không còn «không có nội dung»", ng and "quá ngắn" in (ng[3] or "") and "không có nội dung" not in (ng[3] or "") and "Bài bài" not in (ng[3] or ""), f"{ng[3] if ng else None}")
kh = fb.get("sv.kha@edupilot.local")
ok("R-D03-02 bài rỗng: «Bài nộp không có nội dung để chấm.» (không «Bài bài»)", kh and "Bài nộp không có nội dung để chấm." in (kh[3] or "") and "Bài bài" not in (kh[3] or ""), f"{kh[3] if kh else None} len={kh[1] if kh else None}")
ok("R-D03-03 cờ EMPTY_SUBMISSION ở cả hai bài", all(r[4] and "EMPTY_SUBMISSION" in r[4] for r in rows if r[3]), f"{[r[4] for r in rows]}")
print("w11 done", aid)
