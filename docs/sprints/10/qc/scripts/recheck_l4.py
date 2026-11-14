# Kiểm lại L4: dữ liệu mẫu (cần stack seed lại từ trống)
from pa import *
C = C1
r = lambda q: sql(q)
t = r(f"select count(*), count(*) filter (where jsonb_array_length(citations)>0) from question_bank where course_id='{C}' and review_status='APPROVED'")[0]
ok("RECHECK-L4a ≥80 APPROVED, mọi câu (trừ CODE/SHORT?) có citations", int(t[0]) >= 80 and int(t[1]) >= 80, f"APPROVED={t[0]} có citations={t[1]}")
e = r(f"select count(*) from question_bank where course_id='{C}' and review_status='PENDING' and origin='EXTRACTED'")[0][0]; ok("RECHECK-L4b PENDING có câu EXTRACTED", int(e) > 0, f"{e}")
q = r(f"select status from exams where course_id='{C}' and title ilike 'Quiz%'"); ok("RECHECK-L4c Quiz PUBLISHED", q and q[0][0] == "PUBLISHED", str(q))
l2 = r(f"select count(*) from question_bank where course_id='{C2}'")[0][0]; ov = r(f"select count(*) from question_bank a join question_bank b on a.stem=b.stem and a.course_id='{C1}' and b.course_id='{C2}'")[0][0]
ok("RECHECK-L4d lớp 2 có ngân hàng nhỏ, không trùng lớp 1", int(l2) > 0 and int(ov) == 0, f"lớp 2={l2} câu, trùng={ov}")
a = r(f"select count(*), count(distinct student_id) from practice_attempts where course_id='{C}'")[0]; print("lượt", a)
