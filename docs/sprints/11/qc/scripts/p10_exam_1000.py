# TC-06-21 (Q-S11-2): lớp có ≥ 1.000 bài nộp (34 bài PUBLISHED × 30 SV) → 30 lượt mỗi endpoint #8–#12, p95 ≤ 500 ms; cursor không trùng; không OFFSET.
# Fixture ở lớp riêng QCT4 (GV qct2), xoá bằng `python3 p10_exam_1000.py clean`. KHÔNG đụng lớp seed.
import statistics, subprocess, sys, re
from p10lib import *
step = sys.argv[1] if len(sys.argv) > 1 else "all"
C1 = course("teacher", "761987")
def cid4():
    r = sql("select id from courses where class_code='QCT4'"); return r[0][0] if r else None
if step in ("all", "load"):
    if not cid4():
        c, j, _, _ = api("admin", "POST", "/admin/courses", {"subject_code": "QCT", "class_code": "QCT4", "name": "QC 1.000 bài nộp", "semester": "2026-2027-HK1", "capacity": 100, "teacher_id": uid("qct2")}); assert c == 201, (c, j)
    C4 = cid4(); T = uid("qct2")
    sql(f"""insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot)
      select '{C4}', e.user_id, 'STUDENT','ACTIVE','ADMIN', e.student_code_snapshot from enrollments e where e.course_id='{C1}' and e.role_in_course='STUDENT' and e.status='ACTIVE' on conflict do nothing""")
    sql(f"""insert into question_bank (course_id,type,title,topic,difficulty,stem,answer_key,explanation,origin,review_status,created_by,reviewed_by,reviewed_at)
      select '{C4}', type,title,topic,difficulty,stem,answer_key,explanation,origin,review_status,'{T}','{T}',now() from question_bank where course_id='{C1}' and review_status='APPROVED' and type='MCQ_SINGLE' limit 12""")
    sql(f"""insert into exams (course_id,title,kind,status,opens_at,closes_at,duration_minutes,max_score,rounding_step,multi_scoring,reveal_answers,appeal_days,published_at,created_by)
      select '{C4}', 'QC bài ' || g, 'MCQ','PUBLISHED', now() - ((37 - g) * interval '7 days'), now() - ((37 - g) * interval '7 days') + interval '1 hour', 30, 10, 0.01,'PARTIAL', true, 7, now() - ((37 - g) * interval '7 days') + interval '2 hours', '{T}' from generate_series(1,34) g""")
    sql(f"""insert into exam_items (course_id,exam_id,question_id,position,points)
      select '{C4}', e.id, q.id, q.rn, 1 from exams e cross join (select id, row_number() over (order by id) rn from question_bank where course_id='{C4}') q where e.course_id='{C4}'""")
    # 34 × 30 lượt GRADED; điểm giảm dần theo bài với ~1/6 SV (để declining có kết quả); breakdown theo từng câu
    sql(f"""insert into exam_attempts (course_id,exam_id,student_id,status,started_at,deadline_at,submitted_at,submit_reason,auto_score,breakdown,graded_at)
      select '{C4}', e.id, en.user_id, 'GRADED', e.opens_at + interval '5 minutes', e.opens_at + interval '35 minutes', e.opens_at + interval '25 minutes','MANUAL', sc.score,
        (select jsonb_agg(jsonb_build_object('item_id', i.id, 'max','1.00','earned', case when random() < 0.35 + (i.position % 5) * 0.12 then '1' else '0' end, 'points','1.00') order by i.position) from exam_items i where i.exam_id=e.id),
        e.opens_at + interval '30 minutes'
      from exams e join enrollments en on en.course_id=e.course_id and en.role_in_course='STUDENT' and en.status='ACTIVE'
      cross join lateral (select round((case when abs(hashtext(en.user_id::text)) % 6 = 0 then 9 - (extract(epoch from e.opens_at) - extract(epoch from now() - interval '259 days')) / 86400 / 40 else 4 + random() * 5 end)::numeric, 2) as score) sc
      where e.course_id='{C4}' on conflict do nothing""")
    print("QCT4", C4, "bài thi", sql(f"select count(*) from exams where course_id='{C4}'")[0][0], "lượt GRADED", sql(f"select count(*) from exam_attempts where course_id='{C4}'")[0][0])
if step in ("all", "measure"):
    C4 = cid4(); ex = sql(f"select id from exams where course_id='{C4}' order by opens_at desc limit 1")[0][0]
    def lat(path, n=30, user="qct2"):
        ms = []
        for _ in range(n):
            c, _, t, _ = api(user, "GET", path); assert c == 200, (c, path); ms.append(t)
        ms.sort(); return ms[int(.95 * n) - 1], statistics.median(ms)
    for name, path in (("#8 overview", f"/courses/{C4}/analytics/exams/overview?exam_id={ex}"), ("#9 trend", f"/courses/{C4}/analytics/exams/trend"), ("#10 hard-items", f"/courses/{C4}/analytics/exams/hard-items?exam_id={ex}&limit=100"),
                       ("#11 declining", f"/courses/{C4}/analytics/exams/declining?limit=100"), ("#12 compare", "/analytics/exams/compare")):
        p95, p50 = lat(path); ok(f"TC-06-21 1.000 bài nộp {name}: p95 ≤ 500 ms", p95 <= 500, f"p50={p50:.0f} p95={p95:.0f} ms, load máy={subprocess.run('uptime', capture_output=True, text=True).stdout.split('load averages:')[-1].split()[0]}")
    for name, base in (("hard-items", f"/courses/{C4}/analytics/exams/hard-items?exam_id={ex}"), ("declining", f"/courses/{C4}/analytics/exams/declining")):
        seen, cur, pages = [], None, 0
        while True:
            c, j, _, _ = api("qct2", "GET", base + "&limit=5" if "?" in base else base + "?limit=5") if cur is None else api("qct2", "GET", base + ("&" if "?" in base else "?") + f"limit=5&cursor={cur}")
            assert c == 200, (c, j); seen += [x.get("item_id") or x.get("student_id") or x.get("id") for x in j["items"]]; pages += 1; cur = j.get("next_cursor")
            if not cur or pages > 100: break
        ok(f"TC-06-21 cursor {name}: duyệt hết không trùng", len(seen) == len(set(seen)) and len(seen) > 0, f"{len(seen)} dòng / {pages} trang (limit=5)")
    ok("TC-06-21 không OFFSET trong mã analytics", subprocess.run(["grep", "-rniE", r"\boffset\b", "../../../../../backend-go/internal/exam/analytics.go", "../../../../../backend-go/internal/store/queries/exam_analytics.sql"], capture_output=True, text=True).stdout.strip() == "", "grep OFFSET = 0")
if step in ("all", "clean"):
    C4 = cid4()
    if C4:
        for t in ("exam_attempts", "exam_items", "exams", "question_bank"): sql(f"delete from {t} where course_id='{C4}'")
        sql(f"delete from enrollments where course_id='{C4}' and role_in_course='STUDENT'")
        print("đã dọn fixture QCT4 (giữ lớp)")
