import sys, os, math, random; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
SIM = f"/courses/{CG0}/threads/similar-confirmed"
T = json.load(open(W + "g08_threads.json"))["tids"]
def chunk_of(t): return sql(f"select c.id, regexp_replace(left(c.text,1500), '[[:space:]|]+', ' ', 'g') from knowledge_items k join forum_posts p on p.id=k.source_id join content_chunks c on c.id=k.chunk_id where p.thread_id='{t}'")[0]
CH = [chunk_of(t) for t in T]; q = CH[0][1]
emb0 = [float(x) for x in sql(f"select embedding::text from content_chunks where id='{CH[0][0]}'")[0][0].strip("[]").split(",")]
n0 = math.sqrt(sum(x * x for x in emb0)); u = [x / n0 for x in emb0]
random.seed(7); r = [random.gauss(0, 1) for _ in u]; d = sum(a * b for a, b in zip(r, u)); w = [a - d * b for a, b in zip(r, u)]; nw = math.sqrt(sum(x * x for x in w)); w = [x / nw for x in w]
def vec(c): return [c * a + math.sqrt(1 - c * c) * b for a, b in zip(u, w)]
def setv(cid, c): sql(f"update content_chunks set embedding='[{','.join(f'{x:.7f}' for x in vec(c))}]' where id='{cid}'")
def cos(cid): return float(sql(f"select 1 - (embedding <=> (select embedding from content_chunks where id='{CH[0][0]}')) from content_chunks where id='{cid}'")[0][0])
def sim(user="sv.kha"): return api(user, "POST", SIM, {"q": q})
def ids(j): return [x["thread_id"] for x in j]
flag("THREAD_SUGGEST", True)
# vector: thread0 cos 1 (tham chiếu), thread1 cos 1; còn lại 0,3
for i, c in enumerate([1, 1, 0.3, 0.3, 0.3]): setv(CH[i][0], c) if i else None
before = ids(sim()[1])
sql(f"update knowledge_items set source_type='TICKET' where chunk_id='{CH[1][0]}'"); sql(f"update content_chunks set text = text || ' ZQ-TK-91' where id='{CH[1][0]}'")
c, j, _ = api("sv.kha", "POST", SIM, {"q": q}); after = ids(j); gj = api("sv.kha", "POST", SIM, {"q": "ZQ-TK-91 trả lời của giảng viên về chuỗi cung ứng"})
leak = "ZQ-TK-91" in json.dumps(j, ensure_ascii=False) or "ZQ-TK-91" in json.dumps(gj[1], ensure_ascii=False)
chk("TC-G08-06", T[1] in before and T[1] not in after and not leak, f"mục kho của thread 1 chuyển source_type 'TICKET' (mô phỏng tri thức từ kênh riêng: dựng bằng SQL, không đi qua escalate thật) có chữ lạ «ZQ-TK-91», cos 1 với câu soạn: trước (POST) có thread 1 = {T[1] in before}; sau: gợi ý {[T.index(x) for x in after if x in T]} không có thread 1; chữ «ZQ-TK-91» trong phản hồi (kể cả soạn tiêu đề ZQ-TK-91) = {leak}; phản hồi DTO chỉ {{thread_id,title}}")
sql(f"update knowledge_items set source_type='POST' where chunk_id='{CH[1][0]}'")
# G08-13: mục lỗi thời (embedding NULL, như khi nội dung vừa đổi chờ nhúng lại) không được gợi ý; ẩn thread (staff) → biến mất ≤ 60 s
sql(f"update content_chunks set embedding=null where id='{CH[0][0]}'"); st = ids(sim()[1])
setv(CH[0][0], 1.0); v = int(sql(f"select version from forum_threads where id='{T[0]}'")[0][0]); pre = ids(sim()[1])
t0 = time.time(); hc = api("teacher", "POST", f"/courses/{CG0}/threads/{T[0]}/hide", {"reason": "QC106 ẩn", "version": v})[0]
gone = None
for _ in range(60):
    if T[0] not in ids(sim()[1]): gone = time.time() - t0; break
    time.sleep(1)
chk("TC-G08-13", T[0] not in st and T[0] in pre and gone is not None and gone <= 60, f"mục kho thread 0 đặt embedding NULL (trạng thái «chờ nhúng lại» sau khi nội dung đổi; mô phỏng bằng SQL — chưa đo chuỗi document.changed thật) → không gợi ý ({T[0] not in st}); khôi phục vector → có ({T[0] in pre}); ẩn thread bằng staff (hide {hc}) → biến mất khỏi gợi ý sau {gone if gone is None else round(gone, 2)} s (≤ 60)")
