# QC smoke S10 (11-18): dựng lớp riêng QD10A / QD10B trên ep-qc để KHÔNG đụng 761987/761988 (E5).
# QD10A = lớp 1 (GV teacher@, TA ta@, SV sv.kha/gioi/nguyco/moi + sv31..34, bản sao 107 câu APPROVED đọc-chỉ từ 761987); QD10B = lớp 2 (teacher@, sv31, 12 câu).
import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from qlib import *
def mk(code, students, nq):
    ex = sql(f"select id from courses where class_code='{code}'")
    if ex: cid = ex[0][0]; print("có sẵn", code, cid)
    else:
        tid = sql("select id from users where email='teacher@edupilot.local'")[0][0]; taid = sql("select id from users where email='ta@edupilot.local'")[0][0]
        c, j, _ = api("admin", "POST", "/admin/courses", {"subject_code": "QCD", "class_code": code, "name": f"QC smoke10 {code}", "semester": "2026-2027-HK1", "capacity": 60, "teacher_id": tid})
        assert c in (200, 201), (c, j); cid = j["id"]
        c, j, _ = api("admin", "POST", f"/admin/courses/{cid}/assign", {"ta_ids": [taid]}); print("assign", c)
    jc = sql(f"select join_code from courses where id='{cid}'")[0][0]
    for s in students:
        c, j, _ = api(s, "POST", "/courses/join", {"code": jc}); print("join", s, c)
    have = int(sql(f"select count(*) from question_bank where course_id='{cid}'")[0][0])
    if have == 0:
        sql(f"""insert into question_bank (id,course_id,type,title,topic,difficulty,stem,answer_key,explanation,citations,origin,review_status,created_by,reviewed_by,reviewed_at)
          select uuidv7(),'{cid}',type,title,topic,difficulty,stem,answer_key,explanation,citations,origin,review_status,created_by,reviewed_by,reviewed_at
          from (select q.* from question_bank q join courses c on c.id=q.course_id where c.class_code='761987' and q.review_status='APPROVED' order by q.created_at limit {nq}) s""")
    print(code, cid, "câu:", sql(f"select count(*) from question_bank where course_id='{cid}'")[0][0])
mk("QD10A", ["sv.kha", "sv.gioi", "sv.nguyco", "sv.moi", "sv31", "sv32", "sv33", "sv34"], 107)
mk("QD10B", ["sv31"], 12)
mk("QD10A", ["sv36", "sv37"], 107)   # SV mới chưa có lượt nào (09-05, 09-19)
mk("QD10C", ["sv36"], 0)             # lớp chưa có câu APPROVED (06-23)
