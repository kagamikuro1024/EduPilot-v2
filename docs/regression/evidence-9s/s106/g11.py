import sys, os, re; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
sql("update users set full_name='Dũng Nguyễn' where email='qcgd002@edupilot.local'"); redis_del(f"ep:roster:{CG0}"); redis_del("ep:roster:*")
npii0 = sql(f"select count(*) from pii_events where course_id='{CG0}'")[0][0]
tag = str(time.time())[-6:]
c1, j1, _ = api("sv.kha", "POST", f"/courses/{CG0}/threads", {"title": f"QC106 P11 cụm {tag}", "body": "Em cần áp dụng nguyên tắc đặc quyền tối thiểu cho dịch vụ thuê ngoài trong bài tập ạ."})
c2, j2, _ = api("sv.kha", "POST", f"/courses/{CG0}/threads", {"title": f"QC106 P11 tên {tag}", "body": "Dung Nguyen hỏi về cách trình bày phần phân tích trong bài tập ạ."})
print(c1, lastj(j1, 200)); print(c2, lastj(j2, 250))
def stored(t): return sql(f"select p.body from forum_posts p join forum_threads th on th.id=p.thread_id where th.title='{t}' order by p.created_at limit 1")
s1 = stored(f"QC106 P11 cụm {tag}"); s2 = stored(f"QC106 P11 tên {tag}"); npii = sql(f"select count(*) from pii_events where course_id='{CG0}'")[0][0]
print("stored1", s1, "stored2", s2, "pii", npii0, "→", npii, sql(f"select channel, pii_type, count, action from pii_events where course_id='{CG0}' order by created_at desc limit 3"))
ok1 = c1 in (200, 201) and s1 and "[[SV_" not in s1[0][0] and "áp dụng nguyên tắc" in s1[0][0]
ok2 = (c2 in (200, 201) and s2 and "[[SV_" in s2[0][0] and "Dung Nguyen" not in s2[0][0]) or (c2 == 422)
chk("TC-P11-05", ok1 and ok2 and int(npii) - int(npii0) == 1, f"roster QD9G có «Dũng Nguyễn» (đổi tên qcgd002, xoá cache roster); bài 1 «cần áp dụng nguyên tắc đặc quyền tối thiểu…» → HTTP {c1}, chữ lưu: {s1[0][0][:60] if s1 else None!r}; bài 2 «Dung Nguyen hỏi…» → HTTP {c2} {lastj(j2,120) if c2 not in (200,201) else ''}, chữ lưu: {s2[0][0][:70] if s2 else None!r}; pii_events lớp: {npii0} → {npii} (kỳ vọng +1)")
