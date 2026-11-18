# dọn lượt IN_PROGRESS của SV test (nộp qua API) trước khi chạy script
from qlib import *
C1 = course("sv.kha")
rows = sql("select pa.id, u.email from practice_attempts pa join users u on u.id=pa.student_id where pa.status='IN_PROGRESS' and pa.course_id='%s'" % C1)
print(len(rows), "dở")
for aid, em in rows:
    u = em.split("@")[0]
    try: c, j, _ = api(u, "POST", f"/courses/{C1}/practice/attempts/{aid}/submit", {}); print(u, aid[:8], c)
    except Exception as e: print("err", u, e)
