import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
TH = f"/courses/{CG0}/threads"; flag("THREAD_SUGGEST", True)
tag = str(int(time.time()))[-5:]; base = "Cách phân biệt tấn công chuỗi cung ứng với tấn công trực tiếp là gì"; tids = []
for i in range(5):
    c, j, _ = api("sv.kha", "POST", TH, {"title": f"{base} QCG08{tag} biến thể {i}?", "body": f"Em muốn hiểu rõ hơn về {base.lower()} trong bài tập 03, nhờ thầy cô giải thích mục {i}."}, idem=str(uuid.uuid4()))
    if c == 201: tids.append(j["thread"]["id"])
json.dump({"tag": tag, "tids": tids}, open("/Users/kuro/.config/edupilot-team/qc3-work/g08_threads.json", "w"))
for t in tids:
    sql(f"insert into forum_posts (course_id, thread_id, kind, body, verification_state, citations, confidence) select '{CG0}', '{t}', 'AI', 'Tấn công chuỗi cung ứng nhắm vào nhà cung cấp phần mềm. QCG08 {tag}', 'PENDING', '[]', 0.7 where not exists (select 1 from forum_posts where thread_id='{t}' and kind='AI')")
    p = sql(f"select id from forum_posts where thread_id='{t}' and kind='AI'")[0][0]; api("teacher", "POST", f"/courses/{CG0}/posts/{p}/verify", {})
time.sleep(25); print(len(tids), "thread;", sql(f"select count(*) from knowledge_items k join content_chunks c on c.id=k.chunk_id where c.embedding is not null and k.course_id='{CG0}'")[0][0], "tri thức có embedding")
