import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
Q0 = "Cách phân biệt tấn công chuỗi cung ứng với tấn công trực tiếp là gì"; SIM = f"/courses/{CG0}/threads/similar-confirmed"; TH = f"/courses/{CG0}/threads"
flag("THREAD_SUGGEST", True); tag = str(int(time.time()))[-5:]; tids = []
for i in range(2):
    c, j, _ = api("sv.kha", "POST", TH, {"title": f"{Q0} QCG0813{tag} {i}?", "body": "Em muốn hiểu rõ hơn về chuỗi cung ứng phần mềm."}, idem=str(uuid.uuid4())); tids.append(j["thread"]["id"])
for t in tids:
    sql(f"insert into forum_posts (course_id, thread_id, kind, body, verification_state, citations, confidence) select '{CG0}', '{t}', 'AI', 'Tấn công chuỗi cung ứng nhắm vào nhà cung cấp phần mềm trung gian QCG0813 {tag}', 'PENDING', '[]', 0.7")
    p = sql(f"select id from forum_posts where thread_id='{t}' and kind='AI'")[0][0]; api("teacher", "POST", f"/courses/{CG0}/posts/{p}/verify", {})
time.sleep(35)
ch = [sql(f"select c.id, left(regexp_replace(c.text, '[[:space:]|]+', ' ', 'g'), 1500) from knowledge_items k join forum_posts p on p.id=k.source_id join content_chunks c on c.id=k.chunk_id where p.thread_id='{t}'")[0] for t in tids]
sql(f"update content_chunks set embedding=(select embedding from content_chunks where id='{ch[0][0]}') where id='{ch[1][0]}'"); q = ch[0][1]
def ids(): return [x["thread_id"] for x in api("sv.kha", "POST", SIM, {"q": q})[1]]
pre = ids(); print("trước:", [tids.index(x) for x in pre if x in tids])
t0 = time.time(); dc = api("sv.kha", "DELETE", f"{TH}/{tids[0]}", None)[0]
v = int(sql(f"select version from forum_threads where id='{tids[1]}'")[0][0]); hc = api("teacher", "POST", f"{TH}/{tids[1]}/hide", {"reason": "QC106 G08-13", "version": v})[0]
gone = {}
for _ in range(70):
    cur = ids()
    for i in (0, 1):
        if i not in gone and tids[i] not in cur: gone[i] = round(time.time() - t0, 2)
    if len(gone) == 2: break
    time.sleep(1)
items_left = sql(f"select count(*) from knowledge_items k join forum_posts p on p.id=k.source_id where p.thread_id in ('{tids[0]}','{tids[1]}')")[0][0]
chk("TC-G08-13", tids[0] in pre and tids[1] in pre and len(gone) == 2 and max(gone.values()) <= 60, f"luồng thật: 2 thread có mục kho POST (cos 1 với câu soạn; embedding thread 2 ép bằng SQL) đều có trong gợi ý; chủ bài xoá thread 1 (DELETE {dc}) và GV ẩn thread 2 (hide {hc}): biến khỏi gợi ý sau {gone} s (≤ 60 s); mục kho còn lại cho 2 thread: {items_left}; sự kiện document.changed → rag.BumpVersion/Remove (knowledge.go) đi outbox")
